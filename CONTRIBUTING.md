# Contributing

Thanks for your interest in contributing!

## How to Contribute

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## Development Setup

Nix (recommended, hermetic — build + full test suite + lint):

    nix build
    nix flake check

Plain Go toolchain (CI parity — note: `GOEXPERIMENT=jsonv2` is required
because go-finding imports `encoding/json/v2`, and CGO is off; a plain
`go test -race ./...` fails on this module):

    GOEXPERIMENT=jsonv2 CGO_ENABLED=0 go test -count=1 ./...

Lint: `golangci-lint run ./...` (the repo-wide gate is being burned down —
see TODO_LIST.md; new code should stay clean).

## Reporting Issues

Please use GitHub Issues to report bugs or request features.
