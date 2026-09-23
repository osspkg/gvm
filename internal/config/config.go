/*
 *  Copyright (c) 2024-2026 Mikhail Knyazhev <markus621@yandex.com>. All rights reserved.
 *  Use of this source code is governed by a BSD 3-Clause license that can be found in the LICENSE file.
 */

package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrNoVersion = errors.New("gvm: go version is not configured")
	ErrInvalid   = errors.New("gvm: invalid configuration")
)

var (
	keyPattern     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+(?:\.[0-9]+)?(?:[A-Za-z][A-Za-z0-9.-]*)?$`)
)

// Values is the parsed representation of one .gvmrc file.
type Values struct {
	Path   string
	Fields map[string]string
	Tools  []string
}

// Config is the resolved configuration for one command invocation.
type Config struct {
	Home       string
	GoVersion  string
	Venv       bool
	Tools      []string
	Env        map[string]string
	LocalPath  string
	GlobalPath string
}

// ParseFile reads a restricted dotenv file without executing shell code.
func ParseFile(path string) (Values, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Values{}, fmt.Errorf("read config %q: %w", path, err)
	}
	values, err := Parse(data)
	if err != nil {
		return Values{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	values.Path = path
	return values, nil
}

// Parse parses one-line KEY=VALUE entries. Repeated GVM_TOOLS keys are kept in order.
func Parse(data []byte) (Values, error) {
	values := Values{Fields: make(map[string]string)}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return Values{}, fmt.Errorf("line %d: %w: expected KEY=VALUE", lineNumber, ErrInvalid)
		}
		key = strings.TrimSpace(key)
		if !keyPattern.MatchString(key) {
			return Values{}, fmt.Errorf("line %d: %w: invalid key %q", lineNumber, ErrInvalid, key)
		}

		value, err := parseValue(strings.TrimSpace(rawValue))
		if err != nil {
			return Values{}, fmt.Errorf("line %d: %w: %s", lineNumber, ErrInvalid, err)
		}
		if key == "GVM_TOOLS" {
			if value == "" {
				continue
			}
			if strings.ContainsAny(value, "\r\n") {
				return Values{}, fmt.Errorf("line %d: %w: GVM_TOOLS cannot contain newlines", lineNumber, ErrInvalid)
			}
			values.Tools = append(values.Tools, value)
			continue
		}
		values.Fields[key] = value
	}
	if err := scanner.Err(); err != nil {
		return Values{}, fmt.Errorf("scan config: %w", err)
	}
	return values, nil
}

func parseValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if raw[0] == '\'' {
		end := strings.IndexByte(raw[1:], '\'')
		if end < 0 {
			return "", errors.New("invalid single-quoted value")
		}
		end++
		if suffix := strings.TrimSpace(raw[end+1:]); suffix != "" && !strings.HasPrefix(suffix, "#") {
			return "", errors.New("unexpected content after quoted value")
		}
		return raw[1:end], nil
	}
	if raw[0] == '"' {
		end := -1
		escaped := false
		for index := 1; index < len(raw); index++ {
			if raw[index] == '\\' && !escaped {
				escaped = true
				continue
			}
			if raw[index] == '"' && !escaped {
				end = index
				break
			}
			escaped = false
		}
		if end < 0 {
			return "", errors.New("unterminated double-quoted value")
		}
		if suffix := strings.TrimSpace(raw[end+1:]); suffix != "" && !strings.HasPrefix(suffix, "#") {
			return "", errors.New("unexpected content after quoted value")
		}
		value, err := strconv.Unquote(raw[:end+1])
		if err != nil {
			return "", fmt.Errorf("invalid double-quoted value: %w", err)
		}
		return value, nil
	}
	if strings.ContainsAny(raw, "\r\n") {
		return "", errors.New("unquoted value cannot contain newlines")
	}
	if index := strings.Index(raw, " #"); index >= 0 {
		raw = strings.TrimSpace(raw[:index])
	}
	return raw, nil
}

// Resolve loads the nearest local file, or the global file when no local file exists.
// Process values override file values for keys present in either configuration.
func Resolve(home, cwd string, process []string) (Config, error) {
	if home == "" {
		return Config{}, fmt.Errorf("%w: GVM_HOME is empty", ErrInvalid)
	}
	absoluteHome, err := filepath.Abs(home)
	if err != nil {
		return Config{}, fmt.Errorf("resolve GVM_HOME: %w", err)
	}
	absoluteCWD, err := filepath.Abs(cwd)
	if err != nil {
		return Config{}, fmt.Errorf("resolve current directory: %w", err)
	}

	globalPath := filepath.Join(absoluteHome, ".gvmrc")
	localPath, err := findLocal(absoluteCWD)
	if err != nil {
		return Config{}, err
	}
	var selected Values
	if localPath != "" {
		selected, err = ParseFile(localPath)
	} else if _, statErr := os.Stat(globalPath); statErr == nil {
		selected, err = ParseFile(globalPath)
	} else if !os.IsNotExist(statErr) {
		err = fmt.Errorf("stat global config %q: %w", globalPath, statErr)
	}
	if err != nil {
		return Config{}, err
	}

	processValues := environMap(process)
	merged := make(map[string]string, len(selected.Fields)+len(processValues))
	for key, value := range selected.Fields {
		merged[key] = value
	}
	for key := range selected.Fields {
		if value, ok := processValues[key]; ok {
			merged[key] = value
		}
	}
	for _, key := range []string{"GVM_GO_VERSION", "GVM_VENV", "GVM_TOOLS"} {
		if value, ok := processValues[key]; ok {
			merged[key] = value
		}
	}

	version := strings.TrimSpace(merged["GVM_GO_VERSION"])
	if version == "" {
		return Config{}, fmt.Errorf("%w: set GVM_GO_VERSION in %s", ErrNoVersion, configHint(localPath, globalPath))
	}
	if !versionPattern.MatchString(version) {
		return Config{}, fmt.Errorf("%w: invalid GVM_GO_VERSION %q", ErrInvalid, version)
	}

	venv, err := parseBool(merged["GVM_VENV"])
	if err != nil {
		return Config{}, err
	}
	tools := append([]string(nil), selected.Tools...)
	if value, ok := processValues["GVM_TOOLS"]; ok {
		tools = splitTools(value)
	}
	for _, tool := range tools {
		if strings.TrimSpace(tool) == "" || strings.ContainsAny(tool, "\r\n") {
			return Config{}, fmt.Errorf("%w: invalid GVM_TOOLS entry", ErrInvalid)
		}
	}

	envValues := make(map[string]string, len(merged))
	for key, value := range merged {
		if key == "GVM_GO_VERSION" || key == "GVM_VENV" || key == "GVM_TOOLS" || key == "GVM_HOME" {
			continue
		}
		envValues[key] = value
	}
	return Config{
		Home:       absoluteHome,
		GoVersion:  version,
		Venv:       venv,
		Tools:      tools,
		Env:        envValues,
		LocalPath:  localPath,
		GlobalPath: globalPath,
	}, nil
}

func findLocal(cwd string) (string, error) {
	path := cwd
	for {
		candidate := filepath.Join(path, ".gvmrc")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat local config %q: %w", candidate, err)
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", nil
		}
		path = parent
	}
}

func environMap(values []string) map[string]string {
	result := make(map[string]string, len(values))
	for _, entry := range values {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			result[key] = value
		}
	}
	return result
}

func parseBool(value string) (bool, error) {
	if value == "" {
		return false, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%w: GVM_VENV must be true or false", ErrInvalid)
	}
	return parsed, nil
}

func splitTools(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool { return r == '\n' || r == '\r' || r == ' ' || r == '\t' })
	return append([]string(nil), fields...)
}

func configHint(localPath, globalPath string) string {
	if localPath != "" {
		return localPath
	}
	return globalPath
}

// Write updates reserved values while preserving all other parsed environment values.
func Write(path string, goVersion string, venv *bool, tools []string) error {
	if !versionPattern.MatchString(goVersion) {
		return fmt.Errorf("%w: invalid GVM_GO_VERSION %q", ErrInvalid, goVersion)
	}
	values := Values{Fields: make(map[string]string)}
	if data, err := os.ReadFile(path); err == nil {
		parsed, parseErr := Parse(data)
		if parseErr != nil {
			return fmt.Errorf("read existing config: %w", parseErr)
		}
		values = parsed
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("read existing config: %w", err)
	}
	values.Fields["GVM_GO_VERSION"] = goVersion
	if venv != nil {
		values.Fields["GVM_VENV"] = strconv.FormatBool(*venv)
	}
	if tools != nil {
		values.Tools = append([]string(nil), tools...)
	}
	return writeValues(path, values)
}

func writeValues(path string, values Values) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	keys := make([]string, 0, len(values.Fields))
	for key := range values.Fields {
		if key != "GVM_GO_VERSION" && key != "GVM_VENV" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var builder strings.Builder
	if value := values.Fields["GVM_GO_VERSION"]; value != "" {
		fmt.Fprintf(&builder, "GVM_GO_VERSION=%s\n", value)
	}
	if value, ok := values.Fields["GVM_VENV"]; ok {
		fmt.Fprintf(&builder, "GVM_VENV=%s\n", value)
	}
	for _, tool := range values.Tools {
		fmt.Fprintf(&builder, "GVM_TOOLS=%s\n", tool)
	}
	for _, key := range keys {
		fmt.Fprintf(&builder, "%s=%s\n", key, quoteValue(values.Fields[key]))
	}

	temp, err := os.CreateTemp(filepath.Dir(path), ".gvmrc-*")
	if err != nil {
		return fmt.Errorf("create config temp file: %w", err)
	}
	tempName := temp.Name()
	defer func() { _ = os.Remove(tempName) }()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("set config permissions: %w", err)
	}
	if _, err := temp.WriteString(builder.String()); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync config: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	if err := replaceConfigFile(tempName, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

func quoteValue(value string) string {
	if value == "" {
		return ""
	}
	if strings.ContainsAny(value, " #\t\r\n\"'") {
		return strconv.Quote(value)
	}
	return value
}
