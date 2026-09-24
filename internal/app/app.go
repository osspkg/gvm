package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/osspkg/gvm/internal/config"
	"github.com/osspkg/gvm/internal/env"
	"github.com/osspkg/gvm/internal/progress"
	"github.com/osspkg/gvm/internal/release"
	"github.com/osspkg/gvm/internal/runner"
	"github.com/osspkg/gvm/internal/sdk"
	"github.com/osspkg/gvm/internal/venv"
	"github.com/osspkg/gvm/internal/version"
)

type App struct {
	Out        io.Writer
	Err        io.Writer
	Environ    []string
	CWD        string
	HTTP       *http.Client
	Progress   *progress.Reporter
	Executable string
	Release    *release.Client
}

func New(out, errOut io.Writer) (*App, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current directory: %w", err)
	}
	if out == nil {
		out = os.Stdout
	}
	if errOut == nil {
		errOut = os.Stderr
	}
	client := &http.Client{Timeout: 15 * time.Minute}
	reporter := progress.New(out, isTerminal(out))
	return &App{
		Out:        out,
		Err:        errOut,
		Environ:    os.Environ(),
		CWD:        cwd,
		HTTP:       client,
		Progress:   reporter,
		Executable: "",
		Release:    release.NewClient(client, reporter),
	}, nil
}

func (a *App) RunGVM(ctx context.Context, args []string) error {
	if len(args) == 0 {
		a.printHelp()
		return nil
	}
	if args[0] == "__apply-update" {
		if len(args) != 3 {
			return errors.New("usage: gvm __apply-update <staging> <bin>")
		}
		return a.applyStagedUpdate(args[1], args[2])
	}
	switch args[0] {
	case "help", "-h", "--help":
		a.printHelp()
		return nil
	case "version", "--version":
		_, err := fmt.Fprintln(a.Out, version.Value)
		return err
	case "install":
		return a.install(ctx, args[1:])
	case "list":
		return a.listSDKs(args[1:])
	case "rm":
		return a.removeSDK(args[1:])
	case "default":
		home, err := a.home()
		if err != nil {
			return err
		}
		return a.setVersion(ctx, args[1:], filepath.Join(home, ".gvmrc"))
	case "local":
		return a.setLocalVersion(ctx, args[1:])
	case "venv":
		return a.createVenv(ctx, args[1:])
	case "run":
		return a.runBinary(ctx, args[1:])
	case "update":
		return a.update(ctx)
	default:
		return fmt.Errorf("unknown command %q; use `gvm help`", args[0])
	}
}

func (a *App) RunGo(ctx context.Context, args []string) error {
	cfg, err := a.resolveConfig()
	if err != nil {
		return err
	}
	sdkRoot, err := a.ensureSDK(ctx, cfg.GoVersion)
	if err != nil {
		return err
	}
	prepared, err := a.prepareRuntime(ctx, cfg, sdkRoot)
	if err != nil {
		return err
	}
	goBinary, err := runner.Executable(filepath.Join(sdkRoot, "bin"), "go")
	if err != nil {
		return err
	}
	return runner.Run(ctx, goBinary, a.CWD, args, prepared.Values, a.Out, a.Err)
}

func (a *App) install(ctx context.Context, args []string) error {
	if len(args) > 1 {
		return errors.New("usage: gvm install [version]")
	}
	versionName := ""
	if len(args) == 1 {
		versionName = args[0]
	} else {
		cfg, err := a.resolveConfig()
		if err != nil {
			return err
		}
		versionName = cfg.GoVersion
	}
	path, err := a.ensureSDK(ctx, versionName)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.Out, "Go %s installed at %s\n", versionName, path)
	return err
}

func (a *App) listSDKs(args []string) error {
	if len(args) != 0 {
		return errors.New("usage: gvm list")
	}
	home, err := a.home()
	if err != nil {
		return err
	}
	versions, err := sdk.NewStore(home, a.HTTP, a.Progress).InstalledVersions()
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(a.Out, "Installed Go SDKs:"); err != nil {
		return err
	}
	if len(versions) == 0 {
		_, err = fmt.Fprintln(a.Out, "  none")
		return err
	}
	for _, versionName := range versions {
		if _, err := fmt.Fprintf(a.Out, "  %s\n", versionName); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) removeSDK(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: gvm rm <version>")
	}
	home, err := a.home()
	if err != nil {
		return err
	}
	if err := sdk.NewStore(home, a.HTTP, a.Progress).Remove(args[0]); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.Out, "removed Go SDK %s\n", args[0])
	return err
}

func (a *App) setVersion(ctx context.Context, args []string, path string) error {
	if len(args) != 1 {
		return errors.New("usage: gvm default <version>")
	}
	return a.configureVersion(ctx, args[0], path)
}

func (a *App) setLocalVersion(ctx context.Context, args []string) error {
	if len(args) > 1 {
		return errors.New("usage: gvm local [version]")
	}
	versionName := ""
	if len(args) == 1 {
		versionName = args[0]
	} else {
		var err error
		versionName, err = config.InferGoVersion(a.CWD)
		if err != nil {
			return err
		}
	}
	return a.configureVersion(ctx, versionName, filepath.Join(a.CWD, ".gvmrc"))
}

