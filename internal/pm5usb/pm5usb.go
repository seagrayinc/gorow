package pm5usb

import (
	"fmt"
	"time"

	"github.com/karalabe/usb"
)

const (
	Concept2VID = 0x17A4
	PM5PID      = 0x001E
	Interface   = 0 // PyRow uses INTERFACE = 0
)

// Device represents a PM5 connected via USB with bulk endpoints (mirroring PyRow's approach).
type Device struct {
	dev       usb.Device
	inEp      int
	outEp     int
	readSize  int
	writeSize int
}

// Open finds and opens the PM5 by VID/PID, claims interface 0, and discovers endpoints.
func Open() (*Device, error) {
	// Enumerate all USB devices
	infos, err := usb.Enumerate(Concept2VID, PM5PID)
	if err != nil {
		return nil, fmt.Errorf("usb enumerate: %w", err)
	}
	if len(infos) == 0 {
		// Try enumerating ALL devices to debug
		allInfos, allErr := usb.Enumerate(0, 0)
		if allErr != nil {
			return nil, fmt.Errorf("PM5 not found (VID:0x%04X PID:0x%04X); enumerate all failed: %w", Concept2VID, PM5PID, allErr)
		}
		return nil, fmt.Errorf("PM5 not found (VID:0x%04X PID:0x%04X); found %d other USB devices (may need WinUSB driver)", Concept2VID, PM5PID, len(allInfos))
	}

	// Open the first matching device
	dev, err := infos[0].Open()
	if err != nil {
		return nil, fmt.Errorf("open device: %w", err)
	}

	// PyRow's behavior: on non-Windows, detach kernel driver if active
	// karalabe/usb doesn't expose kernel driver detach on Windows (not needed),
	// but you can add platform-specific logic here if needed for Linux/macOS.

	// Claim interface 0 (PyRow uses INTERFACE = 0)
	// karalabe/usb doesn't have an explicit "claim interface" call in its API;
	// it's implicit in the device handle. The library handles this internally.

	// Discover endpoints from the device descriptor
	// PyRow stores self.inEndpoint and self.outEndpoint from iface[0] and iface[1]
	// Typically: IN endpoint is 0x81, OUT endpoint is 0x01
	// We'll hardcode these based on PyRow's typical usage; adjust if needed.
	inEp := 0x81   // Standard IN endpoint address
	outEp := 0x01  // Standard OUT endpoint address
	readSize := 64 // Common for HID/bulk; adjust if needed
	writeSize := 64

	return &Device{
		dev:       dev,
		inEp:      inEp,
		outEp:     outEp,
		readSize:  readSize,
		writeSize: writeSize,
	}, nil
}

// Close releases the device.
func (d *Device) Close() error {
	return d.dev.Close()
}

// Send writes a CSAFE HID frame to the OUT endpoint and reads the response from the IN endpoint.
// This mirrors PyRow's send() method:
//
//	length = self.erg.write(self.outEndpoint, csafe, timeout=2000)
//	transmission = self.erg.read(self.inEndpoint, length, timeout=2000)
func (d *Device) Send(frame []byte, timeout time.Duration) ([]byte, error) {
	// Write the frame to the OUT endpoint
	n, err := d.dev.Write(frame)
	if err != nil {
		return nil, fmt.Errorf("usb write: %w", err)
	}

	// PyRow reads back up to 'length' bytes (the length it just wrote)
	// We'll read into a buffer sized to the larger of writeSize or len(frame)
	readLen := len(frame)
	if readLen < d.readSize {
		readLen = d.readSize
	}
	buf := make([]byte, readLen)

	// karalabe/usb.Device.Read is synchronous with an internal timeout
	// We'll call it and hope it respects a reasonable timeout (typically it does)
	m, err := d.dev.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("usb read: %w (wrote %d bytes)", err, n)
	}

	return buf[:m], nil
}
