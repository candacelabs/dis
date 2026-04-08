# dis

Simple, plug-and-play dev containers. Part of [Candace Labs](https://github.com/candacelabs).

## Install

```bash
go install github.com/candacelabs/dis/cmd/dis@latest
```

## Quick Start

```bash
cd your-project
dis init        # scaffold .dis/Dockerfile + Justfile
just dev        # start dev environment
just build      # run build inside container
just shell      # drop into container shell
just down       # stop container
```

## License

MIT
