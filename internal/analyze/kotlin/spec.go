// Package kotlin is the home of the Kotlin language: the data cdd init and
// the configuration need, and the package-declaration based prefix detection
// it shares with Java.
package kotlin

import (
	"context"

	"github.com/jonasalessi/cdd-cli/internal/analyze/internal/jvm"
	"github.com/jonasalessi/cdd-cli/internal/config"
)

// extensions are the file extensions the analyzer reads; .kts covers build
// scripts.
var extensions = []string{".kt", ".kts"}

// Spec returns the Kotlin language spec.
func Spec() config.LanguageSpec {
	return config.LanguageSpec{
		ID:              "kotlin",
		DisplayName:     "Kotlin",
		Extensions:      extensions,
		DefaultExcludes: []string{"**/src/test/**", "**/build/**", "**/target/**"},
		Descriptions: map[config.MetricID]string{
			config.MetricCodeBranch:  "if/when, loops, safe calls (?.), elvis (?:)",
			config.MetricInheritance: ": Base() / : Iface, per level",
			config.MetricLambda:      "lambdas and function refs",
		},
		PackageExample: "com.acme.app",
		LimitExamples:  []string{`# ".*/adapters/.*": 8`},
		DetectPackages: detectPackages,
	}
}

// detectPackages reduces the package declarations of the Kotlin sources
// under root to their shortest telling prefixes.
func detectPackages(ctx context.Context, root string) ([]string, error) {
	return jvm.Prefixes(ctx, root, extensions)
}
