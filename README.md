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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).
