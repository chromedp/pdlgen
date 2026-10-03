package docs_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// root is the repository root.
const root = ".."

// markdown returns the paths of every markdown file in the repository, except
// those under the skill folders, which are not ours to edit.
func markdown(t *testing.T) []string {
	t.Helper()
	var v []string
	err := filepath.WalkDir(root, func(name string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir() && (d.Name() == ".git" || d.Name() == ".agents" || d.Name() == ".claude"):
			return filepath.SkipDir
		case !d.IsDir() && strings.HasSuffix(name, ".md"):
			v = append(v, name)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	return v
}

func TestEveryMarkdownLinkResolves(t *testing.T) {
	re := regexp.MustCompile(`\]\(([^)#\s]+)(?:#[^)]*)?\)`)
	for _, name := range markdown(t) {
		buf, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		for _, m := range re.FindAllStringSubmatch(string(buf), -1) {
			link := m[1]
			if strings.Contains(link, "://") || strings.HasPrefix(link, "mailto:") {
				continue
			}
			target := filepath.Join(filepath.Dir(name), link)
			if strings.HasPrefix(link, "/") {
				target = filepath.Join(root, link)
			}
			if _, err := os.Stat(target); err != nil {
				t.Errorf("%s: link %q does not resolve", name, link)
			}
		}
	}
}

func TestTheRootHoldsFiveDocuments(t *testing.T) {
	want := map[string]bool{"README.md": true, "AGENTS.md": true, "CLAUDE.md": true, "CONTRIBUTING.md": true}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") && !want[e.Name()] {
			t.Errorf("%s belongs in docs", e.Name())
		}
	}
	for name := range want {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Errorf("expected %s in the root: %v", name, err)
		}
	}
}

func TestClaudeImportsAgents(t *testing.T) {
	buf, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if got := strings.TrimSpace(string(buf)); got != "@AGENTS.md" {
		t.Errorf("expected CLAUDE.md to hold only @AGENTS.md, got: %q", got)
	}
}

func TestTheDecisionIndexIsComplete(t *testing.T) {
	dir := filepath.Join(root, "docs", "decisions")
	index, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "2*.md"))
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	name := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-[a-z0-9-]+\.md$`)
	for _, f := range files {
		base := filepath.Base(f)
		if !name.MatchString(base) {
			t.Errorf("%s must be named YYYY-MM-DD-short-title.md", base)
			continue
		}
		buf, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		lines := strings.Split(string(buf), "\n")
		if !strings.HasPrefix(lines[0], "# ") || lines[1] != "" || !strings.HasPrefix(lines[2], "Status: ") {
			t.Errorf("%s must open with a title, a blank line and a status", base)
			continue
		}
		title := strings.TrimPrefix(lines[0], "# ")
		status := strings.TrimSuffix(strings.TrimPrefix(lines[2], "Status: "), ".")
		row := "| " + base[:10] + " | [" + title + "](" + base + ") | " + status + " |"
		if !bytes.Contains(index, []byte(row)) {
			t.Errorf("the index lacks this row, or has it wrong:\n%s", row)
		}
	}
	for _, m := range regexp.MustCompile(`\]\((2[^)]+\.md)\)`).FindAllSubmatch(index, -1) {
		if _, err := os.Stat(filepath.Join(dir, string(m[1]))); err != nil {
			t.Errorf("the index names %s, which does not exist", m[1])
		}
	}
}

func TestSkillsAreCopies(t *testing.T) {
	var a, b []string
	for _, d := range []struct {
		dir string
		out *[]string
	}{{".agents", &a}, {".claude", &b}} {
		base := filepath.Join(root, d.dir, "skills")
		err := filepath.WalkDir(base, func(name string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if e.Type()&fs.ModeSymlink != 0 {
				t.Errorf("%s is a link, and it must be a copy", name)
			}
			if !e.IsDir() {
				rel, _ := filepath.Rel(base, name)
				buf, err := os.ReadFile(name)
				if err != nil {
					return err
				}
				*d.out = append(*d.out, rel+"\x00"+string(buf))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	}
	if strings.Join(a, "\n") != strings.Join(b, "\n") {
		t.Errorf("the skills in .agents and .claude differ")
	}
}

// proseRules are the rules of the simple-english skill that a regular
// expression can check. The rest need a person.
var proseRules = []struct {
	name string
	re   *regexp.Regexp
	fix  string
}{
	{"modal", regexp.MustCompile(`(?i)\b(?:should|would|may|might|could)\b`), "use can, will or must"},
	{"semicolon", regexp.MustCompile(`;`), "write two sentences"},
	{"dash", regexp.MustCompile("—|–|\\s--\\s"), "write two sentences"},
	{"contraction", regexp.MustCompile(`(?i)\b[a-z]+n't\b|\b(?:it|that|there|here|what|let)'s\b|\b(?:i|you|we|they)'(?:re|ve|ll|d|m)\b`), "write the words in full"},
	{"bold", regexp.MustCompile(`\*\*`), "remove the bold"},
	{"perfect", regexp.MustCompile(`(?i)\b(?:has|have) been\b`), "use the simple past or the present"},
	{"latin", regexp.MustCompile(`(?i)\b(?:e\.g\.|i\.e\.|etc\.|ie,|eg,)`), "write the words in full"},
}

// prose strips fenced code blocks, inline code, quoted text and link targets.
func prose(s string) string {
	s = regexp.MustCompile("(?s)```.*?```").ReplaceAllString(s, "")
	s = regexp.MustCompile("`[^`]*`").ReplaceAllString(s, "")
	s = regexp.MustCompile(`"[^"\n]*"`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`\]\([^)]*\)`).ReplaceAllString(s, "]")
	s = regexp.MustCompile(`(?m)^\[[^\]]+\]: .*$`).ReplaceAllString(s, "")
	return s
}

func TestProseIsSimpleEnglish(t *testing.T) {
	for _, name := range markdown(t) {
		buf, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		for i, line := range strings.Split(prose(string(buf)), "\n") {
			for _, r := range proseRules {
				if m := r.re.FindString(line); m != "" {
					t.Errorf("%s: line %d: %s %q: %s", name, i+1, r.name, m, r.fix)
				}
			}
		}
	}
}
