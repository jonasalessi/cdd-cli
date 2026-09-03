// Package golang is the home of the Go language: the data cdd init and the
// configuration need, and the go.mod based package detection. The directory
// is named golang because go is a keyword.
package golang

import (
	"bufio"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

// Spec returns the Go language spec. Go has no exceptions and no
// inheritance, so those two metrics never apply.
func Spec() config.LanguageSpec {
	return config.LanguageSpec{
		ID:              "go",
		DisplayName:     "Go",
		Extensions:      []string{".go"},
		NotApplicable:   []config.MetricID{config.MetricExceptionHandling, config.MetricInheritance},
		DefaultExcludes: []string{"**/*_test.go", "vendor/**"},
		Descriptions: map[config.MetricID]string{
			config.MetricCodeBranch: "if/else, switch/select, for",
			config.MetricLambda:     "func literals",
		},
		PackageExample: "github.com/acme/api",
		LimitExamples:  []string{`# ".*/adapters/.*": 8`},
		DetectPackages: detectPackages,
	}
}

// detectPackages reads the module line of <root>/go.mod. A missing file or
// a file without a module line yields no prefixes.
func detectPackages(ctx context.Context, root string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if mod, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "module "); ok {
			return []string{strings.TrimSpace(mod)}, nil
		}
	}
	return nil, sc.Err()
}
