package pm5

import (
	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_GOREADY_CMD = 0x87

func GoReady() Command {
	return csafe.ShortCommand(csafe_GOREADY_CMD)
}
