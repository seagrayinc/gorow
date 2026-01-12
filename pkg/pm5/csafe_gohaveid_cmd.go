package pm5

import (
	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_GOHAVEID_CMD = 0x83

func GoHaveID() Command {
	return csafe.ShortCommand(csafe_GOHAVEID_CMD)
}
