# dis

Simple, plug-and-play dev containers. Part of [Candace Labs](https://github.com/candacelabs).

## What is dis?

A single Go binary that makes dev containers dead simple. No config files — just a
`.dis/Dockerfile` with LABELs. Commands proxy transparently through `just` recipes.

```bash
git clone <repo>
just dev        # start dev environment
just build      # runs inside container, transparently
just shell      # drop into container when you need to
just down       # stop everything
```

## Tech Stack

| Layer | Choice |
|-------|--------|
| Language | Go 1.22+ |
| CLI | cobra |
| Docker | Docker Go SDK (github.com/docker/docker/client) |
| Testing | stdlib + testify |
| Build | `just` (Justfile in repo root) |
| Release | goreleaser |

## Project Structure

```
dis/
├── cmd/dis/              # CLI entrypoint + cobra commands
│   ├── main.go           # Execute()
│   ├── root.go           # Root command + version
│   ├── init.go           # dis init
│   ├── up.go             # dis up
│   ├── down.go           # dis down
│   ├── run.go            # dis run <cmd>
│   ├── shell.go          # dis shell
│   ├── status.go         # dis status
│   ├── rebuild.go        # dis rebuild
│   ├── logs.go           # dis logs
│   ├── easter.go         # dis nuts / dis aster
│   └── helpers.go        # mustResolveConfig, mustCreateManager
├── internal/
│   ├── config/
│   │   └── convention.go # DisConfig, ResolveConfig, DockerfileExists
│   ├── container/
│   │   ├── manager.go    # Docker lifecycle: Up, Down, Exec, Status, Rebuild, Logs
│   │   ├── labels.go     # ParseLabelsFromDockerfile
│   │   └── health.go     # WaitForReady
│   ├── scaffold/
│   │   ├── detect.go     # DetectProjectType (Node/Python/Go/Rust/Unknown)
│   │   ├── templates.go  # DockerfileTemplate per language
│   │   ├── init.go       # Init orchestration
│   │   └── justfile.go   # GenerateJustfile
│   ├── proxy/
│   │   └── exec.go       # RunCommand, InteractiveShell
│   └── forward/
│       ├── git.go        # Git credential forwarding
│       ├── ssh.go        # SSH agent forwarding
│       ├── ports.go      # Port binding from labels
│       └── history.go    # Shell history volume
├── spec/                 # Implementation plan (you are here)
│   ├── plan.yaml         # Canonical DAG
│   ├── scope.md          # Scope decisions
│   └── contracts/        # Proto contracts for agent I/O
├── go.mod
├── Justfile              # dis's own build/test recipes
├── README.md
├── LICENSE
└── .goreleaser.yml
```

## Execution Plan

This project has a DAG-based execution plan at `spec/plan.yaml`.
Proto contracts defining agent I/O boundaries are in `spec/contracts/`.

To execute: read `spec/plan.yaml`, launch worker agents by level (level 0 first, in parallel),
validate output protos at each boundary, gate on tests passing before advancing to the next level.

### DAG Overview

```
Level 0 (parallel):  A0:bootstrap  A1:config  A2:templates  A3:forwarding
                          │            │  │         │              │
Level 1 (parallel):       └──────┬─────┘  └────┬────┘             │
                                 │              │                  │
                           A4:container    A5:scaffold             │
                                 │                                 │
Level 2:                   A6:proxy ◄──────────────────────────────┘
                                 │
Level 3:                   A7:cli_wiring
```

### Build Order
1. **Level 0** — A0 (bootstrap), A1 (config), A2 (templates), A3 (forwarding) — all parallel
2. **Level 1** — A4 (container manager), A5 (scaffold init) — parallel
3. **Level 2** — A6 (proxy exec)
4. **Level 3** — A7 (CLI wiring) — final integration

### Gate Commands
- Level 0: `go test ./internal/config/ ./internal/container/ ./internal/scaffold/ ./internal/forward/`
- Level 1: `go test ./internal/container/ ./internal/scaffold/`
- Level 2: `go test ./internal/proxy/`
- Level 3: `go build ./cmd/dis/ && go test ./... && ./dis --version`

## Conventions

- **Zero config**: All configuration via `.dis/Dockerfile` LABELs — no config files
- **Error messages**: Always suggest the fix (e.g., "Run 'dis init' to set up")
- **Idempotent commands**: `dis up` on running container is a no-op, `dis down` on stopped container is a no-op
- **Exit codes**: `dis run` exits with the same code as the proxied command
- **Docker SDK**: All Docker interaction through Go SDK, never shelling out to `docker` CLI
- **Testing**: Unit tests with mocked Docker client, testify assertions
- **No global state**: Config is always passed explicitly through function parameters

## Quick Commands

```bash
just build      # compile dis binary
just test       # run all tests
just lint       # run golangci-lint
just clean      # remove binary
just install    # go install
```
