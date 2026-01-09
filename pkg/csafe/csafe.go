package csafe

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
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
	Device        hid.Device
	ReportLengths map[byte]int
}

func (t *Transport) Close() error {
	return t.Device.Close()
}

func (t *Transport) Poll(ctx context.Context, reportChan <-chan hid.Report) <-chan ExtendedResponseFrame {
	out := make(chan ExtendedResponseFrame)

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return

			case report, ok := <-reportChan:
				if !ok {
					slog.Info("report channel closed")
					return
				}

				n, ok := t.ReportLengths[report.ID]
				if !ok {
					slog.Warn("unknown report id", slog.Int("id", int(report.ID)))
					continue
				}

				if n < 0 || n > len(report.Data) {
					slog.Warn("report length out of range", slog.Int("expected", n), slog.Int("actual", len(report.Data)))
					continue
				}

				frames, err := parseFrames(report.Data[:n])
				if err != nil {
					slog.Warn("CSAFE frame parsing failed", slog.Any("error", err))
					continue
				}

				for _, f := range frames {
					out <- f
				}
			}
		}
	}()
	return out
}

func parseFrames(b []byte) ([]ExtendedResponseFrame, error) {
	var frames []ExtendedResponseFrame

	frameStartIdx := -1
	frameEndIdx := -1
	fmt.Println(EncodeReportToString(b))
	for i := 0; i < len(b); i++ {
		if b[i] == ExtendedFrameStartFlag {
			frameStartIdx = i
			continue
		}

		if b[i] == StopFrameFlag && frameStartIdx != -1 {
			frameEndIdx = i
			unstuffed, err := byteUnstuff(b[frameStartIdx+1 : frameEndIdx])
			if err != nil {
				frameStartIdx = -1
				frameEndIdx = -1
				slog.Warn("byte unstuffing failed", slog.Any("error", err))
				continue
			}

			fmt.Println(EncodeReportToString(unstuffed))
			frame := ExtendedResponseFrame{
				ResponseStatus: ResponseStatus{
					FrameToggle:         unstuffed[2] & FrameToggleBitMask,
					PreviousFrameStatus: unstuffed[2] & PreviousFrameStatusBitMask,
					StateMachineState:   unstuffed[2] & StateMachineStateBitMask,
				},
				DestinationAddress: unstuffed[0],
				SourceAddress:      unstuffed[1],
				Status:             unstuffed[2],
				checksum:           unstuffed[len(unstuffed)-1],
			}

			checksum := Checksum(unstuffed[2 : len(unstuffed)-1])
			if frame.checksum != checksum {
				frameStartIdx = -1
				frameEndIdx = -1

				slog.Warn("checksum validation failed", slog.Any("payload", frame.checksum), slog.Any("computed", checksum))
				continue
			}

			frames = append(frames, frame)
			frameStartIdx = -1
			frameEndIdx = -1
		}
	}

	return frames, nil
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

func (t *Transport) Send(ctx context.Context, c Command) error {
	return t.Device.WriteReport(ctx, hidReport2(extendedFrame(c.Marshall())))
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

type ResponseStatus struct {
	FrameToggle         byte
	PreviousFrameStatus byte
	StateMachineState   byte
}

type ExtendedResponseFrame struct {
	ResponseStatus      ResponseStatus
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

func hidReport1(b []byte) hid.Report {
	report := make([]byte, 20)
	copy(report, b)
	return hid.Report{
		ID:   0x01,
		Data: report,
	}
}

func hidReport2(b []byte) hid.Report {
	report := make([]byte, 120)
	copy(report, b)
	return hid.Report{
		ID:   0x02,
		Data: report,
	}
}

func hidReport4(b []byte) hid.Report {
	report := make([]byte, 500)
	copy(report, b)
	return hid.Report{
		ID:   0x04,
		Data: report,
	}
}

type Command interface {
	Marshall() []byte
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
