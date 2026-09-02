package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jonasalessi/cdd-cli/internal/analyze"
	"github.com/jonasalessi/cdd-cli/internal/config"
)

// checkMetrics is the metric set every fixture enables, so a fixture's ICPs
// do not move when the default selection does.
const checkMetrics = "code_branch,condition,exception_handling,internal_coupling,external_coupling,inheritance"

// overLimitMark opens the console line of a unit above its limit.
const overLimitMark = "✗"

// cleanSource is one class of 1 ICP: well inside the greenfield limit.
const cleanSource = `export class Greeter {
  greet(name: string): string {
    if (name) {
      return ` + "`hi ${name}`" + `;
    }
    return "hi";
  }
}
`

// overLimitSource is one class of 12 ICPs — six branches and six conditions
// — against the greenfield limit of 10.
const overLimitSource = `export class OrderService {
  price(a: number, b: number, c: number): number {
    if (a > 0 && b > 0) { return a + b; }
    if (a > 1 && b > 1) { return a - b; }
    if (a > 2 && b > 2) { return a * b; }
    if (a > 3 && b > 3) { return a / b; }
    if (a > 4 && b > 4) { return a + c; }
    if (a > 5 && b > 5) { return b + c; }
    return 0;
  }
}
`

// writeTSFixture lays out a TypeScript project in dir: a tsconfig.json with
// one path alias and a configuration written by cdd init, so the file always
// matches the schema the command reads today.
func writeTSFixture(t *testing.T, dir string) {
	t.Helper()
	writeFixtureFile(t, dir, "tsconfig.json", `{"compilerOptions":{"paths":{"@app/*":["src/*"]}}}`+"\n")
	_, stderr, code := runCdd(t, dir, "init", "--yes", "--force",
		"--languages", "typescript",
		"--metrics", checkMetrics,
	)
	require.Equal(t, 0, code, "stderr: %s", stderr)
}

// writeFixtureFile writes one source file, creating its directories.
func writeFixtureFile(t *testing.T, dir, rel, src string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(src), 0o644))
}

// editConfig rewrites the fixture configuration, replacing each old/new pair
// once. A pair that does not match is a broken test, not a silent no-op.
func editConfig(t *testing.T, dir string, pairs ...string) {
	t.Helper()
	path := filepath.Join(dir, "cdd.config.yaml")
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	out := string(data)
	for i := 0; i+1 < len(pairs); i += 2 {
		require.Contains(t, out, pairs[i])
		out = strings.Replace(out, pairs[i], pairs[i+1], 1)
	}
	require.NoError(t, os.WriteFile(path, []byte(out), 0o644))
}

// legacyProject moves a fixture off greenfield, which is the only project
// type that accepts a mode other than strict_all.
func legacyProject(t *testing.T, dir, blockOnCI, mode string) {
	t.Helper()
	editConfig(t, dir,
		"project_type: greenfield", "project_type: legacy",
		"  block_on_ci: true\n  legacy_mode: strict_all\n",
		"  block_on_ci: "+blockOnCI+"\n  legacy_mode: "+mode+"\n",
	)
}

func TestCheckCleanProject(t *testing.T) {
	dir := t.TempDir()
	writeTSFixture(t, dir)
	writeFixtureFile(t, dir, "src/greeter.ts", cleanSource)

	stdout, stderr, code := runCdd(t, dir, "check")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "src/greeter.ts")
	assert.Contains(t, stdout, "0 over limit")
	assert.NotContains(t, stdout, overLimitMark)
	assert.Empty(t, stderr)
}

func TestCheckOverLimitUnitBlocks(t *testing.T) {
	dir := t.TempDir()
	writeTSFixture(t, dir)
	writeFixtureFile(t, dir, "src/order-service.ts", overLimitSource)

	stdout, stderr, code := runCdd(t, dir, "check")
	assert.Equal(t, 1, code)
	assert.Contains(t, stdout, overLimitMark)
	assert.Contains(t, stdout, "OrderService")
	assert.Contains(t, stdout, "1 over limit")
	assert.Empty(t, stderr, "the report already names the violation")
}

func TestCheckEnforcementVariants(t *testing.T) {
	t.Run("measure_only reports without blocking", func(t *testing.T) {
		dir := t.TempDir()
		writeTSFixture(t, dir)
		writeFixtureFile(t, dir, "src/order-service.ts", overLimitSource)
		legacyProject(t, dir, "false", "measure_only")

		stdout, stderr, code := runCdd(t, dir, "check")
		require.Equal(t, 0, code, "stderr: %s", stderr)
		assert.Contains(t, stdout, "OrderService")
		assert.Contains(t, stdout, "1 over limit")
		assert.Contains(t, stderr, "warning:", "a limit of 10 is outside the legacy band")
	})
	t.Run("strict_on_new_only says it is not enforced", func(t *testing.T) {
		dir := t.TempDir()
		writeTSFixture(t, dir)
		writeFixtureFile(t, dir, "src/order-service.ts", overLimitSource)
		legacyProject(t, dir, "true", "strict_on_new_only")

		stdout, _, code := runCdd(t, dir, "check")
		assert.Equal(t, 0, code)
		assert.Contains(t, stdout, "not enforced yet")
		assert.Contains(t, stdout, "1 over limit")
	})
}

