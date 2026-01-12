package pm5

import (
	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_PM_GET_STROKESTATE = 0xBF

func GetStrokeState() Command {
	return wrap(csafe.ShortCommand(csafe_PM_GET_STROKESTATE))
}

type GetStrokeStateResponse struct {
	StrokeState int
}

func parseGetStrokeStateResponse(b []byte) (GetStrokeStateResponse, error) {
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
