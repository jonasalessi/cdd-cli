# CDD CLI: Cognitive-Driven Development Analyzer

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

CDD CLI is a tool designed to measure and manage code complexity based on the principles of **Cognitive-Driven Development (CDD)**. It helps developers identify areas of the code that are difficult to understand and maintain by calculating the **Intrinsic Cognitive Point (ICP)**.

> ### 🎓 Foundations in Research
>
> This tool is a direct implementation of the **Cognitive-Driven Development (CDD)** methodology. It follows the theoretical
> framework established in the seminal paper:
>
> Tavares de Souza, A. L. O., Costa Pinto, V. H. S. 2020.  
> Toward a Definition of Cognitive-Driven Development, 2020 IEEE International Conference on Software Maintenance and
> Evolution (ICSME), pp. 776–778, https://doi.org/10.1109/ICSME46990.2020.00087

## What CDD measures

Every `if`, `&&`, `catch`, coupling to another type or inheritance level adds
Intrinsic Complexity Points to a code unit. The team picks which constructs
count and how much, then sets one limit, usually 10 for a new project. A unit
above the limit is treated like a compile error and gets refactored before it
is merged. The full method, including limit bands for greenfield and legacy
code, is in [docs/cdd.md](docs/cdd.md).

## Install

```sh
go install github.com/jonasalessi/cdd-cli@latest
```

Or from a clone:

```sh
make build      # writes bin/cdd with version, commit and date injected
./bin/cdd version
```

## Usage

```
$ cdd --help
cdd measures code with Cognitive-Driven Development (CDD): every branch,
condition, coupling or exception block adds Intrinsic Complexity Points (ICPs)
to a code unit, and a unit above the agreed limit must be refactored before it
is merged.

The method, the ICP vocabulary and the limit bands are described in
docs/cdd.md. Run "cdd init" to write a cdd.config.yaml for your project.

Usage:
  cdd [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  init        Create a cdd.config.yaml for this project
  version     Print the cdd version

Flags:
      --config string   path to the cdd configuration file (default "cdd.config.yaml")
  -h, --help            help for cdd
  -v, --version         version for cdd

Use "cdd [command] --help" for more information about a command.
```

The configuration file this repository uses on itself is
[cdd.config.yaml](cdd.config.yaml). Every key is commented, and the same text
is what `cdd init` will write.

## Status: init in progress

Round 1 is done: the command tree, the `cdd.config.yaml` schema (types,
vocabulary, rendering, loading with order-preserving patterns, validation
rules V1 to V12) and the golden tests. `cdd init` is a stub that already
accepts its final flag set and exits 1. Round 2 wires the interactive flow and
language detection. Analyzers come after that.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the commit format and the git hook
setup. `make test`, `make lint` and `make cover` are what CI runs.
