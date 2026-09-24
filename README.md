# gvm

[![CI](https://github.com/osspkg/gvm/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/osspkg/gvm/actions/workflows/ci.yml)
[![Latest Release](https://img.shields.io/github/v/release/osspkg/gvm?sort=semver)](https://github.com/osspkg/gvm/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/osspkg/gvm)](https://go.dev/)
[![License](https://img.shields.io/github/license/osspkg/gvm)](LICENSE)

`gvm` is a per-user Go version manager implemented in Go. It installs and selects Go SDKs, prepares project-specific environments, manages Go tools, and runs binaries with the selected environment.

Supported platforms:

- Linux: amd64, arm64
- macOS: amd64, arm64
- Windows: amd64, arm64

## Features

- Install Go SDKs from the official Go download metadata.
- Install the latest stable Go SDK with `gvm install latest`.
- Verify SDK archives with SHA-256 checksums before installation.
- Select global and project-local Go versions.
- Discover `.gvmrc` files from the current directory up to the filesystem root.
- Use safe dotenv configuration without shell evaluation or command execution.
- Create project virtual environments in `.venv/bin`.
- Install tools from repeated `GVM_TOOL` entries.
- Run binaries with deterministic, deduplicated `PATH` ordering.
- List and remove installed SDK versions.
- Update the `gvm`, `go`, and `gofmt` manager binaries from GitHub Releases.
- Use managed `go` and `gofmt` wrappers with recursive-launch protection.

## Installation

### Linux and macOS

The installer uses the latest GitHub Release and configures `.profile`, `.bashrc`, and `.zshrc`.

```sh
curl --fail --location https://raw.githubusercontent.com/osspkg/gvm/master/install.sh | bash
```

To use a custom installation directory:

```sh
export GVM_HOME="$HOME/tools/gvm"
curl --fail --location https://raw.githubusercontent.com/osspkg/gvm/master/install.sh | bash
```

Open a new shell, or reload the profile after installation:

```sh
source "$HOME/.profile"
```

### Windows PowerShell

The installer downloads the latest Windows release, sets the user-level `GVM_HOME` and `PATH`, and updates the PowerShell profile.

```powershell
irm https://raw.githubusercontent.com/osspkg/gvm/master/install.ps1 | iex
```

For a custom installation directory:

```powershell
$env:GVM_HOME = Join-Path $HOME "tools\gvm"
irm https://raw.githubusercontent.com/osspkg/gvm/master/install.ps1 | iex
```

Start a new PowerShell session after installation.

## Quick start

Select a global Go version and use it in a project:

```sh
gvm default 1.22.0

mkdir -p ~/src/example
cd ~/src/example

gvm local 1.22.0
go version
```

Create a project environment with tools:

```sh
cat > .gvmrc <<'EOF'
GVM_GO_VERSION=1.22.0
GVM_VENV=true
GVM_TOOL=golang.org/x/tools/gopls@latest
GVM_TOOL=honnef.co/go/tools/cmd/staticcheck@latest
GOPROXY=https://proxy.golang.org
EOF

gvm venv
go version
gvm run gopls version
```

`gvm venv` creates `.venv/bin`, adds `.venv/` to `.gitignore`, and installs the configured tools into the project environment.

For every newly installed SDK, gvm preserves the official executables as `bin/go.bin` and `bin/gofmt.bin`, then places the gvm `go` and `gofmt` wrappers at `bin/go` and `bin/gofmt`. This keeps IDEs that discover the SDK by its `GOROOT` path inside the managed environment. Existing SDKs are not migrated; the behavior applies only to newly installed SDKs.

The `go` and `gofmt` wrappers pass the internal `GVM_WRAPPER_ACTIVE=1` marker to the real Go process. If another managed wrapper is invoked from that process, it directly uses the matching `GOROOT/bin/<tool>.bin` and does not resolve the configuration again.

## Commands

| Command | Description |
| --- | --- |
| `gvm install [version|latest]` | Install a Go SDK. Without a version, use `GVM_GO_VERSION` from the active `.gvmrc`. |
| `gvm list` | List complete SDKs installed in `$GVM_HOME/.cache/src`. |
| `gvm rm <version>` | Remove one installed SDK, for example `gvm rm 1.21.0`. |
| `gvm remove-all` | Remove all installed SDKs while preserving tools, module cache, and configuration. |
| `gvm default <version>` | Install an SDK and configure the global `$GVM_HOME/.gvmrc`. |
| `gvm local [version]` | Install an SDK and configure `.gvmrc` in the current directory. Without a version, infer it from `go.work` first, then `go.mod`. |
| `gvm venv` | Create `.venv/bin`, enable `GVM_VENV=true`, and install configured tools. |
| `gvm run <binary> [args...]` | Run a binary using the active SDK environment. |
| `gvm update` | Update or repair the `gvm`, `go`, and `gofmt` manager binaries from the latest GitHub Release. |
| `gvm version` | Print the gvm version. |
| `go [args...]` | Run the selected Go SDK with the active environment. |
| `gofmt [args...]` | Run the selected SDK's gofmt with the active environment. |

`gvm rm` accepts a Go version, not a filesystem path. It does not edit `.gvmrc`; if the removed version remains selected, the next `go` invocation may install it again.

`gvm install latest` queries the official Go metadata and installs the newest stable release available for the current platform.

## Configuration

### `GVM_HOME`

`GVM_HOME` selects the per-user installation directory. If it is not set, gvm uses:

```text
$HOME/.gvm
```

The installers export `GVM_HOME` and prepend `$GVM_HOME/bin` to `PATH` in the supported shell or PowerShell profiles.

### `.gvmrc`

`.gvmrc` is a restricted dotenv file. It is parsed as data and is never sourced by a shell. Shell commands, command substitution, and executable expressions are not evaluated.

Example:

```dotenv
GVM_GO_VERSION=1.22.0
GVM_VENV=true
GVM_TOOL=golang.org/x/tools/gopls@latest
GVM_TOOL=honnef.co/go/tools/cmd/staticcheck@latest
GOPROXY=https://proxy.golang.org
```

Configuration resolution works as follows:

1. Process environment values have the highest priority.
2. The nearest `.gvmrc` in the current directory or a parent directory is selected.
3. If no local `.gvmrc` exists, `$GVM_HOME/.gvmrc` is used.
4. `GVM_VENV` defaults to `false` and `GVM_TOOL` defaults to empty.
5. `GVM_GO_VERSION` is required; there is no implicit Go SDK version for normal `go` invocations.

Repeated `GVM_TOOL` entries are preserved in order. The legacy plain-text format containing only a version number is invalid and is not migrated automatically.
The former plural key `GVM_TOOLS` is not migrated; rename it manually to `GVM_TOOL` in existing `.gvmrc` files.

When `gvm local` is called without a version, it searches the current directory and its parents for `go.work` first, then `go.mod`. The first valid `go` directive becomes `GVM_GO_VERSION`; if both files are available, `go.work` wins. Passing a version explicitly remains an override.

### Managed environment

For the active SDK, gvm computes these variables:

```text
GOROOT=$GVM_HOME/.cache/src/go<version>
GOPATH=$GVM_HOME/.cache
GOMODCACHE=$GVM_HOME/.cache/pkg
GOBIN=$GVM_HOME/.cache/bin
```

When `GVM_VENV=true`, `GOBIN` becomes `<project>/.venv/bin`.

The managed `PATH` order is (the active SDK bin is included after the global tool bin):

```text
<project>/.venv/bin   # when GVM_VENV=true
$GVM_HOME/.cache/bin
$GVM_HOME/.cache/src/go<version>/bin
$GVM_HOME/bin
inherited PATH without empty entries or duplicates
```

The path separator is selected for the host platform automatically.

## Directory layout

```text
$GVM_HOME/
├── bin/
│   ├── gvm
│   ├── go
│   └── gofmt
└── .cache/
    ├── bin/              # global tools installed with go install
    ├── pkg/              # Go module cache
    └── src/
        └── go<version>/  # installed Go SDK; bin/go and bin/gofmt are gvm wrappers
                          # bin/go.bin and bin/gofmt.bin are the official SDK executables
```

Project-local state:

```text
<project>/
├── .gvmrc
├── .gitignore
└── .venv/
    └── bin/
```

`gvm venv` adds `.venv/` to `.gitignore` idempotently.

## Running binaries

`gvm run`, the `go` wrapper, and the `gofmt` wrapper use the same environment construction. Binary lookup order is:

1. `<project>/.venv/bin`
2. `$GVM_HOME/.cache/bin`
3. `$GVM_HOME/.cache/src/go<version>/bin`
4. `$GVM_HOME/bin`
5. The inherited `PATH`

Each path is included once, and empty path components are removed.

## Updates and releases

`gvm update` retrieves the latest release from `github.com/osspkg/gvm`, selects the archive for the current operating system and architecture, verifies `checksums.txt`, and updates or repairs only the manager binaries (`gvm`, `go`, and `gofmt`); if one is missing, the current release is downloaded again. Installed Go SDKs are not modified.

Release archives use these names:

```text
gvm_<version>_<os>_<arch>.tar.gz  # Linux and macOS
gvm_<version>_<os>_<arch>.zip     # Windows
```

The release workflow publishes builds for all six supported targets and includes SHA-256 checksums.

## Development

Requirements:

- Go 1.26.8 or newer
- Bash for the Unix installer and local CI commands

Run the standard checks:

```sh
go fmt ./...
go test ./...
go test -race ./...
go vet ./...
git diff --check
```

Build the command binaries:

```sh
go build -o bin/gvm ./cmd/gvm
go build -o bin/go ./cmd/go
go build -o bin/gofmt ./cmd/gofmt
```

Cross-compile an individual target:

```sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./cmd/gvm
```

CI configuration is in [.github/workflows/ci.yml](.github/workflows/ci.yml), and release packaging is in [.github/workflows/release.yml](.github/workflows/release.yml).

## Contributing

Bug reports, feature requests, and pull requests are welcome. Please include the platform, architecture, Go version, command used, and relevant error output when reporting a problem.

## License

gvm is distributed under the [BSD 3-Clause License](LICENSE).
