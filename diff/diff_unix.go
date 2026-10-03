//go:build !windows

package diff

import (
	"os/exec"
	"syscall"
	"unsafe"
)

// getColumns returns the columns for the active terminal, or 0 when stdin is
// not a terminal.
func getColumns() int {
	type size struct {
		R uint16
		C uint16
		X uint16
		Y uint16
	}

	ret := new(size)
	code, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(ret)))
	if int(code) == -1 || err != 0 {
		return 0
	}
	return int(ret.C)
}

// hasDiff takes the command result and error and returns true when exit status
// is 1, which both diff and icdiff use to report a difference.
func hasDiff(_ bool, err error) bool {
	if e, ok := err.(*exec.ExitError); ok {
		if status, ok := e.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus() == 1
		}
	}
	return false
}