func (a *App) configureVersion(ctx context.Context, versionName, path string) error {
	if _, err := a.ensureSDK(ctx, versionName); err != nil {
		return err
	}
	if err := config.Write(path, versionName, nil, nil); err != nil {
		return err
	}
	_, err := fmt.Fprintf(a.Out, "configured Go %s in %s\n", versionName, path)
	return err
}

func (a *App) createVenv(ctx context.Context, args []string) error {
	if len(args) != 0 {
		return errors.New("usage: gvm venv")
	}
	cfg, err := a.resolveConfig()
	if err != nil {
		return err
	}
	sdkRoot, err := a.ensureSDK(ctx, cfg.GoVersion)
	if err != nil {
		return err
	}
	path := filepath.Join(a.CWD, ".gvmrc")
	if cfg.LocalPath != "" {
		path = cfg.LocalPath
	}
	venvEnabled := true
	tools := []string(nil)
	if cfg.LocalPath == "" {
		tools = cfg.Tools
	}
	if err := config.Write(path, cfg.GoVersion, &venvEnabled, tools); err != nil {
		return err
	}
	cfg.Venv = true
	if _, err := venv.Ensure(a.CWD); err != nil {
		return err
	}
	prepared, err := env.Build(cfg, a.CWD, a.Environ)
	if err != nil {
		return err
	}
	if err := venv.EnsureTools(ctx, a.CWD, sdkRoot, cfg.Tools, prepared.Values, a.Out, a.Err); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.Out, "virtual environment ready at %s\n", filepath.Join(a.CWD, ".venv"))
	return err
}

func (a *App) runBinary(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: gvm run <binary> [args...]")
	}
	cfg, err := a.resolveConfig()
	if err != nil {
		return err
	}
	sdkRoot, err := a.ensureSDK(ctx, cfg.GoVersion)
	if err != nil {
		return err
	}
	prepared, err := a.prepareRuntime(ctx, cfg, sdkRoot)
	if err != nil {
		return err
	}
	directories := filepath.SplitList(prepared.PATH)
	binary, err := runner.Find(args[0], directories)
	if err != nil {
		return err
	}
	return runner.Run(ctx, binary, a.CWD, args[1:], prepared.Values, a.Out, a.Err)
}

func (a *App) resolveConfig() (config.Config, error) {
	home, err := a.home()
	if err != nil {
		return config.Config{}, err
	}
	return config.Resolve(home, a.CWD, a.Environ)
}

func (a *App) ensureSDK(ctx context.Context, versionName string) (string, error) {
	home, err := a.home()
	if err != nil {
		return "", err
	}
	store := sdk.NewStore(home, a.HTTP, a.Progress)
	return store.Ensure(ctx, versionName)
}

func (a *App) prepareRuntime(ctx context.Context, cfg config.Config, sdkRoot string) (env.Result, error) {
	if cfg.Venv {
		if _, err := venv.Ensure(a.CWD); err != nil {
			return env.Result{}, err
		}
	}
	prepared, err := env.Build(cfg, a.CWD, a.Environ)
	if err != nil {
		return env.Result{}, err
	}
	if err := venv.EnsureTools(ctx, a.CWD, sdkRoot, cfg.Tools, prepared.Values, a.Out, a.Err); err != nil {
		return env.Result{}, err
	}
	return prepared, nil
}

func (a *App) home() (string, error) {
	values := make(map[string]string, len(a.Environ))
	for _, item := range a.Environ {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			values[key] = value
		}
	}
	if home := values["GVM_HOME"]; home != "" {
		return filepath.Abs(home)
	}
	if home := values["HOME"]; home != "" {
		return filepath.Abs(filepath.Join(home, ".gvm"))
	}
	if home := values["USERPROFILE"]; home != "" {
		return filepath.Abs(filepath.Join(home, ".gvm"))
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find user home: %w", err)
	}
	return filepath.Abs(filepath.Join(home, ".gvm"))
}

func (a *App) update(ctx context.Context) error {
	latest, err := a.Release.Latest(ctx)
	if err != nil {
		return err
	}
	latestVersion := strings.TrimPrefix(latest.TagName, "v")
	if version.Value != "dev" && strings.TrimPrefix(version.Value, "v") == latestVersion {
		_, err := fmt.Fprintf(a.Out, "gvm %s is already current\n", latest.TagName)
		return err
	}
	archive, err := release.CurrentAsset(latest)
	if err != nil {
		return err
	}
	checksums, err := release.ChecksumAsset(latest)
	if err != nil {
		return err
	}
	home, err := a.home()
	if err != nil {
		return err
	}
	updateDir, err := release.DownloadPath(home)
	if err != nil {
		return err
	}
	archivePath := filepath.Join(updateDir, archive.Name)
	checksumsPath := filepath.Join(updateDir, checksums.Name)
	defer func() {
		_ = os.Remove(archivePath)
		_ = os.Remove(checksumsPath)
	}()
	if err := a.Release.Download(ctx, checksums, checksumsPath); err != nil {
		return err
	}
	checksumData, err := os.ReadFile(checksumsPath)
	if err != nil {
		return fmt.Errorf("read release checksums: %w", err)
	}
	expected := release.ParseChecksums(checksumData)[archive.Name]
	if expected == "" {
		return fmt.Errorf("release checksums do not contain %s", archive.Name)
	}
	if err := a.Release.Download(ctx, archive, archivePath); err != nil {
		return err
	}
	if err := release.VerifyChecksum(archivePath, expected); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(updateDir, "stage-")
	if err != nil {
		return fmt.Errorf("create update staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(staging) }()
	if err := release.ExtractManagerArchive(archivePath, staging); err != nil {
		return err
	}
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return fmt.Errorf("create GVM bin directory: %w", err)
	}
	if runtime.GOOS == "windows" {
		return a.startWindowsUpdater(staging, binDir)
	}
	if err := replaceManagerBinaries(staging, binDir); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.Out, "updated gvm to %s\n", latest.TagName)
	return err
}

