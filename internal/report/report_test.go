package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jonasalessi/cdd-cli/internal/analyze"
	"github.com/jonasalessi/cdd-cli/internal/config"
)

// render is Write into a buffer with the default filter: violations only.
func render(t *testing.T, format string, res analyze.RunResult) string {
	t.Helper()
	return renderWith(t, format, res, Options{})
}

// renderAll is Write into a buffer listing every unit, the way --all does.
func renderAll(t *testing.T, format string, res analyze.RunResult) string {
	t.Helper()
	return renderWith(t, format, res, Options{All: true})
}

func renderWith(t *testing.T, format string, res analyze.RunResult, opts Options) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, format, res, opts))
	return buf.String()
}

func TestWriteGolden(t *testing.T) {
	for _, format := range config.ReporterFormats() {
		for suffix, tc := range goldenCases() {
			t.Run(format+suffix, func(t *testing.T) {
				got := renderWith(t, format, tc.res, tc.opts)
				path := goldenPath(format, suffix)
				if updateGolden() {
					require.NoError(t, os.WriteFile(path, []byte(got), 0o644))
				}
				want, err := os.ReadFile(path)
				require.NoError(t, err, "run go test ./internal/report -update to create it")
				assert.Equal(t, string(want), got)
			})
		}
	}
}

func TestWriteEveryFormatIsSupported(t *testing.T) {
	for _, format := range config.ReporterFormats() {
		assert.NotEmpty(t, render(t, format, fullRun()), format)
	}
}

