# Agent instructions

## Repository scope

- Work from the repository root.
- This is the Go module `github.com/osspkg/gvm`. The minimum declared Go version is 1.26.8; CI and lint configuration use Go 1.26.8.
- Keep command entrypoints in `cmd/gvm`, `cmd/go`, and `cmd/gofmt`.
- Keep the `gofmt` wrapper entrypoint in `cmd/gofmt`.
- Keep implementation packages private under `internal/`:
  - `internal/app` orchestrates CLI commands.
  - `internal/config` parses and writes safe dotenv-style `.gvmrc` files.
  - `internal/env` computes the managed runtime environment and PATH.
  - `internal/sdk` installs, lists, validates, and removes Go SDKs.
  - `internal/venv` manages project virtual environments and tools.
  - `internal/release` handles GitHub release archives and updates.
  - `internal/progress`, `internal/runner`, and `internal/version` provide shared runtime behavior.
- Treat `README.md`, `install.sh`, `install.ps1`, and `.github/workflows/` as public operational contracts.

## Configuration and safety contracts

- Parse `.gvmrc` as data. Never add shell sourcing, `eval`, command substitution, or executable configuration behavior.
- Preserve unrelated environment entries when changing only `GVM_GO_VERSION`, `GVM_VENV`, or `GVM_TOOL`.
- When `gvm local` creates a project-local `.gvmrc` without an explicit version, inspect `go.work` first and `go.mod` second, taking the valid `go` directive as `GVM_GO_VERSION`. If both files exist, `go.work` wins; if neither has a valid directive, return an actionable error instead of selecting the system Go implicitly.
- Preserve the resolution order: process environment, nearest local `.gvmrc` from the current directory or its parents, global `$GVM_HOME/.gvmrc`, then defaults. Managed `GOROOT`, `GOPATH`, `GOMODCACHE`, `GOBIN`, and `PATH` must not be overridden by project config.
- The Go and gofmt wrappers set `GVM_WRAPPER_ACTIVE=1` for the selected real Go process; a nested wrapper must use the matching `GOROOT/bin/<tool>.bin` directly and return an error when that preserved binary is absent.
- Keep SDKs under `$GVM_HOME/.cache/src/go<version>`, global tools under `$GVM_HOME/.cache/bin`, module cache under `$GVM_HOME/.cache/pkg`, and manager binaries (`gvm`, `go`, and `gofmt`) under `$GVM_HOME/bin`.
- SDK installation, release updates, and tool installation must stage work safely and clean up temporary files on failure.
- Validate SDK versions before constructing paths. Do not turn user-supplied versions into arbitrary filesystem paths.
- Keep `gvm rm <version>` limited to the requested SDK. Never replace it with a broad recursive deletion or a path-based delete.
- Do not use the real user `GVM_HOME`, shell profiles, or live GitHub releases for tests. Use `t.TempDir()`, injected environments, and fake HTTP servers.

## Code and test conventions

- Use standard Go packages and platform-aware APIs such as `filepath` and `os.PathListSeparator`; do not hard-code Unix path separators in cross-platform code.
- Put tests next to the package they cover in `*_test.go` files. Prefer table-driven tests and temporary directories for filesystem behavior.
- Cover malformed configuration, path traversal, checksum failures, incomplete SDKs, atomic cleanup, PATH ordering, and argument forwarding when changing those areas.
- Keep network tests deterministic with fake HTTP servers. Do not make unit or integration tests depend on the public Go download API or GitHub.
- Update the public README when changing CLI commands, configuration precedence, cache layout, supported platforms, release assets, or installer behavior.
- Avoid committing generated or local output from `bin/`, `dist/`, `.cache/`, `build/`, `coverage.*`, or binaries.

## Development commands

Run commands from the repository root.

For a focused package test:

```sh
go test ./internal/config
go test ./internal/sdk
```

For normal validation:

```sh
go fmt ./...
go test ./...
go vet ./...
git diff --check
```

Before handoff, also run the race suite when the change affects shared state, filesystem lifecycle, process execution, or concurrency:

```sh
go test -race ./...
```

The repository's CI-equivalent target is:

```sh
make ci
```

`make ci` installs `goppy`, runs license setup, linting, tests, and the build pipeline. It may modify generated/license or build outputs; inspect `git status` afterward. The underlying targets are available individually as `make license`, `make lint`, `make tests`, and `make build`.

Build all manager binaries (`gvm`, `go`, and `gofmt`) locally with:

```sh
go build -o bin/gvm ./cmd/gvm
go build -o bin/go ./cmd/go
go build -o bin/gofmt ./cmd/gofmt
```

The release workflow cross-builds these binaries for Linux, macOS, and Windows on amd64 and arm64. When changing release packaging, keep archive names, `checksums.txt`, executable suffixes, and the matrix in `.github/workflows/release.yml` synchronized with the installers and update client.

## Installer and release boundaries

- Keep `install.sh` and `install.ps1` thin and repeatable. They create `GVM_HOME` layout, download a published release, install `gvm`, `go`, and `gofmt`, and configure user profiles/environment variables idempotently.
- Test installer syntax with `bash -n install.sh`. Do not execute installers against the real home directory during validation; use an isolated home or inspect the script.
- Changes to GitHub release behavior must preserve SHA-256 verification and atomic replacement of manager binaries (`gvm`, `go`, and `gofmt`) without altering installed SDKs.
- `gvm update` must repair a missing manager binary even when the installed version is already the latest release.
- Do not run publication, deployment, profile mutation, or release commands merely as validation. These actions require an explicit request.

## Handoff checklist

- Re-read the changed code and documentation.
- Run formatting, focused tests, and the broader checks appropriate to the change.
- Run `go test -race ./...` for lifecycle, process, filesystem, or concurrency changes.
- Run `bash -n install.sh` when the Unix installer changes; use a PowerShell syntax check when the Windows installer changes and the tool is available.
- Run `git diff --check` and inspect `git status` for accidental artifacts.
- Report checks that actually ran and distinguish unavailable platform-specific checks from successful checks.

