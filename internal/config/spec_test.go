package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpecApplicable(t *testing.T) {
	assert.Equal(t, []MetricID{
		"code_branch", "condition", "internal_coupling", "external_coupling", "local_variable", "lambda",
	}, gammaSpec().Applicable())
	assert.Equal(t, Metrics(), alphaSpec().Applicable(), "no exclusions means every metric, in Metrics order")

	assert.False(t, gammaSpec().IsApplicable(MetricExceptionHandling))
	assert.False(t, gammaSpec().IsApplicable(MetricInheritance))
	assert.True(t, alphaSpec().IsApplicable(MetricInheritance))
	assert.False(t, alphaSpec().IsApplicable("nesting"), "unknown metrics never apply")
}

func TestSpecDescription(t *testing.T) {
	assert.Equal(t, "if/else, switch/select, for", gammaSpec().Description(MetricCodeBranch), "own wording")
	assert.Equal(t, MetricDescription(MetricCondition), gammaSpec().Description(MetricCondition), "generic fallback")
	assert.Equal(t, MetricDescription(MetricCondition), LanguageSpec{}.Description(MetricCondition),
		"a spec without descriptions falls back for everything")
	for _, spec := range testSpecs() {
		for _, m := range spec.Applicable() {
			assert.NotEmpty(t, spec.Description(m), "%s/%s", spec.ID, m)
		}
	}
}

func TestFindSpec(t *testing.T) {
	spec, ok := FindSpec(testSpecs(), langBeta)
	assert.True(t, ok)
	assert.Equal(t, "Beta", spec.DisplayName)

	_, ok = FindSpec(testSpecs(), "rust")
	assert.False(t, ok)
	_, ok = FindSpec(nil, langBeta)
	assert.False(t, ok)
}

func TestLanguageIDs(t *testing.T) {
	assert.Equal(t, []Language{langAlpha, langBeta, langGamma, langDelta}, LanguageIDs(testSpecs()))
	assert.Empty(t, LanguageIDs(nil))
}
