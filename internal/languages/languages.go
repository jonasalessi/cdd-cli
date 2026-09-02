// Package languages is the registry of every language cdd supports. It is
// the one existing file a new language touches: create
// internal/analyze/<id>/ with a spec.go, then add a line to All below.
// Every consumer receives the registered specs from cmd; nothing else in the
// repository imports this package.
package languages

import (
	"github.com/jonasalessi/cdd-cli/internal/analyze"
	"github.com/jonasalessi/cdd-cli/internal/analyze/golang"
	"github.com/jonasalessi/cdd-cli/internal/analyze/java"
	"github.com/jonasalessi/cdd-cli/internal/analyze/kotlin"
	"github.com/jonasalessi/cdd-cli/internal/analyze/typescript"
	"github.com/jonasalessi/cdd-cli/internal/config"
)

// Language pairs the data side of a language with its analyzer constructor.
// A nil NewAnalyzer means the language has no analyzer yet; cdd check must
// report that as an error, never as zero ICPs.
type Language struct {
	Spec        config.LanguageSpec
	NewAnalyzer func() analyze.Analyzer
}

// All returns every supported language in registration order, which is the
// order languages are listed and rendered everywhere. Each call returns a
// fresh slice.
func All() []Language {
	return []Language{
		{Spec: golang.Spec()},
		{Spec: java.Spec()},
		{Spec: kotlin.Spec()},
		{Spec: typescript.Spec()},
	}
}

// Specs projects All onto the specs, which is what config, detect, prompt
// and initcmd take.
func Specs() []config.LanguageSpec {
	all := All()
	specs := make([]config.LanguageSpec, len(all))
	for i, l := range all {
		specs[i] = l.Spec
	}
	return specs
}
