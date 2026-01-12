package pm5

import (
	"github.com/seagrayinc/gorow/internal/csafe"
)

const csafe_BADID_CMD = 0x88

func BadID() Command {
	return csafe.ShortCommand(csafe_BADID_CMD)
}
