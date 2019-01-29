package diff

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
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

// FileInfo contains file information.
type FileInfo struct {
	Name string
	Info os.FileInfo
}

// String satisfies fmt.Stringer.
func (fi *FileInfo) String() string {
	return filepath.Base(fi.Name)
}

// FindFilesWithMask walks dir finding all files with the regexp mask, removing
// any exclude'd files.
func FindFilesWithMask(dir, mask string, exclude ...string) ([]*FileInfo, error) {
	var maskRE = regexp.MustCompile(mask)

	// build list of protocol files on disk
	var files []*FileInfo
	dir = strings.TrimSuffix(dir, string(os.PathSeparator)) + string(os.PathSeparator)
	err := filepath.Walk(dir, func(n string, fi os.FileInfo, err error) error {
		switch {
		case os.IsNotExist(err) || n == dir:
			return nil
		case err != nil:
			return err
		case fi.IsDir():
			return nil
		}

		// skip if same as current or doesn't match file mask
		fn := n[len(dir):]
		if !maskRE.MatchString(fn) || contains(exclude, filepath.Base(fn)) {
			return nil
		}

		// add to files
		files = append(files, &FileInfo{
			Name: n,
			Info: fi,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

// WalkAndCompare walks dir, looking for files matching the supplied regexp
// mask, successively comparing each against filename. The first having a diff
// (compared by most recent first) will be returned.
//
// Useful for comparing multiple files to find the most recent difference from
// a set of files matching mask that likely have the same content.
func WalkAndCompare(dir, mask string, filename string, cmp func(*FileInfo, *FileInfo) bool) ([]byte, error) {
	files, err := FindFilesWithMask(dir, mask)
	if err != nil {
		return nil, err
	}

	// if nothing to process, bail
	if len(files) == 0 {
		return nil, nil
	}
	// sort most recent
	sort.Slice(files, func(a, b int) bool {
		return cmp(files[a], files[b])
	})

	// find filename in files
	var i int
	for ; i < len(files); i++ {
		if filepath.Base(files[i].Name) == filepath.Base(filename) {
			break
		}
	}

	// compare and return first with diff
	for i--; i >= 0; i-- {
		buf, err := CompareFiles(files[i].Name, filename)
		if err != nil {
			return nil, err
		}
		if buf != nil && len(bytes.TrimSpace(buf)) > 0 {
			return buf, nil
		}
	}

	return nil, nil
}

// contains determines if s is defined in v.
func contains(v []string, s string) bool {
	for _, z := range v {
		if z == s {
			return true
		}
	}
	return false
}
