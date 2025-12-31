//go:build !windows

package hid

import (
	usbhid "rafaelmartins.com/p/usbhid"
)

type usbManager struct{}

func newManager() (Manager, error) { return &usbManager{}, nil }

func (m *usbManager) List() ([]Info, error) {
	devs, err := usbhid.Enumerate(nil)
	if err != nil {
		return nil, err
	}
	out := make([]Info, 0, len(devs))
	for _, d := range devs {
		out = append(out, Info{
			Path:         d.Path(),
			VendorID:     d.VendorId(),
			ProductID:    d.ProductId(),
			Product:      d.Product(),
			Manufacturer: d.Manufacturer(),
		})
	}
	return out, nil
}

type usbDevice struct{ d *usbhid.Device }

func (m *usbManager) Open(info Info) (Device, error) {
	d, err := usbhid.Get(func(dev *usbhid.Device) bool {
		return dev.Path() == info.Path
	}, true, false)
	if err != nil {
		return nil, err
	}
	return &usbDevice{d}, nil
}

func (m *usbManager) OpenVIDPID(vendorID, productID uint16) (Device, error) {
	d, err := usbhid.Get(func(dev *usbhid.Device) bool {
		return dev.VendorId() == vendorID && dev.ProductId() == productID
	}, true, false)
	if err != nil {
		return nil, err
	}
	return &usbDevice{d}, nil
}

func (d *usbDevice) Write(p []byte) (int, error) {
	// p should include report ID at p[0]; extract rid and data
	if len(p) == 0 {
		return 0, nil
	}
	rid := p[0]
	data := p[1:]
	if err := d.d.SetOutputReport(rid, data); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (d *usbDevice) Read(p []byte) (int, error) {
	_, buf, err := d.d.GetInputReport()
	if err != nil {
		return 0, err
	}
	n := copy(p, buf)
	return n, nil
}

// Advanced
func (d *usbDevice) WriteOutput(reportID byte, data []byte) error {
	return d.d.SetOutputReport(reportID, data)
}
func (d *usbDevice) ReadInput() ([]byte, error) {
	_, buf, err := d.d.GetInputReport()
	return buf, err
}
func (d *usbDevice) WriteFeature(reportID byte, data []byte) error {
	return d.d.SetFeatureReport(reportID, data)
}
func (d *usbDevice) ReadFeature(reportID byte) ([]byte, error) { return d.d.GetFeatureReport(reportID) }
func (d *usbDevice) ReportLens() (int, int, int) {
	return int(d.d.GetInputReportLength()), int(d.d.GetOutputReportLength()), int(d.d.GetFeatureReportLength())
}

func (d *usbDevice) Close() error { return d.d.Close() }
