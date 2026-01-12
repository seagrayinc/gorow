package pm5

import (
	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_GETID_CMD = 0x92

func GetID() Command {
	return csafe.ShortCommand(csafe_GETID_CMD)
}

type GetIDResponse struct {
	ASCIIDigit0 byte
	ASCIIDigit1 byte
	ASCIIDigit2 byte
	ASCIIDigit3 byte
	ASCIIDigit4 byte
}

func parseGetIDResponse(b []byte) (GetIDResponse, error) {
	return GetIDResponse{
		ASCIIDigit0: b[0],
		ASCIIDigit1: b[1],
		ASCIIDigit2: b[2],
		ASCIIDigit3: b[3],
		ASCIIDigit4: b[4],
	}, nil
}
