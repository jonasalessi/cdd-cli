# Coding Standards
	•	Use camelCase for declaring methods, functions, and variables, PascalCase for classes and interfaces, and kebab-case for files and directories.
	•	Avoid abbreviations, but also do not use very long names (more than 25 characters).
	•	Declare constants to represent magic numbers for readability.
	•	Methods and functions must perform a clear, well-defined action, and this should be reflected in their name, which must always start with a verb, never a noun.
	•	Whenever possible, avoid passing more than 3 parameters; prefer using objects if necessary.
	•	Avoid side effects. In general, a method or function should perform either a mutation or a query, never allow a query to have side effects (mutation).
	•	Never nest more than two if/else statements; always prefer early returns.
	•	Never use flag parameters to switch method or function behavior; in such cases, extract them into separate methods or functions with specific behavior.
	•	Avoid long methods (more than 50 lines).
	•	Avoid large classes (more than 300 lines).
	•	Always invert dependencies for external resources in both use cases and interface adapters by applying the Dependency Inversion Principle.
	•	ONLY use comments to explain WHY not HOW
	•	Use Javadoc comments to explain the class or method's purpose when the name is not enough to explain it purposefully.
	•	Never declare more than one variable on the same line.
	•	Declare variables as close as possible to where they will be used.
	•	Prefer composition over inheritance whenever possible.
 