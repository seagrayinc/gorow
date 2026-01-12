package pm5

import (
	"errors"

	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_SETUSERCFG1_CMD = 0x1A

func wrap(c csafe.Command) csafe.Command {
	return csafe.LongCommand(csafe_SETUSERCFG1_CMD, c)
}

func unwrap(r csafe.Response) (csafe.Response, error) {
	if r.Command != csafe_SETUSERCFG1_CMD {
		return r, nil
	}

	responses := csafe.ParseResponses(r.Data)
	if len(responses) < 1 {
		return csafe.Response{}, errors.New("malformed response")
	}

	return responses[0], nil
}
