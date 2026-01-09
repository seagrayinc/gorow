// Package pm5 implements the PM5 CSAFE Protocol defined at
// https://www.concept2.sg/files/pdf/us/monitors/PM5_CSAFECommunicationDefinition.pdf

package pm5

const (
	Concept2VID uint16 = 0x17A4
	PM5PID      uint16 = 0x001E
)

var (
	ReportLengths = map[byte]int{
		0x01: 21,
		0x02: 121,
		0x04: 501,
	}
)
