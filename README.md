# CDD CLI: Cognitive-Driven Development Analyzer

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

CDD CLI is a tool designed to measure and manage code complexity based on the principles of **Cognitive-Driven Development (
CDD)**. It helps developers identify areas of the code that are difficult to understand and maintain by calculating the *
*Intrinsic Cognitive Point (ICP)**.
hb
> ### 🎓 Foundations in Research
>
> This tool is a direct implementation of the **Cognitive-Driven Development (CDD)** methodology. It follows the theoretical
> framework established in the seminal paper:
>
> Tavares de Souza, A. L. O., Costa Pinto, V. H. S. 2020.  
> Toward a Definition of Cognitive-Driven Development, 2020 IEEE International Conference on Software Maintenance and
> Evolution (ICSME), pp. 776–778, https://doi.org/10.1109/ICSME46990.2020.00087

## Key Features

- **Multi-Language Support**: Analyzes both **Java** and **Kotlin** source code.
- **ICP Calculation**: Measures complexity based on branching logic, coupling, and exception handling.
- **SLOC Metrics**: Provides physical Source Lines of Code distribution.
- **Actionable Recommendations**: Suggests refactoring targets based on complexity thresholds.
- **Multiple Output Formats**: Supports Console, JSON, XML, and Markdown.

## Quick Start

### Prerequisites

- **Java**: JRE 21 or higher.

### Installation

1. Download the latest release from the [Releases](https://github.com/jonas/icp-cli/releases) page.
2. You can choose between two versions:
    - **Distribution Zip (`cdd-cli.zip`)**: Unzip this file and add the `bin` folder to your system's `PATH`. This allows you
      to run `cdd-cli` directly from your terminal.
    - **Fat JAR (`cdd-cli.jar`)**: A single executable JAR file containing all dependencies.

### Usage

If you added the distribution's `bin` folder to your `PATH`:

```bash
cdd-cli /path/to/your/code
```

Alternatively, using the fat JAR:

```bash
java -jar cdd-cli.jar /path/to/your/code
```

### Common Options

- `<path>`: Directory or file to analyze (required).
- `--limit <double>`: Set the ICP limit per class (default: 10.0).
- `--sloc-limit <int>`: Set the SLOC limit for methods (default: 24).
- `--format <format>`: Output format: `console`, `json`, `xml`, `markdown` (default: `console`).
- `--output <file>`: Redirect output to a file (default: stdout).
- `--config <file>`: Path to a custom YAML configuration file (default: `.cdd.yml`).
- `--include <pattern>`: Include file pattern (can be repeated).
- `--exclude <pattern>`: Exclude file pattern (can be repeated).
- `--fail-on-violations`: Exit with code 1 if any class exceeds the ICP limit (default: false).
- `-v, --verbose`: Enable verbose output.
- `--debug`: Generate `cdd-debug-[date time].log` for debugging and troubleshooting (default: false).
- `--debug-log-dir <path>`: Custom directory to store the `cdd-debug-[date time].log` file.
- `--version`: Show the version and exit.

## Configuration

CDD CLI can be configured using a `.cdd.yaml`, `.cdd.yml`, or `.cdd.json` file in your project root or via the `--config` option.

```yaml
limit: 10.0 # Maximum ICP value allowed per class
icpTypes: # Optional: custom weights for ICP types
  CODE_BRANCH: 1.0
  CONDITION: 1.0
  EXCEPTION_HANDLING: 1.0
  INTERNAL_COUPLING: 1.0
  EXTERNAL_COUPLING: 0.5
classTypeLimits: { } # Optional: limits per class type (e.g., Service: 15)

internalCoupling:
  autoDetect: true # Enables automatic detection of project packages
  packages: [ ] # List of internal packages to consider for coupling

externalCoupling:
  libraries: # List of external libraries to monitor coupling for
    - "java.util.*"
    - "java.io.*"
    - "java.lang.*"

include: [ ] # List of glob patterns for files to include
exclude: [ ] # List of glob patterns for files to exclude

sloc:
  classLimit: 0 # SLOC limit for classes (0 = disabled)
  methodLimit: 24 # SLOC limit for methods
  warnAtMethod: 15 # SLOC threshold for warnings
  excludeComments: true # Do not count comments in SLOC
  excludeBlankLines: true # Do not count blank lines in SLOC

reporting:
  format: "console" # Output format: console, json, xml, markdown
  outputFile: null # Optional: path to write the report
  verbose: false # Enable detailed reporting
  showLineNumbers: true # Include line numbers in reports
  showSuggestions: true # Provide refactoring suggestions
  showSlocMetrics: true # Include SLOC metrics in report
  showSlocDistribution: true # Show SLOC distribution charts
  showCorrelation: true # Show ICP vs SLOC correlation
```

## Known Limitations

### Non-Standard Code Patterns

Highly dynamic code or code using heavy reflection might not have its coupling fully detected, as the analysis is primarily
static and PSI-based.

## Support

For issues, please visit the [Issue Tracker](https://github.com/jonasalessi/cdd-cli/issues).

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
