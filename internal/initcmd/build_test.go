package initcmd

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

func goAnswers() Answers {
	return Answers{Languages: []config.Language{config.LangGo}, DefaultExcludes: true}
}

func TestBuildGreenfieldDefaults(t *testing.T) {
	cfg, warnings, err := Build(goAnswers())
	require.NoError(t, err)
	assert.Empty(t, warnings)
	assert.Equal(t, config.SchemaVersion, cfg.Version)
	assert.Equal(t, config.ProjectGreenfield, cfg.ProjectType)
	assert.Equal(t, config.ModeStrictAll, cfg.Enforcement.LegacyMode)
	assert.True(t, cfg.Enforcement.BlockOnCI)
	assert.Equal(t, config.DefaultTimeout(), cfg.Timeout)
	assert.Equal(t, config.FormatConsole, cfg.Reporter.Format)
	assert.Nil(t, cfg.Reporter.OutputFile)
	assert.True(t, cfg.InternalCoupling.AutoDetect)
	assert.Empty(t, cfg.Include)
	assert.Equal(t, config.DefaultExcludes(config.LangGo), cfg.Exclude)

	require.Len(t, cfg.Metrics, 1)
	patterns := cfg.Metrics[config.LangGo]
	require.Len(t, patterns, 1)
	assert.Equal(t, config.PatternAll, patterns[0].Pattern)
	assert.Equal(t, map[config.MetricID]float64{
		config.MetricCodeBranch:       1.0,
		config.MetricCondition:        1.0,
		config.MetricInternalCoupling: 1.0,
		config.MetricExternalCoupling: 0.5,
	}, patterns[0].Weights, "exception_handling and inheritance do not apply to go")

	limits := cfg.ICPLimits[config.LangGo]
	require.Len(t, limits, 1)
	assert.Equal(t, config.PatternLimit{Pattern: config.PatternAll, Limit: 10}, limits[0])
}

func TestBuildProjectTypeRules(t *testing.T) {
	tests := map[string]struct {
		projectType, legacyMode string
		wantMode                string
		wantBlock               bool
		wantLimit               int
		wantWarning             string
	}{
		"greenfield forces strict_all and block_on_ci": {
			projectType: config.ProjectGreenfield,
			wantMode:    config.ModeStrictAll, wantBlock: true, wantLimit: 10,
		},
		"greenfield ignores a legacy mode with a warning": {
			projectType: config.ProjectGreenfield, legacyMode: config.ModeMeasureOnly,
			wantMode: config.ModeStrictAll, wantBlock: true, wantLimit: 10,
			wantWarning: "only applies to legacy",
		},
		"legacy defaults to strict_on_new_only": {
			projectType: config.ProjectLegacy,
			wantMode:    config.ModeStrictOnNewOnly, wantBlock: true, wantLimit: 25,
		},
		"measure_only forces block_on_ci false": {
			projectType: config.ProjectLegacy, legacyMode: config.ModeMeasureOnly,
			wantMode: config.ModeMeasureOnly, wantBlock: false, wantLimit: 25,
		},
		"boy_scout warns about the missing baseline": {
			projectType: config.ProjectLegacy, legacyMode: config.ModeBoyScout,
			wantMode: config.ModeBoyScout, wantBlock: true, wantLimit: 25,
			wantWarning: "not supported yet",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			a := goAnswers()
			a.ProjectType = tt.projectType
			a.LegacyMode = tt.legacyMode
			cfg, warnings, err := Build(a)
			require.NoError(t, err)
			assert.Equal(t, tt.wantMode, cfg.Enforcement.LegacyMode)
			assert.Equal(t, tt.wantBlock, cfg.Enforcement.BlockOnCI)
			assert.Equal(t, tt.wantLimit, cfg.ICPLimits[config.LangGo][0].Limit)
			if tt.wantWarning == "" {
				assert.Empty(t, warnings)
			} else {
				require.Len(t, warnings, 1)
				assert.Contains(t, warnings[0], tt.wantWarning)
			}
		})
	}
}

func TestBuildLimit(t *testing.T) {
	t.Run("outside the band warns but is accepted", func(t *testing.T) {
		a := goAnswers()
		a.Limit = 50
		cfg, warnings, err := Build(a)
		require.NoError(t, err)
		assert.Equal(t, 50, cfg.ICPLimits[config.LangGo][0].Limit)
		require.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "outside the greenfield band 7-14")
	})
	t.Run("below 1 is an error", func(t *testing.T) {
		a := goAnswers()
		a.Limit = -3
		_, _, err := Build(a)
		assert.ErrorContains(t, err, "at least 1")
	})
}

