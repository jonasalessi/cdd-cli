package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// run executes the root command with args and returns stdout, stderr and the
// exit code.
func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs(args)
	code := execute(root, &stderr)
	return stdout.String(), stderr.String(), code
}

func TestHelpListsSubcommands(t *testing.T) {
	out, _, code := run(t, "--help")
	assert.Equal(t, 0, code)
	assert.Contains(t, out, "init ")
	assert.Contains(t, out, "version ")
	assert.Contains(t, out, `Run "cdd init"`)
	assert.Contains(t, out, `--config string`)
	assert.Contains(t, out, `(default "cdd.config.yaml")`)
}

func TestVersion(t *testing.T) {
	out, _, code := run(t, "version")
	assert.Equal(t, 0, code)
	assert.Equal(t, "cdd dev (none, unknown)\n", out)

	out, _, code = run(t, "--version")
	assert.Equal(t, 0, code)
	assert.Equal(t, "cdd dev (none, unknown)\n", out)
}

func TestUnknownCommand(t *testing.T) {
	_, stderr, code := run(t, "nope")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, `cdd: unknown command "nope"`)
}

func TestInitHelpFlagSet(t *testing.T) {
	out, _, code := run(t, "init", "--help")
	require.Equal(t, 0, code)
	for _, flag := range []string{
		"--force", "--languages strings", "--project-type string", "--legacy-mode string",
		"--limit int", "--metrics strings", "--weight strings", "--packages strings",
		"--no-default-excludes", "--timeout duration", "--scan-timeout duration",
		"--yes", "--output string",
	} {
		assert.Contains(t, out, flag)
	}
	assert.Contains(t, out, "(default 5m0s)")
	assert.Contains(t, out, "(default 4s)")
}

func TestConfigFlag(t *testing.T) {
	_, _, code := run(t, "--config", "other.yaml", "version")
	assert.Equal(t, 0, code)
	assert.Equal(t, "other.yaml", configPath)
}
