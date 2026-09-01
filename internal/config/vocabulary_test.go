package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLanguages(t *testing.T) {
	want := []Language{"go", "java", "kotlin", "typescript"}
	assert.Equal(t, want, Languages())
	for _, l := range want {
		assert.True(t, IsLanguage(l), l)
	}
	assert.False(t, IsLanguage("rust"))

	Languages()[0] = "mutated"
	assert.Equal(t, want, Languages(), "accessors must return copies")
}

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

func TestApplicable(t *testing.T) {
	assert.Equal(t, []MetricID{
		"code_branch", "condition", "internal_coupling", "external_coupling", "local_variable", "lambda",
	}, Applicable("go"))
	for _, l := range []Language{"java", "kotlin", "typescript"} {
		assert.Equal(t, Metrics(), Applicable(l), l)
	}
	assert.False(t, IsApplicable("go", "exception_handling"))
	assert.False(t, IsApplicable("go", "inheritance"))
	assert.True(t, IsApplicable("java", "inheritance"))
	assert.False(t, IsApplicable("java", "nesting"), "unknown metrics never apply")
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
		for _, l := range Languages() {
			assert.NotEmpty(t, MetricDescription(m, l), "%s/%s", m, l)
		}
	}
	tests := []struct {
		metric MetricID
		lang   Language
		want   string
	}{
		{"code_branch", "kotlin", "if/when, loops, safe calls (?.), elvis (?:)"},
		{"code_branch", "typescript", "if/else, switch, ternary, loops, ?. and ??"},
		{"code_branch", "go", "if/else, switch/select, for"},
		{"code_branch", "java", "if/else, switch case, ternary, loops"},
		{"condition", "typescript", "&&, || and ?? clauses"},
		{"condition", "java", "&& and || clauses"},
		{"exception_handling", "java", "try / catch / finally blocks"},
		{"inheritance", "kotlin", ": Base() / : Iface, per level"},
		{"inheritance", "java", "extends / implements, per level"},
		{"lambda", "go", "func literals"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, MetricDescription(tt.metric, tt.lang), "%s/%s", tt.metric, tt.lang)
	}
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

func TestDefaultExcludes(t *testing.T) {
	tests := map[Language][]string{
		"go":         {"**/*_test.go", "vendor/**"},
		"java":       {"**/src/test/**", "**/build/**", "**/target/**"},
		"kotlin":     {"**/src/test/**", "**/build/**", "**/target/**"},
		"typescript": {"**/*.test.ts", "**/*.spec.ts", "**/*.d.ts", "**/node_modules/**", "**/dist/**"},
	}
	for l, want := range tests {
		require.Equal(t, want, DefaultExcludes(l), l)
	}
	assert.Empty(t, DefaultExcludes("rust"))
}

func TestConstants(t *testing.T) {
	assert.Equal(t, 1, SchemaVersion)
	assert.Equal(t, 3, MinMetrics)
	assert.Equal(t, ".*", PatternAll)
	assert.Equal(t, "regex:", RegexPrefix)
	assert.Equal(t, "glob:", GlobPrefix)
}
