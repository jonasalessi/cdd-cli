package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadGolden(t *testing.T) {
	for name, want := range goldenConfigs() {
		t.Run(name, func(t *testing.T) {
			got, err := Load(goldenPath(name))
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	require.Error(t, err)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestLoadBadYAMLNamesPathAndLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cdd.config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("version: 1\nproject_type: [\n"), 0o644))
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), path)
	assert.Contains(t, err.Error(), "line")
}

func TestParseKeepsPatternOrder(t *testing.T) {
	doc := `
version: 1
metrics:
  alpha:
    ".*/z/.*": {code_branch: 1}
    ".*": {code_branch: 1, condition: 1, inheritance: 1}
    ".*/a/.*": {condition: 2}
icp-limits:
  alpha:
    ".*/z/.*": 3
    ".*": 10
    ".*/a/.*": 7
`
	cfg, err := Parse(strings.NewReader(doc))
	require.NoError(t, err)
	assert.Equal(t, PatternWeights{
		{Pattern: ".*/z/.*", Weights: map[MetricID]float64{MetricCodeBranch: 1}},
		{
			Pattern: PatternAll,
			Weights: map[MetricID]float64{MetricCodeBranch: 1, MetricCondition: 1, MetricInheritance: 1},
		},
		{Pattern: ".*/a/.*", Weights: map[MetricID]float64{MetricCondition: 2}},
	}, cfg.Metrics[langAlpha])
	assert.Equal(t, PatternLimits{
		{Pattern: ".*/z/.*", Limit: 3},
		{Pattern: PatternAll, Limit: 10},
		{Pattern: ".*/a/.*", Limit: 7},
	}, cfg.ICPLimits[langAlpha])
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want string
	}{
		{"empty document", "", "empty document"},
		{"unknown top-level key", "version: 1\nlimit: 10\n", "field limit not found"},
		{"unknown nested key", "version: 1\nreporter:\n  color: red\n", "field color not found"},
		{
			"metrics language is a list",
			"metrics:\n  alpha:\n    - code_branch\n",
			"line 3: expected a mapping of patterns",
		},
		{"limits language is a scalar", "icp-limits:\n  alpha: 10\n", "line 2: expected a mapping of patterns"},
		{
			"duplicate pattern",
			"metrics:\n  alpha:\n    \".*\": {code_branch: 1}\n    \".*\": {condition: 1}\n",
			`line 4: pattern ".*" already defined at line 3`,
		},
		{"weight is not a number", "metrics:\n  alpha:\n    \".*\": {code_branch: heavy}\n", "line 3"},
		{"limit is not a number", "icp-limits:\n  alpha:\n    \".*\": ten\n", "line 3"},
		{"timeout is not a duration", "timeout: soon\n", "line 1"},
		{"version is not a number", "version: one\n", "line 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.doc))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestParseNullSections(t *testing.T) {
	cfg, err := Parse(strings.NewReader("version: 1\nmetrics:\n  alpha:\nicp-limits:\n  alpha:\n"))
	require.NoError(t, err)
	assert.Empty(t, cfg.Metrics[langAlpha])
	assert.Empty(t, cfg.ICPLimits[langAlpha])
	assert.Nil(t, cfg.Reporter.OutputFile)
}

func TestParseOutputFile(t *testing.T) {
	cfg, err := Parse(strings.NewReader("reporter:\n  format: json\n  outputFile: out.json\n"))
	require.NoError(t, err)
	require.NotNil(t, cfg.Reporter.OutputFile)
	assert.Equal(t, "out.json", *cfg.Reporter.OutputFile)
}
