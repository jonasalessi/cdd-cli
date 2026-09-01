// Package prompt renders the interactive cdd init interview with huh. It
// holds no rules of its own: defaults come in through initcmd.Answers, every
// answer goes back out unchanged, and initcmd.Build judges them.
package prompt

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"

	"github.com/jonasalessi/cdd-cli/internal/config"
	"github.com/jonasalessi/cdd-cli/internal/detect"
	"github.com/jonasalessi/cdd-cli/internal/initcmd"
)

// ErrAborted is returned when the user cancels the interview with ctrl-c.
var ErrAborted = huh.ErrUserAborted

// displayNames maps language ids to their prompt labels.
var displayNames = map[config.Language]string{
	config.LangGo:         "Go",
	config.LangJava:       "Java",
	config.LangKotlin:     "Kotlin",
	config.LangTypeScript: "TypeScript",
}

// Run walks through the interview in prompt order and returns the answers.
// defaults pre-fills every question; det adds the file counts and, for a
// cut-short scan, the notice to the language question.
func Run(defaults initcmd.Answers, det detect.Detected) (initcmd.Answers, error) {
	a := defaults
	steps := []func(*initcmd.Answers) error{
		func(a *initcmd.Answers) error { return askLanguages(a, det) },
		askProjectType,
		askLegacyMode,
		askLimit,
		askMetrics,
		askWeights,
		askPackages,
		askExcludes,
	}
	for _, step := range steps {
		if err := step(&a); err != nil {
			return a, err
		}
	}
	return a, nil
}

// ConfirmOverwrite asks whether the existing file at path may be replaced.
// The default answer is no.
func ConfirmOverwrite(path string) (bool, error) {
	overwrite := false
	err := runField(huh.NewConfirm().
		Title(fmt.Sprintf("%s already exists. Overwrite?", path)).
		Value(&overwrite))
	return overwrite, err
}

// runField shows a single question as its own form page.
func runField(f huh.Field) error {
	return huh.NewForm(huh.NewGroup(f)).Run()
}

func askLanguages(a *initcmd.Answers, det detect.Detected) error {
	field := huh.NewMultiSelect[config.Language]().
		Title("Languages").
		Options(languageOptions(det, a.Languages)...).
		Validate(validateLanguages).
		Value(&a.Languages)
	if det.Truncated {
		field = field.Description(truncatedNotice(det.Elapsed))
	}
	return runField(field)
}

func askProjectType(a *initcmd.Answers) error {
	if a.ProjectType == "" {
		a.ProjectType = config.ProjectGreenfield
	}
	greenfieldLabel := config.ProjectGreenfield + ": strict from day one, limit 7-14 (cdd.md 4A)"
	legacyLabel := config.ProjectLegacy + ": measure existing, enforce new, limit 20-40"
	return runField(huh.NewSelect[string]().
		Title("Project type").
		Options(
			huh.NewOption(greenfieldLabel, config.ProjectGreenfield),
			huh.NewOption(legacyLabel, config.ProjectLegacy),
		).
		Value(&a.ProjectType))
}

func askLegacyMode(a *initcmd.Answers) error {
	if a.ProjectType != config.ProjectLegacy {
		return nil
	}
	if a.LegacyMode == "" {
		a.LegacyMode = config.ModeStrictOnNewOnly
	}
	return runField(huh.NewSelect[string]().
		Title("Enforcement mode").
		Options(
			huh.NewOption(config.ModeStrictOnNewOnly+": existing files are measured only, new files must comply",
				config.ModeStrictOnNewOnly),
			huh.NewOption(config.ModeBoyScout+": modified files must not raise their baseline ICP (baseline not yet supported)",
				config.ModeBoyScout),
			huh.NewOption(config.ModeMeasureOnly+": report only, never blocks CI", config.ModeMeasureOnly),
		).
		Value(&a.LegacyMode))
}

func askLimit(a *initcmd.Answers) error {
	if a.Limit == 0 {
		a.Limit = config.DefaultLimit(a.ProjectType)
	}
	raw := strconv.Itoa(a.Limit)
	projectType := a.ProjectType
	err := runField(huh.NewInput().
		Title("ICP limit").
		DescriptionFunc(func() string { return limitDescription(projectType, raw) }, &raw).
		Validate(validateLimit).
		Value(&raw))
	if err != nil {
		return err
	}
	limit, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return err
	}
	a.Limit = limit
	return nil
}

// askMetrics keeps asking until every selected language keeps at least the
// minimum number of applicable metrics; initcmd.Build is the judge.
func askMetrics(a *initcmd.Answers) error {
	if len(a.Metrics) == 0 {
		a.Metrics = config.DefaultSelection()
	}
	note := ""
	for {
		field := huh.NewMultiSelect[config.MetricID]().
			Title("Metrics").
			Options(metricOptions(a.Metrics)...).
			Validate(validateMetrics).
			Value(&a.Metrics)
		if note != "" {
			field = field.Description(note)
		}
		if err := runField(field); err != nil {
			return err
		}
		var few initcmd.ErrTooFewMetrics
		if _, _, err := initcmd.Build(*a); errors.As(err, &few) {
			note = few.Error()
			continue
		}
		return nil
	}
}

