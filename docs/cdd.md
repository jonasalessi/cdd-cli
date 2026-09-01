# Cognitive-Driven Development (CDD): Complete Implementation Guide

**Cognitive-Driven Development (CDD)** is a software design technique created to manage and reduce code complexity by aligning code structures with human cognitive processing capacities. Grounded in **Cognitive Load Theory (CLT)** and the **"Magical Number Seven (± 2)" principle**, CDD posits that every piece of software has an **intrinsic complexity** that imposes a cognitive load on short-term working memory. Because human working memory is constrained, code units (classes, methods, or source files) must be systematically bounded to fit human comprehension.

CDD transforms subjective design debates into an **objective, quantifiable discipline** built around a dual mechanism: **measuring intrinsic complexity** and **enforcing an explicit threshold limit**.

---

## 1. The Core Mechanism: The Metric–Limit Pair

Traditional design practices (such as Clean Code, SOLID, or design patterns) often rely on subjective heuristics that developers interpret differently, leading to inconsistent assessments of whether code is "clean" or "overly complex". CDD resolves this ambiguity through two components:

1. **A Measurable Complexity Metric**: An agreed-upon set of source code elements that add to cognitive load, termed **Intrinsic Complexity Points (ICPs)**.
2. **An Uncompromising Upper-Bound Limit**: A maximum numeric ICP threshold that a single code unit is permitted to accumulate.

Under CDD, a code unit that exceeds the agreed limit carries the same status as a unit that **fails to compile**: it must be refactored before merging or opening a Pull Request.

---

## 2. CDD Variables: Intrinsic Complexity Points (ICPs)

**Intrinsic Complexity Points (ICPs)** are concrete language constructs and dependencies that burden a developer's working memory. CDD does not mandate a single rigid set of ICPs; teams select and calibrate the variables that reflect their stack, experience, and project context. However, to avoid gaming the metric and prevent unmonitored code bloat, teams are advised to select **at least three distinct items**.

### Common ICP Variables and Weighting Scheme

| Variable / Construct | Description & Operational Scope | Typical Cost (Weight) |
| :--- | :--- | :--- |
| **Code Branches** | Structural branching constructs, including `if`, `else`, `switch-case`, ternary operators (`? :`), and loops (`for`, `while`, `do-while`). Resembles Cyclomatic Complexity. | **1.0 ICP** per branch (`if` = 1, `if-else` = 2) |
| **Conditions (Logical Expressions)** | Conjoined logical conditions inside statements (e.g., `&&`, `\|\|`). For instance, `if (a > b && c < d)` scores 3 ICPs: 1 for the `if` and 1 for each Boolean condition. | **1.0 ICP** per conditional clause |
| **Exception & Flow Blocks** | Execution control blocks including `try`, `catch`, and `finally`. A complete `try-catch-finally` block counts as 3 ICPs (1 for each block). | **1.0 ICP** per block |
| **Internal Coupling** | Direct references to domain classes, abstractions, entities, repositories, or services developed within the same project codebase. | **1.0 ICP** per internal dependency reference |
| **External Coupling** | Explicit usage or variable declarations of third-party frameworks, external library units, or standard platform libraries (e.g., Spring framework components, JDK infrastructure). | **0.5 to 1.0 ICP** per declaration |
| **Local Variables & Fields** | Declarations of internal attributes, state variables, or method-level temporary variables. | **0.5 to 1.0 ICP** (optional context variable) |
| **Functional Constructs / Lambdas** | Anonymous functions or lambda expressions. Subject to team consensus (often excluded if team agrees they do not impede comprehension). | **1.0 ICP** (if adopted) or **0.0 ICP** (if excluded) |
| **Inheritance & Hierarchy** | Inherited base classes or implemented interface contracts. | **1.0 ICP** per inheritance level |

---

## 3. Step-by-Step Implementation Workflow

Adopting CDD within a development cycle follows a disciplined seven-step process:

```
+----------------------------------------------------------------+
|  Step 1: Define Team ICP Variables & Costs (Consensus)          |
+----------------------------------------------------------------+
                               |
                               v
+----------------------------------------------------------------+
|  Step 2: Establish the Numeric Limit (e.g., <= 10 ICPs)        |
+----------------------------------------------------------------+
                               |
                               v
+----------------------------------------------------------------+
|  Step 3: Feature-First Implementation (Build functionality)    |
+----------------------------------------------------------------+
                               |
                               v
+----------------------------------------------------------------+
|  Step 4: Compute & Annotate ICPs (@ICP / metadata tags)         |
+----------------------------------------------------------------+
                               |
                               v
+----------------------------------------------------------------+
|  Step 5: Decision Check: Is ICP Total > Limit?                 |
+----------------------------------------------------------------+
              |                                    |
          (Yes: > Limit)                       (No: <= Limit)
              v                                    v
+-----------------------------+       +--------------------------+
| Step 6: Refactor Complexity |       | Step 7: Code Review & PR |
| (Extract, Encapsulate, OOP) |       | (Merge to Main Branch)   |
+-----------------------------+       +--------------------------+
              |
              +---> Loops back to Step 4 until ICP <= Limit
```

### Step 1: Define the Intrinsic Complexity Points (ICPs)
The development team convenes to agree upon the coding elements that directly impact readability and cognitive effort. Select 3 to 5 clear, easily scannable elements. Assign an objective cost (e.g., 1.0 point per branch, 1.0 point per internal class coupling, 0.5 per framework type).

