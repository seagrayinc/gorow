package pm5

import (
	"encoding/hex"
	"errors"
	"log/slog"

	"github.com/seagrayinc/gorow/internal/csafe"
)

const (
	csafe_SETUSERCFG1_CMD = 0x1A
)

func parseResponses(f csafe.ExtendedResponseFrame) ([]interface{}, error) {
	var parsedResponses []interface{}

	parsedResponses = append(parsedResponses, GetStatusResponse(f.ResponseStatus))

	for _, resp := range f.CommandResponses {
		switch resp.Command {
		case csafe_GETVERSION_CMD:
			parsedResp, err := parseGetVersionResponse(resp.Data)
			if err != nil {
				return nil, err
			}
			parsedResponses = append(parsedResponses, parsedResp)

		case csafe_GETPOWER_CMD:
			parsedResp, err := parseGetPowerResponse(resp.Data)
			if err != nil {
				return nil, err
			}
			parsedResponses = append(parsedResponses, parsedResp)

		case csafe_GETID_CMD:
			parsedResp, err := parseGetIDResponse(resp.Data)
			if err != nil {
				return nil, err
			}
			parsedResponses = append(parsedResponses, parsedResp)

		case csafe_SETUSERCFG1_CMD:
			unwrapped, err := unwrap(resp.Data)
			if err != nil {
				return nil, err
			}
			switch unwrapped.Command {
			case csafe_PM_GET_STROKESTATS:
				parsedResp, err := parseGetStrokeStatsResponse(unwrapped.Data)
				if err != nil {
					return nil, err
				}
				parsedResponses = append(parsedResponses, parsedResp)

			case csafe_PM_GET_STROKESTATE:
				parsedResp, err := parseGetStrokeStateResponse(unwrapped.Data)
				if err != nil {
					return nil, err
				}
				parsedResponses = append(parsedResponses, parsedResp)

			case csafe_PM_GET_WORKOUTSTATE:
				parsedResp, err := parseGetWorkoutStateResponse(unwrapped.Data)
				if err != nil {
					return nil, err
				}
				parsedResponses = append(parsedResponses, parsedResp)

			default:
				slog.Warn("unsupported wrapped command response", slog.String("command", hex.EncodeToString([]byte{unwrapped.Command})))
			}

		default:
			slog.Warn("unsupported command response", slog.String("command", hex.EncodeToString([]byte{resp.Command})))
		}
	}

	return parsedResponses, nil
}

func wrap(c csafe.Command) csafe.Command {
	return csafe.LongCommand(csafe_SETUSERCFG1_CMD, c)
}

func unwrap(b []byte) (csafe.Response, error) {
	responses := csafe.ParseResponses(b)
	if len(responses) < 1 {
		return csafe.Response{}, errors.New("malformed response")
	}
	return responses[0], nil
}
