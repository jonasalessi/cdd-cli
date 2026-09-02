package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	want := []MetricID{
		"code_branch", "condition", "exception_handling", "internal_coupling",
		"external_coupling", "inheritance", "local_variable", "lambda",
	}
	assert.Equal(t, want, Metrics())
	for _, m := range want {
		assert.True(t, IsMetric(m), m)
	}
	assert.False(t, IsMetric("nesting"))
}

func TestDefaultWeight(t *testing.T) {
	want := map[MetricID]float64{
		"code_branch":        1.0,
		"condition":          1.0,
		"exception_handling": 1.0,
		"internal_coupling":  1.0,
		"external_coupling":  0.5,
		"inheritance":        1.0,
		"local_variable":     0.5,
		"lambda":             1.0,
	}
	for m, w := range want {
		assert.Equal(t, w, DefaultWeight(m), m)
	}
}

func TestDefaultSelection(t *testing.T) {
	want := []MetricID{
		"code_branch", "condition", "exception_handling", "internal_coupling", "external_coupling", "inheritance",
	}
	assert.Equal(t, want, DefaultSelection())
}

func TestMetricDescription(t *testing.T) {
	for _, m := range Metrics() {
		assert.NotEmpty(t, MetricDescription(m), m)
	}
	assert.Equal(t, "if/else, switch case, ternary, loops", MetricDescription("code_branch"))
	assert.Equal(t, "&& and || clauses", MetricDescription("condition"))
	assert.Equal(t, "try / catch / finally blocks", MetricDescription("exception_handling"))
	assert.Equal(t, "extends / implements, per level", MetricDescription("inheritance"))
	assert.Empty(t, MetricDescription("nesting"))
}

func TestProjectTypes(t *testing.T) {
	assert.Equal(t, []string{"greenfield", "legacy"}, ProjectTypes())
	assert.True(t, IsProjectType("greenfield"))
	assert.True(t, IsProjectType("legacy"))
	assert.False(t, IsProjectType("brownfield"))
}

func TestLimits(t *testing.T) {
	assert.Equal(t, 10, DefaultLimit("greenfield"))
	assert.Equal(t, 25, DefaultLimit("legacy"))
	assert.Equal(t, 10, DefaultLimit("unknown"), "unknown types fall back to greenfield")

	lo, hi := LimitBand("greenfield")
	assert.Equal(t, [2]int{7, 14}, [2]int{lo, hi})
	lo, hi = LimitBand("legacy")
	assert.Equal(t, [2]int{20, 40}, [2]int{lo, hi})
}

func TestLegacyModes(t *testing.T) {
	want := []string{"strict_all", "strict_on_new_only", "boy_scout", "measure_only"}
	assert.Equal(t, want, LegacyModes())
	for _, m := range want {
		assert.True(t, IsLegacyMode(m), m)
	}
	assert.False(t, IsLegacyMode("lenient"))
}

func TestReporterFormats(t *testing.T) {
	want := []string{"console", "json", "xml", "markdown"}
	assert.Equal(t, want, ReporterFormats())
	for _, f := range want {
		assert.True(t, IsReporterFormat(f), f)
	}
	assert.False(t, IsReporterFormat("html"))
}

func TestTimeouts(t *testing.T) {
	assert.Equal(t, 5*time.Minute, DefaultTimeout())
	assert.Equal(t, 4*time.Second, DefaultScanTimeout())
}

func TestConstants(t *testing.T) {
	assert.Equal(t, 1, SchemaVersion)
	assert.Equal(t, 3, MinMetrics)
	assert.Equal(t, ".*", PatternAll)
	assert.Equal(t, "regex:", RegexPrefix)
	assert.Equal(t, "glob:", GlobPrefix)
}
