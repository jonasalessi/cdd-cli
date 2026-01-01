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

## Project Structure

This project is organized into two main Gradle modules:

- **`core`**: Contains the core CDD/ICP analysis logic, domain models, and reporters. It is designed to be easily reusable for future integrations (e.g., IDE plugins, web dashboards) and remains independent of CLI-specific concerns.
- **`cli`**: Contains the command-line interface logic using Clikt. It depends on the `core` module to perform the analysis.

## Quick Start

### Prerequisites

- **Java**: JRE 21 or higher.

### Installation

#### Building From Source

To build the project and generate the distributions:

```bash
./gradlew clean build
```

The artifacts will be generated in:
- **Distribution Zip**: `cli/build/distributions/cdd-cli.zip`
- **Fat JAR**: `cli/build/libs/cdd-cli.jar`

#### Using Releases

1. Download the latest release from the [Releases](https://github.com/jonas/icp-cli/releases) page.
2. You can choose between:
    - **Distribution Zip (`cdd-cli.zip`)**: Unzip this file and add the `bin` folder to your system's `PATH`.
    - **Fat JAR (`cdd-cli.jar`)**: A single executable JAR file.

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

*   `--config <file>`: Path to configuration file (default: `.cdd.yaml` or `.cdd.yml`)
*   `--format <format>`: Output format (`console`, `json`, `xml`, `markdown`). Default: `console`.
*   `--output <file>`: Write output to file instead of stdout.
*   `--fail-on-violations`: Exit with code 1 if violations are found.
*   `--include <pattern>`: Include files matching pattern (glob).
*   `--exclude <pattern>`: Exclude files matching pattern (glob).
*   `-v, --verbose`: Enable verbose logging.
*   `--debug`: Enable debug logging (creates `cdd-debug.log`).

## Configuration

CDD CLI is configured via a YAML file (`.cdd.yaml`). Limits and metrics are now defined per language and file path.

```yaml
# Cognitive Driven Development (CDD) Configuration Template
# This file defines how the ICP (Intrinsic Complexity Points) are calculated and reported.

# -----------------------------------------------------------------------------
# METRICS: ICP WEIGHTS
# -----------------------------------------------------------------------------
# Defines how many points each code construct contributes to the total ICP of a class.
# Higher weights make specific patterns count more towards the limit.
# Structure: metrics -> [language] -> [file_regex_pattern] -> [metric_type]: [weight]
metrics:
  java:
    ".*": # Default weights for all Java files
      code_branch: 1.0        # if/else, switch case, ternary operator
      condition: 1.0          # logical operators (&&, ||) and branch expressions
      internal_coupling: 1.0  # references to other classes within the internal project
      exception_handling: 1.0 # try/catch and finally blocks
  kotlin:
    ".*": # Default weights for all Kotlin files
       code_branch: 1.0       # if/when, loop constructs, safe calls (?.), elvis (?:)
       condition: 1.0         # logical operators and branch expressions
       internal_coupling: 1.0 # references to other internal project classes/packages
       exception_handling: 1.0 # try/catch blocks

# -----------------------------------------------------------------------------
# ICP-LIMITS: CLASS THRESHOLDS
# -----------------------------------------------------------------------------
# Defines the maximum total ICP allowed for a single class before it is flagged.
# If a class's total ICP (sum of all weighted metrics) exceeds this value, 
# it will be reported as a violation.
# Structure: icp-limits -> [language] -> [file_regex_pattern]: [limit_value]
icp-limits:
   java:
      ".*": 12 # Default threshold for Java classes. Recommended range: 8-15.
   kotlin:
      ".*": 12 # Default threshold for Kotlin classes.

# -----------------------------------------------------------------------------
# REPORTER SETTINGS
# -----------------------------------------------------------------------------
reporter:
   # format: The output style. Supported: console, json, xml, markdown
   format: console
   # outputFile: Optional path to save the report. If null, outputs to stdout.
   outputFile: null

# -----------------------------------------------------------------------------
# INTERNAL COUPLING DETECTION
# -----------------------------------------------------------------------------
# Settings to define what is considered "internal" to your project.
internal_coupling:
   # auto_detect: If true, tries to discover internal packages from source files.
   auto_detect: true
   # packages: Explicit list of package prefixes to treat as internal.
   # e.g., ["com.mycompany.app"]
   packages: []

# -----------------------------------------------------------------------------
# FILE FILTERING
# -----------------------------------------------------------------------------
# Patterns follow glob: or regex: syntax. If no prefix is provided, glob: is assumed.
# include: If not empty, only files matching at least one pattern will be analyzed.
include: [] 
# exclude: Files matching these patterns will be skipped (e.g., generated code, tests).
exclude: [] 

# -----------------------------------------------------------------------------
# SLOC (SOURCE LINES OF CODE) METRICS
# -----------------------------------------------------------------------------
sloc:
   # methodLimit: Maximum lines of code allowed in a single method.
   # Flagged separately from ICP in some reports.
   methodLimit: 24
```

### Metrics and Limits
*   **`metrics`**: Customizes how many points each construct contributes to the total ICP.
    *   `code_branch`: Branches (if, switch, ?. (safe call), etc.).
    *   `condition`: Logical operators and conditions in loops.
    *   `internal_coupling`: References to other classes in the project.
    *   `exception_handling`: Usage of try/catch/finally blocks.
*   **`icp-limits`**: Sets the maximum allowed **total ICP** (sum of all weighted metrics) for a class. If a class exceeds this limit, it's flagged as a violation.

Patterns support **Regex** matching for fine-grained control over specific file groups.

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
