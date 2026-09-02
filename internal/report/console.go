package report

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"
)

// The words the console report opens a line with. The format is key-value
// text: one fact per line, no marker, no table and no color, so the reader
// — usually a coding agent — can split a line on spaces and on "=".
const (
	statusPass = "PASS"
	statusFail = "FAIL"
	// labelViolation opens a unit above its limit, labelUnit one within it.
	labelViolation = "violation"
	labelUnit      = "unit"
)

// consoleRecord is one listed unit together with the file it came from: the
// console report orders units across files, so each one carries its path.
type consoleRecord struct {
	path string
	unit Unit
}

// renderConsole writes the key-value report: the verdict on the first line,
// the run's warnings under it, then one record per listed unit — the
// violations first, worst over the limit first — and last what could not be
// measured. Names are identifiers and never contain a space, so a line needs
// no quoting to be split on spaces.
func renderConsole(w io.Writer, doc Report) error {
	p := &printer{w: w}
	consoleHeadline(p, doc)
	for _, warning := range doc.Warnings {
		p.printf("warning: %s\n", warning)
	}
	violations, within := consoleRecords(doc)
	for _, r := range violations {
		consoleUnit(p, r)
	}
	for _, r := range within {
		consoleUnit(p, r)
	}
	consoleWarnings(p, doc)
	return p.err
}

// consoleHeadline writes the verdict: the run fails as soon as one unit is
// over its limit, and the counts are the whole run even when the listing
// below is filtered.
func consoleHeadline(p *printer, doc Report) {
	status := statusPass
	if doc.Summary.Violations > 0 {
		status = statusFail
	}
	p.printf("cdd check: %s violations=%d units=%d root=%s elapsed=%s",
		status, doc.Summary.Violations, doc.Summary.Units, doc.Root, formatElapsed(doc.ElapsedMS))
	if doc.Partial {
		p.printf(" partial=true")
	}
	p.printf("\n")
}

// consoleRecords splits the listed units into the ones over their limit,
// the widest gap first, and the ones within it in the document's own path
// and source order.
func consoleRecords(doc Report) (violations, within []consoleRecord) {
	for _, f := range doc.Files {
		for _, u := range f.Units {
			r := consoleRecord{path: f.Path, unit: u}
			if u.Exceeds {
				violations = append(violations, r)
			} else {
				within = append(within, r)
			}
		}
	}
	slices.SortStableFunc(violations, byOverLimit)
	return violations, within
}

// byOverLimit orders two violations by how far they are above their limit,
// the widest gap first; an equal gap is broken by path and then by line, so
// the report never depends on map or scheduling order.
func byOverLimit(a, b consoleRecord) int {
	if c := cmp.Compare(overLimit(b.unit), overLimit(a.unit)); c != 0 {
		return c
	}
	if c := strings.Compare(a.path, b.path); c != 0 {
		return c
	}
	return cmp.Compare(a.unit.Line, b.unit.Line)
}

// overLimit is how many ICPs a unit is above its limit.
func overLimit(u Unit) float64 {
	return u.Total - float64(u.Limit)
}

// consoleUnit writes one record: where the unit is and what it scored, then
// the metrics that earned the score and, when the report explains itself,
// one "icp" line per counted construct. A blank line opens the record, so
// the reader sees where one ends and the next begins.
func consoleUnit(p *printer, r consoleRecord) {
	u := r.unit
	label, gap := labelUnit, ""
	if u.Exceeds {
		label, gap = labelViolation, " over="+formatNumber(overLimit(u))
	}
	p.printf("\n%s: %s:%d:%d %s %s icp=%s limit=%d%s\n",
		label, r.path, u.Line, u.Col, u.Kind, u.Name, formatNumber(u.Total), u.Limit, gap)
	p.printf("  metrics: %s\n", consoleMetrics(u.Metrics))
	for _, o := range u.constructs() {
		p.printf("  icp: %s\n", occurrenceText(o))
	}
}

// consoleMetrics spells the metrics that earned the ICPs, the heaviest
// first. A metric that never occurred contributed nothing and is left out;
// a unit that scored on nothing reads "none".
func consoleMetrics(ms []Metric) string {
	counted := make([]Metric, 0, len(ms))
	for _, m := range ms {
		if m.Count > 0 {
			counted = append(counted, m)
		}
	}
	if len(counted) == 0 {
		return metricsNone
	}
	slices.SortStableFunc(counted, func(a, b Metric) int { return cmp.Compare(b.Score, a.Score) })
	parts := make([]string, len(counted))
	for i, m := range counted {
		parts[i] = consoleMetric(m)
	}
	return strings.Join(parts, " ")
}

// consoleMetric spells one metric as its bare count when the weight is 1,
// and as "count x weight" otherwise, so the score is always reproducible
// from the line.
func consoleMetric(m Metric) string {
	if m.Score == float64(m.Count) {
		return fmt.Sprintf("%s=%d", m.ID, m.Count)
	}
	return fmt.Sprintf("%s=%dx%s", m.ID, m.Count, formatNumber(m.Score/float64(m.Count)))
}

// consoleWarnings closes the report with what the analyzer could not
// measure, in the document's path order.
func consoleWarnings(p *printer, doc Report) {
	opened := false
	for _, f := range doc.Files {
		for _, warning := range f.Warnings {
			if !opened {
				p.printf("\n")
				opened = true
			}
			p.printf("warning: %s: %s\n", f.Path, warning)
		}
	}
}
