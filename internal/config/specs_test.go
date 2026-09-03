package config

import "context"

// The config tests never import the real registry: they run against the
// synthetic languages below, so adding a real language changes no golden
// here. The wording and defaults mirror what the golden files encode.
const (
	// langAlpha applies every metric and carries JVM-style wording.
	langAlpha Language = "alpha"
	// langBeta applies every metric with its own branch and lambda wording.
	langBeta Language = "beta"
	// langGamma cannot count exceptions or inheritance.
	langGamma Language = "gamma"
	// langDelta applies every metric with module-style coupling wording.
	langDelta Language = "delta"
)

func noPackages(context.Context, string) ([]string, error) { return nil, nil }

func alphaSpec() LanguageSpec {
	return LanguageSpec{
		ID:              langAlpha,
		DisplayName:     "Alpha",
		Extensions:      []string{".alpha"},
		DefaultExcludes: []string{"**/src/test/**", "**/build/**", "**/target/**"},
		Descriptions: map[MetricID]string{
			MetricInternalCoupling: "references to project classes",
			MetricExternalCoupling: "framework / JDK types",
		},
		PackageExample: "com.acme.app",
		LimitExamples:  []string{`# ".*/adapters/.*": 8`, `# ".*Dto\\.java": 20`},
		DetectPackages: noPackages,
	}
}

func betaSpec() LanguageSpec {
	return LanguageSpec{
		ID:              langBeta,
		DisplayName:     "Beta",
		Extensions:      []string{".beta"},
		DefaultExcludes: []string{"**/src/test/**", "**/build/**", "**/target/**"},
		Descriptions: map[MetricID]string{
			MetricCodeBranch:  "if/when, loops, safe calls (?.), elvis (?:)",
			MetricInheritance: ": Base() / : Iface, per level",
			MetricLambda:      "lambdas and function refs",
		},
		PackageExample: "com.acme.app",
		LimitExamples:  []string{`# ".*/adapters/.*": 8`},
		DetectPackages: noPackages,
	}
}

func gammaSpec() LanguageSpec {
	return LanguageSpec{
		ID:              langGamma,
		DisplayName:     "Gamma",
		Extensions:      []string{".gamma"},
		NotApplicable:   []MetricID{MetricExceptionHandling, MetricInheritance},
		DefaultExcludes: []string{"**/*_test.go", "vendor/**"},
		Descriptions: map[MetricID]string{
			MetricCodeBranch: "if/else, switch/select, for",
			MetricLambda:     "func literals",
		},
		PackageExample: "github.com/acme/api",
		LimitExamples:  []string{`# ".*/adapters/.*": 8`},
		DetectPackages: noPackages,
	}
}

func deltaSpec() LanguageSpec {
	return LanguageSpec{
		ID:          langDelta,
		DisplayName: "Delta",
		Extensions:  []string{".delta"},
		DefaultExcludes: []string{
			"**/*.test.ts", "**/*.spec.ts", "**/*.d.ts", "**/node_modules/**", "**/dist/**",
		},
		Descriptions: map[MetricID]string{
			MetricCodeBranch:       "if/else, switch, ternary, loops, ?. and ??",
			MetricCondition:        "&&, || and ?? clauses",
			MetricInternalCoupling: "references to project modules",
			MetricExternalCoupling: "framework / node_modules types",
			MetricLambda:           "arrow functions and callbacks",
		},
		PackageExample: "@app/",
		LimitExamples:  []string{`# ".*/adapters/.*": 8`},
		DetectPackages: noPackages,
	}
}

// testSpecs is the roster every config test runs against, in render order.
func testSpecs() []LanguageSpec {
	return []LanguageSpec{alphaSpec(), betaSpec(), gammaSpec(), deltaSpec()}
}
