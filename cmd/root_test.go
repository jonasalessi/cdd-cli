package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/spf13/cobra"
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
	assert.Contains(t, out, "check ")
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

func TestCheckHelpFlagSet(t *testing.T) {
	out, _, code := run(t, "check", "--help")
	require.Equal(t, 0, code)
	assert.Contains(t, out, "--all")
	assert.Contains(t, out, "--explain")
	assert.Contains(t, out, "list every counted construct with its position and ICPs")
}

func TestConfigFlag(t *testing.T) {
	_, _, code := run(t, "--config", "other.yaml", "version")
	assert.Equal(t, 0, code)
	assert.Equal(t, "other.yaml", configPath)
}

func TestExitCodeErrorPassesItsCodeThroughSilently(t *testing.T) {
	assert.Equal(t, "exit code 130", exitCodeError{code: 130}.Error())

	c := &cobra.Command{
		RunE:          func(*cobra.Command, []string) error { return exitCodeError{code: 130} },
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	c.SetOut(io.Discard)
	c.SetErr(io.Discard)
	var stderr bytes.Buffer
	assert.Equal(t, 130, execute(c, &stderr))
	assert.Empty(t, stderr.String(), "the ctrl-c convention prints nothing")
}

func TestExecuteReadsOsArgs(t *testing.T) {
	args := os.Args
	t.Cleanup(func() { os.Args = args })
	os.Args = []string{"cdd", "version"}
	assert.Equal(t, 0, Execute())
}
