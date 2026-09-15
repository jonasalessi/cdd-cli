# Tree-sitter grammars

The TypeScript, Kotlin and Java analyzers parse with tree-sitter through the
`go-tree-sitter` binding. This page applies only to an analyzer that does the
same; see [analyzer-layout.md](analyzer-layout.md) for the alternative.

## Build the grammar once and pin every name

Each language package builds its grammar once, at package level, and every
analyzer instance shares it. Every node kind, field and anonymous token the
analyzer relies on is resolved by name at that point and pinned by a test
(`TestGrammarResolvesEveryKind` and friends), so a grammar bump that renames
something fails loudly in `make test` instead of silently counting zero.
When you add a kind, add it to that test as well.

## Version pins

Every grammar and the `go-tree-sitter` binding are pinned in `go.mod`. The
binding is pinned to `v0.24.0` and must not be bumped without re-running
every analyzer's tests.

## Grammar provenance

The TypeScript grammar comes from the `tree-sitter/` GitHub organisation, and
`tree-sitter/tree-sitter-java` comes from the same place, so both are
maintained where the parser generator itself is. The Kotlin grammar,
`tree-sitter-grammars/tree-sitter-kotlin`, comes from the community
collective that maintains a fork of `fwcd/tree-sitter-kotlin`; the original
`fwcd` grammar is one release behind and less active, which is why the fork
was chosen. Depending on a grammar from outside `tree-sitter/` is an accepted
risk, and the resolve-by-name test above is the mitigation. State the
provenance of any new grammar here.
