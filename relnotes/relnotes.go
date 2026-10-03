// Package relnotes turns the output of apidiff into the notes of a release of
// cdproto: the message of the commit, the annotation of the tag and the entry
// of the changelog.
package relnotes

import (
	"bufio"
	"fmt"
	"io"
	"slices"
	"strings"
)

// Kind is the kind of a change.
type Kind string

// Kinds of change.
const (
	Added   Kind = "added"
	Removed Kind = "removed"
	Changed Kind = "changed"
)

// Change is one change to an exported name.
type Change struct {
	// Package is the name of the package, such as "input".
	Package string
	// Name is the exported name, such as "DispatchKeyEventParams.Type". A
	// method of a pointer type is "Type.Method".
	Name string
	// Kind is the kind of the change.
	Kind Kind
	// Detail is the text after "changed from", for a changed name.
	Detail string
	// Incompatible is true when apidiff lists the change as incompatible.
	Incompatible bool
}

// Report is the list of the changes between two versions of a module.
type Report struct {
	Changes []Change
}

// Parse reads the output of apidiff -m.
func Parse(r io.Reader) (*Report, error) {
	rep := new(Report)
	var incompatible bool
	sc := bufio.NewScanner(r)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case line == "":
		case line == "Incompatible changes:":
			incompatible = true
		case line == "Compatible changes:":
			incompatible = false
		case strings.HasPrefix(line, "- ./"):
			c, err := parseLine(strings.TrimPrefix(line, "- ./"))
			if err != nil {
				return nil, err
			}
			c.Incompatible = incompatible
			rep.Changes = append(rep.Changes, c)
		default:
			return nil, fmt.Errorf("unexpected line %q", line)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("reading apidiff output: %w", err)
	}
	return rep, nil
}

// parseLine parses "pkg.Name: removed" and its relatives.
func parseLine(s string) (Change, error) {
	name, what, ok := strings.Cut(s, ": ")
	if !ok {
		return Change{}, fmt.Errorf("unexpected change %q", s)
	}
	var c Change
	c.Package, c.Name, _ = strings.Cut(name, ".")
	// a method of a pointer type is written (*Type).Method
	c.Name = strings.NewReplacer("(*", "", ")", "").Replace(c.Name)
	switch {
	case what == "removed":
		c.Kind = Removed
	case what == "added":
		c.Kind = Added
	case strings.HasPrefix(what, "changed from "):
		c.Kind, c.Detail = Changed, strings.Replace(strings.TrimPrefix(what, "changed from "), " to ", " -> ", 1)
	default:
		c.Kind, c.Detail = Changed, what
	}
	return c, nil
}

// Count returns the number of incompatible changes and the number of
// compatible changes.
func (r *Report) Count() (incompatible, compatible int) {
	for _, c := range r.Changes {
		if c.Incompatible {
			incompatible++
		} else {
			compatible++
		}
	}
	return incompatible, compatible
}

// Meta describes a release.
type Meta struct {
	// Version is the version of the release, and Previous is the version of the
	// last release, which is empty for the first.
	Version, Previous string
	// Chromium and V8 are the versions of the protocol, and PrevChromium and
	// PrevV8 are the versions of the previous release.
	Chromium, V8, PrevChromium, PrevV8 string
	// Date is the date of the release, as 2006-01-02.
	Date string
	// AddedPackages and RemovedPackages are the packages that the release adds
	// and removes.
	AddedPackages, RemovedPackages []string
}

// maxNames is the number of names that a list of one package shows before it
// says how many more there are.
const maxNames = 12

// Commit returns the message of the commit of a release.
func (r *Report) Commit(m Meta) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Updating to %s_%s definitions\n\n", m.Chromium, m.V8)
	fmt.Fprintf(&b, "- Release: %s\n", m.Version)
	b.WriteString(r.body(m, true))
	return b.String()
}

// Tag returns the annotation of the tag of a release, with its subject.
func (r *Report) Tag(m Meta) string {
	return fmt.Sprintf("cdproto %s\n\n%s", m.Version, r.body(m, true))
}

