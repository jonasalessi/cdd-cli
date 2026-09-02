package config

import (
	"flag"
	"os"
	"path/filepath"
	"time"
)

// update rewrites the golden files from the fixtures below; UPDATE_GOLDEN=1
// does the same.
var update = flag.Bool("update", false, "rewrite golden files")

func updateGolden() bool {
	return *update || os.Getenv("UPDATE_GOLDEN") == "1"
}

const (
	goldenGreenfield = "greenfield-alpha-beta"
	goldenLegacy     = "legacy-gamma-delta"
)

func goldenPath(name string) string {
	return filepath.Join("testdata", "golden", name+".yaml")
}

// goldenConfigs pairs every golden file with the Config that produces it.
func goldenConfigs() map[string]*Config {
	return map[string]*Config{
		goldenGreenfield: greenfieldAlphaBeta(),
		goldenLegacy:     legacyGammaDelta(),
	}
}

// defaultWeightsFor returns the default selection applicable to spec with
// its default weights, which is what cdd init writes under ".*".
func defaultWeightsFor(spec LanguageSpec) map[MetricID]float64 {
	w := make(map[MetricID]float64)
	for _, id := range DefaultSelection() {
		if spec.IsApplicable(id) {
			w[id] = DefaultWeight(id)
		}
	}
	return w
}

func greenfieldAlphaBeta() *Config {
	return &Config{
		Version:     SchemaVersion,
		ProjectType: ProjectGreenfield,
		Metrics: map[Language]PatternWeights{
			langAlpha: {
				{Pattern: PatternAll, Weights: defaultWeightsFor(alphaSpec())},
				{Pattern: ".*/adapters/.*", Weights: map[MetricID]float64{MetricInternalCoupling: 0.5}},
				{
					Pattern: `.*Dto\.java`,
					Weights: map[MetricID]float64{MetricInternalCoupling: 0.5, MetricExternalCoupling: 0.25},
				},
			},
			langBeta: {
				{Pattern: PatternAll, Weights: defaultWeightsFor(betaSpec())},
				{Pattern: ".*/adapters/.*", Weights: map[MetricID]float64{MetricInternalCoupling: 0.5}},
				{
					Pattern: `.*Dto\.kt`,
					Weights: map[MetricID]float64{MetricInternalCoupling: 0.5, MetricExternalCoupling: 0.25},
				},
			},
		},
		ICPLimits: map[Language]PatternLimits{
			langAlpha: {
				{Pattern: PatternAll, Limit: 10},
				{Pattern: ".*/adapters/.*", Limit: 8},
				{Pattern: `.*Dto\.java`, Limit: 14},
			},
			langBeta: {
				{Pattern: PatternAll, Limit: 10},
				{Pattern: ".*/adapters/.*", Limit: 8},
				{Pattern: `.*Dto\.kt`, Limit: 14},
			},
		},
		Enforcement: Enforcement{BlockOnCI: true, LegacyMode: ModeStrictAll},
		Timeout:     DefaultTimeout(),
		Reporter:    Reporter{Format: FormatConsole},
		InternalCoupling: InternalCoupling{
			AutoDetect: true,
			Packages:   []string{"com.acme.billing"},
		},
		Include: []string{},
		Exclude: alphaSpec().DefaultExcludes,
	}
}

func legacyGammaDelta() *Config {
	return &Config{
		Version:     SchemaVersion,
		ProjectType: ProjectLegacy,
		Metrics: map[Language]PatternWeights{
			langGamma: {
				{Pattern: PatternAll, Weights: defaultWeightsFor(gammaSpec())},
				{Pattern: ".*/adapters/.*", Weights: map[MetricID]float64{MetricExternalCoupling: 1}},
				{Pattern: ".*/usecase/.*", Weights: map[MetricID]float64{MetricInternalCoupling: 0.5}},
			},
			langDelta: {
				{Pattern: PatternAll, Weights: defaultWeightsFor(deltaSpec())},
				{Pattern: ".*/adapters/.*", Weights: map[MetricID]float64{MetricExternalCoupling: 1}},
				{Pattern: ".*/usecase/.*", Weights: map[MetricID]float64{MetricInternalCoupling: 0.5}},
			},
		},
		ICPLimits: map[Language]PatternLimits{
			langGamma: {
				{Pattern: PatternAll, Limit: 25},
				{Pattern: ".*/adapters/.*", Limit: 20},
				{Pattern: ".*/usecase/.*", Limit: 30},
			},
			langDelta: {
				{Pattern: PatternAll, Limit: 25},
				{Pattern: ".*/adapters/.*", Limit: 20},
				{Pattern: ".*/usecase/.*", Limit: 30},
			},
		},
		Enforcement: Enforcement{BlockOnCI: true, LegacyMode: ModeStrictOnNewOnly},
		Timeout:     90 * time.Minute,
		Reporter:    Reporter{Format: FormatConsole},
		InternalCoupling: InternalCoupling{
			AutoDetect: true,
			Packages:   []string{"github.com/acme/billing", "@app/"},
		},
		Include: []string{},
		Exclude: append(gammaSpec().DefaultExcludes, deltaSpec().DefaultExcludes...),
	}
}

// valid returns a minimal configuration that passes every rule without
// warnings, shaped like the one cdd init writes for a single language; the
// validation tests mutate it.
func valid() *Config {
	return &Config{
		Version:     SchemaVersion,
		ProjectType: ProjectGreenfield,
		Metrics: map[Language]PatternWeights{
			langGamma: {{Pattern: PatternAll, Weights: defaultWeightsFor(gammaSpec())}},
		},
		ICPLimits: map[Language]PatternLimits{
			langGamma: {{Pattern: PatternAll, Limit: DefaultLimit(ProjectGreenfield)}},
		},
		Enforcement: Enforcement{BlockOnCI: true, LegacyMode: ModeStrictAll},
		Timeout:     DefaultTimeout(),
		Reporter:    Reporter{Format: FormatConsole},
		InternalCoupling: InternalCoupling{
			AutoDetect: true,
			Packages:   []string{"github.com/acme/api"},
		},
		Include: []string{},
		Exclude: gammaSpec().DefaultExcludes,
	}
}
