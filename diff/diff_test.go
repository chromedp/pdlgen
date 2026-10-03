package diff

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareFilesWithoutTerminal(t *testing.T) {
	if _, err := exec.LookPath("diff"); err != nil {
		t.Skip("diff is not on the path")
	}
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.pdl"), filepath.Join(dir, "b.pdl")
	if err := os.WriteFile(a, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if err := os.WriteFile(b, []byte("one\n2\nthree\n"), 0o644); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	// stdin is not a terminal under test, which used to panic
	buf, err := CompareFiles(a, b)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(string(buf), "two") {
		t.Errorf("expected the changed line, got:\n%s", buf)
	}
	if buf, err = CompareFiles(a, a); err != nil || len(strings.TrimSpace(string(buf))) != 0 {
		t.Errorf("expected no difference, got: %q, %v", buf, err)
	}
}
