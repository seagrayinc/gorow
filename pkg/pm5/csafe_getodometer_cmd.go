package pm5

import (
	"encoding/binary"

	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_GETODOMETER_CMD = 0x9B

func GetOdometer() Command {
	return csafe.ShortCommand(csafe_GETODOMETER_CMD)
}

type GetOdometerResponse struct {
	Distance       uint32
	UnitsSpecifier byte
}

func parseGetOdometerResponse(b []byte) (GetOdometerResponse, error) {
	return GetOdometerResponse{
		Distance:       binary.LittleEndian.Uint32(b[:4]),
		UnitsSpecifier: b[4],
	}, nil
}