func (a *App) startWindowsUpdater(staging, binDir string) error {
	current, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find current gvm executable: %w", err)
	}
	// The parent removes its temporary staging directory when this method
	// returns. Move the payload to a pending path that the helper owns.
	pending := staging + ".pending"
	if err := os.Rename(staging, pending); err != nil {
		return fmt.Errorf("preserve Windows update staging: %w", err)
	}
	restored := false
	defer func() {
		if !restored {
			_ = os.Rename(pending, staging)
		}
	}()
	helper := filepath.Join(filepath.Dir(pending), "gvm-updater.exe")
	if err := copyFile(current, helper); err != nil {
		return fmt.Errorf("prepare Windows updater: %w", err)
	}
	command := exec.Command(helper, "__apply-update", pending, binDir)
	command.Stdout = a.Out
	command.Stderr = a.Err
	if err := command.Start(); err != nil {
		return fmt.Errorf("start Windows updater: %w", err)
	}
	if err := command.Process.Release(); err != nil {
		return fmt.Errorf("detach Windows updater: %w", err)
	}
	restored = true
	_, err = fmt.Fprintln(a.Out, "update staged; restart gvm to finish installation")
	return err
}

func (a *App) applyStagedUpdate(staging, binDir string) error {
	if runtime.GOOS == "windows" {
		for attempt := 0; attempt < 20; attempt++ {
			if err := replaceManagerBinaries(staging, binDir); err == nil {
				_ = os.RemoveAll(staging)
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
		return errors.New("replace Windows manager binaries after retries")
	}
	return replaceManagerBinaries(staging, binDir)
}

func managerBinaryNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"gvm.exe", "go.exe"}
	}
	return []string{"gvm", "go"}
}

func replaceManagerBinaries(staging, binDir string) error {
	names := managerBinaryNames()
	for _, name := range names {
		info, err := os.Stat(filepath.Join(staging, name))
		if err != nil || info.IsDir() {
			return fmt.Errorf("staged update is missing %s", name)
		}
	}

	backups := make(map[string]string, len(names))
	installed := make([]string, 0, len(names))
	rollback := func() {
		for _, path := range installed {
			_ = os.Remove(path)
		}
		for destination, backup := range backups {
			_ = os.Rename(backup, destination)
		}
	}
	for _, name := range names {
		destination := filepath.Join(binDir, name)
		info, err := os.Stat(destination)
		if err == nil {
			if info.IsDir() {
				rollback()
				return fmt.Errorf("manager destination is a directory: %s", destination)
			}
			backupFile, createErr := os.CreateTemp(binDir, ".gvm-backup-*")
			if createErr != nil {
				rollback()
				return fmt.Errorf("create manager backup: %w", createErr)
			}
			backup := backupFile.Name()
			if closeErr := backupFile.Close(); closeErr != nil {
				_ = os.Remove(backup)
				rollback()
				return fmt.Errorf("close manager backup: %w", closeErr)
			}
			_ = os.Remove(backup)
			if renameErr := os.Rename(destination, backup); renameErr != nil {
				rollback()
				return fmt.Errorf("backup %s: %w", name, renameErr)
			}
			backups[destination] = backup
		} else if !os.IsNotExist(err) {
			rollback()
			return fmt.Errorf("inspect manager destination: %w", err)
		}
	}
	for _, name := range names {
		source := filepath.Join(staging, name)
		destination := filepath.Join(binDir, name)
		if err := os.Rename(source, destination); err != nil {
			rollback()
			return fmt.Errorf("replace %s: %w", name, err)
		}
		installed = append(installed, destination)
	}
	for _, backup := range backups {
		if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove manager backup: %w", err)
		}
	}
	return nil
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o700)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		_ = os.Remove(destination)
		return err
	}
	return output.Close()
}

func (a *App) printHelp() {
	_, _ = fmt.Fprintln(a.Out, `gvm - Go version manager

Usage:
  gvm install [version]
  gvm list
  gvm rm <version>
  gvm default <version>
  gvm local [version]
  gvm venv
  gvm run <binary> [args...]
  gvm update
  gvm version`)
}

func isTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