func askWeights(a *initcmd.Answers) error {
	customize := false
	err := runField(huh.NewConfirm().
		Title("Customize weights?").
		Description("Defaults: 1.0, except external_coupling and local_variable at 0.5").
		Value(&customize))
	if err != nil || !customize {
		return err
	}
	values := make([]string, len(a.Metrics))
	fields := make([]huh.Field, len(a.Metrics))
	for i, id := range a.Metrics {
		weight := config.DefaultWeight(id)
		if override, ok := a.Weights[id]; ok {
			weight = override
		}
		values[i] = strconv.FormatFloat(weight, 'f', -1, 64)
		fields[i] = huh.NewInput().Title(string(id)).Validate(validateWeight).Value(&values[i])
	}
	if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
		return err
	}
	if a.Weights == nil {
		a.Weights = make(map[config.MetricID]float64, len(a.Metrics))
	}
	for i, id := range a.Metrics {
		weight, err := parseWeight(values[i])
		if err != nil {
			return err
		}
		a.Weights[id] = weight
	}
	return nil
}

func askPackages(a *initcmd.Answers) error {
	raw := strings.Join(a.Packages, ", ")
	err := runField(huh.NewInput().
		Title("Internal packages").
		Description("Comma-separated prefixes counted as internal coupling").
		Value(&raw))
	if err != nil {
		return err
	}
	a.Packages = parseCSV(raw)
	return nil
}

func askExcludes(a *initcmd.Answers) error {
	return runField(huh.NewConfirm().
		Title("Exclude tests and generated code?").
		Value(&a.DefaultExcludes))
}

// languageOptions lists every known language, labels the detected ones with
// their file count and pre-checks the defaults.
func languageOptions(det detect.Detected, selected []config.Language) []huh.Option[config.Language] {
	out := make([]huh.Option[config.Language], 0, len(config.Languages()))
	for _, lang := range config.Languages() {
		out = append(out, huh.NewOption(languageLabel(lang, det.Counts[lang]), lang).
			Selected(slices.Contains(selected, lang)))
	}
	return out
}

func languageLabel(lang config.Language, count int) string {
	label := displayNames[lang]
	if label == "" {
		label = string(lang)
	}
	switch count {
	case 0:
		return label
	case 1:
		return label + " (1 file)"
	default:
		return fmt.Sprintf("%s (%d files)", label, count)
	}
}

func truncatedNotice(elapsed time.Duration) string {
	return fmt.Sprintf("scan stopped after %s, tick anything missing", elapsed.Round(time.Millisecond))
}

// metricOptions lists every metric with its description and pre-checks the
// current selection.
func metricOptions(selected []config.MetricID) []huh.Option[config.MetricID] {
	out := make([]huh.Option[config.MetricID], 0, len(config.Metrics()))
	for _, id := range config.Metrics() {
		label := fmt.Sprintf("%s: %s", id, config.MetricDescription(id, ""))
		out = append(out, huh.NewOption(label, id).Selected(slices.Contains(selected, id)))
	}
	return out
}

// limitDescription shows the recommended band and, for a value outside it,
// an inline warning that does not block the input.
func limitDescription(projectType, raw string) string {
	lo, hi := config.LimitBand(projectType)
	base := fmt.Sprintf("Recommended %s band: %d-%d", projectType, lo, hi)
	limit, err := strconv.Atoi(strings.TrimSpace(raw))
	if err == nil && (limit < lo || limit > hi) {
		return fmt.Sprintf("%s. Warning: %d is outside the band, accepted anyway", base, limit)
	}
	return base
}

func validateLanguages(selected []config.Language) error {
	if len(selected) == 0 {
		return errors.New("pick at least one language")
	}
	return nil
}

func validateMetrics(selected []config.MetricID) error {
	if len(selected) < config.MinMetrics {
		return fmt.Errorf("pick at least %d metrics", config.MinMetrics)
	}
	return nil
}

func validateLimit(raw string) error {
	limit, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return errors.New("enter a whole number")
	}
	if limit < 1 {
		return errors.New("the limit must be at least 1")
	}
	return nil
}

func validateWeight(raw string) error {
	_, err := parseWeight(raw)
	return err
}

// parseWeight reads a strictly positive number.
func parseWeight(raw string) (float64, error) {
	weight, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, errors.New("enter a number")
	}
	if weight <= 0 {
		return 0, errors.New("the weight must be above 0")
	}
	return weight, nil
}

// parseCSV splits a comma-separated input, trimming entries and dropping
// empty ones.
func parseCSV(raw string) []string {
	var out []string
	for part := range strings.SplitSeq(raw, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
