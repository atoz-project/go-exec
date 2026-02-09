# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

本项目 `github.com/atoz-project/go-exec` 源自 `github.com/FalconOpsLLC/goexec`，在其基础上进行二次开发。

go-exec 是一个 Windows 远程执行多功能工具，通过 Windows RPC 协议（SCMR、TSCH、WMI、DCOM）实现多种远程执行方式，底层使用 go-msrpc 库。用于授权场景下的 Windows 远程命令执行。

## Build & Run

```bash
# Build (always use CGO_ENABLED=0)
CGO_ENABLED=0 go build -ldflags="-s -w" -trimpath

# Install
CGO_ENABLED=0 go install -ldflags="-s -w"

# Run tests
CGO_ENABLED=0 go test ./...

# Run single package test
CGO_ENABLED=0 go test ./pkg/goexec/... -v

# Docker build
docker build . --tag goexec --network host
```

No CGO dependencies — always build with `CGO_ENABLED=0`.

## Architecture

### Module System

The CLI uses cobra with a **module → method** hierarchy: `goexec <module> <method> [target] [flags]`

Four protocol modules, each in `pkg/goexec/<module>/`:

- **scmr** — Service Control Manager (create/change/delete Windows services)
- **tsch** — Task Scheduler (create/demand/change scheduled tasks)
- **wmi** — WMI process creation and method calls
- **dcom** — Multiple DCOM objects (mmc, shellwindows, shellbrowserwindow, htafile, excel, visualstudio)

### Key Interfaces (`pkg/goexec/`)

- `Method` — base: `Connect(ctx)` + `Init(ctx)`
- `ExecutionMethod` — extends Method with `Execute(ctx, *ExecutionIO)`
- `AuxiliaryMethod` — extends Method with `Call(ctx)` (non-execution operations like `scmr delete`, `wmi call`)
- `Clean` variants wrap methods with automatic cleanup via `Cleaner`

Execution flow: `ExecuteCleanMethod()` → `Connect()` → `Init()` → `Execute()` → `Clean()` → output collection

### Client Layer (`pkg/goexec/dce/`, `pkg/goexec/smb/`)

- `dce.Client` — DCE/RPC client wrapping go-msrpc, handles EPM endpoint discovery, SMB/TCP transport, NTLM/Kerberos auth
- `smb.Client` — SMB2 client for output file retrieval via ADMIN$ share

### CLI Wiring (`cmd/`)

- `root.go` — cobra root command, flag sets, auth/logging/proxy init
- `args.go` — reusable argument validators (`argsTarget`, `argsRpcClient`, `argsSmbClient`, `argsOutput`)
- Each module file (`scmr.go`, `tsch.go`, `wmi.go`, `dcom.go`) wires flags and cobra commands to module structs

### Auth

Authentication via `adauth` library — supports password, NT hash, Kerberos AES key, PFX certificate, CCache file. Flags registered globally on root command.

### Output Collection

Optional process output capture: wraps command in `cmd.exe /c ... > tempfile`, then fetches via SMB (`OutputProvider` interface). Configured through `ExecutionIO`.

## Code Conventions

- Module name: `github.com/atoz-project/go-exec`
- `internal/util/` — small helpers (random hostname/string generation)
- `pkg/goexec/` — public library interfaces and types
- Logging: `zerolog` with structured fields, context-propagated loggers
- Flag organization: custom `flagSet` groups with labeled sections in help output
