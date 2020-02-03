// +build !windows

package session

import (
	"os"
	"syscall"
)

var procAttrs = &syscall.SysProcAttr{}

func terminateProc(process *os.Process) error {
	return process.Signal(syscall.SIGINT)
}
