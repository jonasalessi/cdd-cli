# Feature 03 — TypeScript Support: CDD Analyzer on Tree-sitter

## Goal

Implement the first CDD analyzer — TypeScript — on top of Tree-sitter, together
with the language-agnostic analysis pipeline and the `cdd check` command, so
that running `cdd check` in a TypeScript project parses every matched source
file, computes ICPs per unit, enforces the configured `icp-limits`, and reports
results in any of the four reporter formats.

As the first analyzer, this feature also establishes the pipeline every future
analyzer (Go, Java, Kotlin) plugs into.

## Scope

**In:**
- `internal/analyze` — language-agnostic pipeline: analyzer contract, unit and
  result types, include/exclude matcher, pattern-weight resolution, parallel
  file walking, timeout with partial results.
- `internal/analyze/typescript` — the Tree-sitter TypeScript/TSX analyzer.
- `internal/report` — the four reporters: `console`, `json`, `xml`, `markdown`.
- `cmd/check.go` — the `cdd check` command (thin cobra layer, no business rules).
- Vocabulary fix: `??` is a condition, not a code branch.
- CI/build changes required by CGO.

**Out:**
- Go, Java, and Kotlin analyzers.
- `cdd report` command and the `boy_scout` baseline store.
- Incremental parsing / watch mode.
- `.js`, `.jsx`, `.mjs`, `.cjs` files (JavaScript is a separate feature).
- tsconfig `baseUrl`, `extends` chains, and monorepo project references
  (follow-up; only root `tsconfig.json` `paths` are honoured, as `detect.Packages`
  does today).

## Dependencies (pinned)

| Module | Version | Notes |
|---|---|---|
| `github.com/tree-sitter/go-tree-sitter` | `v0.24.0` | Official binding. CGO. Do **not** use `v0.25.0` — phantom tag, deleted from GitHub, unresolvable with `GOPROXY=direct`. |
| `github.com/tree-sitter/tree-sitter-typescript` | `v0.23.2` | One import (`bindings/go`) provides both `LanguageTypescript()` and `LanguageTSX()`. |

Consequences to accept and document:
- `CGO_ENABLED=1` is mandatory; `go install` now needs a C compiler (README note).
- Binary grows roughly 5–12 MB (the TS + TSX parse tables are ~17.5 MB of C).
- Cross-compilation needs a C cross-toolchain (use `zig cc` when a release
  matrix is introduced).

Grammar selection is by extension: `.ts`, `.mts`, `.cts` → `typescript`;
`.tsx` → `tsx`. Both grammars are required: JSX breaks the plain grammar and
legacy `<Type>expr` casts break the tsx grammar.

## Functional requirements

| # | Rule | Where the rule lives |
|---|---|---|
| FR-1 | A TypeScript **unit** is each top-level: `class_declaration`, `abstract_class_declaration`, `interface_declaration`, `enum_declaration`, `type_alias_declaration`, `function_declaration`, `generator_function_declaration`, and each top-level `lexical_declaration` whose declarator initializer is an `arrow_function` or `function_expression` and is exported. `ambient_declaration` and declaration-only signatures are not units. | `internal/analyze/typescript` |
| FR-2 | ICPs are counted per unit using the metric→node mapping below; only metrics present in the merged weights for the file are counted; each occurrence adds the configured weight. | `internal/analyze/typescript` |
| FR-3 | Weights/limits resolution: pattern keys are Go RE2 regexes matched against the file path relative to the config file with `/` separators; every matching entry applies in document order, later entries merging over earlier ones; a metric absent from the merged result is disabled. | `internal/analyze` |
| FR-4 | Include/exclude: entries without a prefix are globs (`glob:` implied), `regex:` marks a regex; exclude always wins over include; defaults come from `config.DefaultExcludes`. | `internal/analyze` (matcher) |
| FR-5 | A file whose root node `HasError()` is skipped from ICP totals and surfaced as a warning naming the file and first error position. | `internal/analyze/typescript` |
| FR-6 | Import classification: relative specifiers (`./`, `../`) are always internal; specifiers matching a configured or detected internal prefix (`internal_coupling` config, `detect.Packages` tsconfig `paths` aliases) are internal; all other bare specifiers are external. Type-only imports count. Side-effect imports (`import "x"`) count as coupling of the resolved kind. | `internal/analyze/typescript` |
| FR-7 | `cdd check` exit codes: `0` when no unit exceeds its limit, `1` when violations exist and enforcement blocks, non-zero with a partial report when `timeout` elapses. Uses `exitCodeError` (`cmd/root.go`). | `cmd/check.go` + `internal/analyze` |
| FR-8 | Reporter format comes from `reporter.format` (`console`, `json`, `xml`, `markdown`); `reporter.output_file` nil ⇒ stdout, otherwise write the file. | `internal/report` |
| FR-9 | No metric/language/mode string literals outside `internal/config/vocabulary.go` (`make check-literals`); no `func init()` self-registration (`gochecknoinits`). Analyzer lookup is an explicit `map[config.Language]…`/`switch` iterating `config.Languages()`. | `internal/analyze` |
| FR-10 | `??` counts as `condition` only; `?.` remains a `code_branch`. The `code_branch` description in the vocabulary drops `??`. | `internal/config/vocabulary.go` |

