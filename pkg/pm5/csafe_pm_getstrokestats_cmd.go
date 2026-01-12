package pm5

import (
	"encoding/binary"

	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_PM_GET_STROKESTATS = 0x6E

func GetStrokeStats() Command {
	return wrap(csafe.LongCommand(csafe_PM_GET_STROKESTATS, []byte{0}))
}

type GetStrokeStatsResponse struct {
	StrokeDistance     int
	StrokeDriveTime    int
	StrokeRecoveryTime int
	StrokeLength       int
	DriveCounter       int
	PeakDriveForce     int
	ImpulseDriveForce  int
	AverageDriveForce  int
	WorkPerStroke      int
}

func parseGetStrokeStatsResponse(b []byte) (GetStrokeStatsResponse, error) {
	return GetStrokeStatsResponse{
		StrokeDistance:     int(binary.LittleEndian.Uint16(b[0:2])),
		StrokeDriveTime:    int(b[2]),
		StrokeRecoveryTime: int(binary.LittleEndian.Uint16(b[3:5])),
		StrokeLength:       int(b[5]),
		DriveCounter:       int(binary.LittleEndian.Uint16(b[6:8])),
		PeakDriveForce:     int(binary.LittleEndian.Uint16(b[8:10])),
		ImpulseDriveForce:  int(binary.LittleEndian.Uint16(b[10:12])),
		AverageDriveForce:  int(binary.LittleEndian.Uint16(b[12:14])),
		WorkPerStroke:      int(binary.LittleEndian.Uint16(b[14:16])),
	}, nil
}
