package config

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// update rewrites the golden files and the dogfood cdd.config.yaml from the
// fixtures below. UPDATE_GOLDEN=1 does the same for the golden files only.
var update = flag.Bool("update", false, "rewrite golden files and cdd.config.yaml")

func updateGolden() bool {
	return *update || os.Getenv("UPDATE_GOLDEN") == "1"
}

const (
	goldenGreenfield = "greenfield-java-kotlin"
	goldenLegacy     = "legacy-go-typescript"
)

func goldenPath(name string) string {
	return filepath.Join("testdata", "golden", name+".yaml")
}

// goldenConfigs pairs every golden file with the Config that produces it.
func goldenConfigs() map[string]*Config {
	return map[string]*Config{
		goldenGreenfield: greenfieldJavaKotlin(),
		goldenLegacy:     legacyGoTypeScript(),
	}
}

// defaultWeights returns the default selection applicable to l with its
// default weights, which is what cdd init writes under ".*".
func defaultWeightsFor(l Language) map[MetricID]float64 {
	w := make(map[MetricID]float64)
	for _, id := range DefaultSelection() {
		if IsApplicable(l, id) {
			w[id] = DefaultWeight(id)
		}
	}
	return w
}

func greenfieldJavaKotlin() *Config {
	return &Config{
		Version:     SchemaVersion,
		ProjectType: ProjectGreenfield,
		Metrics: map[Language]PatternWeights{
			LangJava: {
				{Pattern: PatternAll, Weights: defaultWeightsFor(LangJava)},
				{Pattern: ".*/adapters/.*", Weights: map[MetricID]float64{MetricInternalCoupling: 0.5}},
				{
					Pattern: `.*Dto\.java`,
					Weights: map[MetricID]float64{MetricInternalCoupling: 0.5, MetricExternalCoupling: 0.25},
				},
			},
			LangKotlin: {
				{Pattern: PatternAll, Weights: defaultWeightsFor(LangKotlin)},
				{Pattern: ".*/adapters/.*", Weights: map[MetricID]float64{MetricInternalCoupling: 0.5}},
				{
					Pattern: `.*Dto\.kt`,
					Weights: map[MetricID]float64{MetricInternalCoupling: 0.5, MetricExternalCoupling: 0.25},
				},
			},
		},
		ICPLimits: map[Language]PatternLimits{
			LangJava: {
				{Pattern: PatternAll, Limit: 10},
				{Pattern: ".*/adapters/.*", Limit: 8},
				{Pattern: `.*Dto\.java`, Limit: 14},
			},
			LangKotlin: {
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
		Exclude: DefaultExcludes(LangJava),
	}
}

func legacyGoTypeScript() *Config {
	return &Config{
		Version:     SchemaVersion,
		ProjectType: ProjectLegacy,
		Metrics: map[Language]PatternWeights{
			LangGo: {
				{Pattern: PatternAll, Weights: defaultWeightsFor(LangGo)},
				{Pattern: ".*/adapters/.*", Weights: map[MetricID]float64{MetricExternalCoupling: 1}},
				{Pattern: ".*/usecase/.*", Weights: map[MetricID]float64{MetricInternalCoupling: 0.5}},
			},
			LangTypeScript: {
				{Pattern: PatternAll, Weights: defaultWeightsFor(LangTypeScript)},
				{Pattern: ".*/adapters/.*", Weights: map[MetricID]float64{MetricExternalCoupling: 1}},
				{Pattern: ".*/usecase/.*", Weights: map[MetricID]float64{MetricInternalCoupling: 0.5}},
			},
		},
		ICPLimits: map[Language]PatternLimits{
			LangGo: {
				{Pattern: PatternAll, Limit: 25},
				{Pattern: ".*/adapters/.*", Limit: 20},
				{Pattern: ".*/usecase/.*", Limit: 30},
			},
			LangTypeScript: {
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
		Exclude: append(DefaultExcludes(LangGo), DefaultExcludes(LangTypeScript)...),
	}
}

// dogfood is the configuration of this repository (cdd.config.yaml).
func dogfood() *Config {
	return &Config{
		Version:     SchemaVersion,
		ProjectType: ProjectGreenfield,
		Metrics: map[Language]PatternWeights{
			LangGo: {{Pattern: PatternAll, Weights: defaultWeightsFor(LangGo)}},
		},
		ICPLimits: map[Language]PatternLimits{
			LangGo: {{Pattern: PatternAll, Limit: DefaultLimit(ProjectGreenfield)}},
		},
		Enforcement: Enforcement{BlockOnCI: true, LegacyMode: ModeStrictAll},
		Timeout:     DefaultTimeout(),
		Reporter:    Reporter{Format: FormatConsole},
		InternalCoupling: InternalCoupling{
			AutoDetect: true,
			Packages:   []string{"github.com/jonasalessi/cdd-cli"},
		},
		Include: []string{},
		Exclude: DefaultExcludes(LangGo),
	}
}

// valid returns a minimal configuration that passes every rule without
// warnings; the validation tests mutate it.
func valid() *Config {
	return dogfood()
}

func TestDogfoodMatchesRender(t *testing.T) {
	path := filepath.Join("..", "..", "cdd.config.yaml")
	want, err := Render(dogfood())
	require.NoError(t, err)
	if *update {
		require.NoError(t, os.WriteFile(path, want, 0o644))
	}
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(
		t,
		string(want),
		string(got),
		"cdd.config.yaml drifted from Render; run go test ./internal/config -update",
	)

	cfg, err := Load(path)
	require.NoError(t, err)
	require.Empty(t, Validate(cfg), "the dogfood file must validate without warnings")
}
