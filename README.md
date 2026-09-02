# CDD CLI: Cognitive-Driven Development Analyzer

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

`cdd` measures how much of your code a reader has to hold in their head at
once. It scores every code unit in Intrinsic Complexity Points (ICPs) and
flags the ones above the limit your team picked.

> The method comes from a 2020 ICSME paper:
>
> Tavares de Souza, A. L. O., Costa Pinto, V. H. S. 2020.
> Toward a Definition of Cognitive-Driven Development. 2020 IEEE
> International Conference on Software Maintenance and Evolution (ICSME),
> pp. 776-778. https://doi.org/10.1109/ICSME46990.2020.00087

## What CDD measures

Every `if`, `&&`, `catch`, coupling to another type and inheritance level adds
points to the unit that holds it. The team picks which of those count and how
much, then sets one limit. A unit over the limit fails the way a compile error
fails, and someone refactors it before it merges.

Ten is the usual starting limit for a new project. Legacy code starts higher,
somewhere between 20 and 40, and comes down as the code improves.

## Installation

```sh
go install github.com/jonasalessi/cdd-cli@latest
```

That build needs a C compiler. The TypeScript analyzer embeds Tree-sitter
through cgo, so `CGO_ENABLED=1` and a working toolchain are required:

| Platform | Toolchain |
| --- | --- |
| macOS | clang, from the Xcode command line tools: `xcode-select --install` |
| Debian / Ubuntu | gcc, from the `build-essential` package |
| Windows | gcc, from MSYS2 or MinGW-w64 |

The grammar parse tables are compiled into the binary, which makes it a few
megabytes larger than a pure-Go build.

Or from a clone:

```sh
make build      # writes bin/cdd with version, commit and date injected
./bin/cdd version
```

## Usage

Every command reads `cdd.config.yaml` from the working directory. Pass
`--config path/to/file.yaml` when it lives somewhere else.

```sh
cdd --help      # the command list
cdd version     # version, commit and build date
cdd init        # Initialize the configuration
cdd check       # Measure the project against the configuration
```

### cdd init

`init` writes that file, so it is where a project starts. It walks the first
two steps of CDD, agreeing on which constructs count as ICPs and agreeing on
the limit.

With no flags it asks one question at a time.

1. Which languages to configure. `cdd` scans the project first and ticks what
   it finds, so usually you press enter.
2. Greenfield or legacy. Greenfield keeps the limit tight from the first
   commit and defaults to 10, in a band of 7 to 14. Legacy measures what
   already exists and defaults to 25, in a band of 20 to 40.
3. For legacy only, how hard to enforce: `strict_all`, `strict_on_new_only`,
   `boy_scout` or `measure_only`.
4. The limit itself. Anything outside the band prints a warning and is still
   accepted, so pick 6 if the team wants 6.
5. Which metrics to count, three or more per language. `init` hides the ones
   an analyzer cannot see, which is why Go never offers `exception_handling`
   or `inheritance`.
6. Whether to edit the default weights, which package prefixes count as
   internal, and whether to skip tests and generated code. The defaults suit
   most projects.

The answers come out as commented YAML. [cdd.config.yaml](cdd.config.yaml) is
the file this repository uses on itself, and `init` writes the same comments
into yours.

#### Without the questions

Every answer is also a flag. `--yes` skips the prompts and fills the rest with
defaults, which is what you want in CI or a setup script.

```sh
cdd init --yes \
  --languages go,typescript \
  --project-type legacy \
  --legacy-mode strict_on_new_only \
  --limit 25 \
  --metrics code_branch,condition,internal_coupling,external_coupling
```

It prints one line when the file lands:

```
Created cdd.config.yaml — languages: go, typescript · project: legacy · limit: 25 · metrics: 4
```

| Flag | What it sets |
| --- | --- |
| `--languages` | Languages to configure: `go`, `java`, `kotlin`, `typescript`. |
| `--project-type` | `greenfield` or `legacy`. |
| `--legacy-mode` | Enforcement mode, legacy projects only. |
| `--limit` | ICP limit for every language. `0` takes the default of the project type. |
| `--metrics` | Metric ids to enable, three or more. |
| `--weight id=value` | Overrides one weight. Repeat the flag per metric. |
| `--packages` | Package prefixes that count as internal coupling. |
| `--no-default-excludes` | Keeps tests and generated code in the analysis. |
| `--timeout` | Analysis budget written to the file. Default `5m`. |
| `--scan-timeout` | Budget for detecting languages and packages. Default `4s`. |
| `--force` | Overwrites an existing configuration file. |
| `--output` | Writes the file here instead of the path in `--config`. |
| `--yes` | Skips every prompt. |

#### When the file already exists

