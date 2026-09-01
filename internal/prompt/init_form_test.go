package prompt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/jonasalessi/cdd-cli/internal/config"
	"github.com/jonasalessi/cdd-cli/internal/detect"
)

func TestValidateLimit(t *testing.T) {
	assert.NoError(t, validateLimit("10"))
	assert.NoError(t, validateLimit(" 1 "))
	assert.Error(t, validateLimit("0"))
	assert.Error(t, validateLimit("-2"))
	assert.Error(t, validateLimit("ten"))
	assert.Error(t, validateLimit("1.5"))
	assert.Error(t, validateLimit(""))
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

func TestMetricOptionsCarryDescriptions(t *testing.T) {
	options := metricOptions(config.DefaultSelection())
	assert.Len(t, options, len(config.Metrics()))
	assert.Contains(t, options[0].Key, "code_branch: ")
}
