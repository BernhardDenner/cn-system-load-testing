# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`cn-system-load-testing` is a cloud native system load testing tool written in Go.

This repository is in early development — no source code exists yet. When implementation begins, update this file with build/test/lint commands, module layout, and architectural decisions.

## Planned Language & Tooling

- **Language:** Go
- **Gitignore:** Pre-configured for Go (binaries, test cache, vendor/, go.work, .env)

## Getting Started

Once Go modules are initialized, typical commands will be:

```bash
go mod init github.com/...    # initialize Go modules (fill in module path)
go build ./...                # build all packages
go test ./...                 # run all tests
go vet ./...                  # static analysis
```
