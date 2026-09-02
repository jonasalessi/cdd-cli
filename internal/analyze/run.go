package analyze

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/jonasalessi/cdd-cli/internal/config"
)

// ErrTimeout ends a run that hit its time budget, or whose caller canceled
// it, before every file was analyzed. Run returns it wrapped, together with
// the partial RunResult, so a command can print what was measured and still
// exit non-zero.
var ErrTimeout = errors.New("analysis stopped before every file was analyzed")

// Run analyzes every file under req.Root that a configured language claims
// and the include/exclude patterns keep, and returns the weighed report.
//
// The languages are checked up front: a configured language the registry
// does not know, or one whose analyzer does not exist yet, is an error
// before a single file is read, so the outcome does not depend on which
// files the tree happens to hold.
//
// Files are analyzed in parallel, one analyzer per worker and language, and
// the reports come back in path order. When req.Config.Timeout elapses the
// remaining files are dropped, RunResult.Partial is set and the error wraps
// ErrTimeout; any other failure aborts the run naming the file.
func Run(ctx context.Context, req Request) (RunResult, error) {
	start := time.Now()
	p, err := newPlan(req)
	if err != nil {
		return RunResult{}, err
	}
	if p.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}
	found, err := p.collect(ctx, req.Root)
	if err != nil {
		return RunResult{}, err
	}
	files, err := p.analyze(ctx, req.Root, found)
	if err != nil {
		return RunResult{}, err
	}
	slices.SortFunc(files, func(a, b FileReport) int { return strings.Compare(a.Path, b.Path) })
	result := RunResult{Root: req.Root, Files: files, Warnings: p.warnings, Elapsed: time.Since(start)}
	if ctx.Err() == nil {
		return result, nil
	}
	stopped := stoppedMessage(p.timeout, len(files), len(found)-len(files))
	result.Partial = true
	result.Warnings = append(result.Warnings, stopped)
	return result, fmt.Errorf("%s: %w", stopped, ErrTimeout)
}

// plan is the read-only knowledge a run shares with every worker: which
// language claims which extension, how each language is weighed, limited
// and constructed, and which files the configuration keeps.
type plan struct {
	langs     map[config.Language]Language
	byExt     map[string]config.Language
	resolvers map[config.Language]*Resolver
	prefixes  map[config.Language][]string
	matcher   *Matcher
	timeout   time.Duration
	warnings  []string
}

// newPlan resolves everything a run needs before it touches the tree.
func newPlan(req Request) (*plan, error) {
	if req.Config == nil {
		return nil, errors.New("analyze: no configuration")
	}
	p := &plan{
		langs:     make(map[config.Language]Language),
		byExt:     make(map[string]config.Language),
		resolvers: make(map[config.Language]*Resolver),
		prefixes:  make(map[config.Language][]string),
		timeout:   req.Config.Timeout,
		warnings:  enforcementWarnings(req.Config.Enforcement),
	}
	if err := p.addLanguages(req); err != nil {
		return nil, err
	}
	matcher, err := NewMatcher(req.Config.Include, req.Config.Exclude)
	if err != nil {
		return nil, fmt.Errorf("analyze: %w", err)
	}
	p.matcher = matcher
	return p, nil
}

// addLanguages registers every language the configuration asks for, in
// registry order, and reports the first one the run cannot honor.
func (p *plan) addLanguages(req Request) error {
	cfg := req.Config
	for _, lang := range req.Languages {
		id := lang.Spec.ID
		if _, configured := cfg.Metrics[id]; !configured {
			continue
		}
		if lang.NewAnalyzer == nil {
			return fmt.Errorf("no analyzer for %s yet", id)
		}
		resolver, err := NewResolver(cfg, id)
		if err != nil {
			return err
		}
		prefixes, err := internalPrefixes(cfg, lang.Spec, req.Root)
		if err != nil {
			return err
		}
		p.langs[id] = lang
		p.resolvers[id] = resolver
		p.prefixes[id] = prefixes
		for _, ext := range lang.Spec.Extensions {
			p.byExt[strings.ToLower(ext)] = id
		}
	}
	return p.checkUnknown(cfg)
}

// checkUnknown reports a language the configuration names and the registry
// does not know; config.Validate rejects the same file earlier.
func (p *plan) checkUnknown(cfg *config.Config) error {
	var unknown []string
	for id := range cfg.Metrics {
		if _, known := p.langs[id]; !known {
			unknown = append(unknown, string(id))
		}
	}
	if len(unknown) == 0 {
		return nil
	}
	slices.Sort(unknown)
	return fmt.Errorf("unknown language %s in metrics", strings.Join(unknown, ", "))
}

