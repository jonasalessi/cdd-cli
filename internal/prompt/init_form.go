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

// packageExamples shows, per language, the shape of an internal package
// prefix in the packages question.
var packageExamples = map[config.Language]string{
	config.LangGo:         "github.com/acme/api",
	config.LangJava:       "com.acme.app",
	config.LangKotlin:     "com.acme.app",
	config.LangTypeScript: "@app/",
}

// Run walks through the interview as one multi-page form and returns the
// answers. defaults pre-fills every question; det adds the file counts and,
// for a cut-short scan, the notice to the language question. Pages that do
// not apply to the answers given so far — the enforcement mode of a
// greenfield project, the metrics of an unselected language — stay hidden,
// and shift+tab navigates back.
func Run(defaults initcmd.Answers, det detect.Detected) (initcmd.Answers, error) {
	a := defaults
	if a.ProjectType == "" {
		a.ProjectType = config.ProjectGreenfield
	}
	if a.LegacyMode == "" {
		a.LegacyMode = config.ModeStrictOnNewOnly
	}
	limitRaw := ""
	if a.Limit != 0 {
		limitRaw = strconv.Itoa(a.Limit)
	}
	customize := false

	groups := []*huh.Group{
		languagesGroup(&a, det),
		projectTypeGroup(&a),
		legacyModeGroup(&a).WithHideFunc(func() bool { return hideLegacyMode(a.ProjectType) }),
		limitGroup(&a, &limitRaw),
	}
	selections := make(map[config.Language]*[]config.MetricID, len(config.Languages()))
	for _, lang := range config.Languages() {
		sel := initcmd.SeedMetrics(a, lang)
		selections[lang] = &sel
		groups = append(groups, metricsGroup(lang, selections[lang]).
			WithHideFunc(func() bool { return hideLanguagePage(lang, a.Languages) }))
	}
	groups = append(groups, weightsConfirmGroup(&customize))
	pkgRaws := make(map[config.Language]*string, len(config.Languages()))
	for _, lang := range config.Languages() {
		raw := strings.Join(initcmd.SeedPackages(a, lang), ", ")
		pkgRaws[lang] = &raw
		groups = append(groups, packagesGroup(lang, pkgRaws[lang]).
			WithHideFunc(func() bool { return hideLanguagePage(lang, a.Languages) }))
	}
	groups = append(groups, excludesGroup(&a))
	if err := huh.NewForm(groups...).Run(); err != nil {
		return a, err
	}

	a.Limit = 0
	if raw := strings.TrimSpace(limitRaw); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			return a, err
		}
		a.Limit = limit
	}
	a.MetricsByLanguage = make(map[config.Language][]config.MetricID, len(a.Languages))
	a.PackagesByLanguage = make(map[config.Language][]string, len(a.Languages))
	for _, lang := range a.Languages {
		a.MetricsByLanguage[lang] = *selections[lang]
		if pkgs := parseCSV(*pkgRaws[lang]); len(pkgs) > 0 {
			a.PackagesByLanguage[lang] = pkgs
		}
	}
	a.Packages = nil
	if customize {
		if err := runWeightsForm(&a); err != nil {
			return a, err
		}
	}
	return a, nil
}

// ConfirmOverwrite asks whether the existing file at path may be replaced.
// The default answer is no.
func ConfirmOverwrite(path string) (bool, error) {
	overwrite := false
	err := huh.NewForm(huh.NewGroup(huh.NewConfirm().
		Title(fmt.Sprintf("%s already exists. Overwrite?", path)).
		Value(&overwrite))).Run()
	return overwrite, err
}

func languagesGroup(a *initcmd.Answers, det detect.Detected) *huh.Group {
	description := "Detected languages are pre-checked"
	if det.Truncated {
		description += "; " + truncatedNotice(det.Elapsed)
	}
	return huh.NewGroup(huh.NewMultiSelect[config.Language]().
		Title("Languages").
		Description(description).
		Options(languageOptions(det, a.Languages)...).
		Validate(validateLanguages).
		Value(&a.Languages))
}

func projectTypeGroup(a *initcmd.Answers) *huh.Group {
	greenfieldLabel := config.ProjectGreenfield + ": strict from day one, limit 7-14 (cdd.md 4A)"
	legacyLabel := config.ProjectLegacy + ": measure existing, enforce new, limit 20-40"
	return huh.NewGroup(huh.NewSelect[string]().
		Title("Project type").
		Options(
			huh.NewOption(greenfieldLabel, config.ProjectGreenfield),
			huh.NewOption(legacyLabel, config.ProjectLegacy),
		).
		Value(&a.ProjectType))
}

