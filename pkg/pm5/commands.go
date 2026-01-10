package pm5

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"log/slog"

	"github.com/seagrayinc/pm5-csafe/pkg/csafe"
)

const (
	CSAFE_SETUSERCFG1_CMD     = 0x1A
	CSAFE_PM_GET_STROKESTATS  = 0x6E
	CSAFE_GETSTATUS_CMD       = 0x80
	CSAFE_PM_GET_WORKOUTSTATE = 0x8D
	CSAFE_GETVERSION_CMD      = 0x91
	CSAFE_GETID_CMD           = 0x92
	CSAFE_GETPOWER_CMD        = 0xB4
	CSAFE_PM_GET_STROKESTATE  = 0xBF
)

func ParseResponses(f csafe.ExtendedResponseFrame) ([]interface{}, error) {
	var parsedResponses []interface{}

	for _, resp := range f.CommandResponses {
		switch resp.Command {
		case CSAFE_GETVERSION_CMD:
			parsedResp, err := ParseGetVersionResponse(resp.Data)
			if err != nil {
				return nil, err
			}
			parsedResponses = append(parsedResponses, parsedResp)

		case CSAFE_GETPOWER_CMD:
			parsedResp, err := ParseGetPowerResponse(resp.Data)
			if err != nil {
				return nil, err
			}
			parsedResponses = append(parsedResponses, parsedResp)

		case CSAFE_GETID_CMD:
			parsedResp, err := ParseGetIDResponse(resp.Data)
			if err != nil {
				return nil, err
			}
			parsedResponses = append(parsedResponses, parsedResp)

		case CSAFE_SETUSERCFG1_CMD:
			unwrapped, err := unwrap(resp.Data)
			if err != nil {
				return nil, err
			}
			switch unwrapped.Command {
			case CSAFE_PM_GET_STROKESTATS:
				parsedResp, err := ParseGetStrokeStatsResponse(unwrapped.Data)
				if err != nil {
					return nil, err
				}
				parsedResponses = append(parsedResponses, parsedResp)

			case CSAFE_PM_GET_STROKESTATE:
				parsedResp, err := ParseGetStrokeStateResponse(unwrapped.Data)
				if err != nil {
					return nil, err
				}
				parsedResponses = append(parsedResponses, parsedResp)

			case CSAFE_PM_GET_WORKOUTSTATE:
				parsedResp, err := ParseGetWorkoutStateResponse(unwrapped.Data)
				if err != nil {
					return nil, err
				}
				parsedResponses = append(parsedResponses, parsedResp)

			default:
				slog.Warn("unsupported wrapped command response", slog.String("command", hex.EncodeToString([]byte{unwrapped.Command})))
			}

		default:
			slog.Warn("unsupported command response", slog.String("command", hex.EncodeToString([]byte{resp.Command})))
		}
	}

	return parsedResponses, nil
}

func GetVersion() csafe.Command {
	return csafe.ShortCommand(CSAFE_GETVERSION_CMD)
}

type GetVersionResponse struct {
	ManufacturerID  int
	ClassID         int
	Model           int
	HardwareVersion int
	FirmwareVersion int
}

func ParseGetVersionResponse(b []byte) (GetVersionResponse, error) {
	return GetVersionResponse{
		ManufacturerID:  int(b[0]),
		ClassID:         int(b[1]),
		Model:           int(b[2]),
		HardwareVersion: int(binary.LittleEndian.Uint16(b[3:5])),
		FirmwareVersion: int(binary.LittleEndian.Uint16(b[5:7])),
	}, nil
}

func GetPower() csafe.Command {
	return csafe.ShortCommand(CSAFE_GETPOWER_CMD)
}

type GetPowerResponse struct {
	StrokeWatts    int
	UnitsSpecifier int
}

func ParseGetPowerResponse(b []byte) (GetPowerResponse, error) {
	return GetPowerResponse{
		StrokeWatts:    int(binary.LittleEndian.Uint16(b[:2])),
		UnitsSpecifier: int(b[2]),
	}, nil
}

func GetID() csafe.Command {
	return csafe.ShortCommand(CSAFE_GETID_CMD)
}

type GetIDResponse struct {
	ASCIIDigit0 byte
	ASCIIDigit1 byte
	ASCIIDigit2 byte
	ASCIIDigit3 byte
	ASCIIDigit4 byte
}

func ParseGetIDResponse(b []byte) (GetIDResponse, error) {
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

func GetStrokeStats() csafe.Command {
	return wrap(csafe.LongCommand(CSAFE_PM_GET_STROKESTATS, []byte{0}))
}

func ParseGetStrokeStatsResponse(b []byte) (GetStrokeStatsResponse, error) {
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

func GetStrokeState() csafe.Command {
	return wrap(csafe.ShortCommand(CSAFE_PM_GET_STROKESTATE))
}

type GetStrokeStateResponse struct {
	StrokeState int
}

func ParseGetStrokeStateResponse(b []byte) (GetStrokeStateResponse, error) {
	//Stroke State
	//typedef enum {
	//	STROKESTATE_WAITING_FOR_WHEEL_TO_REACH_MIN_SPEED_STATE, /**< FW to reach min speed state (0). */
	//	STROKESTATE_WAITING_FOR_WHEEL_TO_ACCELERATE_STATE, /**< FW to accelerate state (1). */
	//	STROKESTATE_DRIVING_STATE, /**< Driving state (2). */
	//	STROKESTATE_DWELLING_AFTER_DRIVE_STATE, /**< Dwelling after drive state (3). */
	//	STROKESTATE_RECOVERY_STATE /**< Recovery state (4). */
	//} OBJ_STROKESTATE_T;
	return GetStrokeStateResponse{StrokeState: int(b[0])}, nil
}

func GetWorkoutState() csafe.Command {
	return wrap(csafe.ShortCommand(CSAFE_PM_GET_WORKOUTSTATE))
}

type GetWorkoutStateResponse struct {
	WorkoutState       int
	WorkoutStateString string
}

var WorkoutStateMap = map[int]string{
	0:  "Wait to begin",
	1:  "Workout row",
	2:  "Countdown pause",
	3:  "Interval rest",
	4:  "Interval work time",
	5:  "Interval work distance",
	6:  "Interval rest end to work time",
	7:  "Interval rest end to work distance",
	8:  "Interval work time to rest",
	9:  "Interval work distance to rest",
	10: "Workout end",
	11: "Workout terminate",
	12: "Workout logged",
	13: "Workout rearm",
}

func ParseGetWorkoutStateResponse(b []byte) (GetWorkoutStateResponse, error) {
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
	return GetWorkoutStateResponse{WorkoutState: int(b[2]), WorkoutStateString: WorkoutStateMap[int(b[2])]}, nil
}

func wrap(c csafe.Command) csafe.Command {
	return csafe.LongCommand(CSAFE_SETUSERCFG1_CMD, c)
}

func unwrap(b []byte) (csafe.Response, error) {
	responses := csafe.ParseResponses(b)
	if len(responses) < 1 {
		return csafe.Response{}, errors.New("malformed response")
	}
	return responses[0], nil
}
