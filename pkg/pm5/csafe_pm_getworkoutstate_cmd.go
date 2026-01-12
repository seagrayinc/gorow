package pm5

import (
	"github.com/seagrayinc/pm5-csafe/pkg/csafe"
)

const CSAFE_PM_GET_WORKOUTSTATE = 0x8D

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
	return GetWorkoutStateResponse{WorkoutState: int(b[0]), WorkoutStateString: WorkoutStateMap[int(b[0])]}, nil
}
