package pm5

import (
	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_GOIDLE_CMD = 0x82

func GoIdle() Command {
	return csafe.ShortCommand(csafe_GOIDLE_CMD)
}
