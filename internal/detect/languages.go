// Package detect discovers, before any configuration exists, which languages
// a project contains and which package prefixes are internal to it. cdd init
// uses the result to pre-fill its prompts.
package detect

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

// skipDirs are directory names never worth scanning: VCS metadata,
// dependencies and build output.
var skipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"build":        true,
	"dist":         true,
	"target":       true,
	"out":          true,
}

// extensions is the only language-specific data kept outside config: the
// file extensions each analyzer would read.
var extensions = map[config.Language][]string{
	config.LangGo:         {".go"},
	config.LangJava:       {".java"},
	config.LangKotlin:     {".kt", ".kts"},
	config.LangTypeScript: {".ts", ".tsx", ".mts", ".cts"},
}

// Detected is the result of Languages: how many source files were counted
// per language, whether the scan hit its deadline before finishing, and how
// long it ran.
type Detected struct {
	Counts    map[config.Language]int
	Truncated bool
	Elapsed   time.Duration
}

// Languages returns the detected languages in canonical order.
func (d Detected) Languages() []config.Language {
	var out []config.Language
	for _, lang := range config.Languages() {
		if d.Counts[lang] > 0 {
			out = append(out, lang)
		}
	}
	return out
}

// Languages counts source files per language under root, skipping skipDirs.
// The caller bounds the walk through ctx; when the deadline expires the
// partial counts come back with Truncated set and a nil error, because a
// cut-short scan is still usable.
func Languages(ctx context.Context, root string) (Detected, error) {
	start := time.Now()
	d := Detected{Counts: map[config.Language]int{}}
	byExt := extensionIndex()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && skipDirs[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if lang, ok := byExt[strings.ToLower(filepath.Ext(entry.Name()))]; ok {
			d.Counts[lang]++
		}
		return nil
	})
	d.Elapsed = time.Since(start)
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		d.Truncated = true
		return d, nil
	}
	if err != nil {
		return d, fmt.Errorf("detect: %w", err)
	}
	return d, nil
}

// extensionIndex inverts extensions into extension -> language for the
// languages the config vocabulary knows.
func extensionIndex() map[string]config.Language {
	idx := make(map[string]config.Language)
	for _, lang := range config.Languages() {
		for _, ext := range extensions[lang] {
			idx[ext] = lang
		}
	}
	return idx
}
