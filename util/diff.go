package util

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strconv"
)

// CompareFiles returns the diff between files a, b.
func CompareFiles(a, b string) ([]byte, error) {
	// determine diff tool
	icdiff := true
	diffTool, err := exec.LookPath("icdiff")
	if err != nil {
		diffTool, err = exec.LookPath("diff")
		icdiff = false
	}
	if err != nil || diffTool == "" {
		return nil, errors.New("could not find icdiff or diff on path")
	}
	// build command line options
	opts := []string{"--label", filepath.Base(a), "--label", filepath.Base(b)}
	cols := strconv.Itoa(getColumns())
	if !icdiff {
		opts = append(opts, "--side-by-side", "--width="+cols)
	} else {
		opts = append(opts, "--cols="+cols)
	}
	// log.Printf("DIFF a:%s, b:%s", a, b)
	cmd := exec.Command(diffTool, append(opts, a, b)...)
	buf, err := cmd.CombinedOutput()
	if hasDiff(icdiff, err) {
		return buf, nil
	}
	return nil, nil
}
