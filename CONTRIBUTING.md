# Contributing to cdd-cli

## Run the setup first

Clone the repo, then run this once:

```sh
make setup
```

It sets `core.hooksPath` to `.githooks/`, so git runs the two hooks checked
into this repo: `pre-commit` formats and lints your Go files, and
`commit-msg` rejects any commit whose message does not match the format
below. Skip the setup and git will happily accept unformatted code and a bad
message, and the reviewer will ask you to redo both.

## What the hooks do

`pre-commit` runs `gofmt` and `golangci-lint run --fix` on the staged Go
files, re-stages whatever they fixed, and blocks the commit only when an
issue cannot be fixed automatically. It refuses partially staged Go files,
since re-staging those would commit hunks you left out on purpose.

`commit-msg` checks the message against the format in the next section. A
rejected message creates no commit, so fix the message and commit again.

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

The type is lowercase, followed by a colon and one space. Keep each commit
to one change you can describe in a sentence. Commit messages carry no
trailers; in particular, do not add `Co-Authored-By` when a coding
assistant helped with the change.

## Code style and checks

Follow [Effective Go](https://go.dev/doc/effective_go). The `pre-commit`
hook enforces the mechanical part with `gofmt` and `golangci-lint`.

Before opening a pull request, run all four and make sure the last one
leaves no diff:

```sh
make build
make test
make lint
make fmt
```

`make lint` also runs `make check-literals`, which fails if a language id,
metric id or mode is spelled out as a string outside the places listed in
the next section, or if a language-keyed table appears outside
`internal/analyze/` and `internal/languages/`.

## Adding a language

Every language lives in one directory, `internal/analyze/<id>/`, and is
registered in one file, `internal/languages/languages.go`. No other Go code
knows the language exists: `config`, `detect`, `prompt` and `cmd` all work
from the registered specs they are handed. Two documentation files are
kept by hand; step 4 lists them.

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

   A language may ship with a spec and no analyzer, as `go` does today:
   `cdd init` still configures it but warns that no analyzer exists
   yet, and `cdd check` reports it as an error rather than counting zero
   ICPs. Set `NewAnalyzer` on the same line once the analyzer exists;
   the next section describes how to write one.

3. Run `make test` and `make lint`. The registry tests fail naming the
   directory if the line is missing, naming the id if the directory is
   missing, and naming the field if the spec is incomplete. The literal
   check fails if the id leaked outside `spec.go`.

4. Update the two hand-maintained lists: the `--languages` row in the
   README's flag table, and the language comments in
   `internal/config/templates/cdd.config.yaml.tmpl`.

### Writing an analyzer

An analyzer implements `analyze.Analyzer` and is registered by setting
`NewAnalyzer` on the language's line in `languages.go`. The TypeScript,
Kotlin and Java analyzers are built on tree-sitter; the grammar-agnostic
mechanics they share (the parse budget, the cursor walk, child accessors,
source ranges, syntax-error reporting and the occurrence sort) live in
`internal/analyze/internal/treesitter`. Rules that belong to a platform
rather than to one language live beside them:
`internal/analyze/internal/jvm` holds the package-prefix detection and the
per-unit import attribution the Java and Kotlin analyzers share, while the
grammar walk that reads a path, a binding and a star out of an import stays
in each language package. The node-kind table, the metric rules and the
unit rules stay there too: copy the shape of
`internal/analyze/kotlin/parser.go`, not its values.

An analyzer also decides which of its imports are the standard library, and
it does so with a predicate it hands to the classifier it shares:
`jvm.NewImports(prefixes, stdlib)` for the JVM languages, its own `classify`
for TypeScript. The list that predicate reads lives in the language package's
`stdlib.go`, except for the JDK table Java and Kotlin share, which lives in
`internal/analyze/internal/jvm/stdlib.go`. `internal/analyze/*/stdlib.go` is
exempt from the literal check the way `spec.go` is, because module names such
as Node's `console` collide with vocabulary ids. A configured project prefix
always wins over the standard library, so a project that lists `java` or
`path` among its own packages keeps counting it as internal coupling.

Each language package builds its grammar once, at package level, and every
analyzer instance shares it. Every node kind, field and anonymous token the
analyzer relies on is resolved by name at that point and pinned by a test
(`TestGrammarResolvesEveryKind` and friends), so a grammar bump that renames
something fails loudly in `make test` instead of silently counting zero.
When you add a kind, add it to that test as well.

Every grammar and the `go-tree-sitter` binding are pinned in `go.mod`. The
binding is pinned to `v0.24.0` and must not be bumped without re-running
every analyzer's tests.

### Grammar provenance

The TypeScript grammar comes from the `tree-sitter/` GitHub organisation,
and `tree-sitter/tree-sitter-java` comes from the same place, so both are
maintained where the parser generator itself is. The Kotlin grammar, `tree-sitter-grammars/tree-sitter-kotlin`, comes from
the community collective that maintains a fork of `fwcd/tree-sitter-kotlin`;
the original `fwcd` grammar is one release behind and less active, which is
why the fork was chosen. Depending on a grammar from outside `tree-sitter/`
is an accepted risk, and the resolve-by-name test above is the mitigation.
