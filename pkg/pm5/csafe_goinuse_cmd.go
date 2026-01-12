package pm5

import (
	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_GOINUSE_CMD = 0x85

func GoInUse() Command {
	return csafe.ShortCommand(csafe_GOINUSE_CMD)
}
