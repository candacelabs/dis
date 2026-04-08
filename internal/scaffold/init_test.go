package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit_NewProject(t *testing.T) {
	dir := t.TempDir()

	err := Init(dir, InitOptions{})
	if err != nil {
		t.Fatalf("Init returned unexpected error: %v", err)
	}

	// .dis/Dockerfile must exist
	dockerfilePath := filepath.Join(dir, ".dis", "Dockerfile")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		t.Error(".dis/Dockerfile was not created")
	}

	// Justfile must exist
	justfilePath := filepath.Join(dir, "Justfile")
	if _, err := os.Stat(justfilePath); os.IsNotExist(err) {
		t.Error("Justfile was not created")
	}
}

func TestInit_DetectsNode(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "package.json")

	err := Init(dir, InitOptions{})
	if err != nil {
		t.Fatalf("Init returned unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".dis", "Dockerfile"))
	if err != nil {
		t.Fatalf("failed to read Dockerfile: %v", err)
	}

	if !strings.Contains(string(content), "FROM node:") {
		t.Error("Dockerfile should contain 'FROM node:' for a Node project")
	}
}

func TestInit_DetectsPython(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "requirements.txt")

	err := Init(dir, InitOptions{})
	if err != nil {
		t.Fatalf("Init returned unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".dis", "Dockerfile"))
	if err != nil {
		t.Fatalf("failed to read Dockerfile: %v", err)
	}

	if !strings.Contains(string(content), "FROM python:") {
		t.Error("Dockerfile should contain 'FROM python:' for a Python project")
	}
}

func TestInit_ExistingDockerfile_NoForce(t *testing.T) {
	dir := t.TempDir()

	// Pre-create .dis/Dockerfile
	disDir := filepath.Join(dir, ".dis")
	if err := os.MkdirAll(disDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(disDir, "Dockerfile"), []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	err := Init(dir, InitOptions{Force: false})
	if err == nil {
		t.Fatal("expected error when .dis/Dockerfile exists without --force")
	}

	if !strings.Contains(err.Error(), "already initialized") {
		t.Errorf("error message should mention 'already initialized', got: %v", err)
	}
}

func TestInit_ExistingDockerfile_Force(t *testing.T) {
	dir := t.TempDir()

	// Pre-create .dis/Dockerfile with old content
	disDir := filepath.Join(dir, ".dis")
	if err := os.MkdirAll(disDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(disDir, "Dockerfile"), []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	err := Init(dir, InitOptions{Force: true})
	if err != nil {
		t.Fatalf("Init with --force returned unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(disDir, "Dockerfile"))
	if err != nil {
		t.Fatalf("failed to read Dockerfile: %v", err)
	}

	if string(content) == "old content" {
		t.Error("Dockerfile should have been overwritten with --force")
	}

	if !strings.Contains(string(content), "FROM") {
		t.Error("overwritten Dockerfile should contain a FROM directive")
	}
}

func TestInit_ExistingJustfile_NoForce(t *testing.T) {
	dir := t.TempDir()

	// Pre-create a Justfile
	originalContent := "# my custom justfile"
	if err := os.WriteFile(filepath.Join(dir, "Justfile"), []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	err := Init(dir, InitOptions{})
	if err != nil {
		t.Fatalf("Init returned unexpected error: %v", err)
	}

	// Dockerfile should still be created
	dockerfilePath := filepath.Join(dir, ".dis", "Dockerfile")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		t.Error(".dis/Dockerfile was not created")
	}

	// Justfile should NOT be overwritten
	content, err := os.ReadFile(filepath.Join(dir, "Justfile"))
	if err != nil {
		t.Fatalf("failed to read Justfile: %v", err)
	}

	if string(content) != originalContent {
		t.Error("Justfile should not have been overwritten without --force")
	}
}

func TestInit_OverrideProjectType(t *testing.T) {
	dir := t.TempDir()

	// Create a Node marker, but override to Go
	touch(t, dir, "package.json")

	err := Init(dir, InitOptions{ProjectType: ProjectTypeGo})
	if err != nil {
		t.Fatalf("Init returned unexpected error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".dis", "Dockerfile"))
	if err != nil {
		t.Fatalf("failed to read Dockerfile: %v", err)
	}

	if !strings.Contains(string(content), "FROM golang:") {
		t.Error("Dockerfile should contain 'FROM golang:' when ProjectType is overridden to Go")
	}

	if strings.Contains(string(content), "FROM node:") {
		t.Error("Dockerfile should NOT contain 'FROM node:' when ProjectType is overridden to Go")
	}
}

func TestInit_CreatesDisDirectory(t *testing.T) {
	dir := t.TempDir()

	// Confirm .dis/ does not exist
	disDir := filepath.Join(dir, ".dis")
	if _, err := os.Stat(disDir); !os.IsNotExist(err) {
		t.Fatal(".dis/ should not exist before Init")
	}

	err := Init(dir, InitOptions{})
	if err != nil {
		t.Fatalf("Init returned unexpected error: %v", err)
	}

	info, err := os.Stat(disDir)
	if os.IsNotExist(err) {
		t.Fatal(".dis/ directory was not created")
	}
	if !info.IsDir() {
		t.Error(".dis should be a directory")
	}
}
