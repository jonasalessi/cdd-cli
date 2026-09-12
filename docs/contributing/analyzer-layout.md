# Analyzer layout

An analyzer implements `analyze.Analyzer` and is registered by setting
`NewAnalyzer` on the language's line in `internal/languages/languages.go`.
Every analyzer keeps the same file layout inside its package:

| File          | Holds                                                       |
| ------------- | ----------------------------------------------------------- |
| `spec.go`     | the `config.LanguageSpec`, the only place ids are spelled out |
| `analyzer.go` | the `Analyzer` type, its constructor and the file walk      |
| `units.go`    | which nodes are units (classes, functions, types)           |
| `metrics.go`  | the rule that maps a node to a metric occurrence            |
| `imports.go`  | the walk that reads a path, a binding and a star out of an import |
| `stdlib.go`   | the standard-library predicate; Java and Kotlin share one in `internal/analyze/internal/jvm/` instead, see [stdlib-classification.md](stdlib-classification.md) |
| `parser.go`   | tree-sitter analyzers only: the grammar and its node-kind table |

Every analyzer produces the same `Occurrence` spans, so the report and the
limit check never learn which language produced them.

## What is shared and what is not

Grammar-agnostic mechanics live in `internal/analyze/internal/treesitter`:
the parse budget, the cursor walk, child accessors, source ranges,
syntax-error reporting and the occurrence sort. Rules that belong to a
platform rather than to one language live beside them:
`internal/analyze/internal/jvm` holds the package-prefix detection and the
per-unit import attribution the Java and Kotlin analyzers share.

The node-kind table, the metric rules, the unit rules and the import walk
stay in each language package. When adding a language, copy the shape of
`internal/analyze/kotlin/parser.go`, not its values.

## Parsing without tree-sitter

An analyzer is not required to use tree-sitter. `internal/analyze/golang`
parses with `go/parser` from the standard library, which pins no grammar and
adds nothing to the binary. It keeps the file layout above and the same
`analyze.Analyzer` contract, but has no `parser.go`, no
`TestGrammarResolvesEveryKind` and no `io.Closer`, and reuses only
`treesitter.SortOccurrences` and `treesitter.Span`. Everything in
[tree-sitter-grammars.md](tree-sitter-grammars.md) applies to a tree-sitter
analyzer alone.
