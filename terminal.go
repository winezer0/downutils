package downutils

import (
	"golang.org/x/term"
	"os"
)

func terminalProgressAvailable() bool {
	return term.IsTerminal(int(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb"
}
