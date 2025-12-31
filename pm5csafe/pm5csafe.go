package pm5csafe

import (
	"errors"
	"time"
)

const (
	// From Table 5 - Unique Frame Flags
	ExtendedFrameStartFlag = 0xF0
	StandardFrameStartFlag = 0xF1
	StopFrameFlag          = 0xF2
	ByteStuffingFlag       = 0xF3

	// From Table 9 – Response Status Byte Bit-Mapping
	FrameToggleBitMask = 0x80 // Toggles between 0 and 1 on alternate frames

	PreviousFrameStatusBitMask = 0x30
	FrameStatusOK              = 0x00
	FrameStatusReject          = 0x10
	FrameStatusBad             = 0x20
	FrameStatusNotReady        = 0x30

	StateMachineStateBitMask = 0x0F
	MachineStateError        = 0x00
	MachineStateReady        = 0x01
	MachineStateIdle         = 0x02
	MachineStateHaveID       = 0x03
	MachineStateInUse        = 0x05
	MachineStatePause        = 0x06
	MachineStateFinish       = 0x07
	MachineStateManual       = 0x08
	MachineStateOffline      = 0x09

	// From Table 10 - CSAFE Concept2 PM Information
	CSAFEManufacturerID  = 22
	CSAFEClassIdentifier = 2
	CSAFEModelPM3        = 3
	CSAFEModelPM4        = 4
	CSAFEModelPM5        = 5
	MaxFrameLength       = 120 // bytes
	MinimumInterframeGap = 50 * time.Millisecond

	// Commands
	CSAFE_SETUSERCFG1_CMD = 0x1A
	CSAFE_SETPMCFG_CMD    = 0x76
	CSAFE_SETPMDATA_CMD   = 0x77
	CSAFE_GETPMCFG_CMD    = 0x7E
	CSAFE_GETPMDATA_CMD   = 0x7F
	CSAFE_GETID_CMD       = 0x92
)

var (
	// From Table 6 - Byte Stuffing Values
	ByteStuffingValues = map[byte][]byte{
		ExtendedFrameStartFlag: {ByteStuffingFlag, 0x00},
		StandardFrameStartFlag: {ByteStuffingFlag, 0x01},
		StopFrameFlag:          {ByteStuffingFlag, 0x02},
		ByteStuffingFlag:       {ByteStuffingFlag, 0x03},
	}
)

type StandardFrame struct {
	StartFlag     []byte
	FrameContents []byte
	Checksum      []byte
	StopFlag      []byte
}

type ExtendedFrame struct {
	ExtendedStartFlag  []byte
	DestinationAddress []byte
	SourceAddress      []byte
	FrameContents      []byte
	Checksum           []byte
	StopFlag           []byte
}

func Checksum(bytes []byte) byte {
	var checksum byte
	for _, b := range bytes {
		checksum ^= b
	}

	return checksum
}

type LongCommand struct {
	LongCommand   byte
	DataByteCount byte
	Data          []byte
}

type ShortCommand struct {
	ShortCommand byte
}

func (sc ShortCommand) Bytes() []byte {
	return []byte{sc.ShortCommand}
}

type ResponseFrame struct {
	Status              byte
	CommandResponseData []byte
}

func (rf ResponseFrame) FrameToggle() byte {
	return rf.Status & FrameToggleBitMask
}

func (rf ResponseFrame) PreviousFrameStatus() byte {
	return rf.Status & PreviousFrameStatusBitMask
}

func (rf ResponseFrame) StateMachineState() byte {
	return rf.Status & StateMachineStateBitMask
}

func (rf ResponseFrame) CommandResponses() ([]IndividualCommandResponse, error) {
	var cmdIdx int
	var responses []IndividualCommandResponse

	for {
		dataByteCount := rf.CommandResponseData[cmdIdx+1]
		resp := IndividualCommandResponse{
			Command:       rf.CommandResponseData[cmdIdx],
			DataByteCount: dataByteCount,
			Data:          make([]byte, dataByteCount),
		}

		copy(resp.Data, rf.CommandResponseData[cmdIdx+2:cmdIdx+2+int(dataByteCount)])

		responses = append(responses, resp)
		if cmdIdx > len(rf.CommandResponseData) {
			break
		}
	}

	return responses, nil
}

type IndividualCommandResponse struct {
	Command       byte
	DataByteCount byte
	Data          []byte
}

type PMExtension struct {
}

func NewStandardFrame(command ShortCommand) []byte {
	frame := []byte{StandardFrameStartFlag}
	frame = append(frame, command.Bytes()...)
	frame = append(frame, Checksum(command.Bytes()))
	frame = append(frame, StopFrameFlag)
	return frame
}

// NewHIDReport converts the byte to an HID message and selects the appropriate byte size given the payload.
func NewHIDReport(b []byte) []byte {
	// hard-code to use report ID 2 for now
	report := make([]byte, 121)
	report[0] = 0x02
	copy(report[1:], b)
	return report
}

func CSAFEGetID() ShortCommand {
	return ShortCommand{
		ShortCommand: CSAFE_GETID_CMD,
	}
}

func findFrameEnd(frame []byte) (int, error) {
	for i, b := range frame {
		if b == StopFrameFlag {
			return i, nil
		}
	}

	return 0, errors.New("could not find frame end")
}

func ParseHIDResponse(b []byte) (ResponseFrame, error) {
	// skip first byte, it's the HID report id
	if b[1] != StandardFrameStartFlag {
		return ResponseFrame{}, errors.New("could not find frame start")
	}

	frameStop, err := findFrameEnd(b)
	if err != nil {
		return ResponseFrame{}, err
	}

	return ResponseFrame{
		Status:              b[2],
		CommandResponseData: b[3:frameStop],
	}, nil
}
