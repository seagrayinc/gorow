package csafe

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/seagrayinc/pm5-csafe/pkg/hid"
)

const (
	// From Table 4 - Extended Frame Addressing
	ExtendedFrameAddressPCHostPrimary    = 0x00
	ExtendedFrameAddressDefaultSecondary = 0xFD
	ExtendedFrameAddressBroadcast        = 0xFF

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
)

type Transport struct {
	Device hid.Device
}

func (t Transport) Close() error {
	return t.Device.Close()
}

func EncodeReportToString(b []byte) string {
	hexDigits := hex.EncodeToString(b)
	var builder strings.Builder
	for i, r := range hexDigits {
		switch {
		case i > 0 && i%2 == 0:
			builder.WriteString("-")
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func Send[T any](_ context.Context, t Transport, cmd Command[T]) (T, ExtendedResponseFrame, error) {
	var zero T

	report := hidReport1(extendedFrame(cmd.Marshall()))
	fmt.Println(EncodeReportToString(report))
	_, err := t.Device.Write(report)
	if err != nil {
		return zero, ExtendedResponseFrame{}, fmt.Errorf("send failed: %w", err)
	}

	resp := make([]byte, len(report))
	_, err = t.Device.Read(resp)

	fmt.Println(EncodeReportToString(resp))
	frame, err := ParseExtendedHIDResponse(resp)
	if err != nil {
		return zero, ExtendedResponseFrame{}, fmt.Errorf("failed to parse extended frame: %w", err)
	}

	responses := frame.CommandResponses()
	var commandResponse T
	switch len(responses) {
	case 0:
		// If there's no command data to parse, we just unmarshall an empty byte array. This is really only necessary
		// because of how CSAFE_GETSTATUS_CMD is implemented on the PM (it returns just the status and no command data).
		commandResponse, err = cmd.Unmarshall([]byte{})
	default:
		commandResponse, err = cmd.Unmarshall(responses[0].Data)
	}

	return commandResponse, frame, err
}

type ExtendedFrame struct {
	ExtendedStartFlag  []byte
	DestinationAddress byte
	SourceAddress      byte
	FrameContents      []byte
	Checksum           byte
	StopFlag           byte
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

func NewLongCommand(command byte, data []byte) LongCommand {
	return LongCommand{
		LongCommand:   command,
		DataByteCount: byte(len(data)),
		Data:          data,
	}
}

func (lc LongCommand) Bytes() []byte {
	b := []byte{lc.LongCommand, lc.DataByteCount}
	b = append(b, lc.Data...)
	return b
}

type ProprietaryCommand interface {
	Bytes() []byte
}

func GetPMDataCommand(p ProprietaryCommand) []byte {
	cBytes := p.Bytes()
	b := []byte{0x1A, byte(len(cBytes))}
	b = append(b, cBytes...)
	return b
}

type ShortCommand struct {
	ShortCommand byte
}

func (sc ShortCommand) Bytes() []byte {
	return []byte{sc.ShortCommand}
}

type ExtendedResponseFrame struct {
	Status              byte
	DestinationAddress  byte
	SourceAddress       byte
	checksum            byte
	commandResponseData []byte
	parsedResponses     []IndividualCommandResponse
}

func (rf *ExtendedResponseFrame) FrameToggle() byte {
	return rf.Status & FrameToggleBitMask
}

func (rf *ExtendedResponseFrame) PreviousFrameStatus() byte {
	return rf.Status & PreviousFrameStatusBitMask
}

func (rf *ExtendedResponseFrame) StateMachineState() byte {
	return rf.Status & StateMachineStateBitMask
}

func (rf *ExtendedResponseFrame) CommandResponses() []IndividualCommandResponse {
	return rf.parsedResponses
}

func (rf *ExtendedResponseFrame) parseResponses() error {
	var cmdIdx int
	var responses []IndividualCommandResponse

	for {
		dataByteCount := rf.commandResponseData[cmdIdx+1]
		resp := IndividualCommandResponse{
			Command:       rf.commandResponseData[cmdIdx],
			DataByteCount: dataByteCount,
			Data:          make([]byte, dataByteCount),
		}

		dataStart := cmdIdx + 1 + 1 // starts after the command and data size
		dataEnd := dataStart + int(dataByteCount)

		copy(resp.Data, rf.commandResponseData[dataStart:dataEnd])
		responses = append(responses, resp)
		cmdIdx = dataEnd + 1
		if cmdIdx > len(rf.commandResponseData) {
			break
		}
	}

	rf.parsedResponses = responses
	return nil
}

type IndividualCommandResponse struct {
	Command       byte
	DataByteCount byte
	Data          []byte
}

func extendedFrame(b []byte) []byte {
	var frameContents []byte
	frameContents = append(frameContents, ExtendedFrameAddressDefaultSecondary)
	frameContents = append(frameContents, ExtendedFrameAddressPCHostPrimary)
	frameContents = append(frameContents, b...)
	frameContents = append(frameContents, Checksum(b))

	frame := []byte{ExtendedFrameStartFlag}
	frame = append(frame, byteStuff(frameContents)...)
	frame = append(frame, StopFrameFlag)
	return frame
}

func hidReport1(b []byte) []byte {
	report := make([]byte, 21)
	report[0] = 0x01
	copy(report[1:], b)
	return report
}

func hidReport2(b []byte) []byte {
	report := make([]byte, 121)
	report[0] = 0x02
	copy(report[1:], b)
	return report
}

func hidReport4(b []byte) []byte {
	report := make([]byte, 501)
	report[0] = 0x04
	copy(report[1:], b)
	return report
}

type Command[T any] interface {
	Marshall() []byte
	Unmarshall([]byte) (T, error)
}

func findFrameEnd(frame []byte) (int, error) {
	for i, b := range frame {
		if b == StopFrameFlag {
			return i, nil
		}
	}

	return 0, errors.New("could not find frame end")
}

func byteStuff(input []byte) []byte {
	out := make([]byte, 0, len(input))

	for _, b := range input {
		switch b {
		case 0xF0:
			out = append(out, ByteStuffingFlag, 0x00)
		case 0xF1:
			out = append(out, ByteStuffingFlag, 0x01)
		case 0xF2:
			out = append(out, ByteStuffingFlag, 0x02)
		case 0xF3:
			out = append(out, ByteStuffingFlag, 0x03)
		default:
			out = append(out, b)
		}
	}

	return out
}

func byteUnstuff(input []byte) ([]byte, error) {
	out := make([]byte, 0, len(input))

	for i := 0; i < len(input); i++ {
		b := input[i]

		if b != ByteStuffingFlag {
			out = append(out, b)
			continue
		}

		// Escape byte must be followed by a stuffing value
		if i+1 >= len(input) {
			return nil, fmt.Errorf("truncated escape sequence")
		}

		i++
		switch input[i] {
		case 0x00:
			out = append(out, 0xF0)
		case 0x01:
			out = append(out, 0xF1)
		case 0x02:
			out = append(out, 0xF2)
		case 0x03:
			out = append(out, 0xF3)
		default:
			return nil, fmt.Errorf("invalid escape value 0x%02X", input[i])
		}
	}

	return out, nil
}

func ParseExtendedHIDResponse(b []byte) (ExtendedResponseFrame, error) {
	// skip first byte, it's the HID report id
	if b[1] != ExtendedFrameStartFlag {
		return ExtendedResponseFrame{}, errors.New("could not find frame start")
	}

	frameStop, err := findFrameEnd(b)
	if err != nil {
		return ExtendedResponseFrame{}, err
	}

	unstuffed, err := byteUnstuff(b[2:frameStop])
	if err != nil {
		return ExtendedResponseFrame{}, fmt.Errorf("byte unstuffing failed: %w", err)
	}

	frame := ExtendedResponseFrame{
		DestinationAddress: unstuffed[0],
		SourceAddress:      unstuffed[1],
		Status:             unstuffed[2],
		checksum:           unstuffed[len(unstuffed)-1],
	}

	// checksum excludes address information, but includes status and command response data
	if frame.checksum != Checksum(unstuffed[2:len(unstuffed)-1]) {
		return ExtendedResponseFrame{}, fmt.Errorf("checksum mismatch")
	}

	// it's possible there was no command response data (e.g. CSAFE_GETSTATUS_CMD).
	if len(unstuffed) <= 4 {
		return frame, nil
	}

	frame.commandResponseData = unstuffed[3 : len(unstuffed)-2]
	if err := frame.parseResponses(); err != nil {
		return ExtendedResponseFrame{}, errors.New("failed to parse command responses in frame")
	}

	return frame, nil
}
