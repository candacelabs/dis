package scaffold

import (
	"strings"
	"testing"
)

func TestDockerfileTemplate_Node(t *testing.T) {
	tmpl := DockerfileTemplate(ProjectTypeNode)

	for _, want := range []string{"FROM node:", "dis.ports", "dis.post_create"} {
		if !strings.Contains(tmpl, want) {
			t.Errorf("Node template missing %q", want)
		}
	}
}

func TestDockerfileTemplate_Python(t *testing.T) {
	tmpl := DockerfileTemplate(ProjectTypePython)

	for _, want := range []string{"FROM python:", "dis.post_create"} {
		if !strings.Contains(tmpl, want) {
			t.Errorf("Python template missing %q", want)
		}
	}
}

func TestDockerfileTemplate_Go(t *testing.T) {
	tmpl := DockerfileTemplate(ProjectTypeGo)

	if !strings.Contains(tmpl, "FROM golang:") {
		t.Error("Go template missing FROM golang:")
	}
}

func TestDockerfileTemplate_Rust(t *testing.T) {
	tmpl := DockerfileTemplate(ProjectTypeRust)

	for _, want := range []string{"FROM rust:", "pkg-config"} {
		if !strings.Contains(tmpl, want) {
			t.Errorf("Rust template missing %q", want)
		}
	}
}

func TestDockerfileTemplate_Unknown(t *testing.T) {
	tmpl := DockerfileTemplate(ProjectTypeUnknown)

	if !strings.Contains(tmpl, "FROM ubuntu:") {
		t.Error("Unknown template missing FROM ubuntu:")
	}
}

func TestDockerfileTemplate_AllHaveWorkdir(t *testing.T) {
	for _, pt := range []ProjectType{
		ProjectTypeNode,
		ProjectTypePython,
		ProjectTypeGo,
		ProjectTypeRust,
		ProjectTypeUnknown,
	} {
		tmpl := DockerfileTemplate(pt)
		if !strings.Contains(tmpl, "WORKDIR /workspace") {
			t.Errorf("%s template missing WORKDIR /workspace", pt)
		}
	}
}

func TestDockerfileTemplate_AllHaveShell(t *testing.T) {
	for _, pt := range []ProjectType{
		ProjectTypeNode,
		ProjectTypePython,
		ProjectTypeGo,
		ProjectTypeRust,
		ProjectTypeUnknown,
	} {
		tmpl := DockerfileTemplate(pt)
		if !strings.Contains(tmpl, "dis.shell") {
			t.Errorf("%s template missing dis.shell LABEL", pt)
		}
	}
}