// internalPrefixes merges the configured internal package prefixes with the
// ones the language detects under root when auto_detect is on. The result
// is sorted and deduplicated so two runs feed the analyzer the same list.
func internalPrefixes(cfg *config.Config, spec config.LanguageSpec, root string) ([]string, error) {
	prefixes := slices.Clone(cfg.InternalCoupling.Packages)
	if cfg.InternalCoupling.AutoDetect && spec.DetectPackages != nil {
		detected, err := spec.DetectPackages(root)
		if err != nil {
			return nil, fmt.Errorf("detect %s packages: %w", spec.ID, err)
		}
		prefixes = append(prefixes, detected...)
	}
	slices.Sort(prefixes)
	return slices.Compact(prefixes), nil
}

// enforcementWarnings reports an enforcement setting the pipeline cannot
// honor yet, so the caller prints the report instead of blocking on it.
func enforcementWarnings(e config.Enforcement) []string {
	if !e.BlockOnCI || e.Blocks() {
		return nil
	}
	return []string{fmt.Sprintf("legacy_mode %s is not enforced yet; reporting only", e.LegacyMode)}
}

// analyze runs the worker pool over found and returns the finished reports
// in completion order.
func (p *plan) analyze(ctx context.Context, root string, found []candidate) ([]FileReport, error) {
	if len(found) == 0 {
		return nil, nil
	}
	workers := min(runtime.GOMAXPROCS(0), len(found))
	g, gctx := errgroup.WithContext(ctx)
	// One slot per worker plus the producer, which would otherwise wait for
	// a slot no worker ever releases.
	g.SetLimit(workers + 1)
	jobs := make(chan candidate)
	g.Go(func() error { return produce(gctx, jobs, found) })

	var (
		mu    sync.Mutex
		files []FileReport
	)
	emit := func(f FileReport) {
		mu.Lock()
		defer mu.Unlock()
		files = append(files, f)
	}
	for range workers {
		g.Go(func() error { return p.work(gctx, root, jobs, emit) })
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return files, nil
}

// produce feeds the workers until the files run out or the run stops.
func produce(ctx context.Context, jobs chan<- candidate, found []candidate) error {
	defer close(jobs)
	for _, c := range found {
		select {
		case jobs <- c:
		case <-ctx.Done():
			return nil
		}
	}
	return nil
}

// work analyzes files until the channel closes or the run stops. Analyzers
// are not safe for concurrent use, so each worker builds its own per
// language and releases them all on the way out.
func (p *plan) work(ctx context.Context, root string, jobs <-chan candidate, emit func(FileReport)) (err error) {
	analyzers := make(map[config.Language]Analyzer)
	defer func() { err = errors.Join(err, closeAnalyzers(analyzers)) }()
	for c := range jobs {
		if ctx.Err() != nil {
			return nil
		}
		report, fileErr := p.analyzeFile(ctx, analyzers, root, c)
		if fileErr != nil {
			if stoppedEarly(fileErr) || ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("analyze %s: %w", c.path, fileErr)
		}
		emit(report)
	}
	return nil
}

// analyzeFile reads one file, counts it with the worker's analyzer for that
// language, and weighs the counts against the patterns the path resolves to.
func (p *plan) analyzeFile(
	ctx context.Context,
	analyzers map[config.Language]Analyzer,
	root string,
	c candidate,
) (FileReport, error) {
	analyzer, built := analyzers[c.lang]
	if !built {
		analyzer = p.langs[c.lang].NewAnalyzer(Options{InternalPrefixes: p.prefixes[c.lang]})
		analyzers[c.lang] = analyzer
	}
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(c.path)))
	if err != nil {
		return FileReport{}, err
	}
	result, err := analyzer.Analyze(ctx, c.path, src)
	if err != nil {
		return FileReport{}, err
	}
	return FileReport{
		Path:     c.path,
		Language: c.lang,
		Units:    p.resolvers[c.lang].Resolve(c.path, result.Units),
		Warnings: result.Warnings,
	}, nil
}

// closeAnalyzers releases the analyzers that hold resources, which for a
// parser binding are outside the Go heap.
func closeAnalyzers(analyzers map[config.Language]Analyzer) error {
	var errs []error
	for _, a := range analyzers {
		if closer, ok := a.(io.Closer); ok {
			errs = append(errs, closer.Close())
		}
	}
	return errors.Join(errs...)
}

// stoppedEarly reports whether err is the run ending rather than failing.
func stoppedEarly(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// stoppedMessage says why a run ended early and how much of it finished.
func stoppedMessage(timeout time.Duration, analyzed, skipped int) string {
	if timeout > 0 {
		return fmt.Sprintf("timeout of %s elapsed; %d files analyzed, %d skipped", timeout, analyzed, skipped)
	}
	return fmt.Sprintf("run canceled; %d files analyzed, %d skipped", analyzed, skipped)
}