## Metric → Tree-sitter node mapping

Node kinds are the exact grammar names from `tree-sitter-typescript v0.23.2`.

| MetricID | Counted nodes / rule |
|---|---|
| `code_branch` | `if_statement` +1; `else_clause` +1 only when its body is not another `if_statement` (avoid double-counting `else if`); `switch_case` +1 each (`switch_default` +0); `ternary_expression` +1; `for_statement`, `for_in_statement` (covers both `for…in` and `for…of`), `while_statement`, `do_statement` +1 each; `optional_chain` (`?.`) +1 |
| `condition` | `binary_expression` whose `operator` field is `&&`, `\|\|`, or `??` +1 each; `augmented_assignment_expression` with `&&=`, `\|\|=`, `??=` +1 each |
| `exception_handling` | +1 per block: the `try_statement` body, each `catch_clause`, each `finally_clause` (full try/catch/finally = 3) |
| `internal_coupling` | +1 per `import_statement` classified internal (FR-6) |
| `external_coupling` | +weight per `import_statement` classified external (FR-6) |
| `inheritance` | `class_heritage` clauses (`extends` / `implements`) +1 per level; interface `extends` +1 per listed parent |
| `local_variable` | declarators of `lexical_declaration` / `variable_declaration` inside a unit body, plus `public_field_definition` class fields, +weight each |
| `lambda` | `arrow_function` and `function_expression` in expression/argument position +weight each (those that are themselves units per FR-1 are not double-counted) |

Reference check from `docs/cdd.md`: `if (a > b && c < d)` = 3 ICPs (1 branch + 2
conditions — the `&&` plus… see docs example) — encode the doc's worked examples
as test fixtures verbatim.

## Implementation guidance (binding discipline)

- The binding has **no finalizers**: every `Parser`, `Tree`, `TreeCursor`,
  `Query`, `QueryCursor` must be `Close()`d (`defer tree.Close()` per parse).
  Leaks are C-heap leaks invisible to the Go GC.
- Never let a `*Node` outlive its `*Tree` — copy out `Kind()`, `ByteRange()`,
  `StartPosition()`, and text into plain result structs.
- One long-lived `Parser` and cursor per worker goroutine (`Parser` is not
  thread-safe); reuse via `cursor.Reset`. Parallelise with `errgroup`
  (already in the module graph).
- Resolve node-kind ids once at startup with `Language.IdForNodeKind(kind, true)`
  and compare `node.KindId()` (uint16) in the traversal loop; string
  comparison across the CGO boundary is the hot-path cost.
- Guard pathological (minified/generated) files with `parser.SetTimeoutMicros`.

## Deliverables

