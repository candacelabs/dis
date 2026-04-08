package scaffold

// DockerfileTemplate returns a Dockerfile template string appropriate for the
// given ProjectType.  Every template starts with a FROM directive and ends with
// a WORKDIR /workspace directive.  Language-specific dis.* LABELs are included
// so that the dis runtime can discover ports, shells, and post-create hooks.
func DockerfileTemplate(projectType ProjectType) string {
	switch projectType {
	case ProjectTypeNode:
		return nodeDockerfile
	case ProjectTypePython:
		return pythonDockerfile
	case ProjectTypeGo:
		return goDockerfile
	case ProjectTypeRust:
		return rustDockerfile
	default:
		return unknownDockerfile
	}
}

const nodeDockerfile = `FROM node:20-bookworm

LABEL dis.ports="3000"
LABEL dis.shell="/bin/bash"
LABEL dis.post_create="npm install"

RUN apt-get update && apt-get install -y git curl && rm -rf /var/lib/apt/lists/*
WORKDIR /workspace
`

const pythonDockerfile = `FROM python:3.12-bookworm

LABEL dis.shell="/bin/bash"
LABEL dis.post_create="pip install -r requirements.txt"

RUN apt-get update && apt-get install -y git curl && rm -rf /var/lib/apt/lists/*
WORKDIR /workspace
`

const goDockerfile = `FROM golang:1.22-bookworm

LABEL dis.ports="8080"
LABEL dis.shell="/bin/bash"

RUN apt-get update && apt-get install -y git curl && rm -rf /var/lib/apt/lists/*
WORKDIR /workspace
`

const rustDockerfile = `FROM rust:1.77-bookworm

LABEL dis.ports="8080"
LABEL dis.shell="/bin/bash"

RUN apt-get update && apt-get install -y git curl pkg-config libssl-dev && rm -rf /var/lib/apt/lists/*
WORKDIR /workspace
`

const unknownDockerfile = `FROM ubuntu:22.04

LABEL dis.shell="/bin/bash"

RUN apt-get update && apt-get install -y git curl build-essential && rm -rf /var/lib/apt/lists/*
WORKDIR /workspace
`
