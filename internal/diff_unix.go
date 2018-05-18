// +build !windows

package internal

import (
	"os/exec"
	"syscall"
	"unsafe"
)

// GetColumns returns the columns for the active terminal.
func GetColumns() int {
	type size struct {
		R uint16
		C uint16
		X uint16
		Y uint16
	}

	ret := new(size)
	code, _, err := syscall.Syscall(syscall.SYS_IOCTL, uintptr(syscall.Stdin), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(ret)))
	if int(code) == -1 {
		panic(err)
	}
	return int(ret.C)
}

// HasDiff takes the command result and error and returns true when exit status
// is 1.
func HasDiff(icdiff bool, err error) bool {
	if icdiff {
		return err == nil
	}

	if e, ok := err.(*exec.ExitError); ok {
		if status, ok := e.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus() == 1
		}
	}
	return false
}
