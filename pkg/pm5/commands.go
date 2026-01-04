package pm5

import (
	"encoding/binary"
	"errors"

	"github.com/seagrayinc/pm5-csafe/pkg/csafe"
)

const (
	CSAFE_GETSTATUS_CMD = 0x80
)

type GetVersion struct{}

type GetVersionResponse struct {
	ManufacturerID  int
	ClassID         int
	Model           int
	HardwareVersion int
	FirmwareVersion int
}

func (g GetVersion) Marshall() []byte {
	return csafe.ShortCommand{
		ShortCommand: 0x91,
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
	return csafe.ShortCommand{
		ShortCommand: 0xB4,
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
	return csafe.ShortCommand{
		ShortCommand: 0x92,
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

type StateMachineState = byte

const (
	MachineStateError   StateMachineState = 0x00
	MachineStateReady   StateMachineState = 0x01
	MachineStateIdle    StateMachineState = 0x02
	MachineStateHaveID  StateMachineState = 0x03
	MachineStateInUse   StateMachineState = 0x05
	MachineStatePause   StateMachineState = 0x06
	MachineStateFinish  StateMachineState = 0x07
	MachineStateManual  StateMachineState = 0x08
	MachineStateOffline StateMachineState = 0x09
)

type GetStatus struct{}

type GetStatusResponse struct {
	Status StateMachineState
}

func (g GetStatus) Marshall() []byte {
	return csafe.ShortCommand{
		ShortCommand: CSAFE_GETSTATUS_CMD,
	}.Bytes()
}

func (g GetStatus) Unmarshall(_ []byte) (GetStatusResponse, error) {
	return GetStatusResponse{}, nil
}

type GetStrokeStats struct{}

func (g GetStrokeStats) Unmarshall(in []byte) (GetStrokeStatsResponse, error) {
	b := in[2:] // TODO: this should probably be done at the transport layer.... It's required here because the proprietary commands are wrapped

	if len(b) < 16 {
		return GetStrokeStatsResponse{}, errors.New("malformed response")
	}

	return GetStrokeStatsResponse{
		StrokeDistance:     int(binary.LittleEndian.Uint16(b[0:2])),
		StrokeDriveTime:    int(b[2]),
		StrokeRecoveryTime: int(binary.LittleEndian.Uint16(b[3:5])),
		StrokeLength:       int(b[5]),
		DriveCounter:       int(binary.LittleEndian.Uint16(b[6:8])),
		PeakDriveForce:     int(binary.LittleEndian.Uint16(b[8:10])),
		ImpulseDriveForce:  int(binary.LittleEndian.Uint16(b[10:12])),
		AverageDriveForce:  int(binary.LittleEndian.Uint16(b[12:14])),
		WorkPerStroke:      int(binary.LittleEndian.Uint16(b[14:16])),
	}, nil
}

type GetStrokeStatsResponse struct {
	StrokeDistance     int
	StrokeDriveTime    int
	StrokeRecoveryTime int
	StrokeLength       int
	DriveCounter       int
	PeakDriveForce     int
	ImpulseDriveForce  int
	AverageDriveForce  int
	WorkPerStroke      int
}

func (g GetStrokeStats) Marshall() []byte {
	return csafe.GetPMDataCommand(csafe.NewLongCommand(0x6E, []byte{0}))
}

type GetStrokeState struct{}

type GetStrokeStateResponse struct {
	StrokeState int
}

func (g GetStrokeState) Marshall() []byte {
	return csafe.GetPMDataCommand(csafe.ShortCommand{ShortCommand: 0xBF})
}

func (g GetStrokeState) Unmarshall(b []byte) (GetStrokeStateResponse, error) {
	//Stroke State
	//typedef enum {
	//	STROKESTATE_WAITING_FOR_WHEEL_TO_REACH_MIN_SPEED_STATE, /**< FW to reach min speed state (0). */
	//	STROKESTATE_WAITING_FOR_WHEEL_TO_ACCELERATE_STATE, /**< FW to accelerate state (1). */
	//	STROKESTATE_DRIVING_STATE, /**< Driving state (2). */
	//	STROKESTATE_DWELLING_AFTER_DRIVE_STATE, /**< Dwelling after drive state (3). */
	//	STROKESTATE_RECOVERY_STATE /**< Recovery state (4). */
	//} OBJ_STROKESTATE_T;
	return GetStrokeStateResponse{StrokeState: int(b[2])}, nil
}

type GetWorkoutState struct{}

type GetWorkoutStateResponse struct {
	WorkoutState int
}

func (g GetWorkoutState) Marshall() []byte {
	return csafe.GetPMDataCommand(csafe.ShortCommand{ShortCommand: 0x8D})
}

func (g GetWorkoutState) Unmarshall(b []byte) (GetWorkoutStateResponse, error) {
	//typedef enum {
	//	WORKOUTSTATE_WAITTOBEGIN, /**< Wait to begin state (0). */
	//	WORKOUTSTATE_WORKOUTROW, /**< Workout row state (1). */
	//	WORKOUTSTATE_COUNTDOWNPAUSE, /**< Countdown pause state (2). */
	//	WORKOUTSTATE_INTERVALREST, /**< Interval rest state (3). */
	//	WORKOUTSTATE_INTERVALWORKTIME, /**< Interval work time state (4). */
	//	WORKOUTSTATE_INTERVALWORKDISTANCE, /**< Interval work distance state (5). */
	//	WORKOUTSTATE_INTERVALRESTENDTOWORKTIME, /**< Interval rest end to work time state (6). */
	//	WORKOUTSTATE_INTERVALRESTENDTOWORKDISTANCE, /**< Interval rest end to work distance state (7). */
	//	WORKOUTSTATE_INTERVALWORKTIMETOREST, /**< Interval work time to rest state (8). */
	//	WORKOUTSTATE_INTERVALWORKDISTANCETOREST, /**< Interval work distance to rest state (9). */
	//	WORKOUTSTATE_WORKOUTEND, /**< Workout end state (10). */
	//	WORKOUTSTATE_TERMINATE, /**< Workout terminate state (11). */
	//	WORKOUTSTATE_WORKOUTLOGGED, /**< Workout logged state (12). */
	//	WORKOUTSTATE_REARM, /**< Workout rearm state (13). */
	//} OBJ_WORKOUTSTATE_T;
	return GetWorkoutStateResponse{WorkoutState: int(b[2])}, nil
}
