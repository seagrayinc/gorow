package pm5

import (
	"encoding/binary"

	"github.com/seagrayinc/pm5-csafe/internal/csafe"
)

const CSAFE_GETVERSION_CMD = 0x91

func GetVersion() Command {
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
