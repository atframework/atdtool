Follow `AGENTS.md` at the repository root as the canonical shared ruleset for this repository.

Keep this file intentionally short to save context. Put task-specific workflows in `.agents/skills/` and file-specific rules in `.github/instructions/` instead of expanding repository-wide instructions.

When behavior changes, update nearby tests, prefer focused `go test` runs first, and use `go test ./...` for final repo-wide verification when shared logic or cross-package behavior is affected.
