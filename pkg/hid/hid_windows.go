//go:build windows

package hid

import (
	"context"
	"fmt"
	"log/slog"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	hid      = windows.NewLazySystemDLL("hid.dll")
	setupapi = windows.NewLazySystemDLL("setupapi.dll")

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

func (m *winManager) list() ([]Info, error) {
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

func (d *winDevice) WriteReport(_ context.Context, r Report) error {
	// PM5 expects reports of specific sizes (21, 63, or 121 bytes) based on frame length
	// The frame already includes padding to the right size from BuildCSAFEReport
	// So we send exactly len(data)+1 bytes (reportID + data)
	report := make([]byte, len(r.Data)+1)
	report[0] = r.ID
	copy(report[1:], r.Data)

	// Use WriteFile for interrupt OUT transfers
	var written uint32
	err := windows.WriteFile(d.handle, report, &written, nil)
	if err != nil {
		return fmt.Errorf("WriteFile failed: %v", err)
	}
	return nil
}

func (m *winManager) open(info Info) (Device, error) {
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

	//fmt.Printf("%+v", caps)
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
	devs, err := m.list()
	if err != nil {
		return nil, err
	}
	for _, d := range devs {
		if d.VendorID == vendorID && d.ProductID == productID {
			return m.open(d)
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

// Report represents an individual report. The Data field includes the complete buffer based on the device's
// descriptors.
type Report struct {
	ID   byte
	Data []byte
}

func (r Report) Bytes() []byte {
	b := make([]byte, len(r.Data)+1)
	b[0] = r.ID
	copy(b[1:], r.Data)
	return b
}

// PollReports starts a goroutine that constantly reads reports from the device and emits them to the returned channel.
func (d *winDevice) PollReports(ctx context.Context) <-chan Report {
	out := make(chan Report)

	go func() {
		<-ctx.Done()
		_ = d.Close()
	}()

	go func() {
		defer close(out)

		for {
			var read uint32
			buf := make([]byte, d.inputLen)
			err := windows.ReadFile(d.handle, buf, &read, nil)
			if err != nil {
				slog.Info("reading report failed", slog.Any("error", err))
				return
			}

			report := Report{
				ID:   buf[0],
				Data: append([]byte(nil), buf[1:]...),
			}

			out <- report
		}
	}()
	return out
}

func (d *winDevice) Close() error {
	return windows.CloseHandle(d.handle)
}
