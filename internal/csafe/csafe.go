package csafe

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	hid2 "github.com/seagrayinc/gorow/internal/hid"
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
	Device        hid2.Device
	ReportLengths map[byte]int
	SendTimeout   time.Duration // Minimum time between sends if no message received (default 100ms)
	SendBuffer    int           // Size of send buffer (default 100)

	mu                    sync.Mutex
	lastSendTime          time.Time
	receivedSinceLastSend bool

	sendOnce sync.Once
	cmdChan  chan Command
}

func (t *Transport) Close() error {
	return t.Device.Close()
}

func (t *Transport) Poll(ctx context.Context, reportChan <-chan hid2.Report) <-chan ExtendedResponseFrame {
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

				// Signal that we've received a message
				t.mu.Lock()
				t.receivedSinceLastSend = true
				t.mu.Unlock()

				if _, ok := t.ReportLengths[report.ID]; !ok {
					slog.Warn("unknown report id", slog.Int("id", int(report.ID)))
					continue
				}

				frames, err := parseFrames(report.Data)
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

	frameStartIdx, frameEndIdx := -1, -1
	slog.Debug("parsing frames", slog.String("bytes", EncodeReportToString(b)))
	for i := 0; i < len(b); i++ {
		if b[i] == ExtendedFrameStartFlag {
			frameStartIdx = i
			continue
		}

		if b[i] == StopFrameFlag && frameStartIdx != -1 {
			frameEndIdx = i
			slog.Debug("frame found", slog.String("stuffed bytes", EncodeReportToString(b[frameStartIdx+1:frameEndIdx])))
			unstuffed, err := byteUnstuff(b[frameStartIdx+1 : frameEndIdx])
			if err != nil {
				frameStartIdx = -1
				frameEndIdx = -1
				slog.Warn("byte unstuffing failed", slog.Any("error", err))
				continue
			}

			slog.Debug("frame found", slog.String("unstuffed bytes", EncodeReportToString(unstuffed)))
			computedChecksum := Checksum(unstuffed[2 : len(unstuffed)-1])
			declaredChecksum := unstuffed[len(unstuffed)-1]
			if declaredChecksum != computedChecksum {
				frameStartIdx = -1
				frameEndIdx = -1

				slog.Warn("checksum validation failed", slog.Any("payload", declaredChecksum), slog.Any("computed", computedChecksum))
				continue
			}

			frames = append(frames, ExtendedResponseFrame{
				ResponseStatus: ResponseStatus{
					FrameToggle:         unstuffed[2] & FrameToggleBitMask,
					PreviousFrameStatus: unstuffed[2] & PreviousFrameStatusBitMask,
					StateMachineState:   unstuffed[2] & StateMachineStateBitMask,
				},
				DestinationAddress: unstuffed[0],
				SourceAddress:      unstuffed[1],
				Status:             unstuffed[2],
				CommandResponses:   ParseResponses(unstuffed[3 : len(unstuffed)-1]),
			})

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

type Command []byte

// StartSender starts the background goroutine that processes buffered commands.
// It must be called before Send. The context controls the lifetime of the sender.
func (t *Transport) StartSender(ctx context.Context) {
	t.sendOnce.Do(func() {
		bufSize := t.SendBuffer
		if bufSize <= 0 {
			bufSize = 100
		}
		t.cmdChan = make(chan Command, bufSize)

		go t.sendLoop(ctx)
	})
}

func (t *Transport) sendLoop(ctx context.Context) {
	timeout := t.SendTimeout
	if timeout == 0 {
		timeout = 100 * time.Millisecond
	}

	for {
		select {
		case <-ctx.Done():
			return
		case cmd, ok := <-t.cmdChan:
			if !ok {
				return
			}

			// Collect this command and any others available in the buffer
			commands := []Command{cmd}
		collectLoop:
			for {
				select {
				case c, ok := <-t.cmdChan:
					if !ok {
						break collectLoop
					}
					commands = append(commands, c)
				default:
					break collectLoop
				}
			}

			// Wait until we can send
			for {
				t.mu.Lock()
				canSend := t.receivedSinceLastSend || t.lastSendTime.IsZero() || time.Since(t.lastSendTime) >= timeout
				if canSend {
					t.lastSendTime = time.Now()
					t.receivedSinceLastSend = false
					t.mu.Unlock()
					break
				}
				waitTime := timeout - time.Since(t.lastSendTime)
				t.mu.Unlock()

				select {
				case <-ctx.Done():
					return
				case <-time.After(waitTime):
					// Continue loop to re-check conditions
				}
			}

			// Always use report ID 0x02 and length 120 for sending. Report ID 0x01 is too short for long commands and
			// causes checksum failures on response (which also comes on report ID 0x01). Report ID 0x04 doesn't always
			// result in a response.
			report := hidReport(0x02, 120, extendedFrame(commands))
			if err := t.Device.WriteReport(ctx, report); err != nil {
				slog.Warn("failed to write report", slog.Any("error", err))
			}
		}
	}
}

// extendedFrame creates a frame containing multiple commands
func extendedFrame(commands []Command) []byte {
	var cmdBytes []byte
	for _, cmd := range commands {
		cmdBytes = append(cmdBytes, cmd...)
	}

	var frameContents []byte
	frameContents = append(frameContents, ExtendedFrameAddressDefaultSecondary)
	frameContents = append(frameContents, ExtendedFrameAddressPCHostPrimary)
	frameContents = append(frameContents, cmdBytes...)
	frameContents = append(frameContents, Checksum(cmdBytes))

	frame := []byte{ExtendedFrameStartFlag}
	frame = append(frame, byteStuff(frameContents)...)
	frame = append(frame, StopFrameFlag)
	return frame
}

// hidReport creates a report with the given ID and length
func hidReport(id byte, length int, data []byte) hid2.Report {
	report := make([]byte, length)
	copy(report, data)
	return hid2.Report{
		ID:   id,
		Data: report,
	}
}

// Send buffers commands for sending. It is non-blocking.
// StartSender must be called before Send.
func (t *Transport) Send(_ context.Context, commands ...Command) error {
	for _, c := range commands {
		select {
		case t.cmdChan <- c:
			slog.Debug("sending command", slog.String("command", hex.EncodeToString(c)))
		default:
			slog.Warn("send buffer full, dropping command")
			return errors.New("send buffer full")
		}
	}

	return nil
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

func LongCommand(id byte, data []byte) Command {
	b := []byte{id, byte(len(data))}
	b = append(b, data...)
	return b
}

func ShortCommand(id byte) Command {
	return []byte{id}
}

type ResponseStatus struct {
	FrameToggle         byte
	PreviousFrameStatus byte
	StateMachineState   byte
}

type ExtendedResponseFrame struct {
	ResponseStatus     ResponseStatus
	Status             byte
	DestinationAddress byte
	SourceAddress      byte
	CommandResponses   []Response
}

func ParseResponses(frameContents []byte) []Response {
	if len(frameContents) == 0 {
		return nil
	}

	var cmdIdx int
	var responses []Response

	for {
		dataByteCount := frameContents[cmdIdx+1]
		resp := Response{
			Command:       frameContents[cmdIdx],
			DataByteCount: dataByteCount,
			Data:          make([]byte, dataByteCount),
		}

		dataStart := cmdIdx + 1 + 1                   // starts after the command and data size
		dataEnd := dataStart + int(dataByteCount) - 1 // inclusive end index

		copy(resp.Data, frameContents[dataStart:dataEnd+1])
		responses = append(responses, resp)
		cmdIdx = dataEnd + 1
		if cmdIdx >= len(frameContents) {
			break
		}
	}

	return responses
}

type Response struct {
	Command       byte
	DataByteCount byte
	Data          []byte
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