### Step 2: Establish the Upper-Bound Limit
Select a strict ceiling for the maximum accumulated points a single code unit (class/file) can possess:
- **Standard Baseline**: Recommended at **10 points** (or 7 to 14 points), matching the constraints of human short-term working memory.
- The threshold must be considered a non-negotiable compilation barrier.

### Step 3: Implement Feature-First
Developers focus primarily on fulfilling functional requirements, domain rules, and unit tests. Do not pre-optimize or artificially fragment code prematurely during initial prototyping: write the feature code first to confirm correctness.

### Step 4: Compute and Annotate the ICPs
Once the code behaves as expected, calculate the ICPs across the modified or created class:
- In manual or semi-automated environments, use explicit annotations such as `@ICP(n)` at the class level, field declarations, and method declarations to maintain visibility.
- Tally all occurrences of defined branches, conditions, exception handlers, and internal couplings.

### Step 5: Evaluate Against the Threshold
Apply the core evaluation rule:
\\[\text{Calculated ICP} \le \text{Limit}\\]
- **If \\(\text{ICP} \le \text{Limit}\\)**: The code unit conforms to cognitive bounds; proceed to code review and Pull Request submission.
- **If \\(\text{ICP} > \text{Limit}\\)**: The code unit violates cognitive limits; proceed immediately to Step 6.

### Step 6: Refactor Guided by CDD
Refactor the class to distribute complexity to other coherent units:
- **Encapsulate State Operations**: Move inline state inspections and chained method calls directly into the owner domain objects to enhance cohesion.
- **Algorithmic Simplification**: Replace nested if-else structures with early returns, combined boolean conditions, or inline evaluations.
- **Polymorphism and Double Dispatch**: Replace chains of conditional type checks (`instanceof` / enum switches) with polymorphic method dispatches across classes or enumerations.
- **Delegation and Extraction**: Split auxiliary duties into focused collaborators, moving complexity out of the primary class.
- Recompute ICPs (Step 4) iteratively until all units reside comfortably within the limit.

### Step 7: Code Review, Retrospectives, and Governance
Inspect compliance during code reviews and CI/CD pipelines:
- Verify that automated tests pass, functionality is intact, and no class exceeds the limit.
- Hold recurring architectural retrospectives (e.g., during sprint planning or dedicated architecture intervals) to evaluate whether limits or ICP definitions need calibration based on production feedback.

---

## 4. Contextual Application Strategies

### A. New Projects (Greenfield) vs. Legacy Codebases

| Dimension | Greenfield Projects | Legacy Systems |
| :--- | :--- | :--- |
| **Initial Limit** | Strict, restrictive limit (**7 to 10 ICPs**, up to 14). | Relaxed, elevated limit (**20 to 40 ICPs**). |
| **Existing Code Policy** | All code units must strictly adhere to the limit from day one. | **Measure-only at first**: do not force retroactive refactorings on stable, revenue-generating legacy files. |
| **New Code Policy** | Every new class must pass the limit before review. | Newly created classes must adhere to the new standard. |
| **Transition Strategy** | Automated CI checks enforcing zero-tolerance limits. | Progressively tighten limits over time as the team builds familiarity and establishes shared patterns. |

### B. Layer-Specific Calibrations
In architectures with distinct structural boundaries (such as Clean Architecture, Hexagonal, or Layered Architectures), ICP thresholds can be calibrated to reflect the intent of each layer:
- **External / Edge Adapters (Controllers, Request Handlers)**: Enforce strict, low limits (**7 to 10 ICPs**). Their purpose is solely receiving, parsing, validating payloads, and delegating; they should contain minimal business branches.
- **Application / Use Case Layer**: Set moderate-to-higher limits (**14 to 20 ICPs**). This layer coordinates business flows, orchestrates domain entities, and invokes infrastructure services (databases, message queues, notifications).
- **Domain Entity Layer**: Set intermediate limits. While domain logic contains validations and state invariants, operations should remain tightly focused on the entity's encapsulated state.

---

## 5. Known Exceptions and Edge-Case Strategies

Field implementations in production settings demonstrate that certain code structures resist conventional ICP limits:

1. **Rich Contract Classes (DTOs / Serializers)**:
   - Data Transfer Objects interacting with established external JSON/API schemas often aggregate numerous fields and validations that cannot be subdivided without altering the wire format.
   - **Remedy**: Define specialized sub-limits (e.g., raising the limit from 10 to **20 ICPs**) specifically for contract and transfer classes.
2. **Core Domain Orchestrators**:
   - Classes implementing intricate algorithms or foundational domain engines can exceed limits despite extensive decomposition.
   - **Remedy**: Where language ecosystems allow, use language constructs like partial classes to segment distinct sub-behaviors across separate files while maintaining unified runtime contracts, or require explicit architectural justification comments in source records.

---

## 6. Observed Engineering Impacts

Empirical observation of teams adopting CDD demonstrates measurable structural shifts:
- **Controlled Class and Method Sizes**: Classes plateau at predictable size bounds (averaging ~59 lines of code), and methods naturally cluster under 24 SLOC (averaging under 7 SLOC), directly reducing bug propensity.
- **Early Elimination of God Classes**: Explicit ICP ceilings prevent classes from quietly accumulating secondary responsibilities.
- **Streamlined Unit Testing**: Bounding cyclomatic paths and internal coupling directly lowers test suite setup complexity, resulting in smaller, more maintainable test cases.