func TestBuildVocabularyErrors(t *testing.T) {
	tests := map[string]struct {
		mutate  func(*Answers)
		wantErr string
	}{
		"no languages":     {func(a *Answers) { a.Languages = nil }, "at least one language"},
		"unknown language": {func(a *Answers) { a.Languages = []config.Language{"rust"} }, `"rust"`},
		"unknown project type": {
			func(a *Answers) { a.ProjectType = "brownfield" }, `"brownfield"`,
		},
		"unknown legacy mode": {
			func(a *Answers) { a.ProjectType = config.ProjectLegacy; a.LegacyMode = "yolo" }, `"yolo"`,
		},
		"unknown legacy mode on greenfield": {
			func(a *Answers) { a.ProjectType = config.ProjectGreenfield; a.LegacyMode = "measur_only" },
			`"measur_only" is not one of`,
		},
		"unknown metric": {
			func(a *Answers) { a.Metrics = []config.MetricID{"karma", config.MetricCondition, config.MetricLambda} },
			`"karma"`,
		},
		"too few metrics overall": {
			func(a *Answers) { a.Metrics = []config.MetricID{config.MetricCondition, config.MetricLambda} },
			"at least 3 are required",
		},
		"weight for unknown metric": {
			func(a *Answers) { a.Weights = map[config.MetricID]float64{"karma": 1} }, `"karma"`,
		},
		"weight for unselected metric": {
			func(a *Answers) { a.Weights = map[config.MetricID]float64{config.MetricLambda: 1} },
			"not selected",
		},
		"weight of zero": {
			func(a *Answers) { a.Weights = map[config.MetricID]float64{config.MetricCodeBranch: 0} },
			"must be above 0",
		},
		"negative timeout": {func(a *Answers) { a.Timeout = -time.Second }, "must not be negative"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			a := goAnswers()
			tt.mutate(&a)
			_, _, err := Build(a)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestBuildTooFewMetricsForLanguage(t *testing.T) {
	a := goAnswers()
	a.Metrics = []config.MetricID{
		config.MetricExceptionHandling, config.MetricInheritance, config.MetricCondition,
	}
	_, _, err := Build(a)
	var few ErrTooFewMetrics
	require.ErrorAs(t, err, &few)
	assert.Equal(t, config.LangGo, few.Language)
	assert.Equal(t, 1, few.Have)
	assert.Equal(t, config.MinMetrics, few.Need)
	assert.Contains(t, err.Error(), "go")
}

func TestBuildMetricsByLanguage(t *testing.T) {
	t.Run("each language keeps its own selection", func(t *testing.T) {
		a := Answers{
			Languages: []config.Language{config.LangGo, config.LangJava},
			MetricsByLanguage: map[config.Language][]config.MetricID{
				config.LangGo: {
					config.MetricCodeBranch, config.MetricCondition, config.MetricLambda,
				},
				config.LangJava: {
					config.MetricCodeBranch, config.MetricInheritance,
					config.MetricExceptionHandling, config.MetricInternalCoupling,
				},
			},
		}
		cfg, _, err := Build(a)
		require.NoError(t, err)
		assert.Len(t, cfg.Metrics[config.LangGo][0].Weights, 3)
		assert.Len(t, cfg.Metrics[config.LangJava][0].Weights, 4)
		assert.Contains(t, cfg.Metrics[config.LangJava][0].Weights, config.MetricInheritance)
		assert.NotContains(t, cfg.Metrics[config.LangGo][0].Weights, config.MetricInheritance)
	})
	t.Run("a language missing from the map falls back to the global list", func(t *testing.T) {
		a := Answers{
			Languages: []config.Language{config.LangGo, config.LangJava},
			Metrics: []config.MetricID{
				config.MetricCodeBranch, config.MetricCondition, config.MetricInternalCoupling,
			},
			MetricsByLanguage: map[config.Language][]config.MetricID{
				config.LangJava: {
					config.MetricCodeBranch, config.MetricInheritance, config.MetricExceptionHandling,
				},
			},
		}
		cfg, _, err := Build(a)
		require.NoError(t, err)
		assert.Len(t, cfg.Metrics[config.LangGo][0].Weights, 3)
		assert.Contains(t, cfg.Metrics[config.LangJava][0].Weights, config.MetricInheritance)
	})
	t.Run("without map and list every language gets the default selection", func(t *testing.T) {
		a := Answers{Languages: []config.Language{config.LangGo, config.LangJava}}
		cfg, _, err := Build(a)
		require.NoError(t, err)
		assert.Len(t, cfg.Metrics[config.LangGo][0].Weights, 4,
			"default selection minus exception_handling and inheritance")
		assert.Len(t, cfg.Metrics[config.LangJava][0].Weights, len(config.DefaultSelection()))
	})
	t.Run("unknown id in a map entry names it", func(t *testing.T) {
		a := goAnswers()
		a.MetricsByLanguage = map[config.Language][]config.MetricID{
			config.LangGo: {"karma", config.MetricCondition, config.MetricCodeBranch},
		}
		_, _, err := Build(a)
		assert.ErrorContains(t, err, `"karma"`)
	})
	t.Run("map entry left under the minimum after filtering", func(t *testing.T) {
		a := goAnswers()
		a.MetricsByLanguage = map[config.Language][]config.MetricID{
			config.LangGo: {
				config.MetricCondition, config.MetricExceptionHandling, config.MetricInheritance,
			},
		}
		_, _, err := Build(a)
		var few ErrTooFewMetrics
		require.ErrorAs(t, err, &few)
		assert.Equal(t, config.LangGo, few.Language)
		assert.Equal(t, 1, few.Have)
	})
}

func TestBuildWeightsAgainstUnion(t *testing.T) {
	t.Run("a weight for a metric one language selected is applied there only", func(t *testing.T) {
		a := Answers{
			Languages: []config.Language{config.LangGo, config.LangJava},
			MetricsByLanguage: map[config.Language][]config.MetricID{
				config.LangGo: {
					config.MetricCodeBranch, config.MetricCondition, config.MetricInternalCoupling,
				},
				config.LangJava: {
					config.MetricCodeBranch, config.MetricCondition, config.MetricInheritance,
				},
			},
			Weights: map[config.MetricID]float64{config.MetricInheritance: 2},
		}
		cfg, _, err := Build(a)
		require.NoError(t, err)
		assert.Equal(t, 2.0, cfg.Metrics[config.LangJava][0].Weights[config.MetricInheritance])
		assert.NotContains(t, cfg.Metrics[config.LangGo][0].Weights, config.MetricInheritance)
	})
	t.Run("a weight for a metric no language selected is rejected", func(t *testing.T) {
		a := goAnswers()
		a.Weights = map[config.MetricID]float64{config.MetricLambda: 2}
		_, _, err := Build(a)
		assert.ErrorContains(t, err, "not selected")
	})
	t.Run("a weight no selected language can count is rejected, not dropped", func(t *testing.T) {
		a := goAnswers()
		a.Weights = map[config.MetricID]float64{config.MetricInheritance: 2}
		_, _, err := Build(a)
		assert.ErrorContains(t, err, "none of the selected languages can count",
			"inheritance is in the default selection but go cannot count it")
	})
}

func TestSeedMetrics(t *testing.T) {
	t.Run("defaults filtered to what go can count", func(t *testing.T) {
		assert.Equal(t, []config.MetricID{
			config.MetricCodeBranch, config.MetricCondition,
			config.MetricInternalCoupling, config.MetricExternalCoupling,
		}, SeedMetrics(Answers{}, config.LangGo))
	})
	t.Run("java keeps the full default selection", func(t *testing.T) {
		assert.Equal(t, config.DefaultSelection(), SeedMetrics(Answers{}, config.LangJava))
	})
	t.Run("a global list is respected, filtered and kept in order", func(t *testing.T) {
		a := Answers{Metrics: []config.MetricID{
			config.MetricLambda, config.MetricInheritance, config.MetricCodeBranch,
		}}
		assert.Equal(t, []config.MetricID{config.MetricLambda, config.MetricCodeBranch},
			SeedMetrics(a, config.LangGo))
	})
}

func TestBuildWeightOverride(t *testing.T) {
	a := goAnswers()
	a.Weights = map[config.MetricID]float64{config.MetricExternalCoupling: 0.25}
	cfg, _, err := Build(a)
	require.NoError(t, err)
	weights := cfg.Metrics[config.LangGo][0].Weights
	assert.Equal(t, 0.25, weights[config.MetricExternalCoupling])
	assert.Equal(t, 1.0, weights[config.MetricCodeBranch], "unlisted metrics keep the default")
}

func TestBuildExcludes(t *testing.T) {
	t.Run("java and kotlin share their globs without duplicates", func(t *testing.T) {
		a := Answers{
			Languages:       []config.Language{config.LangKotlin, config.LangJava},
			DefaultExcludes: true,
		}
		cfg, _, err := Build(a)
		require.NoError(t, err)
		assert.Equal(t, config.DefaultExcludes(config.LangJava), cfg.Exclude)
	})
	t.Run("disabled leaves the list empty", func(t *testing.T) {
		a := goAnswers()
		a.DefaultExcludes = false
		cfg, _, err := Build(a)
		require.NoError(t, err)
		assert.Empty(t, cfg.Exclude)
	})
}

func TestBuildPackagesPassthrough(t *testing.T) {
	a := goAnswers()
	a.Packages = []string{" com.acme ", "", "com.acme", "org.other"}
	cfg, _, err := Build(a)
	require.NoError(t, err)
	assert.Equal(t, []string{"com.acme", "org.other"}, cfg.InternalCoupling.Packages)
}

func TestBuildPackagesByLanguage(t *testing.T) {
	t.Run("merged in canonical order without duplicates", func(t *testing.T) {
		a := Answers{
			Languages: []config.Language{config.LangJava, config.LangGo},
			Packages:  []string{"shared.prefix"},
			PackagesByLanguage: map[config.Language][]string{
				config.LangGo:   {"github.com/acme/api", "shared.prefix"},
				config.LangJava: {"com.acme.app"},
			},
		}
		cfg, _, err := Build(a)
		require.NoError(t, err)
		assert.Equal(t, []string{"shared.prefix", "github.com/acme/api", "com.acme.app"},
			cfg.InternalCoupling.Packages)
	})
	t.Run("entries of unselected languages are dropped", func(t *testing.T) {
		a := goAnswers()
		a.PackagesByLanguage = map[config.Language][]string{
			config.LangGo:   {"github.com/acme/api"},
			config.LangJava: {"com.acme.app"},
		}
		cfg, _, err := Build(a)
		require.NoError(t, err)
		assert.Equal(t, []string{"github.com/acme/api"}, cfg.InternalCoupling.Packages)
	})
}

func TestSeedPackages(t *testing.T) {
	byLang := Answers{
		Packages: []string{"flag.prefix"},
		PackagesByLanguage: map[config.Language][]string{
			config.LangGo: {"github.com/acme/api"},
		},
	}
	assert.Equal(t, []string{"github.com/acme/api"}, SeedPackages(byLang, config.LangGo))
	assert.Empty(t, SeedPackages(byLang, config.LangJava),
		"a language absent from the map seeds empty when the map exists")

	flat := Answers{Packages: []string{"flag.prefix"}}
	assert.Equal(t, []string{"flag.prefix"}, SeedPackages(flat, config.LangGo),
		"without the map every page seeds from the flat list")
}

func TestBuildDuplicateInputsAreDropped(t *testing.T) {
	a := Answers{
		Languages: []config.Language{config.LangGo, config.LangGo},
		Metrics: []config.MetricID{
			config.MetricCodeBranch, config.MetricCodeBranch,
			config.MetricCondition, config.MetricInternalCoupling,
		},
		DefaultExcludes: true,
	}
	cfg, _, err := Build(a)
	require.NoError(t, err)
	require.Len(t, cfg.Metrics, 1)
	assert.Len(t, cfg.Metrics[config.LangGo][0].Weights, 3)
}

func TestBuildCustomTimeout(t *testing.T) {
	a := goAnswers()
	a.Timeout = 30 * time.Second
	cfg, _, err := Build(a)
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
}

func TestBuildResultValidates(t *testing.T) {
	a := Answers{
		Languages:   []config.Language{config.LangJava, config.LangKotlin},
		ProjectType: config.ProjectLegacy,
		LegacyMode:  config.ModeMeasureOnly,
		Packages:    []string{"com.acme"},
	}
	cfg, _, err := Build(a)
	require.NoError(t, err)
	issues := config.Validate(cfg)
	assert.False(t, issues.HasErrors(), "issues: %v", issues)
}

func TestErrTooFewMetricsIsError(t *testing.T) {
	err := error(ErrTooFewMetrics{Language: config.LangGo, Have: 2, Need: 3})
	assert.False(t, errors.Is(err, ErrExists))
	assert.ErrorContains(t, err, "at least 3")
}
