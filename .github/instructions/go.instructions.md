---
applyTo: "**/*.go,go.mod,go.sum"
---

- Prefer small, package-local changes and reuse patterns from neighboring files before adding new helpers or dependencies.
- Preserve exported names, Cobra flag names, YAML keys, and Helm/value-merge behavior unless the task explicitly changes them.
- Use `filepath` helpers and Windows-safe cleanup for filesystem logic.
- If behavior changes, add or update tests in the nearest relevant package.
