package hid

import (
	"context"
)

// Device represents an opened HID device capable of report I/O.
type Device interface {
	Close() error
	WriteReport(context.Context, Report) error
	PollReports(context.Context) <-chan Report
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
	OpenVIDPID(vendorID, productID uint16) (Device, error)
}

// NewManager returns the OS-specific HID manager.
func NewManager() (Manager, error) {
	return newManager()
}
