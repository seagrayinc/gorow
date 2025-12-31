package hid

// Device represents an opened HID device capable of report I/O.
type Device interface {
	Write([]byte) (int, error) // send output report
	Read([]byte) (int, error)  // read input report
	Close() error
}

// Advanced exposes report-specific operations and lengths when available.
// Implementations may choose to support only a subset.
type Advanced interface {
	WriteOutput(reportID byte, data []byte) error
	ReadInput() ([]byte, error)
	WriteFeature(reportID byte, data []byte) error
	ReadFeature(reportID byte) ([]byte, error)
	ReportLens() (inLen, outLen, featLen int)
}

// Info represents a HID device descriptor.
type Info struct {
	Path         string
	VendorID     uint16
	ProductID    uint16
	Product      string
	Manufacturer string
}

// Manager enumerates and opens HID devices.
type Manager interface {
	List() ([]Info, error)
	Open(info Info) (Device, error)
	OpenVIDPID(vendorID, productID uint16) (Device, error)
}

// NewManager returns the OS-specific HID manager.
func NewManager() (Manager, error) {
	return newManager()
}
