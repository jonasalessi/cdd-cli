# Editor integration

This is the contract the IntelliJ and VS Code plugins read. A plugin runs

```sh
cdd check --all --explain --format json
```

and turns each occurrence into an inline hint. Pass the files that were just
saved as paths to re-check them alone:

```sh
cdd check src/order/service.ts,src/order/repository.ts --all --explain --format json
```

The two flags are independent: `--all` chooses which units are listed,
`--explain` details the ones that are. A plugin wants both, so that a unit
within its limit still shows where its points come from.

## The occurrence record

In `json` and `xml` the detail is per unit rather than per line, and the
top-level `explain` field says whether it was asked for at all:

```json
{
  "unit": "OrderService",
  "metric": "code_branch",
  "line": 5, "col": 5, "end_line": 7, "end_col": 6,
  "count": 1, "score": 1
}
```

The range is 1-based and its end points just past the construct, the way an
editor selects text. `unit` repeats the name of the owning unit so a plugin
can flatten every occurrence of a file into one list. Without `--explain` the
`occurrences` key is absent; with it, a unit that counted nothing carries an
empty list. A unit whose constructs the analyzer could not locate gains no
occurrence, so the report never invents a position.

## Fields to read

- `filter` is `violations` or `all`, so a reader knows why a unit is missing.
- `blocked` says whether the run failed under the configured enforcement.
- `partial` is true when the timeout cut the run short; the report covers the
  files analyzed in time.
- `warnings`, at the report and at each file, lists what the analyzer could
  not read without failing the run.
