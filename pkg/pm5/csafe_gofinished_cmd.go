package pm5

import (
	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_GOFINISHED_CMD = 0x86

func GoFinished() Command {
	return csafe.ShortCommand(csafe_GOFINISHED_CMD)
}
