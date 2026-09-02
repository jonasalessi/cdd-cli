// Package report renders the outcome of an analysis run in the formats
// reporter.format allows: console, json, xml and markdown. Every format
// carries the same document — the model in schema.go — so switching format
// changes the shape of the output, never its content.
package report

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jonasalessi/cdd-cli/internal/analyze"
	"github.com/jonasalessi/cdd-cli/internal/config"
)

// outputFileMode is the permission of a report written to a file. A report
// is meant to be read by humans and by CI, never kept secret.
const outputFileMode = 0o644

// Write renders res to w in the given format. An unknown format is an error
// naming it.
func Write(w io.Writer, format string, res analyze.RunResult) error {
	doc := newReport(res)
	switch format {
	case config.FormatConsole:
		return renderConsole(w, doc)
	case config.FormatJSON:
		return renderJSON(w, doc)
	case config.FormatXML:
		return renderXML(w, doc)
	case config.FormatMarkdown:
		return renderMarkdown(w, doc)
	default:
		return fmt.Errorf("unknown reporter format %q", format)
	}
}

// Emit renders res as r asks and returns the path it wrote. A nil
// r.OutputFile means stdout and the returned path is empty; otherwise the
// file is created or truncated, its parent directory must already exist,
// and the caller can print the returned path as a receipt.
func Emit(stdout io.Writer, r config.Reporter, res analyze.RunResult) (string, error) {
	if r.OutputFile == nil {
		return "", Write(stdout, r.Format, res)
	}
	var buf bytes.Buffer
	if err := Write(&buf, r.Format, res); err != nil {
		return "", err
	}
	path := *r.OutputFile
	if err := writeFile(path, buf.Bytes()); err != nil {
		return "", err
	}
	return path, nil
}

// writeFile creates or truncates path. The report is rendered before the
// file is touched, so a format error never leaves an empty file behind.
func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("output directory %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("output directory %s: not a directory", dir)
	}
	if err := os.WriteFile(path, data, outputFileMode); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}

// formatNumber renders an ICP value with as few digits as possible: 2 reads
// "2" and 2.5 reads "2.5". Every format uses it, so a total is spelled the
// same way everywhere.
func formatNumber(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// formatElapsed renders a duration in milliseconds the way Go spells
// durations, e.g. "1.234s".
func formatElapsed(ms int64) string {
	return (time.Duration(ms) * time.Millisecond).String()
}

// metricText spells one metric for a human: "code_branch 4×0.5=2", or the
// bare count when the metric is enabled but never occurred, since no weight
// was applied to it.
func metricText(m Metric) string {
	if m.Count == 0 {
		return m.ID + " 0"
	}
	weight := m.Score / float64(m.Count)
	return fmt.Sprintf("%s %d×%s=%s", m.ID, m.Count, formatNumber(weight), formatNumber(m.Score))
}

// printer writes a report linearly and remembers the first failure, so a
// renderer checks for an error once instead of after every line.
type printer struct {
	w   io.Writer
	err error
}

// printf writes one formatted chunk unless an earlier write already failed.
func (p *printer) printf(format string, args ...any) {
	if p.err != nil {
		return
	}
	_, p.err = fmt.Fprintf(p.w, format, args...)
}
