package pm5

import (
	"encoding/binary"

	"github.com/seagrayinc/pm5-csafe/pkg/csafe"
)

const CSAFE_GETPOWER_CMD = 0xB4

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
