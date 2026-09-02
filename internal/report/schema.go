package report

import (
	"encoding/xml"

	"github.com/jonasalessi/cdd-cli/internal/analyze"
	"github.com/jonasalessi/cdd-cli/internal/config"
)

// Report is the document every format renders and the schema of the json
// and xml outputs. It is the run seen from outside: where it ran, what it
// measured, what it could not measure, and how long it took.
type Report struct {
	XMLName xml.Name `json:"-" xml:"report"`
	// Root is the directory every File.Path is relative to.
	Root string `json:"root" xml:"root,attr"`
	// Partial is true when the timeout elapsed before every file was
	// analyzed: Files then holds only what was finished in time.
	Partial bool `json:"partial" xml:"partial,attr"`
	// ElapsedMS is the wall-clock time of the run in milliseconds.
	ElapsedMS int64 `json:"elapsed_ms" xml:"elapsed_ms,attr"`
	// Files are the analyzed files in the order the run produced them.
	Files []File `json:"files" xml:"files>file"`
	// Warnings concern the run rather than one file.
	Warnings []string `json:"warnings,omitempty" xml:"warnings>warning,omitempty"`
	// Summary counts what the reader wants first.
	Summary Summary `json:"summary" xml:"summary"`
}

// File is one analyzed file. A file with warnings and no units was skipped.
type File struct {
	// Path is slash-separated and relative to Report.Root.
	Path string `json:"path" xml:"path,attr"`
	// Language is the canonical language id the analyzer was chosen by.
	Language string `json:"language" xml:"language,attr"`
	// Units are the measured units in source order.
	Units []Unit `json:"units" xml:"units>unit"`
	// Warnings are the analyzer's findings for this file.
	Warnings []string `json:"warnings,omitempty" xml:"warnings>warning,omitempty"`
}

// Unit is one measured code unit after weights and limits were applied.
type Unit struct {
	// Name identifies the unit for the reader.
	Name string `json:"name" xml:"name,attr"`
	// Kind says what the unit is, e.g. a class, a function or a file.
	Kind string `json:"kind" xml:"kind,attr"`
	// Line and Col locate the unit's declaration, 1-based.
	Line int `json:"line" xml:"line,attr"`
	Col  int `json:"col"  xml:"col,attr"`
	// Total is the sum of the metric scores: the unit's ICPs.
	Total float64 `json:"total" xml:"total,attr"`
	// Limit is the ICP limit resolved for the file.
	Limit int `json:"limit" xml:"limit,attr"`
	// Exceeds is Total > Limit.
	Exceeds bool `json:"exceeds" xml:"exceeds,attr"`
	// Metrics are the metrics enabled for the file, in the canonical order
	// of config.Metrics(). A metric disabled for the file is absent, a
	// metric that never occurred has count and score zero.
	Metrics []Metric `json:"metrics" xml:"metric"`
}

// Metric is one metric of one unit. Score is Count multiplied by the weight
// the configuration gives the metric for that file, so the weight itself is
// Score divided by Count whenever Count is not zero.
type Metric struct {
	// ID is the canonical metric id.
	ID string `json:"metric" xml:"id,attr"`
	// Count is the raw number of occurrences, before any weight.
	Count int `json:"count" xml:"count,attr"`
	// Score is the weighed contribution of the metric to Unit.Total.
	Score float64 `json:"score" xml:"score,attr"`
}

// Summary counts the whole run.
type Summary struct {
	// Units is every measured unit.
	Units int `json:"units" xml:"units,attr"`
	// Violations is the units above their limit.
	Violations int `json:"violations" xml:"violations,attr"`
}

// newReport turns a run into the document every format renders.
func newReport(res analyze.RunResult) Report {
	files := make([]File, 0, len(res.Files))
	for _, f := range res.Files {
		files = append(files, newFile(f))
	}
	return Report{
		Root:      res.Root,
		Partial:   res.Partial,
		ElapsedMS: res.Elapsed.Milliseconds(),
		Files:     files,
		Warnings:  res.Warnings,
		Summary:   Summary{Units: res.UnitCount(), Violations: res.Violations()},
	}
}

// newFile turns one weighed file into its document entry.
func newFile(f analyze.FileReport) File {
	units := make([]Unit, 0, len(f.Units))
	for _, u := range f.Units {
		units = append(units, newUnit(u))
	}
	return File{
		Path:     f.Path,
		Language: string(f.Language),
		Units:    units,
		Warnings: f.Warnings,
	}
}

// newUnit turns one weighed unit into its document entry, its metrics
// ordered by config.Metrics() so every format lists them the same way.
func newUnit(u analyze.UnitReport) Unit {
	metrics := make([]Metric, 0, len(u.Counts))
	for _, id := range config.Metrics() {
		count, ok := u.Counts[id]
		if !ok {
			continue
		}
		metrics = append(metrics, Metric{ID: string(id), Count: count, Score: u.Scores[id]})
	}
	return Unit{
		Name:    u.Name,
		Kind:    u.Kind,
		Line:    u.Line,
		Col:     u.Col,
		Total:   u.Total,
		Limit:   u.Limit,
		Exceeds: u.Exceeds,
		Metrics: metrics,
	}
}
