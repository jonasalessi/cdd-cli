// Package treesitter holds the tree-sitter mechanics every analyzer built on
// the go-tree-sitter binding shares and none of which depends on a grammar:
// the parse budget, the cursor walk, the small child accessors, source
// ranges, syntax-error reporting and the source-order sort of occurrences.
// Node kinds, fields, metric rules and unit rules belong to each language
// package; this one knows nothing about metrics or configuration.
package treesitter
