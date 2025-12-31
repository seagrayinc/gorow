package csafe

// Minimal CSAFE codec stubs.
import "fmt"

const (
	StartByte = 0xF0
	StopByte  = 0xF2
)

// Frame represents a CSAFE message.
type Frame struct {
	Cmds []byte // payload commands and data
}

// sum8 computes the 8-bit sum of bytes.
func sum8(b []byte) byte {
	var s uint16
	for _, v := range b {
		s += uint16(v)
	}
	return byte(s & 0xFF)
}

// lrc computes the CSAFE checksum (two's complement such that len+payload+chk == 0 mod 256).
func lrc(length byte, payload []byte) byte {
	s := uint16(length) + uint16(sum8(payload))
	return byte((^byte(s & 0xFF)) + 1) // two's complement
}

// Build encodes a CSAFE frame: [F0][LEN][payload...][CHK][F2]
func Build(payload []byte) []byte {
	ln := byte(len(payload))
	buf := make([]byte, 0, len(payload)+4)
	buf = append(buf, StartByte)
	buf = append(buf, ln)
	buf = append(buf, payload...)
	buf = append(buf, lrc(ln, payload))
	buf = append(buf, StopByte)
	return buf
}

// Parse validates a CSAFE frame and returns payload.
func Parse(frame []byte) ([]byte, error) {
	// Expect [F0][LEN][payload...][CHK][F2]
	if len(frame) < 4 {
		return nil, ErrShortFrame
	}
	if frame[0] != StartByte {
		return nil, ErrBadStart
	}
	if frame[len(frame)-1] != StopByte {
		return nil, ErrBadStop
	}
	ln := int(frame[1])
	// Compute expected minimal size and tolerate extra trailing bytes before Stop only if they are trimmed upstream
	if 2+ln+2 != len(frame) {
		// strict length check; caller should slice exact frame
		return nil, ErrBadLength
	}
	payload := frame[2 : 2+ln]
	chk := frame[2+ln]
	// Validate two's complement: (len + sum(payload) + chk) mod 256 == 0
	total := uint16(frame[1]) + uint16(sum8(payload)) + uint16(chk)
	if byte(total&0xFF) != 0x00 {
		return nil, ErrBadChecksum
	}
	return payload, nil
}

// Some basic commands (IDs are examples; confirm against CSAFE spec)
const (
	CmdGetID     = 0x91 // CSAFE_GETID
	CmdGetStatus = 0x80 // CSAFE_GETSTATUS
)

// Errors
var (
	ErrShortFrame  = fmt.Errorf("csafe: short frame")
	ErrBadStart    = fmt.Errorf("csafe: bad start byte")
	ErrBadStop     = fmt.Errorf("csafe: bad stop byte")
	ErrBadLength   = fmt.Errorf("csafe: bad length")
	ErrBadChecksum = fmt.Errorf("csafe: bad checksum")
)
