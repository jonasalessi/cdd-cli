package report

import (
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/jonasalessi/cdd-cli/internal/analyze"
	"github.com/jonasalessi/cdd-cli/internal/config"
)

// update rewrites the golden files from the fixtures below; UPDATE_GOLDEN=1
// does the same.
var update = flag.Bool("update", false, "rewrite golden files")

func updateGolden() bool {
	return *update || os.Getenv("UPDATE_GOLDEN") == "1"
}

// goldenNames maps a reporter format to the base name of its golden files;
// a partial run adds the "-partial" suffix.
var goldenNames = map[string]string{
	config.FormatConsole:  "console.txt",
	config.FormatJSON:     "report.json",
	config.FormatXML:      "report.xml",
	config.FormatMarkdown: "report.md",
}

// goldenPath locates the golden file of format for one of the fixtures.
func goldenPath(format, suffix string) string {
	name := goldenNames[format]
	ext := filepath.Ext(name)
	return filepath.Join("testdata", "golden", name[:len(name)-len(ext)]+suffix+ext)
}

// fixtureElapsed is fixed so the goldens never depend on the clock.
const fixtureElapsed = 1234 * time.Millisecond

// fixtureLanguage is the language id the fixture files carry; the report
// package never interprets it.
const fixtureLanguage = config.Language("typescript")

// goldenRuns pairs every golden-file suffix with the run it renders.
func goldenRuns() map[string]analyze.RunResult {
	return map[string]analyze.RunResult{
		"":         fullRun(),
		"-partial": partialRun(),
	}
}

// fullRun is a finished run over two files: three units, one of them over
// its limit, one file warning and one run warning.
func fullRun() analyze.RunResult {
	return analyze.RunResult{
		Root: "/projects/shop",
		Files: []analyze.FileReport{
			{
				Path:     "src/checkout.ts",
				Language: fixtureLanguage,
				Units: []analyze.UnitReport{
					greetUnit(),
					checkoutServiceUnit(),
				},
				Warnings: []string{"unsupported syntax at 42:7, unit skipped"},
			},
			{
				Path:     "src/money.ts",
				Language: fixtureLanguage,
				Units:    []analyze.UnitReport{formatMoneyUnit()},
			},
		},
		Warnings: []string{"the legacy mode is reported, not enforced yet"},
		Elapsed:  fixtureElapsed,
	}
}

// partialRun is a run the timeout cut short before any file was analyzed.
func partialRun() analyze.RunResult {
	return analyze.RunResult{
		Root:    "/projects/shop",
		Partial: true,
		Elapsed: fixtureElapsed,
	}
}

// greetUnit stays within its limit and enables a metric it never uses.
func greetUnit() analyze.UnitReport {
	return analyze.UnitReport{
		Name: "greet", Kind: "function", Line: 1, Col: 1,
		Counts: map[config.MetricID]int{
			config.MetricCodeBranch:       2,
			config.MetricCondition:        0,
			config.MetricExternalCoupling: 1,
		},
		Scores: map[config.MetricID]float64{
			config.MetricCodeBranch:       2,
			config.MetricCondition:        0,
			config.MetricExternalCoupling: 0.5,
		},
		Total: 2.5, Limit: 10,
	}
}

// checkoutServiceUnit is the unit over its limit.
func checkoutServiceUnit() analyze.UnitReport {
	return analyze.UnitReport{
		Name: "CheckoutService", Kind: "class", Line: 12, Col: 3,
		Counts: map[config.MetricID]int{
			config.MetricCodeBranch:       6,
			config.MetricCondition:        3,
			config.MetricExternalCoupling: 11,
		},
		Scores: map[config.MetricID]float64{
			config.MetricCodeBranch:       6,
			config.MetricCondition:        3,
			config.MetricExternalCoupling: 5.5,
		},
		Total: 14.5, Limit: 10, Exceeds: true,
	}
}

// formatMoneyUnit is the only unit of the second file.
func formatMoneyUnit() analyze.UnitReport {
	return analyze.UnitReport{
		Name: "formatMoney", Kind: "function", Line: 3, Col: 1,
		Counts: map[config.MetricID]int{
			config.MetricCodeBranch: 1,
			config.MetricCondition:  1,
		},
		Scores: map[config.MetricID]float64{
			config.MetricCodeBranch: 1,
			config.MetricCondition:  1,
		},
		Total: 2, Limit: 10,
	}
}
