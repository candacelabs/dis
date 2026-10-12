# dis

Dev containers without the baggage.

No VS Code plugin. No Node runtime. No JSON schema with 400 properties.
Just a Dockerfile, a Justfile, and a single binary that gets out of your way.

```
git clone <repo> && just dev && just build
```

That's it. That's the pitch.

Part of [Candace Labs](https://github.com/candacelabs).

---

## Philosophy

Dev containers are a good idea buried under terrible execution. The core promise — *"clone a repo and start coding immediately"* — is right. Everything else about the current ecosystem is wrong:

- **VS Code lock-in** — devcontainer.json is a VS Code feature, not a standard. Use Neovim? JetBrains? Helix? Good luck.
- **Node runtime for a Docker wrapper** — the devcontainer CLI is a Node.js app that shells out to Docker. Why.
- **Config file sprawl** — devcontainer.json, docker-compose.yml, features, extensions, lifecycle scripts, forwarded ports, mount configs. It's a Dockerfile with extra steps.
- **"Features" system** — pulling random shell scripts from a registry and running them as root inside your container. What could go wrong.

`dis` strips all of that away. Your dev environment is a Dockerfile. Your commands go through `just`. Everything else is convention, not configuration.

### Principles

1. **A Dockerfile is the dev environment.** If you can write a Dockerfile, you can use `dis`. No new config language to learn.
2. **Your tools, not ours.** `dis` doesn't mandate a task runner, an editor, or a workflow. If you have `just` installed, `dis init` generates a Justfile with shortcuts. If you don't, `dis run` works fine on its own.
3. **Zero config by default.** Ports, env vars, mounts, shell preference — all declared as `LABEL`s in your Dockerfile. One file, one source of truth.
4. **Edit on host, run in container.** Your code lives on your machine. Your editor is your editor. The container is just a runtime.
5. **Single binary, no dependencies.** `dis` is a Go binary that talks directly to the Docker API. No Node. No Python. No runtime. Just `dis`.

## Install

```bash
go install github.com/candacelabs/dis/cmd/dis@latest
```

Or grab a binary from [releases](https://github.com/candacelabs/dis/releases).

Requires: Docker. Optionally [just](https://github.com/casey/just) for convenient aliases.

## Quick Start

```bash
cd your-project

# Scaffold a .dis/Dockerfile (+ Justfile if just is installed)
dis init

# Start your dev environment
dis up

# Run commands inside the container — transparently
dis run make build
dis run make test
dis run make lint

# Need a shell? Sure
dis shell

# Done for the day
dis down
```

If you have [just](https://github.com/casey/just) installed, `dis init` generates a Justfile so you can use `just dev`, `just build`, `just test` as shortcuts. But it's entirely optional — `dis` works standalone.

## How It Works

### 1. `dis init` scaffolds your project

Detects your project type and generates two files:

**`.dis/Dockerfile`** — a real Dockerfile with `LABEL`s for configuration:
```dockerfile
FROM node:20-bookworm

LABEL dis.ports="3000,5173"
LABEL dis.env="NODE_ENV=development"
LABEL dis.shell="/bin/bash"
LABEL dis.post_create="npm install"

RUN apt-get update && apt-get install -y git curl
WORKDIR /workspace
```

**`Justfile`** *(optional — generated when `just` is installed)* — shortcuts that proxy through `dis`:
```just
# Start the dev environment
dev:
    dis up

# Build the project (inside container)
build *args:
    dis run make build {{args}}

# Run tests (inside container)
test *args:
    dis run make test {{args}}

# Drop into container shell
shell:
    dis shell

# Stop the dev environment
down:
    dis down
```

### 2. `dis up` starts the container

Builds the image from `.dis/Dockerfile` (if needed), starts the container with:
- Bind-mounted project directory at `/workspace`
- Port forwarding from `dis.ports` label
- Environment variables from `dis.env` label
- Git config and credentials forwarded from host
- SSH agent forwarded from host
- Shell history persisted in a named volume

### 3. `dis run` proxies commands

`dis run npm test` executes `npm test` inside the running container. Exit codes propagate correctly. The `Justfile` wraps this so developers just type `just test`.

### 4. `dis` detects when to rebuild

Changed your Dockerfile? `dis up` notices and rebuilds automatically. Or force it with `dis rebuild`.

## Configuration

There is no configuration file. Everything is a Dockerfile `LABEL`:

| Label | Example | Description |
|-------|---------|-------------|
| `dis.ports` | `"3000,5173,8080"` | Ports to forward from container |
| `dis.env` | `"NODE_ENV=development,DEBUG=1"` | Environment variables |
| `dis.mounts` | `"./data:/data,./config:/etc/app"` | Additional bind mounts |
| `dis.tmpfs` | `"/tmp,/home/dev/.cache"` | Container paths mounted in memory, writable by every user, emptied when the container stops |
| `dis.shell` | `"/bin/zsh"` | Default shell for `dis shell` |
| `dis.post_create` | `"npm install && npm run setup"` | Run once after container creation |

No labels? Sensible defaults. `dis` works with a bare Dockerfile.

## Commands

| Command | What it does |
|---------|-------------|
| `dis init` | Scaffold `.dis/Dockerfile` + `Justfile` for your project |
| `dis up` | Build (if needed) and start the dev container |
| `dis down` | Stop and remove the container |
| `dis run <cmd>` | Execute a command inside the container |
| `dis shell` | Open an interactive shell in the container |
| `dis status` | Show container state, ports, mounts |
| `dis rebuild` | Force rebuild the image and restart |
| `dis logs` | Show container logs (`-f` to follow) |

## What dis is NOT

- ❌ Not a cloud IDE (not Gitpod, not Codespaces)
- ❌ Not a VS Code extension
- ❌ Not a Kubernetes tool
- ❌ Not a package manager (not Nix, not devbox)
- ❌ No "features" marketplace (no random scripts from the internet)
- ❌ No Windows native support (use WSL2)
- ❌ No GUI, no web dashboard, no electron app

## Compared to devcontainers

| | devcontainers | dis |
|---|---|---|
| Config | JSON + YAML + lifecycle scripts | Dockerfile LABELs |
| Runtime | Node.js CLI | Single Go binary |
| Editor | VS Code (or suffer) | Any — your code is on host |
| Dependencies | Node, npm, VS Code | Docker (just optional) |
| "Features" | Registry of shell scripts | Write your own Dockerfile |
| Learning curve | devcontainer.json schema | Dockerfile (you already know it) |
| Transparency | Magic | `dis run` = `docker exec` |

## Contributing

```bash
just build      # compile
just test       # run tests
just lint       # lint
```

## License

MIT
