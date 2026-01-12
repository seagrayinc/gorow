package pm5

import (
	"encoding/binary"

	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_GETPOWER_CMD = 0xB4

func GetPower() Command {
	return csafe.ShortCommand(csafe_GETPOWER_CMD)
}

type GetPowerResponse struct {
	StrokeWatts    int
	UnitsSpecifier int
}

func parseGetPowerResponse(b []byte) (GetPowerResponse, error) {
	return GetPowerResponse{
		StrokeWatts:    int(binary.LittleEndian.Uint16(b[:2])),
		UnitsSpecifier: int(b[2]),
	}, nil
}
