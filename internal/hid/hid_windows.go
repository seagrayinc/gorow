//go:build windows

package hid

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows HID implementation using pure Go syscalls (no CGO)
// This directly calls SetupAPI, HID API, and file I/O functions

var (
	hid      = windows.NewLazySystemDLL("hid.dll")
	setupapi = windows.NewLazySystemDLL("setupapi.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procHidD_GetHidGuid                  = hid.NewProc("HidD_GetHidGuid")
	procHidD_GetAttributes               = hid.NewProc("HidD_GetAttributes")
	procHidD_GetProductString            = hid.NewProc("HidD_GetProductString")
	procHidD_GetManufacturerString       = hid.NewProc("HidD_GetManufacturerString")
	procHidD_GetSerialNumberString       = hid.NewProc("HidD_GetSerialNumberString")
	procHidD_GetPreparsedData            = hid.NewProc("HidD_GetPreparsedData")
	procHidD_FreePreparsedData           = hid.NewProc("HidD_FreePreparsedData")
	procHidP_GetCaps                     = hid.NewProc("HidP_GetCaps")
	procHidD_SetOutputReport             = hid.NewProc("HidD_SetOutputReport")
	procHidD_GetInputReport              = hid.NewProc("HidD_GetInputReport")
	procHidD_SetFeature                  = hid.NewProc("HidD_SetFeature")
	procHidD_GetFeature                  = hid.NewProc("HidD_GetFeature")
	procSetupDiGetClassDevsW             = setupapi.NewProc("SetupDiGetClassDevsW")
	procSetupDiEnumDeviceInterfaces      = setupapi.NewProc("SetupDiEnumDeviceInterfaces")
	procSetupDiGetDeviceInterfaceDetailW = setupapi.NewProc("SetupDiGetDeviceInterfaceDetailW")
	procSetupDiDestroyDeviceInfoList     = setupapi.NewProc("SetupDiDestroyDeviceInfoList")
)

const (
	DIGCF_PRESENT         = 0x00000002
	DIGCF_DEVICEINTERFACE = 0x00000010
	INVALID_HANDLE_VALUE  = ^uintptr(0)
)

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type HIDD_ATTRIBUTES struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

type SP_DEVICE_INTERFACE_DATA struct {
	CbSize             uint32
	InterfaceClassGuid GUID
	Flags              uint32
	Reserved           uintptr
}

type SP_DEVICE_INTERFACE_DETAIL_DATA struct {
	CbSize     uint32
	DevicePath [1]uint16 // Variable length
}

type HIDP_CAPS struct {
	Usage                     uint16
	UsagePage                 uint16
	InputReportByteLength     uint16
	OutputReportByteLength    uint16
	FeatureReportByteLength   uint16
	Reserved                  [17]uint16
	NumberLinkCollectionNodes uint16
	NumberInputButtonCaps     uint16
	NumberInputValueCaps      uint16
	NumberInputDataIndices    uint16
	NumberOutputButtonCaps    uint16
	NumberOutputValueCaps     uint16
	NumberOutputDataIndices   uint16
	NumberFeatureButtonCaps   uint16
	NumberFeatureValueCaps    uint16
	NumberFeatureDataIndices  uint16
}

type winManager struct{}

func newManager() (Manager, error) {
	return &winManager{}, nil
}

func (m *winManager) List() ([]Info, error) {
	var hidGuid GUID
	procHidD_GetHidGuid.Call(uintptr(unsafe.Pointer(&hidGuid)))

	devInfo, _, err := procSetupDiGetClassDevsW.Call(
		uintptr(unsafe.Pointer(&hidGuid)),
		0,
		0,
		DIGCF_PRESENT|DIGCF_DEVICEINTERFACE,
	)
	if devInfo == 0 || devInfo == INVALID_HANDLE_VALUE {
		return nil, fmt.Errorf("SetupDiGetClassDevsW failed: %v", err)
	}
	defer procSetupDiDestroyDeviceInfoList.Call(devInfo)

	var devices []Info
	var devInterfaceData SP_DEVICE_INTERFACE_DATA
	devInterfaceData.CbSize = uint32(unsafe.Sizeof(devInterfaceData))

	for i := uint32(0); ; i++ {
		r, _, _ := procSetupDiEnumDeviceInterfaces.Call(
			devInfo,
			0,
			uintptr(unsafe.Pointer(&hidGuid)),
			uintptr(i),
			uintptr(unsafe.Pointer(&devInterfaceData)),
		)
		if r == 0 {
			break
		}

		// Get required size
		var requiredSize uint32
		procSetupDiGetDeviceInterfaceDetailW.Call(
			devInfo,
			uintptr(unsafe.Pointer(&devInterfaceData)),
			0,
			0,
			uintptr(unsafe.Pointer(&requiredSize)),
			0,
		)

		// Allocate detail buffer
		detailData := make([]byte, requiredSize)
		detail := (*SP_DEVICE_INTERFACE_DETAIL_DATA)(unsafe.Pointer(&detailData[0]))
		// CbSize must be sizeof(SP_DEVICE_INTERFACE_DETAIL_DATA) which is different on 32/64 bit
		// On 64-bit Windows, it's 8 bytes (4-byte uint32 + 4-byte padding + variable data)
		// On 32-bit Windows, it's 5 bytes (4-byte uint32 + 1 uint16 + variable data)
		if unsafe.Sizeof(uintptr(0)) == 8 {
			detail.CbSize = 8
		} else {
			detail.CbSize = 6 // Sizeof struct with 1 WCHAR (2 bytes) + DWORD (4 bytes) with padding
		}

		r, _, err = procSetupDiGetDeviceInterfaceDetailW.Call(
			devInfo,
			uintptr(unsafe.Pointer(&devInterfaceData)),
			uintptr(unsafe.Pointer(detail)),
			uintptr(requiredSize),
			0,
			0,
		)
		if r == 0 {
			continue
		}

		// Convert device path
		pathPtr := &detail.DevicePath[0]
		path := windows.UTF16PtrToString(pathPtr)

		// Open device to get attributes
		h, err := windows.CreateFile(
			pathPtr,
			0, // No access needed for attributes
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
			nil,
			windows.OPEN_EXISTING,
			0,
			0,
		)
		if err != nil {
			continue
		}

		var attrs HIDD_ATTRIBUTES
		attrs.Size = uint32(unsafe.Sizeof(attrs))
		r, _, _ = procHidD_GetAttributes.Call(uintptr(h), uintptr(unsafe.Pointer(&attrs)))

		var manufacturer, product string
		if r != 0 {
			mfr := make([]uint16, 256)
			procHidD_GetManufacturerString.Call(uintptr(h), uintptr(unsafe.Pointer(&mfr[0])), uintptr(len(mfr)*2))
			manufacturer = windows.UTF16ToString(mfr)

			prod := make([]uint16, 256)
			procHidD_GetProductString.Call(uintptr(h), uintptr(unsafe.Pointer(&prod[0])), uintptr(len(prod)*2))
			product = windows.UTF16ToString(prod)
		}

		windows.CloseHandle(h)

		if r != 0 {
			devices = append(devices, Info{
				Path:         path,
				VendorID:     attrs.VendorID,
				ProductID:    attrs.ProductID,
				Manufacturer: manufacturer,
				Product:      product,
			})
		}
	}

	return devices, nil
}

func (m *winManager) Open(info Info) (Device, error) {
	pathPtr, err := windows.UTF16PtrFromString(info.Path)
	if err != nil {
		return nil, err
	}

	h, err := windows.CreateFile(
		pathPtr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_EXISTING,
		0, // Synchronous I/O
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateFile failed: %v", err)
	}

	// Get capabilities
	var preparsedData uintptr
	r, _, _ := procHidD_GetPreparsedData.Call(uintptr(h), uintptr(unsafe.Pointer(&preparsedData)))
	if r == 0 {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("HidD_GetPreparsedData failed")
	}

	var caps HIDP_CAPS
	r, _, _ = procHidP_GetCaps.Call(preparsedData, uintptr(unsafe.Pointer(&caps)))
	procHidD_FreePreparsedData.Call(preparsedData)

	const HIDP_STATUS_SUCCESS = 0x00110000
	if r != HIDP_STATUS_SUCCESS {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("HidP_GetCaps failed: 0x%X", r)
	}

	return &winDevice{
		handle:     h,
		path:       info.Path,
		inputLen:   int(caps.InputReportByteLength),
		outputLen:  int(caps.OutputReportByteLength),
		featureLen: int(caps.FeatureReportByteLength),
	}, nil
}

func (m *winManager) OpenVIDPID(vendorID, productID uint16) (Device, error) {
	devs, err := m.List()
	if err != nil {
		return nil, err
	}
	for _, d := range devs {
		if d.VendorID == vendorID && d.ProductID == productID {
			return m.Open(d)
		}
	}
	return nil, fmt.Errorf("device not found (VID:0x%04X PID:0x%04X)", vendorID, productID)
}

type winDevice struct {
	handle     windows.Handle
	path       string
	inputLen   int
	outputLen  int
	featureLen int
}

func (d *winDevice) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	rid := p[0]
	data := p[1:]
	if err := d.WriteOutput(rid, data); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (d *winDevice) Read(p []byte) (int, error) {
	// Windows HID ReadFile requires buffer to match the device's input report length
	// Use the actual report length from the device descriptor
	var read uint32
	buf := make([]byte, d.outputLen)
	err := windows.ReadFile(d.handle, buf, &read, nil)
	if err != nil {
		return 0, fmt.Errorf("ReadFile failed: %v", err)
	}

	copy(p, buf)
	return len(p), nil
}

func (d *winDevice) WriteOutput(reportID byte, data []byte) error {
	// PM5 expects reports of specific sizes (21, 63, or 121 bytes) based on frame length
	// The frame already includes padding to the right size from BuildCSAFEReport
	// So we send exactly len(data)+1 bytes (reportID + data)
	report := make([]byte, len(data)+1)
	report[0] = reportID
	copy(report[1:], data)

	// Use WriteFile for interrupt OUT transfers
	var written uint32
	err := windows.WriteFile(d.handle, report, &written, nil)
	if err != nil {
		return fmt.Errorf("WriteFile failed: %v", err)
	}
	return nil
}

func (d *winDevice) ReadInput() ([]byte, error) {
	// Windows HID ReadFile requires buffer to match the device's input report length
	// Use the actual report length from the device descriptor
	report := make([]byte, d.inputLen)
	var read uint32
	err := windows.ReadFile(d.handle, report, &read, nil)
	if err != nil {
		return nil, fmt.Errorf("ReadFile failed: %v", err)
	}
	// Report includes report ID at byte 0
	// Return only the actual data received, excluding the report ID
	if read > 1 {
		return report[1:read], nil
	}
	return nil, nil
}

func (d *winDevice) WriteFeature(reportID byte, data []byte) error {
	report := make([]byte, d.featureLen)
	report[0] = reportID
	copy(report[1:], data)

	r, _, err := procHidD_SetFeature.Call(
		uintptr(d.handle),
		uintptr(unsafe.Pointer(&report[0])),
		uintptr(len(report)),
	)
	if r == 0 {
		return fmt.Errorf("HidD_SetFeature failed: %v", err)
	}
	return nil
}

func (d *winDevice) ReadFeature(reportID byte) ([]byte, error) {
	report := make([]byte, d.featureLen)
	report[0] = reportID

	r, _, err := procHidD_GetFeature.Call(
		uintptr(d.handle),
		uintptr(unsafe.Pointer(&report[0])),
		uintptr(len(report)),
	)
	if r == 0 {
		return nil, fmt.Errorf("HidD_GetFeature failed: %v", err)
	}
	return report[1:], nil
}

func (d *winDevice) ReportLens() (int, int, int) {
	return d.inputLen, d.outputLen, d.featureLen
}

func (d *winDevice) Close() error {
	return windows.CloseHandle(d.handle)
}