func TestCheckConfigFlagSetsTheRoot(t *testing.T) {
	dir := t.TempDir()
	writeTSFixture(t, dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "app"), 0o755))
	require.NoError(t, os.Rename(
		filepath.Join(dir, "cdd.config.yaml"),
		filepath.Join(dir, "app", "cdd.config.yaml"),
	))
	writeFixtureFile(t, dir, "outside.ts", strings.ReplaceAll(cleanSource, "Greeter", "OutsideService"))
	writeFixtureFile(t, dir, "app/src/inside.ts", strings.ReplaceAll(cleanSource, "Greeter", "InsideService"))

	stdout, stderr, code := runCdd(t, dir, "check", "--config", filepath.Join("app", "cdd.config.yaml"))
	require.Equal(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "InsideService")
	assert.NotContains(t, stdout, "OutsideService", "a file above the configuration is outside the run")
}

func TestCheckJSONReportToFile(t *testing.T) {
	dir := t.TempDir()
	writeTSFixture(t, dir)
	writeFixtureFile(t, dir, "src/greeter.ts", cleanSource)
	editConfig(t, dir,
		"  format: console", "  format: json",
		"  outputFile: null", `  outputFile: "report.json"`,
	)

	stdout, stderr, code := runCdd(t, dir, "check")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "Report written to report.json")

	data, err := os.ReadFile(filepath.Join(dir, "report.json"))
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(data, &doc), "the report must be valid JSON")
	assert.Contains(t, string(data), "Greeter")
}

func TestCheckTimeoutReportsPartially(t *testing.T) {
	dir := t.TempDir()
	writeTSFixture(t, dir)
	writeFixtureFile(t, dir, "src/greeter.ts", cleanSource)
	editConfig(t, dir, "\ntimeout: 5m\n", "\ntimeout: 1ns\n")

	stdout, stderr, code := runCdd(t, dir, "check")
	assert.Equal(t, 2, code)
	assert.Contains(t, stdout, "partial")
	assert.Contains(t, stderr, "timeout")
}

func TestCheckLanguageWithoutAnalyzer(t *testing.T) {
	dir := t.TempDir()
	writeGoFixture(t, dir)
	_, stderr, code := runCdd(t, dir, "init", "--yes", "--force", "--languages", "go")
	require.Equal(t, 0, code, "stderr: %s", stderr)

	_, stderr, code = runCdd(t, dir, "check")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "no analyzer for go yet")
}

func TestCheckMissingConfig(t *testing.T) {
	_, stderr, code := runCdd(t, t.TempDir(), "check", "--config", "nowhere.yaml")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "nowhere.yaml")
}

func TestCheckInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	writeTSFixture(t, dir)
	editConfig(t, dir, "version: 1", "version: 2")

	_, stderr, code := runCdd(t, dir, "check")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "is invalid")
	assert.Contains(t, stderr, "version")
}

func TestCheckUnwritableReport(t *testing.T) {
	dir := t.TempDir()
	writeTSFixture(t, dir)
	writeFixtureFile(t, dir, "src/greeter.ts", cleanSource)
	editConfig(t, dir, "  outputFile: null", `  outputFile: "nowhere/report.json"`)

	_, stderr, code := runCdd(t, dir, "check")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "output directory")
}

func TestCommandContextFallsBackToBackground(t *testing.T) {
	c, _, _ := silentCmd()
	assert.NotNil(t, commandContext(c), "a command cobra never ran carries no context")
}

func TestEmitReportRejectsAnUnknownFormat(t *testing.T) {
	c, _, _ := silentCmd()
	assert.ErrorContains(t, emitReport(c, config.Reporter{Format: "toml"}, analyze.RunResult{}), "toml")
}

func TestCheckSyntaxErrorIsAWarning(t *testing.T) {
	dir := t.TempDir()
	writeTSFixture(t, dir)
	writeFixtureFile(t, dir, "src/broken.ts", "export class Broken { if if if (\n")

	stdout, stderr, code := runCdd(t, dir, "check")
	require.Equal(t, 0, code, "stderr: %s", stderr)
	assert.Contains(t, stdout, "syntax error")
	assert.Contains(t, stdout, "src/broken.ts")
	assert.Contains(t, stdout, "0 over limit")
}