// Changelog returns the entry of a release for CHANGELOG.md.
func (r *Report) Changelog(m Meta) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s - %s\n\n", m.Version, m.Date)
	b.WriteString(r.body(m, false))
	return b.String()
}

// body writes the text of the notes. The lists of names are in the full text,
// and not in the shorter one.
func (r *Report) body(m Meta, full bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "- Chromium: %s%s\n", m.Chromium, was(m.PrevChromium, m.Chromium))
	fmt.Fprintf(&b, "- V8: %s%s\n", m.V8, was(m.PrevV8, m.V8))
	if m.Previous == "" {
		b.WriteString("- API changes: first tagged release\n")
	} else {
		inc, comp := r.Count()
		fmt.Fprintf(&b, "- API changes since %s: %d incompatible, %d compatible\n", m.Previous, inc, comp)
	}
	if len(m.AddedPackages) != 0 {
		fmt.Fprintf(&b, "- Packages added: %s\n", strings.Join(m.AddedPackages, ", "))
	}
	if len(m.RemovedPackages) != 0 {
		fmt.Fprintf(&b, "- Packages removed: %s\n", strings.Join(m.RemovedPackages, ", "))
	}
	if len(r.Changes) == 0 {
		return b.String()
	}
	b.WriteString("\nBy package, as removed, changed and added names:\n\n")
	for _, p := range r.packages() {
		fmt.Fprintf(&b, "  %-24s %4d removed %4d changed %4d added\n", p.name, p.count(Removed), p.count(Changed), p.count(Added))
	}
	if !full {
		return b.String()
	}
	for _, section := range []struct {
		title string
		kind  Kind
	}{{"Removed", Removed}, {"Changed", Changed}, {"Added", Added}} {
		var lines []string
		for _, p := range r.packages() {
			if names := p.names(section.kind); len(names) != 0 {
				lines = append(lines, fmt.Sprintf("  %s: %s", p.name, strings.Join(names, ", ")))
			}
		}
		if len(lines) != 0 {
			fmt.Fprintf(&b, "\n%s:\n\n%s\n", section.title, strings.Join(lines, "\n"))
		}
	}
	return b.String()
}

// was returns " (was prev)" when prev is a different version.
func was(prev, cur string) string {
	if prev == "" || prev == cur {
		return ""
	}
	return " (was " + prev + ")"
}

// pkg holds the changes of one package.
type pkg struct {
	name    string
	changes []Change
}

// count returns the number of changes of the kind.
func (p *pkg) count(k Kind) int {
	var n int
	for _, c := range p.changes {
		if c.Kind == k {
			n++
		}
	}
	return n
}

// names returns the changed names of the kind, ready to print. A method that
// many types in the package lose or gain together, such as UnmarshalJSON, is
// one entry that counts the types.
func (p *pkg) names(k Kind) []string {
	methods := make(map[string][]string)
	var names []string
	for _, c := range p.changes {
		if c.Kind != k {
			continue
		}
		if typ, method, ok := strings.Cut(c.Name, "."); ok && k != Changed && method != "" && !strings.Contains(method, ".") {
			methods[method] = append(methods[method], typ)
			continue
		}
		if c.Detail != "" {
			names = append(names, c.Name+" ("+c.Detail+")")
		} else {
			names = append(names, c.Name)
		}
	}
	for method, types := range methods {
		if len(types) > 3 {
			names = append(names, fmt.Sprintf("%s on %d types", method, len(types)))
			continue
		}
		for _, typ := range types {
			names = append(names, typ+"."+method)
		}
	}
	slices.Sort(names)
	if len(names) > maxNames {
		more := len(names) - maxNames
		names = append(names[:maxNames:maxNames], fmt.Sprintf("and %d more", more))
	}
	return names
}

// packages returns the changes grouped by package, in order of the package
// name.
func (r *Report) packages() []*pkg {
	byName := make(map[string]*pkg)
	var list []*pkg
	for _, c := range r.Changes {
		p := byName[c.Package]
		if p == nil {
			p = &pkg{name: c.Package}
			byName[c.Package] = p
			list = append(list, p)
		}
		p.changes = append(p.changes, c)
	}
	slices.SortFunc(list, func(a, b *pkg) int { return strings.Compare(a.name, b.name) })
	return list
}
