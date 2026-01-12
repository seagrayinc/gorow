package pm5

import (
	"encoding/hex"
	"log/slog"

	"github.com/seagrayinc/gorow/internal/csafe"
)

type parserFunc func([]byte) (any, error)

// wrappedParser is a helper to convert a typed parser function into a generic parserFunc.
func wrappedParser[T any](f func([]byte) (T, error)) parserFunc {
	return func(b []byte) (any, error) {
		return f(b)
	}
}

var (
	parserMap = map[byte]parserFunc{
		csafe_GETVERSION_CMD:      wrappedParser(parseGetVersionResponse),
		csafe_GETPOWER_CMD:        wrappedParser(parseGetPowerResponse),
		csafe_GETID_CMD:           wrappedParser(parseGetIDResponse),
		csafe_GETUNITS_CMD:        wrappedParser(parseGetUnitsResponse),
		csafe_PM_GET_STROKESTATS:  wrappedParser(parseGetStrokeStatsResponse),
		csafe_PM_GET_STROKESTATE:  wrappedParser(parseGetStrokeStateResponse),
		csafe_PM_GET_WORKOUTSTATE: wrappedParser(parseGetWorkoutStateResponse),
	}
)

func parseResponses(f csafe.ExtendedResponseFrame) ([]any, error) {
	// Every response frame includes a status byte that we can use to construct a GetStatusResponse. Even in the case
	// where GetStatus is explicitly requested, this results in an empty CSAFE data payload and just the status byte.
	// For this reason, we'll always create _at least_ a GetStatusResponse event for every response frame.
	responses := []any{GetStatusResponse(f.ResponseStatus)}

	// Now parse out any additional command responses included in the frame.
	for _, resp := range f.CommandResponses {
		r, err := unwrap(resp)
		if err != nil {
			slog.Error("failed to unwrap response", slog.Any("error", err))
			continue
		}

		parser, ok := parserMap[r.Command]
		if !ok {
			slog.Warn("unsupported command response", slog.String("command", hex.EncodeToString([]byte{resp.Command})))
			continue
		}

		parsedResp, err := parser(r.Data)
		if err != nil {
			return nil, err
		}

		responses = append(responses, parsedResp)
	}

	return responses, nil
}