```
internal/analyze/
    analyze.go          Analyzer contract, Unit/FileResult/RunResult types
    matcher.go          include/exclude matching (FR-4)
    weights.go          pattern weight/limit resolution (FR-3)
    run.go              file walk, worker pool, timeout partial results (FR-7)
    *_test.go           table-driven tests
internal/analyze/typescript/
    parser.go           grammar-by-extension parser wiring
    units.go            unit extraction (FR-1)
    metrics.go          counters (mapping table)
    imports.go          coupling classification (FR-6)
    *_test.go, testdata/  rich fixtures per metric + docs/cdd.md worked examples
internal/report/
    console.go json.go xml.go markdown.go, golden tests (FR-8)
cmd/
    check.go check_test.go   thin cobra layer + e2e binary tests
internal/config/vocabulary.go   FR-10 description fix
cdd.config.yaml + internal/config/testdata/golden/* + cmd/testdata/golden/*
    regenerated after FR-10 (CI dogfood gate)
.github/workflows/ci.yml, Makefile, README.md   CGO_ENABLED=1, C-compiler prereq
```

## Tasks

- **T1 — Dependencies & parse spike.** Add the two pinned modules; parse one
  `.ts` and one `.tsx` fixture in a test proving grammar selection and error
  detection. Verify CI builds with CGO.
  *Accept:* `make build` and `make test` pass locally and in CI.

- **T2 — Pipeline types & matching.** `internal/analyze`: analyzer contract,
  result types, weight/limit resolution (FR-3), include/exclude matcher (FR-4).
  *Accept:* table-driven tests cover glob, regex, exclude-wins, last-match-wins
  merge, disabled-metric cases; coverage ≥ 90 %.

- **T3 — Unit extraction.** FR-1 in `internal/analyze/typescript`.
  *Accept:* a fixture containing every unit kind (plus non-units: ambient
  declarations, non-exported arrow consts, nested functions) yields exactly the
  expected unit names and positions.

- **T4 — Metric counters.** The full mapping table.
  *Accept:* one fixture per metric with hand-computed totals; the worked
  examples in `docs/cdd.md` reproduce exactly (e.g. try/catch/finally = 3).

- **T5 — Coupling resolver.** FR-6 using `detect.Packages` + `internal_coupling`
  config.
  *Accept:* fixtures with a JSONC `tsconfig.json` (`paths` aliases), relative,
  aliased, and bare imports classify correctly.

- **T6 — Vocabulary `??` fix.** FR-10; regenerate golden files and the root
  `cdd.config.yaml`.
  *Accept:* CI dogfood reproducibility gate stays green.

- **T7 — Reporters.** Four formats with golden-file tests (reuse the `-update`
  / `UPDATE_GOLDEN=1` convention).
  *Accept:* a shared fixture `RunResult` renders to four checked-in goldens.

- **T8 — `cdd check`.** Cobra wiring on `cmd/root.go`'s `AddCommand`, exit
  codes (FR-7), timeout partial report; e2e binary tests following the
  `cmd/init_test.go` `TestMain` pattern.
  *Accept:* e2e on a TS fixture project: over-limit unit ⇒ exit 1 with the unit
  named; clean project ⇒ exit 0; `--config` respected.

- **T9 — Docs & CI.** README: C-compiler prerequisite for `go install`,
  binary-size note; CI: `CGO_ENABLED=1` explicit; Makefile untouched targets
  keep working.
  *Accept:* README documents the prerequisite; CI green end-to-end.

## Definition of done

- [ ] `make build`
- [ ] `make test` (race detector on, as today)
- [ ] `make lint` (including `check-literals`)
- [ ] `make fmt` leaves no diff
- [ ] Coverage ≥ 90 % for `internal/analyze`, `internal/analyze/typescript`, `internal/report`
- [ ] CI dogfood gate (`cdd init … && git diff --exit-code cdd.config.yaml`) green
- [ ] `cmd/check.go` contains no business rules (mirrors the `init` layering)

## Suggested order

T1 → T2 → T3 → T4 → T5 → T6 → T7 → T8 → T9. T6 is independent and can land any
time after T1; T7 only depends on T2's result types.
