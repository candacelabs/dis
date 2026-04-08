package scaffold

import (
	"os"
	"path/filepath"
)

// ProjectType represents the detected type of a software project.
type ProjectType string

const (
	ProjectTypeNode    ProjectType = "node"
	ProjectTypePython  ProjectType = "python"
	ProjectTypeGo      ProjectType = "go"
	ProjectTypeRust    ProjectType = "rust"
	ProjectTypeUnknown ProjectType = "unknown"
)

// DetectProjectType inspects projectRoot for well-known marker files and
// returns the first matching ProjectType.  Detection order (first match wins):
//  1. package.json        → Node
//  2. pyproject.toml / requirements.txt / setup.py → Python
//  3. go.mod              → Go
//  4. Cargo.toml          → Rust
//  5. (none of the above) → Unknown
func DetectProjectType(projectRoot string) ProjectType {
	// Node
	if fileExists(projectRoot, "package.json") {
		return ProjectTypeNode
	}

	// Python – any of the three markers
	if fileExists(projectRoot, "pyproject.toml") ||
		fileExists(projectRoot, "requirements.txt") ||
		fileExists(projectRoot, "setup.py") {
		return ProjectTypePython
	}

	// Go
	if fileExists(projectRoot, "go.mod") {
		return ProjectTypeGo
	}

	// Rust
	if fileExists(projectRoot, "Cargo.toml") {
		return ProjectTypeRust
	}

	return ProjectTypeUnknown
}

// fileExists returns true when name exists directly inside dir.
func fileExists(dir, name string) bool {
	info, err := os.Stat(filepath.Join(dir, name))
	if err != nil {
		return false
	}
	return !info.IsDir()
}
