// Package pm5 implements the PM5 CSAFE Protocol defined at
// https://www.concept2.sg/files/pdf/us/monitors/PM5_CSAFECommunicationDefinition.pdf

package pm5

import (
	"context"
	"errors"

	"github.com/seagrayinc/pm5-csafe/pkg/csafe"
	"github.com/seagrayinc/pm5-csafe/pkg/hid"
)

const (
	Concept2VID uint16 = 0x17A4
	PM5PID      uint16 = 0x001E
)

var (
	ReportLengths = map[byte]int{
		0x01: 21,
		0x02: 121,
		0x04: 501,
	}
)

type PM5 struct {
	transport csafe.Transport
}

func (p PM5) GetVersion(ctx context.Context) (GetVersionResponse, error) {
	response, _, err := csafe.Send(ctx, p.transport, GetVersion{})
	return response, err
}

func (p PM5) GetPower(ctx context.Context) (GetPowerResponse, error) {
	response, _, err := csafe.Send(ctx, p.transport, GetPower{})
	return response, err
}

func (p PM5) GetID(ctx context.Context) (GetIDResponse, error) {
	response, _, err := csafe.Send(ctx, p.transport, GetID{})
	return response, err
}

func (p PM5) GetStatus(ctx context.Context) (GetStatusResponse, error) {
	_, frame, err := csafe.Send(ctx, p.transport, GetStatus{})
	if err != nil {
		return GetStatusResponse{}, err
	}

	return GetStatusResponse{Status: frame.StateMachineState()}, nil
}

func Open(mgr hid.Manager) (PM5, error) {
	dev, err := mgr.OpenVIDPID(Concept2VID, PM5PID)
	if err != nil {
		return PM5{}, errors.New("performance monitor not found")
	}

	return PM5{
		transport: csafe.Transport{Device: dev},
	}, nil
}

func (p PM5) Close() error {
	return p.transport.Close()
}

func (p PM5) GetStrokeStats(ctx context.Context) (GetStrokeStatsResponse, error) {
	response, _, err := csafe.Send(ctx, p.transport, GetStrokeStats{})
	return response, err
}

func (p PM5) GetStrokeState(ctx context.Context) (GetStrokeStateResponse, error) {
	response, _, err := csafe.Send(ctx, p.transport, GetStrokeState{})
	return response, err
}

func (p PM5) GetWorkoutState(ctx context.Context) (GetWorkoutStateResponse, error) {
	response, _, err := csafe.Send(ctx, p.transport, GetWorkoutState{})
	return response, err
}
