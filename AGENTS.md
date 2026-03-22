# Repository Guidelines

## Project Structure & Module Organization

- `main.go`: CLI entrypoint.
- `cmd/`: Cobra commands and flag wiring (e.g., `cmd/scmr.go`, `cmd/tsch.go`).
- `pkg/goexec/`: Library code used by the CLI.
  - Protocol/modules live under `pkg/goexec/{smb,scmr,tsch,dcom,wmi,dce}`.
- `internal/`: Private helpers (e.g., `internal/util`).
- Tests live next to code as `*_test.go` under `cmd/` and `pkg/`.

## Build, Test, and Development Commands

- `go test ./...`: run all unit tests.
- `CGO_ENABLED=0 go test ./...`: matches CI (runs on Ubuntu + Windows).
- `go run . --help`: run the CLI locally.
- `CGO_ENABLED=0 go build ./...`: sanity-check builds without CGO.
- `gofmt -w $(git ls-files '*.go')`: format all Go files (required).

## Coding Style & Naming Conventions

- Go style: `gofmt` formatting, idiomatic names (`CamelCase` exports, short locals).
- Error handling: no `panic` for expected/configuration errors; wrap with context
  (`fmt.Errorf("open service: %w", err)`).
- Contexts: propagate `context.Context`; avoid `context.TODO()` in production code.
- Cleanup: prefer `goexec.Cleaner`/`AddCleaners` and ensure cleanup runs on error
  paths. `Cleaner` is LIFO (defer-like); aggregate multiple cleanup errors with
  `errors.Join`.

## Testing Guidelines

- Use the standard `testing` package; table-driven tests where it fits.
- Keep tests hermetic: avoid requiring a live Windows target or network access.
  Prefer fakes, small seams, and unit tests for edge cases (timeouts, cleanup, etc.).
- Run a focused package while iterating: `go test ./pkg/goexec/scmr -run TestName`.

## Commit & Pull Request Guidelines

- Prefer Conventional Commits as used in recent history: `fix:`, `feat:`,
  `refactor:`, `test:`, `ci:`.
- PRs should include:
  - What changed and why (behavior, compatibility impact).
  - How to test (commands + expected outcome).
  - Tests updated/added for bug fixes.

## Security & Configuration Notes

This project interacts with remote execution and credentials. Avoid logging
secrets, use timeouts for network operations, and default to least-privilege
access rights when calling Windows RPC/SMB APIs.
