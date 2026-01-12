package pm5

import (
	"github.com/seagrayinc/pm5-csafe/internal/csafe"
)

const CSAFE_GETSTATUS_CMD = 0x80

const (
	MachineStateError   byte = 0x00
	MachineStateReady   byte = 0x01
	MachineStateIdle    byte = 0x02
	MachineStateHaveID  byte = 0x03
	MachineStateInUse   byte = 0x05
	MachineStatePause   byte = 0x06
	MachineStateFinish  byte = 0x07
	MachineStateManual  byte = 0x08
	MachineStateOffline byte = 0x09

	FrameToggleOff = 0x00
	FrameToggleOn  = 0x80

	FrameStatusOk       = 0x00
	FrameStatusReject   = 0x10
	FrameStatusBad      = 0x20
	FrameStatusNotReady = 0x30
)

func GetStatus() Command {
	return csafe.ShortCommand(CSAFE_GETSTATUS_CMD)
}

type GetStatusResponse csafe.ResponseStatus
