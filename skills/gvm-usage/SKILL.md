---
name: gvm-usage
description: Use this repository's gvm Go version manager to select, install, inspect, remove, and run Go SDKs and project tools; apply when work involves GVM_HOME, .gvmrc, venv, SDK selection, gvm list/rm/remove-all, or wrapper diagnostics.
---

# GVM Usage

Use this skill when a task requires operating the repository's `gvm` Go version manager or diagnosing its Go wrapper. Do not activate it for unrelated Go development merely because the project uses Go.

## Core contract

- `GVM_HOME` is read from the process environment; when unset, it defaults to `$HOME/.gvm`.
- The manager supports Linux, macOS, and Windows on amd64 and arm64.
- A selected SDK is stored at `$GVM_HOME/.cache/src/go<version>`.
- Global installed tools are stored in `$GVM_HOME/.cache/bin`; the module cache is `$GVM_HOME/.cache/pkg`.
- `GVM_GO_VERSION` is required. Do not assume that the system Go version is the project version.
- `gvm` and `go` are normally the managed binaries in `$GVM_HOME/bin`.

Before changing state, discover the active installation:

`@sh
gvm version
gvm list
printf '%s\n' "$GVM_HOME"
command -v gvm
command -v go
`

If `GVM_HOME` is custom, export it before running manager commands. On PowerShell use `$env:GVM_HOME`.

## Configuration and `.gvmrc`

`.gvmrc` is a dotenv-style configuration file, not a shell script.

Configuration precedence is:

1. Process environment.
2. The nearest `.gvmrc` in the current directory or its parents.
3. `$GVM_HOME/.gvmrc`.
4. Built-in defaults.

The parser does not execute commands, substitutions, or shell code. Never use `source`, `eval`, or an equivalent operation on `.gvmrc`.

A typical project configuration is:

`dotenv
GVM_GO_VERSION=1.22.0
GVM_VENV=true
GVM_TOOL=golang.org/x/tools/gopls@latest
GVM_TOOL=honnef.co/go/tools/cmd/staticcheck@latest
GOPROXY=https://proxy.golang.org
`

`GVM_TOOL` may be repeated; each occurrence adds one tool. Preserve unrelated variables when changing only `GVM_GO_VERSION`.
The former plural key `GVM_TOOLS` is not migrated; rename it manually in existing `.gvmrc` files.

Use:

`sh
gvm default 1.22.0
`

to install the SDK and update the global `$GVM_HOME/.gvmrc`, or:

`sh
gvm local 1.22.0
`

to install the SDK and update `.gvmrc` in the current project directory. Both commands preserve other configuration entries.

## Installation and SDK lifecycle

Unix bootstrap:

`sh
curl --fail --location https://raw.githubusercontent.com/osspkg/gvm/master/install.sh | bash
`

PowerShell bootstrap:

`powershell
irm https://raw.githubusercontent.com/osspkg/gvm/master/install.ps1 | iex
`

For a custom location:

`sh
export GVM_HOME="$HOME/.local/gvm"
`

`powershell
$env:GVM_HOME = "$HOME\.local\gvm"
`

After installation, reload the shell profile if `gvm` is not found.

Install one SDK explicitly:

`sh
gvm install 1.22.0
`

With no version argument, `gvm install` uses the active `.gvmrc`.

Install the newest stable release from the official Go metadata:

`sh
gvm install latest
`

List installed SDKs:

`sh
gvm list
`

The list contains only complete SDK installations. Temporary, incomplete, and unrelated entries under `.cache/src` are ignored.

Remove exactly one SDK:

`sh
gvm rm 1.22.0
`

Pass a version, never a filesystem path. The command validates the version and refuses traversal or symlink targets. It does not edit `.gvmrc`. If the removed SDK is still selected, the next managed `go` invocation may install it again.

Because `rm` is destructive, ask for confirmation when the user has not explicitly requested removal. For an explicit removal request, remove only the specified version and verify that other SDK directories remain.

Remove all installed SDKs explicitly:

`sh
gvm remove-all
`

This preserves `$GVM_HOME/.cache/bin`, `$GVM_HOME/.cache/pkg`, and `.gvmrc`; it only removes versioned SDK directories.

Update the manager binaries without changing installed SDKs:

`sh
gvm update
`

## Virtual environments and tools

Create a project Go virtual environment with:

`sh
gvm venv
`

This creates `<project>/.venv/bin`, adds `.venv/` to an existing `.gitignore` idempotently, enables `GVM_VENV=true` in the local configuration, and installs the configured `GVM_TOOL`.

When `GVM_VENV=true`, tools are installed into the project `.venv/bin`. Otherwise tools use the global `$GVM_HOME/.cache/bin`.

Use the managed wrapper for Go commands:

`sh
go version
go env GOROOT GOPATH GOMODCACHE GOBIN
`

Use `gvm run` for a tool or binary with the same resolved configuration:

`sh
gvm run gopls ./...
gvm run staticcheck ./...
`

Arguments after the binary name are passed through unchanged.

## Runtime environment and PATH

For an active SDK, the wrapper computes:

`text
GOROOT=$GVM_HOME/.cache/src/go<version>
GOPATH=$GVM_HOME/.cache
GOMODCACHE=$GVM_HOME/.cache/pkg
GOBIN=$GVM_HOME/.cache/bin
`

With a project virtual environment, `GOBIN` becomes `<project>/.venv/bin`.

The effective `PATH` is ordered as follows:

1. `<project>/.venv/bin` when the virtual environment is enabled.
2. `$GVM_HOME/.cache/bin`.
3. `$GVM_HOME/.cache/src/go<version>/bin`.
4. `$GVM_HOME/bin`.
5. The inherited PATH, with empty elements and duplicates removed.

This ordering ensures project tools win over global tools, SDK commands are available, and the managed wrapper wins over a system Go binary.

## Recommended workflow

1. Check `GVM_HOME`, `gvm version`, and `gvm list`.
2. Select a project SDK with `gvm local VERSION`, or configure the global default with `gvm default VERSION`.
3. Put project-specific Go settings and repeated `GVM_TOOL` entries in `.gvmrc`.
4. Run `gvm venv` when tools must be isolated per project.
5. Verify `go version` and `go env GOROOT GOPATH GOMODCACHE GOBIN`.
6. Use `go` for Go commands and `gvm run BINARY ...` for managed binaries.

## Troubleshooting

- `GVM_GO_VERSION is required`: create a local or global `.gvmrc`, or set `GVM_GO_VERSION` in the process environment.
- The wrong `go` is running: inspect `command -v go`, ensure `$GVM_HOME/bin` precedes the system PATH, reload the profile, and rerun `gvm version`.
- An SDK is missing: run `gvm list`, then `gvm install VERSION`.
- A tool is missing: inspect `GVM_TOOL`, enable the project environment with `gvm venv`, and retry through `gvm run`.
- A removed SDK reappears: it is still selected by `.gvmrc`; select another version or remove the corresponding configuration entry.

Prefer read-only diagnostics before mutating installation state. Do not manually delete SDK directories with `rm -rf`; use `gvm rm` with the exact version. Do not alter unrelated environment variables or project configuration entries.

