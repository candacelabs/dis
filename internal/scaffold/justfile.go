package scaffold

// GenerateJustfile returns the contents of a standard dis Justfile with
// recipes that proxy common development commands through the dis container.
func GenerateJustfile() string {
	return justfileContent
}

const justfileContent = `# Development environment powered by dis (https://github.com/candacelabs/dis)

default:
    @just --list

# Start the dev environment
dev:
    dis up

# Drop into container shell
shell:
    dis shell

# Stop the dev environment
down:
    dis down

# Proxy any command through the dev container
run *args:
    dis run {{args}}

# Build the project (inside container)
build *args:
    dis run make build {{args}}

# Run tests (inside container)
test *args:
    dis run make test {{args}}

# Run linter (inside container)
lint:
    dis run make lint

# Show dev environment status
status:
    dis status

# Force rebuild dev environment
rebuild:
    dis rebuild
`
