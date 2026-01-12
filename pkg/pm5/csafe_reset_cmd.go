package pm5

import (
	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_RESET_CMD = 0x81

func Reset() Command {
	return csafe.ShortCommand(csafe_RESET_CMD)
}
