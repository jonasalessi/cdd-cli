package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		in   time.Duration
		want string
	}{
		{0, "0s"},
		{30 * time.Second, "30s"},
		{5 * time.Minute, "5m"},
		{90 * time.Minute, "1h30m"},
		{2 * time.Hour, "2h"},
		{time.Hour + 30*time.Minute + 15*time.Second, "1h30m15s"},
		{500 * time.Millisecond, "500ms"},
		{-time.Minute, "-1m"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, formatDuration(tt.in), tt.in)
		if tt.in >= 0 {
			d, err := time.ParseDuration(tt.want)
			require.NoError(t, err)
			assert.Equal(t, tt.in, d, "must read back")
		}
	}
}

func TestFormatWeight(t *testing.T) {
	tests := map[float64]string{1: "1.0", 0.5: "0.5", 0.25: "0.25", 10: "10.0", 1.5: "1.5"}
	for in, want := range tests {
		assert.Equal(t, want, formatWeight(in))
	}
}

func TestQuote(t *testing.T) {
	assert.Equal(t, `"**/*_test.go"`, quote("**/*_test.go"))
	assert.Equal(t, `".*Dto\\.java"`, quote(`.*Dto\.java`))
	assert.Equal(t, `"say \"hi\""`, quote(`say "hi"`))
}

func TestRenderNil(t *testing.T) {
	_, err := Render(nil)
	require.Error(t, err)
}

func TestRenderGolden(t *testing.T) {
	for name, cfg := range goldenConfigs() {
		t.Run(name, func(t *testing.T) {
			got, err := Render(cfg)
			require.NoError(t, err)
			path := goldenPath(name)
			if updateGolden() {
				require.NoError(t, os.WriteFile(path, got, 0o644))
			}
			want, err := os.ReadFile(path)
			require.NoError(t, err, "run go test ./internal/config -update to create it")
			assert.Equal(t, string(want), string(got))
		})
	}
}

func TestRenderLayout(t *testing.T) {
	cfg := greenfieldJavaKotlin()
	out, err := Render(cfg)
	require.NoError(t, err)
	s := string(out)

	assert.True(t, strings.HasPrefix(s, "# Cognitive Driven Development (CDD) Configuration"))
	assert.Contains(t, s, "\nversion: 1\n")
	assert.Contains(t, s, "\nproject_type: greenfield\n")
	assert.Contains(t, s, "\ntimeout: 5m\n", "never 5m0s")
	assert.Contains(t, s, "\n  outputFile: null\n")
	assert.Contains(t, s, "\n  packages:\n    - \"com.acme.billing\"\n")
	assert.Contains(t, s, "\ninclude: []\n")
	assert.Contains(t, s, "\nexclude:\n  - \"**/src/test/**\"\n")
	assert.Contains(t, s, "\n    # \".*/adapters/.*\": 8\n    # \".*Dto\\\\.java\": 20\n")
	assert.Contains(t, s, "# format: The output style. Supported: console, json, xml, markdown")

	java := strings.Index(s, "\n  java:\n")
	kotlin := strings.Index(s, "\n  kotlin:\n")
	assert.Greater(t, kotlin, java, "languages in canonical order")

	// Pattern order and metric order inside a pattern.
	lines := strings.Split(s, "\n")
	start := indexOf(lines, `    ".*":`)
	require.GreaterOrEqual(t, start, 0)
	assert.Equal(t, "      code_branch: 1.0        # if/else, switch case, ternary, loops", lines[start+1])
	assert.Equal(t, "      condition: 1.0          # && and || clauses", lines[start+2])
	assert.Equal(t, "      exception_handling: 1.0 # try / catch / finally blocks", lines[start+3])
	assert.Equal(t, "      internal_coupling: 1.0  # references to project classes", lines[start+4])
	assert.Equal(t, "      external_coupling: 0.5  # framework / JDK types", lines[start+5])
	assert.Equal(t, "      inheritance: 1.0        # extends / implements, per level", lines[start+6])
	assert.Equal(t, `    ".*/adapters/.*":`, lines[start+7])
	assert.Equal(t, "      internal_coupling: 0.5  # references to project classes", lines[start+8])
	assert.Equal(t, `    ".*Dto\\.java":`, lines[start+9])
}

func TestRenderEmptyListsAndOutputFile(t *testing.T) {
	cfg := dogfood()
	out := "report.json"
	cfg.Reporter.OutputFile = &out
	cfg.InternalCoupling.Packages = nil
	cfg.Exclude = nil
	cfg.Include = []string{"src/**"}
	cfg.Timeout = 0

	got, err := Render(cfg)
	require.NoError(t, err)
	s := string(got)
	assert.Contains(t, s, "\n  outputFile: \"report.json\"\n")
	assert.Contains(t, s, "\n  packages: []\n")
	assert.Contains(t, s, "\nexclude: []\n")
	assert.Contains(t, s, "\ninclude:\n  - \"src/**\"\n")
	assert.Contains(t, s, "\ntimeout: 0s\n")
	assert.True(t, strings.HasSuffix(s, "\n"), "file ends with one newline")
	assert.False(t, strings.HasSuffix(s, "\n\n"))
}

func TestRenderLimitExamplesFollowFirstLanguage(t *testing.T) {
	got, err := Render(legacyGoTypeScript())
	require.NoError(t, err)
	s := string(got)
	assert.Equal(t, 1, strings.Count(s, `# ".*/adapters/.*": 8`), "one example, under the first language")
	assert.NotContains(t, s, "Dto", "the java DTO example is not shown for go/typescript")
}

func TestFormatList(t *testing.T) {
	assert.Equal(t, " []", formatList(nil, 2))
	assert.Equal(t, "\n  - \"a\"\n  - \"b\"", formatList([]string{"a", "b"}, 2))
	assert.Equal(t, "\n    - \"a\"", formatList([]string{"a"}, 4))
}

func TestRenderSkipsUnknownLanguages(t *testing.T) {
	cfg := dogfood()
	cfg.Metrics["rust"] = PatternWeights{{Pattern: PatternAll}}
	got, err := Render(cfg)
	require.NoError(t, err)
	assert.NotContains(t, string(got), "rust")
}

func indexOf(lines []string, want string) int {
	for i, l := range lines {
		if l == want {
			return i
		}
	}
	return -1
}
