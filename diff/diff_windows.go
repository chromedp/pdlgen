// +build windows

package diff

// getColumns returns the columns for the active terminal.
func getColumns() int {
	return 0
}

// hasDiff takes the command result and error and returns true when exit status
// is 1.
func hasDiff(bool, error) bool {
	return false
}
