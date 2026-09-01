package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

// cddBin is the binary built once in TestMain. The init tests run it in
// temporary directories, so stdin is not a TTY and nothing touches the
// repository.
var cddBin string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "cdd-e2e-")
	if err != nil {
		panic(err)
	}
	cddBin = filepath.Join(tmp, "cdd")
	build := exec.CommandContext(context.Background(), "go", "build", "-o", cddBin, ".")
	build.Dir = ".."
	if out, buildErr := build.CombinedOutput(); buildErr != nil {
		fmt.Fprintf(os.Stderr, "building cdd: %v\n%s", buildErr, out)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(tmp)
	os.Exit(code)
}

// runCdd executes the built binary in dir.
func runCdd(t *testing.T, dir string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	c := exec.CommandContext(context.Background(), cddBin, args...)
	c.Dir = dir
	var outBuf, errBuf bytes.Buffer
	c.Stdout, c.Stderr = &outBuf, &errBuf
	err := c.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
	default:
		t.Fatalf("running %s: %v", cddBin, err)
	}
	return outBuf.String(), errBuf.String(), code
}

// writeGoFixture lays out a minimal Go project.
func writeGoFixture(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"),
		[]byte("module example.com/fixture\n\ngo 1.23\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"),
		[]byte("package main\n\nfunc main() {}\n"), 0o644))
}

// loadConfig loads and validates the file the command wrote.
func loadConfig(t *testing.T, dir string) *config.Config {
	t.Helper()
	cfg, err := config.Load(filepath.Join(dir, "cdd.config.yaml"))
	require.NoError(t, err)
	issues := config.Validate(cfg)
	require.False(t, issues.HasErrors(), "issues: %v", issues)
	return cfg
}

func TestInitYesDetectsGoProject(t *testing.T) {
	dir := t.TempDir()
	writeGoFixture(t, dir)

	stdout, stderr, code := runCdd(t, dir, "init", "--yes")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "Created cdd.config.yaml")
	assert.Empty(t, stderr)

	cfg := loadConfig(t, dir)
	require.Len(t, cfg.Metrics, 1)
	assert.Contains(t, cfg.Metrics, config.LangGo)
	assert.Equal(t, []string{"example.com/fixture"}, cfg.InternalCoupling.Packages)
	assert.Equal(t, config.ProjectGreenfield, cfg.ProjectType)
}

func TestInitFullFlagsMatchesGolden(t *testing.T) {
	dir := t.TempDir()
	_, stderr, code := runCdd(t, dir, "init",
		"--languages", "java,kotlin",
		"--project-type", "greenfield",
		"--limit", "10",
		"--metrics", "code_branch,condition,exception_handling,internal_coupling,external_coupling,inheritance",
		"--packages", "com.acme",
		"--force",
	)
	require.Equal(t, 0, code, "stderr: %s", stderr)

	got, err := os.ReadFile(filepath.Join(dir, "cdd.config.yaml"))
	require.NoError(t, err)
	want, err := os.ReadFile(filepath.Join("testdata", "golden", "greenfield-java-kotlin.yaml"))
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))
}

func TestInitMeasureOnlyDisablesCI(t *testing.T) {
	dir := t.TempDir()
	writeGoFixture(t, dir)
	_, stderr, code := runCdd(t, dir, "init", "--project-type", "legacy", "--legacy-mode", "measure_only")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	cfg := loadConfig(t, dir)
	assert.False(t, cfg.Enforcement.BlockOnCI)
	assert.Equal(t, "measure_only", cfg.Enforcement.LegacyMode)
}

func TestInitNoLanguagesFails(t *testing.T) {
	dir := t.TempDir()
	_, stderr, code := runCdd(t, dir, "init", "--yes")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "--languages")
	assert.NoFileExists(t, filepath.Join(dir, "cdd.config.yaml"))
}

func TestInitExistingFileGuard(t *testing.T) {
	dir := t.TempDir()
	writeGoFixture(t, dir)
	path := filepath.Join(dir, "cdd.config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o644))

	_, stderr, code := runCdd(t, dir, "init", "--yes")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "--force")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "old", string(data))

	_, stderr, code = runCdd(t, dir, "init", "--yes", "--force")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "version: 1")
}

func TestInitZeroWeightFails(t *testing.T) {
	dir := t.TempDir()
	writeGoFixture(t, dir)
	_, stderr, code := runCdd(t, dir, "init", "--yes", "--weight", "code_branch=0")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "code_branch")
}

func TestInitTimeoutFlag(t *testing.T) {
	dir := t.TempDir()
	writeGoFixture(t, dir)
	_, stderr, code := runCdd(t, dir, "init", "--yes", "--timeout", "30s")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	data, err := os.ReadFile(filepath.Join(dir, "cdd.config.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "\ntimeout: 30s\n")

	_, stderr, code = runCdd(t, dir, "init", "--yes", "--force", "--timeout", "-1s")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "timeout")
}

func TestInitScanTimeoutTruncation(t *testing.T) {
	t.Run("with --languages the result is unaffected", func(t *testing.T) {
		dir := t.TempDir()
		writeGoFixture(t, dir)
		_, stderr, code := runCdd(t, dir, "init", "--yes", "--scan-timeout", "1ns", "--languages", "go")
		require.Equal(t, 0, code, "stderr: %s", stderr)
		assert.Contains(t, stderr, "scan stopped")
		cfg := loadConfig(t, dir)
		assert.Contains(t, cfg.Metrics, config.LangGo)
	})
	t.Run("without --languages nothing is detected in time", func(t *testing.T) {
		dir := t.TempDir()
		writeGoFixture(t, dir)
		_, stderr, code := runCdd(t, dir, "init", "--yes", "--scan-timeout", "1ns")
		assert.Equal(t, 1, code)
		assert.Contains(t, stderr, "scan stopped")
		assert.Contains(t, stderr, "--languages")
	})
}
