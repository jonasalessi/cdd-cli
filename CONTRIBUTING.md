# Contributing to cdd-cli

## Run the setup first

Clone the repo, then run this once:

```sh
make setup
```

It sets `core.hooksPath` to `.githooks/`, so git runs the `commit-msg` hook
checked into this repo. The hook rejects any commit whose message does not
match the format below. Skip the setup and git will happily accept a bad
message, and the reviewer will ask you to rewrite it.

## What the hooks do

`pre-commit` runs `gofmt` and `golangci-lint run --fix` on the staged Go
files, re-stages whatever they fixed, and blocks the commit only when an
issue cannot be fixed automatically. It refuses partially staged Go files,
since re-staging those would commit hunks you left out on purpose.

`commit-msg` checks the message format below.

## Commit message format

```
<type>: <description>
```

Allowed types: `feat`, `fix`, `refactor`, `perf`, `docs`, `test`, `build`, `ci`.

```
feat: add init command
fix: handle missing config file
docs: describe ICP calculation
```

The type is lowercase, followed by a colon and one space. Make one commit per
remediation batch. Do not add a `Co-Authored-By` trailer.

If the hook rejects your message, do not `git commit --amend`. Fix the message
and create a new commit.

## Code style

Follow [Effective Go](https://go.dev/doc/effective_go).

## Adding a language

Every language lives in one directory, `internal/analyze/<id>/`, and is
registered in one file, `internal/languages/languages.go`. Nothing else in
the repository knows the language exists: `config`, `detect`, `prompt` and
`cmd` all work from the registered specs they are handed.

1. Create `internal/analyze/<id>/` with a `spec.go`. The directory name is
   the language id, except that `go` lives in `golang/` because `go` is a
   keyword. The file exports one function returning the language's data:

   ```go
   package rust

   import "github.com/jonasalessi/cdd-cli/internal/config"

   func Spec() config.LanguageSpec {
       return config.LanguageSpec{
           ID:              "rust",
           DisplayName:     "Rust",
           Extensions:      []string{".rs"},
           NotApplicable:   []config.MetricID{config.MetricInheritance},
           DefaultExcludes: []string{"target/**"},
           Descriptions:    map[config.MetricID]string{config.MetricLambda: "closures"},
           PackageExample:  "acme_billing",
           LimitExamples:   []string{`# ".*/adapters/.*": 8`},
           DetectPackages:  detectPackages, // guesses internal prefixes from Cargo.toml
       }
   }
   ```

   `NotApplicable` hides the metrics the analyzer cannot count, and at least
   three must remain. `Descriptions` only lists the metrics whose constructs
   have a language-specific name; the rest use the generic wording. Ids may
   be spelled out as string literals in this file and nowhere else.

2. Add one line to `All()` in `internal/languages/languages.go`:

   ```go
   {Spec: rust.Spec()},
   ```

   `NewAnalyzer` may stay nil until the analyzer exists; `cdd check` reports
   a language without one as an error rather than counting zero ICPs.

3. Run `make test`. The registry tests fail naming the directory if the line
   is missing, naming the id if the directory is missing, and naming the
   field if the spec is incomplete. `make check-literals` fails if a language
   id, metric id or mode is spelled out anywhere else, or if a
   language-keyed table appears outside `internal/analyze/` and
   `internal/languages/`.

The language list in the README (the `--languages` flag row) and the
comments in `internal/config/templates/cdd.config.yaml.tmpl` are the only
places kept by hand.
