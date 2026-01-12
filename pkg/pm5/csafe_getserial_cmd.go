package pm5

import (
	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_GETSERIAL_CMD = 0x94

func GetSerial() Command {
	return csafe.ShortCommand(csafe_GETSERIAL_CMD)
}

type GetSerialResponse struct {
	SerialNumber string
}

func parseGetSerialResponse(b []byte) (GetSerialResponse, error) {
	return GetSerialResponse{
		SerialNumber: string(b[:9]),
	}, nil
}
