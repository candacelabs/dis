package scaffold

import (
	"os"
	"path/filepath"
	"testing"
)

// helper: touch creates an empty file inside dir.
func touch(t *testing.T, dir, name string) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("touch %s: %v", name, err)
	}
	f.Close()
}

func TestDetectProjectType_Node(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "package.json")

	got := DetectProjectType(dir)
	if got != ProjectTypeNode {
		t.Errorf("expected %q, got %q", ProjectTypeNode, got)
	}
}

func TestDetectProjectType_Python_Pyproject(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "pyproject.toml")

	got := DetectProjectType(dir)
	if got != ProjectTypePython {
		t.Errorf("expected %q, got %q", ProjectTypePython, got)
	}
}

func TestDetectProjectType_Python_Requirements(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "requirements.txt")

	got := DetectProjectType(dir)
	if got != ProjectTypePython {
		t.Errorf("expected %q, got %q", ProjectTypePython, got)
	}
}

func TestDetectProjectType_Go(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "go.mod")

	got := DetectProjectType(dir)
	if got != ProjectTypeGo {
		t.Errorf("expected %q, got %q", ProjectTypeGo, got)
	}
}

func TestDetectProjectType_Rust(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "Cargo.toml")

	got := DetectProjectType(dir)
	if got != ProjectTypeRust {
		t.Errorf("expected %q, got %q", ProjectTypeRust, got)
	}
}

func TestDetectProjectType_Unknown(t *testing.T) {
	dir := t.TempDir()

	got := DetectProjectType(dir)
	if got != ProjectTypeUnknown {
		t.Errorf("expected %q, got %q", ProjectTypeUnknown, got)
	}
}

func TestDetectProjectType_Priority(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "package.json")
	touch(t, dir, "go.mod")

	got := DetectProjectType(dir)
	if got != ProjectTypeNode {
		t.Errorf("expected %q (first-match priority), got %q", ProjectTypeNode, got)
	}
}
