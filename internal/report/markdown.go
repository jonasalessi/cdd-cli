package report

import (
	"io"
	"strings"
)

// The table header of a file section, and the status words of a unit.
const (
	tableHeader = "| Unit | Kind | Location | ICPs | Limit | Status |\n" +
		"| --- | --- | --- | --- | --- | --- |\n"
	statusOK   = "ok"
	statusOver = "over limit"
	// statusNone stands in for the metric list of a unit that scored on no
	// metric at all.
	statusNone = "none"
)

// renderMarkdown writes the report as a GitHub-flavored document: one
// table per file, the units over their limit in bold, warnings as bullet
// lists and a closing summary paragraph.
func renderMarkdown(w io.Writer, doc Report) error {
	p := &printer{w: w}
	p.printf("# cdd check\n\nRoot: `%s`\n", doc.Root)
	for _, f := range doc.Files {
		markdownFile(p, f)
	}
	markdownWarnings(p, "\n## Warnings\n", doc.Warnings)
	markdownSummary(p, doc)
	return p.err
}

// markdownFile writes the section of one file: its table, the metrics that
// produced the ICPs, and the analyzer's warnings.
func markdownFile(p *printer, f File) {
	p.printf("\n## `%s` (%s)\n\n", escapeCell(f.Path), escapeCell(f.Language))
	if len(f.Units) == 0 {
		p.printf("No unit was measured.\n")
	} else {
		p.printf("%s", tableHeader)
		for _, u := range f.Units {
			markdownUnit(p, u)
		}
		markdownMetrics(p, f.Units)
	}
	markdownWarnings(p, "\nWarnings:\n", f.Warnings)
}

// markdownUnit writes one table row, bold when the unit is over its limit.
func markdownUnit(p *printer, u Unit) {
	name := "`" + escapeCell(u.Name) + "`"
	total := formatNumber(u.Total)
	status := statusOK
	if u.Exceeds {
		name = "**" + name + "**"
		total = "**" + total + "**"
		status = "**" + statusOver + "**"
	}
	p.printf("| %s | %s | %d:%d | %s | %d | %s |\n",
		name, escapeCell(u.Kind), u.Line, u.Col, total, u.Limit, status)
}

// markdownMetrics writes the per-unit breakdown under the table: one bullet
// per unit, naming the metrics that earned its ICPs.
func markdownMetrics(p *printer, units []Unit) {
	p.printf("\nMetrics:\n\n")
	for _, u := range units {
		p.printf("- `%s` — %s\n", escapeCell(u.Name), countedMetrics(u.Metrics))
	}
}

// countedMetrics spells the metrics that actually occurred. A metric that is
// enabled and never seen adds nothing to the total, so listing it would bury
// the ones that do under a row of zeros.
func countedMetrics(ms []Metric) string {
	parts := make([]string, 0, len(ms))
	for _, m := range ms {
		if m.Count > 0 {
			parts = append(parts, metricText(m))
		}
	}
	if len(parts) == 0 {
		return statusNone
	}
	return strings.Join(parts, ", ")
}

// markdownWarnings writes heading followed by one bullet per warning, and
// nothing at all when there is no warning.
func markdownWarnings(p *printer, heading string, warnings []string) {
	if len(warnings) == 0 {
		return
	}
	p.printf("%s\n", heading)
	for _, warning := range warnings {
		p.printf("- %s\n", warning)
	}
}

// markdownSummary closes the document with the counts, the elapsed time and,
// when the run was cut short, what that means.
func markdownSummary(p *printer, doc Report) {
	p.printf("\n## Summary\n\n")
	if doc.Partial {
		p.printf("Partial result: the timeout elapsed before every file was analyzed.\n\n")
	}
	p.printf("%d units analyzed, %d over limit, elapsed %s.\n",
		doc.Summary.Units, doc.Summary.Violations, formatElapsed(doc.ElapsedMS))
}

// escapeCell keeps a value inside its table cell: a pipe would end the cell
// early.
func escapeCell(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}
