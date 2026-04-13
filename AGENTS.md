# atdtool agent guide

This is the shared, low-noise instruction file for AI coding agents in this repository. Keep it short, and move task-specific workflows into canonical skills under `.agents/skills/` instead of growing this file.

## Repository map

- `cmd/atdtool/`: CLI commands, template rendering, and command wiring.
- `cli/values/`: path normalization and `--set` parsing.
- `internal/pkg/util/`: Helm/chart value merge logic.
- `internal/pkg/noncloudnative/`: `deploy.yaml` loading and render-time values.
- `pkg/`: reusable libraries and package-level tests.
- `docs/`: usage, structure, and reference documentation.

## Working style

- Explore first. For multi-file or unclear tasks, inspect the relevant package and outline a short plan before editing.
- Keep diffs small. Preserve existing CLI behavior, merge semantics, and file layout unless the task explicitly changes them.
- Prefer local patterns from neighboring files over new abstractions or new dependencies.
- Update tests for every behavior change. Update docs when CLI behavior, merge precedence, template runtime values, or documented layout expectations change.
- Do not reformat or rewrite unrelated files.

## Go code

- Keep functions focused and error messages actionable.
- Preserve public flags, YAML field names, and output filenames unless the change requires them.
- Use `filepath` helpers and platform-neutral paths for filesystem work.

## Testing and verification

- Prefer focused verification first: run the smallest relevant package or test case before broader runs.
- Add or adjust tests near the changed code. Prefer table-driven tests for precedence and parser permutations.
- Use `t.TempDir()`, `filepath.Join`, and deterministic fixtures. Avoid hard-coded POSIX temp paths and flaky timing assumptions.
- Close files before cleanup; Windows is strict about open handles.
- Final repo-wide verification command: `go test ./...`

## Documentation

- Document `atdtool`'s actual behavior and generic input layout.
- Do not rely on external sample repositories when updating docs.

## Context discipline

- Keep always-on instructions minimal.
- Read extra docs or references only when they are directly relevant to the current task.