`init` never overwrites a `cdd.config.yaml` by accident.

- `--force` overwrites it and says nothing about it.
- Without `--force`, a run in a terminal asks first. Answer no and it prints
  `aborted` and exits 0.
- A run with no prompts, so `--yes` or CI, has nobody to ask. It fails with
  `cdd.config.yaml exists; pass --force to overwrite` and exits 1.

#### The metric vocabulary

| Metric id | Weight | Languages | Counts |
| --- | --- | --- | --- |
| `code_branch` | 1.0 | all | `if`/`else`, `switch`, ternary, loops, and `?.`/`??` in Kotlin and TypeScript |
| `condition` | 1.0 | all | `&&`, `\|\|` and `??` clauses inside a branch |
| `exception_handling` | 1.0 | not Go | `try` / `catch` / `finally` blocks |
| `internal_coupling` | 1.0 | all | References to types that belong to this project |
| `external_coupling` | 0.5 | all | Framework, platform and third-party types |
| `inheritance` | 1.0 | not Go | `extends` / `implements`, counted per level |
| `local_variable` | 0.5 | all | Locals and fields. Off by default |
| `lambda` | 1.0 | all | Lambdas, method references, func literals. Off by default |

The first six are ticked by default. Weights are per language and per file
pattern, so a DTO package can count coupling at half the weight of everything
else without a second file.

### cdd check

`check` walks the last two steps of CDD: it computes the ICPs of every code
unit and compares each one with the limit its file resolves to. It analyzes
the tree rooted at the configuration file's directory, so a configuration in a
subdirectory measures that subdirectory alone.

```
$ cdd check
cdd check: FAIL violations=1 units=2 root=. elapsed=1ms

violation: src/order-service.ts:4:8 class OrderService icp=16.5 limit=10 over=6.5
  metrics: condition=10 code_branch=5 internal_coupling=1 external_coupling=1x0.5
```

The report lists the units over their limit and nothing else, worst first,
because those are the ones to refactor. Each one names where it is, what it
scored and by how much it is over, and the `metrics` line breaks the score
down: a bare `condition=10` counts ten conditions at weight 1, while
`external_coupling=1x0.5` counts one coupling at weight 0.5. The first line
counts the whole run either way, so `units=2` includes what is not listed.

`--all` adds every unit within its limit, after the violations:

```
$ cdd check --all
cdd check: FAIL violations=1 units=2 root=. elapsed=1ms

violation: src/order-service.ts:4:8 class OrderService icp=16.5 limit=10 over=6.5
  metrics: condition=10 code_branch=5 internal_coupling=1 external_coupling=1x0.5

unit: src/greeter.ts:1:8 class Greeter icp=1 limit=10
  metrics: code_branch=1
```

A file the analyzer could not read is reported as `warning: <path>: <text>`
at the end, and a run cut short by the timeout adds `partial=true` to the
first line. The `json` and `xml` reports carry the same filter under a
`filter` field, `violations` or `all`, so a reader knows why a unit is
missing.

| Flag | What it does |
| --- | --- |
| `--all` | Lists every unit, not only the ones over their limit. |
| `--config` | Path to the configuration file. Default `cdd.config.yaml`. |

#### What it reads from the configuration

| Key | What `check` does with it |
| --- | --- |
| `metrics` | Which constructs count per language and file pattern, and their weights. A metric absent from the merged weights is not counted. |
| `icp-limits` | The limit each unit is compared with. The last matching pattern wins. |
| `enforcement` | Whether a unit over its limit fails the run. |
| `timeout` | Wall-clock budget for the whole run. `0s` removes the budget. |
| `reporter` | `format` picks `console`, `json`, `xml` or `markdown`; `outputFile` writes the report to that path instead of stdout. |
| `internal_coupling` | Which import prefixes count as internal coupling rather than external. |
| `include` / `exclude` | Which files are analyzed. `exclude` wins over `include`. |

Only `strict_all` blocks today. `strict_on_new_only` and `boy_scout` need the
git history and the baseline store, neither of which exists yet, so `check`
reports their violations and says they are not enforced.

#### Exit codes

| Code | When |
| --- | --- |
| `0` | No unit is over its limit, or the enforcement only reports them. |
| `1` | A unit is over its limit while `block_on_ci` is true and `legacy_mode` is `strict_all`. |
| `2` | The timeout elapsed. The report printed first covers the files analyzed in time. |
| `1` | Usage error, missing configuration file, or a configuration that fails validation. |

#### Language support

TypeScript is the only language with an analyzer today. `init` still configures
Go, Java and Kotlin, and `check` stops with `no analyzer for <language> yet`
rather than reporting zero ICPs for files it cannot read.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
