package prompt

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/jonasalessi/cdd-cli/internal/config"
	"github.com/jonasalessi/cdd-cli/internal/detect"
	"github.com/jonasalessi/cdd-cli/internal/initcmd"
)

func TestValidateLimit(t *testing.T) {
	assert.NoError(t, validateLimit("10"))
	assert.NoError(t, validateLimit(" 1 "))
	assert.Error(t, validateLimit("0"))
	assert.Error(t, validateLimit("-2"))
	assert.Error(t, validateLimit("ten"))
	assert.Error(t, validateLimit("1.5"))
	assert.NoError(t, validateLimit(""), "empty keeps the project-type default")
	assert.NoError(t, validateLimit("  "))
}

func TestParseWeight(t *testing.T) {
	w, err := parseWeight(" 0.5 ")
	assert.NoError(t, err)
	assert.Equal(t, 0.5, w)

	_, err = parseWeight("0")
	assert.Error(t, err)
	_, err = parseWeight("-1")
	assert.Error(t, err)
	_, err = parseWeight("heavy")
	assert.Error(t, err)
	assert.Error(t, validateWeight("0"))
	assert.NoError(t, validateWeight("2"))
}

func TestParseCSV(t *testing.T) {
	assert.Equal(t, []string{"a", "b.c"}, parseCSV(" a , b.c ,, "))
	assert.Nil(t, parseCSV(""))
	assert.Nil(t, parseCSV(" , "))
}

func TestValidateLanguages(t *testing.T) {
	assert.Error(t, validateLanguages(nil))
	assert.NoError(t, validateLanguages([]config.Language{config.LangGo}))
}

func TestValidateMetrics(t *testing.T) {
	assert.Error(t, validateMetrics([]config.MetricID{config.MetricCondition}))
	assert.NoError(t, validateMetrics([]config.MetricID{
		config.MetricCondition, config.MetricCodeBranch, config.MetricLambda,
	}))
}

func TestLimitDescription(t *testing.T) {
	inBand := limitDescription(config.ProjectGreenfield, "10")
	assert.Contains(t, inBand, "7-14")
	assert.NotContains(t, inBand, "Warning")

	outOfBand := limitDescription(config.ProjectGreenfield, "50")
	assert.Contains(t, outOfBand, "Warning: 50 is outside the band")

	junk := limitDescription(config.ProjectLegacy, "abc")
	assert.Contains(t, junk, "20-40")
	assert.NotContains(t, junk, "Warning")
}

func TestLanguageLabel(t *testing.T) {
	assert.Equal(t, "Go", languageLabel(config.LangGo, 0))
	assert.Equal(t, "Java (1 file)", languageLabel(config.LangJava, 1))
	assert.Equal(t, "TypeScript (12 files)", languageLabel(config.LangTypeScript, 12))
	assert.Equal(t, "elm", languageLabel(config.Language("elm"), 0), "unknown ids fall back to the id")
}

func TestTruncatedNotice(t *testing.T) {
	notice := truncatedNotice(4 * time.Second)
	assert.Contains(t, notice, "scan stopped after 4s")
}

func TestLanguageOptionsPreselection(t *testing.T) {
	det := detect.Detected{Counts: map[config.Language]int{config.LangGo: 2}}
	options := languageOptions(det, []config.Language{config.LangGo})
	assert.Len(t, options, len(config.Languages()))
	assert.Equal(t, "Go (2 files)", options[0].Key)
	assert.Equal(t, config.LangGo, options[0].Value)
}

// TestGroupBuildersConstruct exercises every page builder; the forms
// themselves are not driven in tests.
func TestGroupBuildersConstruct(t *testing.T) {
	a := initcmd.Answers{Languages: []config.Language{config.LangGo}}
	det := detect.Detected{Truncated: true, Elapsed: time.Second,
		Counts: map[config.Language]int{config.LangGo: 2}}
	raw := ""
	sel := []config.MetricID{config.MetricCodeBranch}
	customize := false
	assert.NotNil(t, languagesGroup(&a, det))
	assert.NotNil(t, projectTypeGroup(&a))
	assert.NotNil(t, legacyModeGroup(&a))
	assert.NotNil(t, limitGroup(&a, &raw))
	assert.NotNil(t, metricsGroup(config.LangGo, &sel))
	assert.NotNil(t, weightsConfirmGroup(&customize))
	assert.NotNil(t, packagesGroup(config.LangGo, &raw))
	assert.NotNil(t, excludesGroup(&a))
}

