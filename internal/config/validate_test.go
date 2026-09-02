package config

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rules returns the issues of Validate(cfg, testSpecs()) that carry rule.
func rules(t *testing.T, cfg *Config, rule string) Issues {
	t.Helper()
	var out Issues
	for _, i := range Validate(cfg, testSpecs()) {
		if i.Rule == rule {
			out = append(out, i)
		}
	}
	return out
}

func assertOnly(t *testing.T, cfg *Config, rule, severity string, contains string) {
	t.Helper()
	issues := Validate(cfg, testSpecs())
	require.NotEmpty(t, issues, "expected a %s finding", rule)
	for _, i := range issues {
		assert.Equal(t, rule, i.Rule, i)
		assert.Equal(t, severity, i.Severity, i)
	}
	assert.Contains(t, issues.String(), contains)
}

func TestValidateValid(t *testing.T) {
	assert.Empty(t, Validate(valid(), testSpecs()))
	assert.Empty(t, Validate(legacyGammaDelta(), testSpecs()))
}

func TestValidateNil(t *testing.T) {
	issues := Validate(nil, testSpecs())
	require.Len(t, issues, 1)
	assert.Equal(t, RuleVersion, issues[0].Rule)
	assert.True(t, issues.HasErrors())
}

func TestValidateAggregates(t *testing.T) {
	cfg := valid()
	cfg.Version = 2
	cfg.ProjectType = "brownfield"
	cfg.Reporter.Format = "html"
	issues := Validate(cfg, testSpecs())
	assert.Len(t, issues, 3, "never fail-fast: %s", issues)
	assert.Len(t, issues.Errors(), 3)
	assert.Empty(t, issues.Warnings())
}

func TestV1Version(t *testing.T) {
	cfg := valid()
	cfg.Version = 2
	assertOnly(t, cfg, RuleVersion, SeverityError, "version: got 2, want 1")
	cfg.Version = 1
	assert.Empty(t, rules(t, cfg, RuleVersion))
}

func TestV2ProjectType(t *testing.T) {
	cfg := valid()
	cfg.ProjectType = "brownfield"
	issues := rules(t, cfg, RuleProjectType)
	require.Len(t, issues, 1)
	assert.Equal(t, SeverityError, issues[0].Severity)
	assert.Contains(t, issues[0].Message, `"brownfield"`)

	for _, pt := range ProjectTypes() {
		cfg := legacyGammaDelta()
		cfg.ProjectType = pt
		cfg.Enforcement.LegacyMode = ModeStrictAll
		assert.Empty(t, rules(t, cfg, RuleProjectType), pt)
	}
}

func TestV3Languages(t *testing.T) {
	cfg := valid()
	cfg.Metrics = nil
	cfg.ICPLimits = nil
	assertOnly(t, cfg, RuleLanguages, SeverityError, "at least one language")

	cfg = valid()
	cfg.Metrics["rust"] = PatternWeights{
		{Pattern: PatternAll, Weights: map[MetricID]float64{MetricCodeBranch: 1, MetricCondition: 1, MetricLambda: 1}},
	}
	cfg.ICPLimits["rust"] = PatternLimits{{Pattern: PatternAll, Limit: 10}}
	issues := rules(t, cfg, RuleLanguages)
	require.Len(t, issues, 2, "metrics and icp-limits both name the language")
	assert.Contains(t, issues[0].Message, `metrics: "rust"`)
	assert.Contains(t, issues[1].Message, `icp-limits: "rust"`)

	assert.Empty(t, rules(t, valid(), RuleLanguages))

	cfg = valid()
	issues = rules(t, cfg, RuleLanguages)
	assert.Empty(t, issues)
	issues = Validate(cfg, nil)
	assert.Contains(t, issues.String(), `metrics: "gamma" is not one of`, "a language absent from specs is unknown")
}

func TestV4MetricIDs(t *testing.T) {
	cfg := valid()
	cfg.Metrics[langGamma][0].Weights["nesting"] = 1
	assertOnly(t, cfg, RuleMetricIDs, SeverityError, `"nesting" is not one of`)

	cfg = valid()
	cfg.Metrics[langGamma][0].Weights[MetricExceptionHandling] = 1
	assertOnly(t, cfg, RuleMetricIDs, SeverityError, "exception_handling does not apply to gamma")

	cfg = valid()
	cfg.Metrics[langGamma][0].Weights[MetricLambda] = 1
	assert.Empty(t, rules(t, cfg, RuleMetricIDs))
}

