package scaffold

import (
	"strings"
	"testing"
)

func TestGenerateJustfile_HasDevRecipe(t *testing.T) {
	jf := GenerateJustfile()
	if !strings.Contains(jf, "dev:") {
		t.Error("Justfile missing dev: recipe")
	}
	if !strings.Contains(jf, "dis up") {
		t.Error("Justfile dev recipe missing 'dis up'")
	}
}

func TestGenerateJustfile_HasShellRecipe(t *testing.T) {
	jf := GenerateJustfile()
	if !strings.Contains(jf, "shell:") {
		t.Error("Justfile missing shell: recipe")
	}
	if !strings.Contains(jf, "dis shell") {
		t.Error("Justfile shell recipe missing 'dis shell'")
	}
}

func TestGenerateJustfile_HasRunProxy(t *testing.T) {
	jf := GenerateJustfile()
	if !strings.Contains(jf, "run *args:") {
		t.Error("Justfile missing run *args: recipe")
	}
	if !strings.Contains(jf, "dis run") {
		t.Error("Justfile run recipe missing 'dis run'")
	}
}

func TestGenerateJustfile_HasBuildRecipe(t *testing.T) {
	jf := GenerateJustfile()
	if !strings.Contains(jf, "build") {
		t.Error("Justfile missing build recipe")
	}
	if !strings.Contains(jf, "dis run make build") {
		t.Error("Justfile build recipe missing 'dis run make build'")
	}
}

func TestGenerateJustfile_HasTestRecipe(t *testing.T) {
	jf := GenerateJustfile()
	if !strings.Contains(jf, "test") {
		t.Error("Justfile missing test recipe")
	}
	if !strings.Contains(jf, "dis run make test") {
		t.Error("Justfile test recipe missing 'dis run make test'")
	}
}