func TestWriteUnknownFormat(t *testing.T) {
	err := Write(&bytes.Buffer{}, "yaml", fullRun(), Options{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"yaml"`)
}

// errWriter fails every write, so a renderer must surface the failure.
type errWriter struct{}

var errWrite = errors.New("no room left")

func (errWriter) Write([]byte) (int, error) { return 0, errWrite }

func TestWriteReportsWriteErrors(t *testing.T) {
	for _, format := range config.ReporterFormats() {
		assert.ErrorIs(t, Write(errWriter{}, format, fullRun(), Options{All: true}), errWrite, format)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	data, err := os.ReadFile(goldenPath(config.FormatJSON, ""))
	require.NoError(t, err)

	var doc Report
	require.NoError(t, json.Unmarshal(data, &doc))

	assert.Equal(t, "/projects/shop", doc.Root)
	assert.Equal(t, filterViolations, doc.Filter)
	assert.False(t, doc.Partial)
	assert.Equal(t, int64(1234), doc.ElapsedMS)
	assert.Equal(t, Summary{Units: 3, Violations: 1}, doc.Summary, "the summary counts the run, not the listing")
	assert.Equal(t, []string{"the legacy mode is reported, not enforced yet"}, doc.Warnings)

	require.Len(t, doc.Files, 1, "src/money.ts has no violation and no warning")
	first := doc.Files[0]
	assert.Equal(t, "src/checkout.ts", first.Path)
	assert.Equal(t, string(fixtureLanguage), first.Language)
	assert.Len(t, first.Warnings, 1)

	require.Len(t, first.Units, 1)
	over := first.Units[0]
	assert.Equal(t, "CheckoutService", over.Name)
	assert.True(t, over.Exceeds)
	assert.InDelta(t, 14.5, over.Total, 0)
	assert.Equal(t, 10, over.Limit)
	assert.Equal(t, []Metric{
		{ID: string(config.MetricCodeBranch), Count: 6, Score: 6},
		{ID: string(config.MetricCondition), Count: 3, Score: 3},
		{ID: string(config.MetricExternalCoupling), Count: 11, Score: 5.5},
	}, over.Metrics)
}

func TestFilterListsOnlyViolationsByDefault(t *testing.T) {
	doc := newReport(fullRun(), Options{})
	require.Len(t, doc.Files, 1)
	require.Len(t, doc.Files[0].Units, 1)
	assert.Equal(t, "CheckoutService", doc.Files[0].Units[0].Name)
	assert.Equal(t, filterViolations, doc.Filter)
}

func TestFilterAllListsEveryUnit(t *testing.T) {
	doc := newReport(fullRun(), Options{All: true})
	require.Len(t, doc.Files, 2)
	assert.Len(t, doc.Files[0].Units, 2)
	assert.Len(t, doc.Files[1].Units, 1)
	assert.Equal(t, filterAll, doc.Filter)
}

func TestFilterKeepsAFileForItsWarningAlone(t *testing.T) {
	res := analyze.RunResult{
		Root: "/projects/shop",
		Files: []analyze.FileReport{
			{Path: "src/clean.ts", Language: fixtureLanguage, Units: []analyze.UnitReport{greetUnit()}},
			{Path: "src/broken.ts", Language: fixtureLanguage, Warnings: []string{"syntax error at 1:1"}},
		},
		Elapsed: fixtureElapsed,
	}
	doc := newReport(res, Options{})
	require.Len(t, doc.Files, 1, "the clean file says nothing, the broken one warns")
	assert.Equal(t, "src/broken.ts", doc.Files[0].Path)
	assert.Empty(t, doc.Files[0].Units)
}

func TestFilterLeavesTheSummaryAlone(t *testing.T) {
	violations := newReport(fullRun(), Options{}).Summary
	assert.Equal(t, newReport(fullRun(), Options{All: true}).Summary, violations)
	assert.Equal(t, Summary{Units: 3, Violations: 1}, violations)
}

func TestFilterFieldIsRendered(t *testing.T) {
	assert.Contains(t, render(t, config.FormatJSON, fullRun()), `"filter": "violations"`)
	assert.Contains(t, renderAll(t, config.FormatJSON, fullRun()), `"filter": "all"`)
	assert.Contains(t, render(t, config.FormatXML, fullRun()), `filter="violations"`)
	assert.Contains(t, renderAll(t, config.FormatXML, fullRun()), `filter="all"`)
}

func TestJSONPartialHasEmptyFileList(t *testing.T) {
	out := render(t, config.FormatJSON, partialRun())
	assert.Contains(t, out, `"files": []`)
	assert.Contains(t, out, `"partial": true`)
	assert.NotContains(t, out, `"warnings"`)
	assert.True(t, strings.HasSuffix(out, "}\n"))
}

func TestMetricsFollowVocabularyOrder(t *testing.T) {
	doc := newReport(fullRun(), Options{All: true})
	var ids []string
	for _, m := range doc.Files[0].Units[0].Metrics {
		ids = append(ids, m.ID)
	}
	assert.Equal(t, []string{
		string(config.MetricCodeBranch),
		string(config.MetricCondition),
		string(config.MetricExternalCoupling),
	}, ids, "canonical order, disabled metrics absent")
}

func TestConsoleShapesTheRun(t *testing.T) {
	out := renderAll(t, config.FormatConsole, fullRun())
	assert.Contains(t, out, passMark+" src/checkout.ts:1:1  function greet  2.5/10\n")
	assert.Contains(t, out, failMark+" src/checkout.ts:12:3  class CheckoutService  14.5/10\n")
	assert.Contains(t, out, "\n    external_coupling 11×0.5=5.5\n", "breakdown of the unit over its limit")
	assert.NotContains(t, out, "\n    code_branch 2×1=2\n", "a unit within its limit stays on one line")
	assert.Contains(t, out, "warning: src/checkout.ts: unsupported syntax at 42:7, unit skipped\n")
	assert.Contains(t, out, "warning: the legacy mode is reported, not enforced yet\n")
	assert.True(t, strings.HasSuffix(out, "3 units analyzed, 1 over limit, elapsed 1.234s\n"))
	assert.NotContains(t, out, "partial result")
}

func TestConsolePartialAnnouncesTheTimeout(t *testing.T) {
	out := render(t, config.FormatConsole, partialRun())
	assert.Contains(t, out, "partial result: timeout elapsed\n")
	assert.Contains(t, out, "0 units analyzed, 0 over limit, elapsed 1.234s\n")
}

func TestMarkdownShapesTheRun(t *testing.T) {
	out := renderAll(t, config.FormatMarkdown, fullRun())
	assert.True(t, strings.HasPrefix(out, "# cdd check\n"))
	assert.Contains(t, out, "| Unit | Kind | Location | ICPs | Limit | Status |\n")
	assert.Contains(t, out, "| `greet` | function | 1:1 | 2.5 | 10 | ok |\n")
	assert.Contains(t, out, "| **`CheckoutService`** | class | 12:3 | **14.5** | 10 | **over limit** |\n")
	assert.Contains(t, out, "- unsupported syntax at 42:7, unit skipped\n")
	assert.Contains(t, out, "\n## Summary\n")
	assert.True(t, strings.HasSuffix(out, "3 units analyzed, 1 over limit, elapsed 1.234s.\n"))
}

func TestMarkdownSkippedFile(t *testing.T) {
	res := analyze.RunResult{
		Root: "/projects/shop",
		Files: []analyze.FileReport{{
			Path:     "src/broken.ts",
			Language: fixtureLanguage,
			Warnings: []string{"parse error at 1:1"},
		}},
		Elapsed: fixtureElapsed,
	}
	out := render(t, config.FormatMarkdown, res)
	assert.Contains(t, out, "No unit was measured.\n")
	assert.Contains(t, out, "- parse error at 1:1\n")
	assert.NotContains(t, out, "| Unit |")
}

func TestMarkdownListsOnlyCountedMetrics(t *testing.T) {
	unit := func(counts map[config.MetricID]int) analyze.UnitReport {
		return analyze.UnitReport{
			Name: "noop", Kind: "function", Line: 1, Col: 1, Limit: 10,
			Counts: counts,
			Scores: map[config.MetricID]float64{config.MetricCodeBranch: 0, config.MetricCondition: 2},
		}
	}
	res := func(u analyze.UnitReport) analyze.RunResult {
		return analyze.RunResult{
			Root: "/projects/shop",
			Files: []analyze.FileReport{{
				Path:     "src/empty.ts",
				Language: fixtureLanguage,
				Units:    []analyze.UnitReport{u},
			}},
		}
	}
	t.Run("a unit that scored on nothing reads none", func(t *testing.T) {
		out := renderAll(t, config.FormatMarkdown, res(unit(nil)))
		assert.Contains(t, out, "- `noop` — none\n")
	})
	t.Run("a metric that never occurred is left out", func(t *testing.T) {
		out := renderAll(t, config.FormatMarkdown, res(unit(map[config.MetricID]int{
			config.MetricCodeBranch: 0,
			config.MetricCondition:  2,
		})))
		assert.Contains(t, out, "- `noop` — condition 2×1=2\n")
		assert.NotContains(t, out, "code_branch")
	})
}

func TestXMLShapesTheRun(t *testing.T) {
	out := render(t, config.FormatXML, fullRun())
	assert.True(t, strings.HasPrefix(out, `<?xml version="1.0" encoding="UTF-8"?>`))
	assert.Contains(t, out, `<report root="/projects/shop" filter="violations" partial="false" elapsed_ms="1234">`)
	assert.Contains(t, out, `<summary units="3" violations="1">`)
	assert.Contains(t, out, `<metric id="external_coupling" count="11" score="5.5">`)
	assert.True(t, strings.HasSuffix(out, "</report>\n"))
}

func TestEscapeCell(t *testing.T) {
	assert.Equal(t, `a\|b`, escapeCell("a|b"))
	assert.Equal(t, "plain", escapeCell("plain"))
}

func TestFormatNumber(t *testing.T) {
	assert.Equal(t, "2", formatNumber(2))
	assert.Equal(t, "2.5", formatNumber(2.5))
	assert.Equal(t, "0", formatNumber(0))
	assert.Equal(t, "0.25", formatNumber(0.25))
}

func TestFormatElapsed(t *testing.T) {
	assert.Equal(t, "1.234s", formatElapsed(1234))
	assert.Equal(t, "0s", formatElapsed(0))
	assert.Equal(t, (5 * time.Minute).String(), formatElapsed(300000))
}

func TestMetricText(t *testing.T) {
	id := string(config.MetricCodeBranch)
	assert.Equal(t, id+" 4×1=4", metricText(Metric{ID: id, Count: 4, Score: 4}))
	assert.Equal(t, id+" 3×0.5=1.5", metricText(Metric{ID: id, Count: 3, Score: 1.5}))
	assert.Equal(t, id+" 0", metricText(Metric{ID: id}))
}