func TestV5Weights(t *testing.T) {
	for _, w := range []float64{0, -1} {
		cfg := valid()
		cfg.Metrics[langGamma][0].Weights[MetricCodeBranch] = w
		assertOnly(t, cfg, RuleWeights, SeverityError, "metrics.gamma.\".*\".code_branch: weight")
	}
	cfg := valid()
	cfg.Metrics[langGamma][0].Weights[MetricCodeBranch] = 0.1
	assert.Empty(t, rules(t, cfg, RuleWeights))
}

func TestV6Limits(t *testing.T) {
	for _, limit := range []int{0, -5} {
		cfg := valid()
		cfg.ICPLimits[langGamma][0].Limit = limit
		assertOnly(t, cfg, RuleLimits, SeverityError, "must be >= 1")
	}

	for _, limit := range []int{6, 15} {
		cfg := valid()
		cfg.ICPLimits[langGamma][0].Limit = limit
		assertOnly(t, cfg, RuleLimits, SeverityWarning, "outside the greenfield band 7-14")
		assert.False(t, Validate(cfg, testSpecs()).HasErrors())
	}

	for _, limit := range []int{7, 10, 14} {
		cfg := valid()
		cfg.ICPLimits[langGamma][0].Limit = limit
		assert.Empty(t, rules(t, cfg, RuleLimits), limit)
	}

	cfg := legacyGammaDelta()
	cfg.ICPLimits[langGamma][0].Limit = 41
	assertOnly(t, cfg, RuleLimits, SeverityWarning, "outside the legacy band 20-40")
}

func TestV7Patterns(t *testing.T) {
	cfg := valid()
	cfg.Metrics[langGamma] = append(
		cfg.Metrics[langGamma],
		PatternWeight{Pattern: "(", Weights: map[MetricID]float64{MetricCodeBranch: 1}},
	)
	assertOnly(t, cfg, RulePatterns, SeverityError, `metrics.gamma."(": invalid regex`)

	cfg = valid()
	cfg.ICPLimits[langGamma] = append(cfg.ICPLimits[langGamma], PatternLimit{Pattern: "[", Limit: 10})
	assertOnly(t, cfg, RulePatterns, SeverityError, `icp-limits.gamma."[": invalid regex`)

	cfg = valid()
	cfg.Include = []string{RegexPrefix + "("}
	assertOnly(t, cfg, RulePatterns, SeverityError, "include[0]: invalid regex")

	cfg = valid()
	cfg.Exclude = append(cfg.Exclude, " ")
	assertOnly(t, cfg, RulePatterns, SeverityError, "exclude[2]: empty pattern")

	cfg = valid()
	cfg.Include = []string{GlobPrefix + "src/**", RegexPrefix + `^src/.*\.go$`, "(unbalanced glob is fine"}
	cfg.Metrics[langGamma] = append(
		cfg.Metrics[langGamma],
		PatternWeight{Pattern: `.*/adapters/.*`, Weights: map[MetricID]float64{MetricCodeBranch: 1}},
	)
	assert.Empty(t, rules(t, cfg, RulePatterns))
}

func TestV8DefaultEntry(t *testing.T) {
	cfg := valid()
	cfg.Metrics[langGamma] = PatternWeights{}
	assertOnly(t, cfg, RuleDefaultEntry, SeverityError, `metrics.gamma: no patterns`)

	cfg = valid()
	cfg.Metrics[langGamma][0].Pattern = ".*/adapters/.*"
	assertOnly(t, cfg, RuleDefaultEntry, SeverityError, `metrics.gamma: first pattern is ".*/adapters/.*"`)

	cfg = valid()
	cfg.Metrics[langGamma][0].Weights = map[MetricID]float64{MetricCodeBranch: 1, MetricCondition: 1}
	assertOnly(t, cfg, RuleDefaultEntry, SeverityError, "2 metrics configured, at least 3 are required")

	cfg = valid()
	cfg.ICPLimits[langGamma] = nil
	assertOnly(t, cfg, RuleDefaultEntry, SeverityError, `icp-limits.gamma: no patterns`)

	cfg = valid()
	cfg.ICPLimits[langGamma][0].Pattern = ".*/adapters/.*"
	assertOnly(t, cfg, RuleDefaultEntry, SeverityError, `icp-limits.gamma: first pattern is`)

	cfg = valid()
	cfg.Metrics[langGamma] = append(
		cfg.Metrics[langGamma],
		PatternWeight{Pattern: ".*/adapters/.*", Weights: map[MetricID]float64{MetricCodeBranch: 1}},
	)
	assert.Empty(t, rules(t, cfg, RuleDefaultEntry), "later patterns may have fewer metrics")
}

