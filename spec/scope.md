# dis — Scope Decisions

## What dis IS

A simple, plug-and-play alternative to VS Code dev containers. Single binary (Go),
zero config (convention over configuration via Dockerfile LABELs), transparent command
proxying through `just` so the developer never has to manually `docker exec` into anything.

**Core UX**: `git clone <repo> && just dev && just build` — that's it.

## All Design Decisions (Locked)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Interaction model | Hybrid — background container + transparent proxy via `just` | Best of both worlds: invisible when running commands, accessible when debugging |
| Name | `dis` | "dis up", "dis mount", "dis connect", "dis shell". Part of Candace Labs. |
| Config format | Zero config — convention via `.dis/Dockerfile` LABELs | Eliminates config file drift, single source of truth |
| Multi-service | User writes compose, dis owns dev container | dis doesn't reinvent orchestration |
| Filesystem | Bind mount (code on host) | Edit on host, run in container. macOS perf tradeoff accepted. |
| Target audience | Team onboarding (clone → one command → working env) | Good error messages, auto-detection, `dis init` scaffolder |
| Editor integration | None for MVP | Edit on host with whatever editor you want |
| Language | Go with Docker Go SDK | Single binary, no runtime deps, direct Docker API access |
| Relationship to just | just is required, dis generates the Justfile | just IS the UX layer, dis IS the engine |

## What dis is NOT

- ❌ Not a cloud hosting service (not Gitpod/Coder/Codespaces)
- ❌ Not a Kubernetes tool
- ❌ Not a package manager (not Nix)
- ❌ Not a VS Code extension
- ❌ No "features" system (no pulling random scripts from registries)
- ❌ No Windows native support (WSL2 only)
- ❌ No GUI or web dashboard
- ❌ No custom image registry
- ❌ No remote container support

## MVP Scope

The MVP delivers the core UX loop:

1. `dis init` — scaffold `.dis/Dockerfile` + `Justfile` with project type detection
2. `dis up` — build image (if needed) + start container + run post_create hooks
3. `dis run <cmd>` — execute command inside running container
4. `dis shell` — drop into interactive shell
5. `dis down` — stop and remove container
6. `dis status` — show container status, ports, mounts
7. `dis rebuild` — force rebuild + restart
8. `dis logs` — show container logs
9. `dis nuts` / `dis aster` — easter eggs

### MVP Features
- Dockerfile LABEL parsing for ports, env, mounts, shell, post_create
- Git credential forwarding (mount host git config)
- SSH agent forwarding (mount SSH_AUTH_SOCK)
- Shell history persistence (named Docker volume)
- Port forwarding from LABELs
- Smart rebuild (detect Dockerfile changes)
- Project type detection (Node/Python/Go/Rust)

### v2 (NOT in MVP)
- Docker Compose integration (detect + manage alongside)
- Custom template registry
- Team config sharing
- Health check endpoints
- Container resource limits from LABELs
- macOS filesystem performance optimization (mutagen/etc)
