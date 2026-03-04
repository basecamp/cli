---
name: context-linker
description: |
  Automatically link code changes to {{.Name}} items.
  Use when: committing code, creating PRs, resolving issues.
  Detects item IDs from branch names, commit messages, and PR descriptions.
---

# Context Linker Agent

Connect code changes to {{.Name}} items.

## Detection Patterns

Look for references in:

1. **Branch names**: `feature/todo-12345-description`, `fix/12345-bug`
2. **Commit messages**: `[#12345] Fix bug`, `Fixes #12345`
3. **PR descriptions**: `Closes #12345`, `Related: <url>`

## Workflow: On Commit

1. Extract item ID from branch name:
   ```bash
   BRANCH=$(git branch --show-current)
   ITEM_ID=$(echo "$BRANCH" | grep -oE '[0-9]+' | head -1)
   ```

2. If found, offer to link:
   ```bash
   COMMIT=$(git rev-parse --short HEAD)
   MSG=$(git log -1 --format=%s)
   # Add comment or update linked item
   ```

## Workflow: On PR Creation

1. Check branch name and PR description for item references
2. For each referenced item, add PR link
3. Offer to update item status when PR is merged