func TestV9LegacyMode(t *testing.T) {
	cfg := valid()
	cfg.Enforcement.LegacyMode = "lenient"
	assertOnly(t, cfg, RuleLegacyMode, SeverityError, `"lenient" is not one of`)

	cfg = valid()
	cfg.Enforcement.LegacyMode = ModeStrictOnNewOnly
	assertOnly(t, cfg, RuleLegacyMode, SeverityError, "greenfield project requires strict_all")

	cfg = legacyGammaDelta()
	cfg.Enforcement.LegacyMode = ModeMeasureOnly
	assertOnly(t, cfg, RuleLegacyMode, SeverityError, "block_on_ci: must be false when legacy_mode is measure_only")
	cfg.Enforcement.BlockOnCI = false
	assert.Empty(t, Validate(cfg, testSpecs()))

	cfg = legacyGammaDelta()
	cfg.Enforcement.LegacyMode = ModeBoyScout
	assertOnly(t, cfg, RuleLegacyMode, SeverityWarning, "boy_scout needs the baseline store")
	assert.False(t, Validate(cfg, testSpecs()).HasErrors())

	for _, mode := range []string{ModeStrictAll, ModeStrictOnNewOnly} {
		cfg := legacyGammaDelta()
		cfg.Enforcement.LegacyMode = mode
		assert.Empty(t, rules(t, cfg, RuleLegacyMode), mode)
	}
}

func TestV10Reporter(t *testing.T) {
	cfg := valid()
	cfg.Reporter.Format = "html"
	assertOnly(
		t,
		cfg,
		RuleReporter,
		SeverityError,
		`reporter.format: "html" is not one of console, json, xml, markdown`,
	)

	for _, f := range ReporterFormats() {
		cfg := valid()
		cfg.Reporter.Format = f
		assert.Empty(t, rules(t, cfg, RuleReporter), f)
	}
}

func TestV11LanguageSets(t *testing.T) {
	cfg := valid()
	cfg.ICPLimits = nil
	assertOnly(t, cfg, RuleLanguageSets, SeverityError, "icp-limits: missing entry for gamma")

	cfg = valid()
	cfg.ICPLimits[langAlpha] = PatternLimits{{Pattern: PatternAll, Limit: 10}}
	assertOnly(t, cfg, RuleLanguageSets, SeverityError, "metrics: missing entry for alpha")

	assert.Empty(t, rules(t, legacyGammaDelta(), RuleLanguageSets))
}

func TestV12Timeout(t *testing.T) {
	cfg := valid()
	cfg.Timeout = -time.Second
	assertOnly(t, cfg, RuleTimeout, SeverityError, "timeout: -1s must be >= 0")

	for _, d := range []time.Duration{0, time.Minute} {
		cfg := valid()
		cfg.Timeout = d
		assert.Empty(t, rules(t, cfg, RuleTimeout), d)
	}
}

func TestIssues(t *testing.T) {
	issues := Issues{
		{Rule: RuleLimits, Severity: SeverityWarning, Message: "a"},
		{Rule: RuleVersion, Severity: SeverityError, Message: "b"},
	}
	assert.True(t, issues.HasErrors())
	assert.False(t, issues.Warnings().HasErrors())
	assert.Equal(t, "V6 [warning]: a\nV1 [error]: b", issues.String())
	assert.Equal(t, "V1 [error]: b", issues.Errors().String())
}

func TestRoundTrip(t *testing.T) {
	for name, cfg := range goldenConfigs() {
		t.Run(name, func(t *testing.T) {
			for _, patterns := range cfg.Metrics {
				require.Len(t, patterns, 3)
			}
			out, err := Render(cfg, testSpecs())
			require.NoError(t, err)
			parsed, err := Parse(bytes.NewReader(out))
			require.NoError(t, err)
			assert.False(t, Validate(parsed, testSpecs()).HasErrors(), Validate(parsed, testSpecs()))
			assert.Equal(t, cfg, parsed)

			again, err := Render(parsed, testSpecs())
			require.NoError(t, err)
			assert.Equal(t, string(out), string(again), "render is stable")
		})
	}
}
