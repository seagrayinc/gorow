// Package pm5csafe implements the PM5 CSAFE Protocol defined at
// https://www.concept2.sg/files/pdf/us/monitors/PM5_CSAFECommunicationDefinition.pdf

package pm5

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/seagrayinc/pm5-csafe/pkg/hid"
)

const (
	Concept2VID uint16 = 0x17A4
	PM5PID      uint16 = 0x001E

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
	CSAFE_GETVERSION_CMD  = 0x91
	CSAFE_GETPOWER_CMD    = 0xB4
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

type PM5 struct {
	device hid.Device
}

func (p PM5) GetVersion(ctx context.Context) (GetVersionResponse, error) {
	return Send(ctx, p, GetVersion{})
}

func (p PM5) GetPower(ctx context.Context) (GetPowerResponse, error) {
	return Send(ctx, p, GetPower{})
}

func (p PM5) GetID(ctx context.Context) (GetIDResponse, error) {
	return Send(ctx, p, GetID{})
}

func Send[T any](_ context.Context, pm PM5, cmd Command[T]) (T, error) {
	var zero T

	report := hidReport2(extendedFrame(cmd.Marshall()))
	_, err := pm.device.Write(report)
	if err != nil {
		return zero, fmt.Errorf("send failed: %w", err)
	}

	resp := make([]byte, len(report))
	_, err = pm.device.Read(resp)

	frame, err := ParseExtendedHIDResponse(resp)
	if err != nil {
		return zero, fmt.Errorf("failed to parse extended frame: %w", err)
	}

	return cmd.Unmarshall(frame.CommandResponses()[0].Data)
}

func Open(mgr hid.Manager) (PM5, error) {
	dev, err := mgr.OpenVIDPID(Concept2VID, PM5PID)
	if err != nil {
		return PM5{}, errors.New("performance monitor not found")
	}

	return PM5{
		device: dev,
	}, nil
}

func (p PM5) Close() error {
	return p.device.Close()
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
	Checksum            byte
	CommandResponseData []byte
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
		dataByteCount := rf.CommandResponseData[cmdIdx+1]
		resp := IndividualCommandResponse{
			Command:       rf.CommandResponseData[cmdIdx],
			DataByteCount: dataByteCount,
			Data:          make([]byte, dataByteCount),
		}

		dataStart := cmdIdx + 1 + 1 // starts after the command and data size
		dataEnd := dataStart + int(dataByteCount)

		copy(resp.Data, rf.CommandResponseData[dataStart:dataEnd])
		responses = append(responses, resp)
		cmdIdx = dataEnd + 1
		if cmdIdx > len(rf.CommandResponseData) {
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
	frame := []byte{ExtendedFrameStartFlag}
	frame = append(frame, ExtendedFrameAddressDefaultSecondary)
	frame = append(frame, ExtendedFrameAddressPCHostPrimary)
	frame = append(frame, b...)
	frame = append(frame, Checksum(b))
	frame = append(frame, StopFrameFlag)
	return frame
}

func hidReport2(b []byte) []byte {
	report := make([]byte, 121)
	report[0] = 0x02
	copy(report[1:], b)
	return report
}

type Command[T any] interface {
	Marshall() []byte
	Unmarshall([]byte) (T, error)
}

type GetVersion struct{}

type GetVersionResponse struct {
	ManufacturerID  int
	ClassID         int
	Model           int
	HardwareVersion int
	FirmwareVersion int
}

func (g GetVersion) Marshall() []byte {
	return ShortCommand{
		ShortCommand: CSAFE_GETVERSION_CMD,
	}.Bytes()
}

func (g GetVersion) Unmarshall(b []byte) (GetVersionResponse, error) {
	return GetVersionResponse{
		ManufacturerID:  int(b[0]),
		ClassID:         int(b[1]),
		Model:           int(b[2]),
		HardwareVersion: int(binary.LittleEndian.Uint16(b[3:5])),
		FirmwareVersion: int(binary.LittleEndian.Uint16(b[5:7])),
	}, nil
}

type GetPower struct{}

type GetPowerResponse struct {
	StrokeWatts    int
	UnitsSpecifier int
}

func (g GetPower) Marshall() []byte {
	return ShortCommand{
		ShortCommand: CSAFE_GETPOWER_CMD,
	}.Bytes()
}

func (g GetPower) Unmarshall(b []byte) (GetPowerResponse, error) {
	return GetPowerResponse{
		StrokeWatts:    int(binary.LittleEndian.Uint16(b[:2])),
		UnitsSpecifier: int(b[2]),
	}, nil
}

type GetID struct{}

type GetIDResponse struct {
	ASCIIDigit0 byte
	ASCIIDigit1 byte
	ASCIIDigit2 byte
	ASCIIDigit3 byte
	ASCIIDigit4 byte
}

func (g GetID) Marshall() []byte {
	return ShortCommand{
		ShortCommand: CSAFE_GETID_CMD,
	}.Bytes()
}

func (g GetID) Unmarshall(b []byte) (GetIDResponse, error) {
	return GetIDResponse{
		ASCIIDigit0: b[0],
		ASCIIDigit1: b[1],
		ASCIIDigit2: b[2],
		ASCIIDigit3: b[3],
		ASCIIDigit4: b[4],
	}, nil
}

func findFrameEnd(frame []byte) (int, error) {
	for i, b := range frame {
		if b == StopFrameFlag {
			return i, nil
		}
	}

	return 0, errors.New("could not find frame end")
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

	frame := ExtendedResponseFrame{
		DestinationAddress:  b[2],
		SourceAddress:       b[3],
		Status:              b[4],
		CommandResponseData: b[5 : frameStop-1],
		Checksum:            b[frameStop-1],
	}

	if err := frame.parseResponses(); err != nil {
		return ExtendedResponseFrame{}, errors.New("failed to parse command responses in frame")
	}
	return frame, nil
}
