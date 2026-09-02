package report

import "io"

// The marks that open a unit line. They are the only decoration the console
// report uses: no ANSI color, so a CI log reads like a terminal.
const (
	passMark = "✓"
	failMark = "✗"
)

// renderConsole writes the human-readable report: one line per unit, a
// metric breakdown under the units that exceed their limit, the warnings,
// and a closing summary.
func renderConsole(w io.Writer, doc Report) error {
	p := &printer{w: w}
	p.printf("cdd check %s\n", doc.Root)
	for _, f := range doc.Files {
		consoleFile(p, f)
	}
	for _, warning := range doc.Warnings {
		p.printf("warning: %s\n", warning)
	}
	if doc.Partial {
		p.printf("partial result: timeout elapsed\n")
	}
	p.printf("%d units analyzed, %d over limit, elapsed %s\n",
		doc.Summary.Units, doc.Summary.Violations, formatElapsed(doc.ElapsedMS))
	return p.err
}

// consoleFile writes the units of one file and then its warnings.
func consoleFile(p *printer, f File) {
	for _, u := range f.Units {
		consoleUnit(p, f.Path, u)
	}
	for _, warning := range f.Warnings {
		p.printf("warning: %s: %s\n", f.Path, warning)
	}
}

// consoleUnit writes one unit on one line, and the metrics that earned its
// ICPs underneath when it is over the limit: a unit within its limit stays
// compact, a unit over it explains itself.
func consoleUnit(p *printer, path string, u Unit) {
	mark := passMark
	if u.Exceeds {
		mark = failMark
	}
	p.printf("%s %s:%d:%d  %s %s  %s/%d\n",
		mark, path, u.Line, u.Col, u.Kind, u.Name, formatNumber(u.Total), u.Limit)
	if !u.Exceeds {
		return
	}
	for _, m := range u.Metrics {
		if m.Count == 0 {
			continue
		}
		p.printf("    %s\n", metricText(m))
	}
}
