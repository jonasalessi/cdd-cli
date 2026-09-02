package analyze

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/jonasalessi/cdd-cli/internal/config"
	"github.com/jonasalessi/cdd-cli/internal/detect"
)

// candidate is one file the walk selected: its slash-separated path
// relative to the run root, and the language whose extension claims it.
type candidate struct {
	path string
	lang config.Language
}

// collect walks root and returns every file a configured language claims
// and the matcher includes. Directories detect.SkipDir names, version
// control and build output among them, are never entered. A canceled
// context ends the walk with what it found so far, because Run turns that
// into a partial result rather than an error.
func (p *plan) collect(ctx context.Context, root string) ([]candidate, error) {
	var found []candidate
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && detect.SkipDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		lang, claimed := p.byExt[strings.ToLower(filepath.Ext(entry.Name()))]
		if !claimed {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if slash := filepath.ToSlash(rel); p.matcher.Match(slash) {
			found = append(found, candidate{path: slash, lang: lang})
		}
		return nil
	})
	if err != nil && !stoppedEarly(err) {
		return nil, fmt.Errorf("walk %s: %w", root, err)
	}
	return found, nil
}
