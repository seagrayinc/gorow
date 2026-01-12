package pm5

import (
	"encoding/binary"

	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_GETERRORCODE_CMD = 0x9C

func GetErrorCode() Command {
	return csafe.ShortCommand(csafe_GETERRORCODE_CMD)
}

type GetErrorCodeResponse struct {
	ErrorCode uint32 // 3 bytes (LSB, middle, MSB) stored as uint32
}

func parseGetErrorCodeResponse(b []byte) (GetErrorCodeResponse, error) {
	// Byte 0: Error Code (LSB)
	// Byte 1: Error Code
	// Byte 2: Error Code (MSB)
	// Pad to 4 bytes for LittleEndian.Uint32
	padded := []byte{b[0], b[1], b[2], 0}
	return GetErrorCodeResponse{
		ErrorCode: binary.LittleEndian.Uint32(padded),
	}, nil
}
