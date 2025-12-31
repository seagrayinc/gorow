package pm5

import (
	"bytes"
)

// VID/PID for Concept2 PM
const (
	Concept2VID uint16 = 0x17A4
	PM5PID      uint16 = 0x001E
)

// BuildGetID returns a CSAFE GETID command frame.
func BuildGetID() []byte {
	payload := []byte{0x92} // CSAFE_GETID_CMD per provided map
	return BuildCSAFEReport(payload)
}

// BuildGetID returns a CSAFE GETID command frame.
func BuildGetForcePlotData() []byte {
	payload := []byte{0xA6} // CSAFE_GETID_CMD per provided map
	return BuildCSAFEReport(payload)
}

// BuildGetStatus returns a CSAFE GETSTATUS command frame.
func BuildGetStatus() []byte {
	payload := []byte{0x80} // CSAFE_GETSTATUS_CMD
	return BuildCSAFEReport(payload)
}

// BuildCSAFEReport builds a full HID report with CSAFE framing using XOR checksum and byte stuffing, per Python reference.
// It selects a report ID and pads to the appropriate size (21, 63, or 121) based on message length.
func BuildCSAFEReport(message []byte) []byte {
	// Build CSAFE frame payload: start flag + message (with lengths for long commands handled by caller) + checksum + stop flag
	// Byte-stuff 0xF0..0xF3 using 0xF3 and low 2 bits of original
	// Compute XOR checksum over unstuffed message bytes
	// Use Standard Frame Start Flag 0xF1
	const (
		StandardStart = 0xF1
		StopFlag      = 0xF2
		StuffFlag     = 0xF3
	)

	// Compute XOR checksum over message bytes
	var checksum byte
	for _, b := range message {
		checksum ^= b
	}

	// Byte stuffing
	var stuffed bytes.Buffer
	for _, b := range message {
		if b >= 0xF0 && b <= 0xF3 {
			stuffed.WriteByte(StuffFlag)
			stuffed.WriteByte(b & 0x03)
		} else {
			stuffed.WriteByte(b)
		}
	}

	// Append checksum (note: checksum calculated before stuffing per reference)
	stuffed.WriteByte(checksum)

	// Build frame: Start, payload..., Stop
	frame := make([]byte, 0, stuffed.Len()+2)
	frame = append(frame, StandardStart)
	frame = append(frame, stuffed.Bytes()...)
	frame = append(frame, StopFlag)

	// Wrap in HID report: choose report ID and pad
	// Sizes include report ID
	msgLen := len(frame) + 1
	var rid byte
	var total int
	switch {
	case msgLen <= 21:
		rid = 0x01
		total = 21
	case msgLen <= 63:
		rid = 0x04
		total = 63
	case msgLen <= 121:
		rid = 0x02
		total = 121
	default:
		// too long; return empty to signal error
		return nil
	}
	report := make([]byte, total)
	report[0] = rid
	copy(report[1:], frame)
	// zero padding already present in allocated slice
	return report
}
