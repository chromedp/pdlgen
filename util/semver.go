package util

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
)

// VerRE is a regular expression for matching version numbers.
var VerRE = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(\.[0-9]+)?$`)

// MakeSemver makes a semver for v.
func MakeSemver(v string) *semver.Version {
	// replace last . with -
	if strings.Count(v, ".") > 2 {
		n := strings.LastIndex(v, ".")
		v = v[:n] + "-" + v[n+1:]
	}
	ver, err := semver.NewVersion(v)
	if err != nil {
		panic(fmt.Sprintf("could not make %s into semver: %v", v, err))
	}
	return ver
}

// CompareSemver returns true if the semver of a is less than the semver of b.
func CompareSemver(a, b string) bool {
	return MakeSemver(b).GreaterThan(MakeSemver(a))
}
