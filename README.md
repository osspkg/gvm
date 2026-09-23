# gvm

`gvm` is a Go version manager implemented in Go. It supports Linux, macOS and Windows on amd64 and arm64.

## Install

Unix:

```sh
curl --fail --location https://raw.githubusercontent.com/osspkg/gvm/master/install.sh | bash
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/osspkg/gvm/master/install.ps1 | iex
```

Set `GVM_HOME` before installation to use a custom location. The default is `$HOME/.gvm`.

The installer creates:

```text
$GVM_HOME/
  bin/
  .cache/bin/
  .cache/pkg/
  .cache/src/
```

It adds an idempotent environment block to `.profile`, `.bashrc`, `.zshrc`, and the PowerShell profile where applicable.

## Commands

```text
gvm install [version]       # install a version or the active .gvmrc version
gvm default <version>       # install and select the global version
gvm local <version>         # install and select the project version
gvm venv                    # create .venv and install configured tools
gvm run <binary> [args...]  # run a binary using the active environment
gvm update                  # update gvm and go from the latest GitHub release
gvm version
```

The `go` executable resolves the active configuration, installs the SDK if needed, prepares tools, and executes the selected SDK.

## `.gvmrc`

`.gvmrc` is a restricted dotenv file. It is never sourced by a shell and does not execute substitutions or commands.

```dotenv
GVM_GO_VERSION=1.22.0
GVM_VENV=true
GVM_TOOLS=golang.org/x/tools/gopls@latest
GVM_TOOLS=honnef.co/go/tools/cmd/staticcheck@latest
GOPROXY=https://proxy.golang.org
```

The nearest `.gvmrc` in the current directory or its parents is used. If none exists, `$GVM_HOME/.gvmrc` is used. Process environment values take precedence over configuration files.

When `GVM_VENV=true`, tools are installed in `.venv/bin`; otherwise they are installed in `$GVM_HOME/.cache/bin`. `.venv/` is added to the project `.gitignore` by `gvm venv`.

## Development

```sh
go test ./...
go vet ./...
go build -o bin/gvm ./cmd/gvm
go build -o bin/go ./cmd/go
```

Release CI publishes platform archives and SHA256 checksums for the latest GitHub Release.
