---
name: docs-sync-after-code-change
description: Synchronize documentation after code changes in this repository. Use when implementation changes affect commands, values precedence, template runtime behavior, project structure docs, or examples.
---

# Docs sync after code change

1. Identify the user-visible behavior that changed before editing docs.
2. Update `README.md` as the entry point when commands, workflows, or doc links changed.
3. Update the narrowest matching doc pages under `docs/usage/`, `docs/reference/`, or `docs/structure/`.
4. Keep examples generic and consistent with current code and tests; do not rely on external sample repositories.
5. Prefer describing real behavior over aspirational behavior. When unsure, confirm from code or tests first.
6. If code and docs disagree, fix both in the same task or leave a clear follow-up note.
7. Check edited Markdown for diagnostics before finishing.
