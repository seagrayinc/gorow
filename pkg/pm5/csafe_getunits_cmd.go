package pm5

import (
	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_GETUNITS_CMD = 0x93

func GetUnits() Command {
	return csafe.ShortCommand(csafe_GETUNITS_CMD)
}

type GetUnitsResponse struct {
	UnitsType byte
}

func parseGetUnitsResponse(b []byte) (GetUnitsResponse, error) {
	return GetUnitsResponse{
		UnitsType: b[0],
	}, nil
}
