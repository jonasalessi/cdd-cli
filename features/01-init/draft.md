Task specification and architecture requirements tailored specifically for building the CLI in **Golang**, designed to support static analysis across **Java, Kotlin, Go, and TypeScript** in future phases.

---

# Task Specification: CDD CLI `init` Command (Go / Multi-Language Architecture)

## 1. Overview & Objective
Build an interactive CLI tool in **Golang** providing an `init` subcommand (`cdd init`). The command guides teams through setting up a project-level `cdd.config.yaml` file based on CDD principles:
- Branching based on project maturity (**Greenfield** vs. **Legacy**).
- Establishing a bounded **ICP threshold limit** grounded in cognitive load limits.
- Selecting and calibrating **Intrinsic Complexity Point (ICP)** variables (minimum of 3).
- Laying the foundational schema for future AST analyzers targeting **Go**, **Java**, **Kotlin**, and **TypeScript**.

---

## 2. Go Tech Stack & Libraries
- **CLI Framework**: [`spf13/cobra`](https://github.com/spf13/cobra) (for command structure, flags, and help texts).
- **Interactive Terminal UI**: [`charmbracelet/huh`](https://github.com/charmbracelet/huh) or [`AlecAivazis/survey/v2`](https://github.com/AlecAivazis/survey) (for multiselects, selects, numeric inputs, and validation).
- **YAML Serializer**: [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) (preserves clean indentation and inline comments).

---

## 3. Functional Requirements

### FR-1: Interactive Initialization (`cdd init`)
- **Existing File Check**: Checks for `./cdd.config.yaml`.
  - If present, prompts: `"A cdd.config.yaml already exists. Overwrite? (y/N)"`.
- **Target Language Detection / Selection**:
  - Automatically inspects the current repository for extensions (`.go`, `.java`, `.kt`, `.ts`, `.tsx`).
  - Presents a multi-select prompt to confirm which languages this project uses:
    - `[ ] Go`
    - `[ ] Java`
    - `[ ] Kotlin`
    - `[ ] TypeScript`

---

### FR-2: Project Context Branching

The user chooses their project maturity branch:

#### Branch A: **New Project (Greenfield)**
- **Suggested Limit**: Recommends a default ceiling between **7 and 10 ICPs** (matching human short-term working memory bounds).
- **Enforcement Policy**:
  - `block_on_ci: true`
  - `legacy_mode: "strict_all"` (every file must comply from day one).

#### Branch B: **Legacy Codebase**
- **Suggested Limit**: Recommends a starting ceiling between **20 and 40 ICPs**.
- **Legacy Scope Strategy**:
  - Prompts how to handle existing code:
    1. **`measure_only`**: Calculate and report metrics across legacy files without failing builds.
    2. **`strict_on_new_only`**: Strictly enforce limits only on newly added files.
    3. **`boy_scout`**: Require that modified legacy files do not increase their baseline ICP total.

---

### FR-3: ICP Metric Selection & Calibration
Presents an interactive checklist to choose which intrinsic elements contribute to cognitive load. **Enforces a minimum selection of 3 metrics**:

1. **`branches`** (Default: `1.0`): `if`, `else`, `switch/case`, ternary `? :`, loops (`for`, `while`).
2. **`conditions`** (Default: `1.0`): Conjoined boolean expressions (`&&`, `||`).
3. **`exceptions`** (Default: `1.0`): `try`, `catch`, `finally` blocks.
4. **`internal_coupling`** (Default: `1.0`): Coupling to internal domain classes/types.
5. **`external_coupling`** (Default: `0.5`): Framework/library imports and types.
6. **`variables`** (Default: `0.0` or `0.5`): Local variable and field declarations.
7. **`lambdas`** (Default: `0.0` or `1.0`): Anonymous functions and closures.

*Users can accept default weights or customize individual costs.*

---

### FR-4: YAML File Generation
Generates a structured `cdd.config.yaml` ready to feed into subsequent multi-language analyzer modules.

#### Sample Greenfield Output (`cdd.config.yaml`):
```yaml
# CDD Configuration - Cognitive-Driven Development
version: 1
project_type: greenfield
languages:
  - java
  - kotlin
global_limit: 10

enforcement:
  block_on_ci: true
  legacy_mode: strict_all

metrics:
  branches:
    enabled: true
    cost: 1.0
  conditions:
    enabled: true
    cost: 1.0
  exceptions:
    enabled: true
    cost: 1.0
  internal_coupling:
    enabled: true
    cost: 1.0
  external_coupling:
    enabled: true
    cost: 0.5
  variables:
    enabled: false
    cost: 0.0
  lambdas:
    enabled: false
    cost: 0.0
```

#### Sample Legacy Output (`cdd.config.yaml`):
```yaml
# CDD Configuration - Cognitive-Driven Development
version: 1
project_type: legacy
languages:
  - golang
  - typescript
global_limit: 25

enforcement:
  block_on_ci: true
  legacy_mode: strict_on_new_only

metrics:
  branches:
    enabled: true
    cost: 1.0
  conditions:
    enabled: true
    cost: 1.0
  exceptions:
    enabled: true
    cost: 1.0
  internal_coupling:
    enabled: true
    cost: 1.0
  external_coupling:
    enabled: true
    cost: 0.5
  variables:
    enabled: false
    cost: 0.0
  lambdas:
    enabled: false
    cost: 0.0
```

---

## 4. Go Architecture & Multi-Language Readiness

To make adding static analyzers for **Java, Kotlin, Go, and TypeScript** straightforward later, the CLI should be structured cleanly around an engine/plugin design:

```
cdd-cli/
├── cmd/
│   ├── root.go             # Base cobra command
│   └── init.go             # 'init' command implementation
├── internal/
│   ├── config/
│   │   ├── config.go       # Struct definitions & YAML tags
│   │   └── generator.go    # Template/serializer logic
│   ├── prompt/
│   │   └── interactive.go  # huh / survey terminal form logic
│   └── analyzer/           # Prepared for future analysis commands
│       ├── analyzer.go     # Common interface: Analyze(filePath string) (ICPScore, error)
│       ├── golang/         # (Future) Uses go/parser and go/ast
│       ├── typescript/     # (Future) Tree-sitter or TS compiler host
│       ├── java/           # (Future) Tree-sitter-java / JavaParser binding
│       └── kotlin/         # (Future) Tree-sitter-kotlin / Kast
└── main.go
```

### Go Configuration Structs (`internal/config/config.go`):
```go
package config

type Config struct {
	Version      int              `yaml:"version"`
	ProjectType  string           `yaml:"project_type"`
	Languages    []string         `yaml:"languages"`
	GlobalLimit  int              `yaml:"global_limit"`
	Enforcement  Enforcement      `yaml:"enforcement"`
	Metrics      map[string]Metric `yaml:"metrics"`
}

type Enforcement struct {
	BlockOnCI  bool   `yaml:"block_on_ci"`
	LegacyMode string `yaml:"legacy_mode"`
}

type Metric struct {
	Enabled bool    `yaml:"enabled"`
	Cost    float64 `yaml:"cost"`
}
```

---

## 5. Implementation Task Checklist

- [ ] **Task 1: Go Project Initialization & Cobra Setup**
  - Initialize Go module (`go mod init github.com/your-org/cdd`).
  - Install dependencies (`github.com/spf13/cobra`, `gopkg.in/yaml.v3`, and `charmbracelet/huh` or `survey/v2`).
  - Wire `cdd init` subcommand under root.
- [ ] **Task 2: Language Detection & Context Prompts**
  - Implement heuristic repository scanner to pre-check languages (`.go`, `.java`, `.kt`, `.ts`).
  - Build interactive branch selector (`greenfield` vs. `legacy`).
  - Implement dynamic limit input with context-aware default recommendations (7–10 vs. 20–40).
- [ ] **Task 3: ICP Metric Selection & Validation**
  - Render multi-select checklist for the 7 ICP elements.
  - Add validator rejecting progression if `< 3` metrics are selected.
  - Add numeric weight configuration loop.
- [ ] **Task 4: YAML Serialization & Persistence**
  - Map interactive answers to `config.Config` struct.
  - Implement file writer checking for existing `./cdd.config.yaml` with overwrite guards.
- [ ] **Task 5: CLI Unit & Integration Tests**
  - Test validation rules (e.g., metric count validator, numeric ceiling boundaries).
  - Test YAML marshaling to verify matching config structure.
