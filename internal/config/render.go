package config

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"time"
)

//go:embed templates/cdd.config.yaml.tmpl
var templateText string

var configTemplate = template.Must(template.New("cdd.config.yaml").Parse(templateText))

// Render writes cfg as a commented cdd.config.yaml. Languages come out in
// Languages order, patterns in slice order and metrics in Metrics order;
// each weight carries its MetricDescription as an aligned inline comment.
func Render(cfg *Config) ([]byte, error) {
	if cfg == nil {
		return nil, errors.New("config: render: nil config")
	}
	var buf bytes.Buffer
	if err := configTemplate.Execute(&buf, newTemplateData(cfg)); err != nil {
		return nil, fmt.Errorf("config: render: %w", err)
	}
	return buf.Bytes(), nil
}

// templateData is the view model the template ranges over. Every string is
// already quoted and aligned so the template stays a plain document.
type templateData struct {
	Version     int
	ProjectType string
	Metrics     []languageWeights
	Limits      []languageLimits
	BlockOnCI   bool
	LegacyMode  string
	Timeout     string
	Formats     string
	Format      string
	OutputFile  string
	AutoDetect  bool
	Packages    string
	Include     string
	Exclude     string
}

type languageWeights struct {
	Language Language
	Patterns []patternLines
}

type patternLines struct {
	Pattern string
	Lines   []string
}

type languageLimits struct {
	Language Language
	Patterns []patternLimitView
	Examples []string
}

type patternLimitView struct {
	Pattern string
	Limit   int
}

func newTemplateData(cfg *Config) templateData {
	outputFile := "null"
	if cfg.Reporter.OutputFile != nil {
		outputFile = quote(*cfg.Reporter.OutputFile)
	}
	return templateData{
		Version:     cfg.Version,
		ProjectType: cfg.ProjectType,
		Metrics:     metricsView(cfg.Metrics),
		Limits:      limitsView(cfg.ICPLimits),
		BlockOnCI:   cfg.Enforcement.BlockOnCI,
		LegacyMode:  cfg.Enforcement.LegacyMode,
		Timeout:     formatDuration(cfg.Timeout),
		Formats:     strings.Join(ReporterFormats(), ", "),
		Format:      cfg.Reporter.Format,
		OutputFile:  outputFile,
		AutoDetect:  cfg.InternalCoupling.AutoDetect,
		Packages:    formatList(cfg.InternalCoupling.Packages, 4),
		Include:     formatList(cfg.Include, 2),
		Exclude:     formatList(cfg.Exclude, 2),
	}
}

// metricsView lays out every weight line and pads them to a common column
// so the inline comments line up across the whole section.
func metricsView(m map[Language]PatternWeights) []languageWeights {
	type entry struct {
		text, comment string
	}
	var out []languageWeights
	var cells [][]entry
	width := 0
	for _, lang := range Languages() {
		patterns, ok := m[lang]
		if !ok {
			continue
		}
		lw := languageWeights{Language: lang}
		for _, p := range patterns {
			var block []entry
			for _, id := range Metrics() {
				w, ok := p.Weights[id]
				if !ok {
					continue
				}
				text := fmt.Sprintf("%s: %s", id, formatWeight(w))
				width = max(width, len(text)+1)
				block = append(block, entry{text: text, comment: MetricDescription(id, lang)})
			}
			lw.Patterns = append(lw.Patterns, patternLines{Pattern: quote(p.Pattern)})
			cells = append(cells, block)
		}
		out = append(out, lw)
	}
	i := 0
	for li := range out {
		for pi := range out[li].Patterns {
			for _, e := range cells[i] {
				line := fmt.Sprintf("%-*s# %s", width, e.text, e.comment)
				out[li].Patterns[pi].Lines = append(out[li].Patterns[pi].Lines, line)
			}
			i++
		}
	}
	return out
}

func limitsView(m map[Language]PatternLimits) []languageLimits {
	var out []languageLimits
	for _, lang := range Languages() {
		patterns, ok := m[lang]
		if !ok {
			continue
		}
		ll := languageLimits{Language: lang}
		for _, p := range patterns {
			ll.Patterns = append(ll.Patterns, patternLimitView{Pattern: quote(p.Pattern), Limit: p.Limit})
		}
		if len(out) == 0 {
			ll.Examples = limitExamples(lang)
		}
		out = append(out, ll)
	}
	return out
}

// limitExamples are the commented layer overrides shown under the first
// language of icp-limits (docs/cdd.md sections 4B and 5.1).
func limitExamples(lang Language) []string {
	examples := []string{`# ".*/adapters/.*": 8`}
	if lang == LangJava {
		examples = append(examples, `# ".*Dto\\.java": 20`)
	}
	return examples
}

// formatList renders a YAML sequence: " []" inline when empty, otherwise
// one quoted item per line indented by indent spaces.
func formatList(items []string, indent int) string {
	if len(items) == 0 {
		return " []"
	}
	var b strings.Builder
	for _, item := range items {
		fmt.Fprintf(&b, "\n%s- %s", strings.Repeat(" ", indent), quote(item))
	}
	return b.String()
}

// formatWeight prints w with the fewest digits that round-trip, and always
// at least one decimal: 1 -> "1.0", 0.5 -> "0.5", 0.25 -> "0.25".
func formatWeight(w float64) string {
	s := strconv.FormatFloat(w, 'f', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return s
}

// formatDuration prints d in the shortest Go duration syntax that
// time.ParseDuration reads back: "0s", "30s", "5m", "1h30m". Fractions of a
// second fall back to time.Duration.String.
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	if d < 0 {
		return "-" + formatDuration(-d)
	}
	if d%time.Second != 0 {
		return d.String()
	}
	h := d / time.Hour
	m := (d % time.Hour) / time.Minute
	s := (d % time.Minute) / time.Second
	var b strings.Builder
	if h > 0 {
		fmt.Fprintf(&b, "%dh", h)
	}
	if m > 0 {
		fmt.Fprintf(&b, "%dm", m)
	}
	if s > 0 {
		fmt.Fprintf(&b, "%ds", s)
	}
	return b.String()
}

// quote returns s as a double-quoted YAML scalar. Go escapes are a subset of
// YAML double-quoted escapes, so strconv.Quote is enough.
func quote(s string) string {
	return strconv.Quote(s)
}
