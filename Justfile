# dis development recipes

default:
    @just --list

build:
    go build -o dis ./cmd/dis/

test:
    go test ./... -v

lint:
    golangci-lint run ./...

clean:
    rm -f dis
    go clean

install:
    go install ./cmd/dis/