func TestMetricOptionsOnlyApplicable(t *testing.T) {
	options := metricOptionsFor(config.LangGo, config.DefaultSelection())
	assert.Len(t, options, len(config.Applicable(config.LangGo)))
	assert.Equal(t, "code_branch: if/else, switch/select, for", options[0].Key,
		"labels use the language's own wording")
	for _, opt := range options {
		assert.NotEqual(t, config.MetricInheritance, opt.Value)
		assert.NotEqual(t, config.MetricExceptionHandling, opt.Value)
	}

	javaOptions := metricOptionsFor(config.LangJava, nil)
	assert.Len(t, javaOptions, len(config.Metrics()))
}

func TestHidePredicates(t *testing.T) {
	assert.True(t, hideLegacyMode(config.ProjectGreenfield))
	assert.False(t, hideLegacyMode(config.ProjectLegacy))

	selected := []config.Language{config.LangGo}
	assert.False(t, hideLanguagePage(config.LangGo, selected))
	assert.True(t, hideLanguagePage(config.LangJava, selected))
}

func TestLimitPlaceholder(t *testing.T) {
	assert.Equal(t, "10", limitPlaceholder(config.ProjectGreenfield))
	assert.Equal(t, "25", limitPlaceholder(config.ProjectLegacy))
}

func TestPackagesHintFor(t *testing.T) {
	assert.Contains(t, packagesHintFor(config.LangGo), "github.com/acme/api")
	assert.Contains(t, packagesHintFor(config.LangTypeScript), "@app/")
	assert.NotContains(t, packagesHintFor(config.LangGo), "com.acme.app")

	assert.NotContains(t, packagesHintFor(config.Language("elm")), "e.g.",
		"unknown languages get no example")
}

func TestExcludesDescription(t *testing.T) {
	desc := excludesDescription([]config.Language{config.LangJava, config.LangKotlin})
	assert.Contains(t, desc, "**/src/test/**")
	assert.Equal(t, 1, strings.Count(desc, "**/build/**"), "shared globs listed once")
	assert.NotContains(t, desc, "vendor/**")

	assert.Empty(t, excludesDescription(nil))
}

func TestMetricsUnion(t *testing.T) {
	a := initcmd.Answers{
		Languages: []config.Language{config.LangGo, config.LangJava},
		MetricsByLanguage: map[config.Language][]config.MetricID{
			config.LangGo:   {config.MetricCondition, config.MetricCodeBranch},
			config.LangJava: {config.MetricInheritance, config.MetricCondition},
		},
	}
	assert.Equal(t, []config.MetricID{
		config.MetricCodeBranch, config.MetricCondition, config.MetricInheritance,
	}, metricsUnion(a), "canonical order, no duplicates")
}

func TestWeightDescription(t *testing.T) {
	a := initcmd.Answers{
		Languages: []config.Language{config.LangGo, config.LangJava},
		MetricsByLanguage: map[config.Language][]config.MetricID{
			config.LangGo:   {config.MetricCodeBranch, config.MetricCondition},
			config.LangJava: {config.MetricCodeBranch, config.MetricInheritance},
		},
	}
	shared := weightDescription(a, config.MetricCodeBranch)
	assert.NotContains(t, shared, "applies to", "metrics counted by every language carry no note")
	assert.Equal(t, config.MetricDescription(config.MetricCodeBranch, ""), shared)

	partial := weightDescription(a, config.MetricInheritance)
	assert.Contains(t, partial, "applies to: java")

	goOnly := weightDescription(a, config.MetricCondition)
	assert.Equal(t, config.MetricDescription(config.MetricCondition, config.LangGo)+" — applies to: go", goOnly,
		"a single owner gets its language's wording")
}
