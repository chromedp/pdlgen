// +build windows

package internal

// GetColumns returns the columns for the active terminal.
func GetColumns() int {
	return 0
}

// HasDiff takes the command result and error and returns true when exit status
// is 1.
func HasDiff(bool, error) bool {
	return false
}