func legacyModeGroup(a *initcmd.Answers) *huh.Group {
	return huh.NewGroup(huh.NewSelect[string]().
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

func limitGroup(a *initcmd.Answers, raw *string) *huh.Group {
	// The char limit keeps huh's width math non-negative on very narrow
	// terminals, where rendering an empty input's placeholder panics.
	return huh.NewGroup(huh.NewInput().
		Title("ICP limit").
		CharLimit(4).
		PlaceholderFunc(func() string { return limitPlaceholder(a.ProjectType) }, &a.ProjectType).
		DescriptionFunc(func() string { return limitDescription(a.ProjectType, *raw) },
			&struct {
				ProjectType *string
				Raw         *string
			}{&a.ProjectType, raw}).
		Validate(validateLimit).
		Value(raw))
}

func metricsGroup(lang config.Language, sel *[]config.MetricID) *huh.Group {
	name := languageLabel(lang, 0)
	return huh.NewGroup(huh.NewMultiSelect[config.MetricID]().
		Title("Metrics — " + name).
		Description(fmt.Sprintf("Only metrics the %s analyzer can count; pick at least %d", name, config.MinMetrics)).
		Options(metricOptionsFor(lang, *sel)...).
		Validate(validateMetrics).
		Value(sel))
}

func weightsConfirmGroup(customize *bool) *huh.Group {
	return huh.NewGroup(huh.NewConfirm().
		Title("Customize weights?").
		Description("Defaults: 1.0, except external_coupling and local_variable at 0.5").
		Value(customize))
}

func packagesGroup(lang config.Language, raw *string) *huh.Group {
	return huh.NewGroup(huh.NewInput().
		Title("Internal packages — " + languageLabel(lang, 0)).
		Description(packagesHintFor(lang)).
		Value(raw))
}

func excludesGroup(a *initcmd.Answers) *huh.Group {
	return huh.NewGroup(huh.NewConfirm().
		Title("Exclude tests and generated code?").
		DescriptionFunc(func() string { return excludesDescription(a.Languages) }, &a.Languages).
		Value(&a.DefaultExcludes))
}

// runWeightsForm asks one weight per metric selected by any language and
// stores the overrides.
func runWeightsForm(a *initcmd.Answers) error {
	union := metricsUnion(*a)
	values := make([]string, len(union))
	fields := make([]huh.Field, len(union))
	for i, id := range union {
		weight := config.DefaultWeight(id)
		if override, ok := a.Weights[id]; ok {
			weight = override
		}
		values[i] = strconv.FormatFloat(weight, 'f', -1, 64)
		fields[i] = huh.NewInput().
			Title(string(id)).
			Description(weightDescription(*a, id)).
			Validate(validateWeight).
			Value(&values[i])
	}
	if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
		return err
	}
	if a.Weights == nil {
		a.Weights = make(map[config.MetricID]float64, len(union))
	}
	for i, id := range union {
		weight, err := parseWeight(values[i])
		if err != nil {
			return err
		}
		a.Weights[id] = weight
	}
	return nil
}

// hideLegacyMode hides the enforcement-mode page for greenfield projects.
func hideLegacyMode(projectType string) bool {
	return projectType != config.ProjectLegacy
}

// hideLanguagePage hides a per-language page of an unselected language.
func hideLanguagePage(lang config.Language, selected []config.Language) bool {
	return !slices.Contains(selected, lang)
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

// metricOptionsFor lists the metrics the language's analyzer can count, each
// described in the language's own wording, and pre-checks the seed.
func metricOptionsFor(lang config.Language, selected []config.MetricID) []huh.Option[config.MetricID] {
	applicable := config.Applicable(lang)
	out := make([]huh.Option[config.MetricID], 0, len(applicable))
	for _, id := range applicable {
		label := fmt.Sprintf("%s: %s", id, config.MetricDescription(id, lang))
		out = append(out, huh.NewOption(label, id).Selected(slices.Contains(selected, id)))
	}
	return out
}

// metricsUnion joins the per-language selections in canonical metric order.
func metricsUnion(a initcmd.Answers) []config.MetricID {
	present := map[config.MetricID]bool{}
	for _, lang := range a.Languages {
		for _, id := range a.MetricsByLanguage[lang] {
			present[id] = true
		}
	}
	var out []config.MetricID
	for _, id := range config.Metrics() {
		if present[id] {
			out = append(out, id)
		}
	}
	return out
}

// weightDescription describes a weight input: the language's own wording
// when exactly one selected language counts the metric, and a note naming
// the languages it applies to when it is not counted by all of them.
func weightDescription(a initcmd.Answers, id config.MetricID) string {
	var owners []config.Language
	for _, lang := range a.Languages {
		if slices.Contains(a.MetricsByLanguage[lang], id) {
			owners = append(owners, lang)
		}
	}
	lang := config.Language("")
	if len(owners) == 1 {
		lang = owners[0]
	}
	description := config.MetricDescription(id, lang)
	if len(owners) < len(a.Languages) {
		ids := make([]string, len(owners))
		for i, owner := range owners {
			ids[i] = string(owner)
		}
		description += " — applies to: " + strings.Join(ids, ", ")
	}
	return description
}

// packagesHintFor shows the prefix format of the language.
func packagesHintFor(lang config.Language) string {
	base := "Comma-separated prefixes counted as internal coupling"
	example, ok := packageExamples[lang]
	if !ok {
		return base
	}
	return base + ", e.g. " + example
}

// excludesDescription lists the globs a yes answer writes for the selected
// languages, deduplicated in canonical language order.
func excludesDescription(langs []config.Language) string {
	var globs []string
	seen := map[string]bool{}
	for _, lang := range config.Languages() {
		if !slices.Contains(langs, lang) {
			continue
		}
		for _, glob := range config.DefaultExcludes(lang) {
			if !seen[glob] {
				seen[glob] = true
				globs = append(globs, glob)
			}
		}
	}
	if len(globs) == 0 {
		return ""
	}
	return "Will exclude: " + strings.Join(globs, ", ")
}

// limitPlaceholder shows the default limit of the chosen project type.
func limitPlaceholder(projectType string) string {
	return strconv.Itoa(config.DefaultLimit(projectType))
}

// limitDescription shows the recommended band and, for a value outside it,
// an inline warning that does not block the input.
func limitDescription(projectType, raw string) string {
	lo, hi := config.LimitBand(projectType)
	base := fmt.Sprintf("Recommended %s band: %d-%d (empty keeps the default %d)",
		projectType, lo, hi, config.DefaultLimit(projectType))
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

// validateLimit accepts a whole number of at least 1 or an empty input,
// which keeps the default of the project type.
func validateLimit(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	limit, err := strconv.Atoi(raw)
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
