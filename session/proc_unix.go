// +build !windows

package session

import (
	"os"
	"os/signal"
)

var procAttrs = nil

func terminateProc(process *os.Process) error {
	process.Signal(syscall.SIGINT)
}
