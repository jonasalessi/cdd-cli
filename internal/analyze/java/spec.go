// Package java is the home of the Java language: the data cdd init and the
// configuration need, and the package-declaration based prefix detection it
// shares with Kotlin.
package java

import (
	"context"

	"github.com/jonasalessi/cdd-cli/internal/analyze/internal/jvm"
	"github.com/jonasalessi/cdd-cli/internal/config"
)

// extensions are the file extensions the analyzer reads.
var extensions = []string{".java"}

// Spec returns the Java language spec.
func Spec() config.LanguageSpec {
	return config.LanguageSpec{
		ID:              "java",
		DisplayName:     "Java",
		Extensions:      extensions,
		DefaultExcludes: []string{"**/src/test/**", "**/build/**", "**/target/**"},
		Descriptions: map[config.MetricID]string{
			config.MetricInternalCoupling: "references to project classes",
			config.MetricExternalCoupling: "framework / JDK types",
		},
		PackageExample: "com.acme.app",
		LimitExamples:  []string{`# ".*/adapters/.*": 8`, `# ".*Dto\\.java": 20`},
		DetectPackages: detectPackages,
	}
}

// detectPackages reduces the package declarations of the Java sources under
// root to their shortest telling prefixes.
func detectPackages(ctx context.Context, root string) ([]string, error) {
	return jvm.Prefixes(ctx, root, extensions)
}
