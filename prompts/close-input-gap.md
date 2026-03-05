# Close an Input Convention Gap

You are closing a specific gap in a Go CLI's compliance with the 37signals CLI input conventions.

## Input

- **Criterion ID**: e.g., "I2" (short flag for primary text)
- **Command**: e.g., "todo add", "journal write"
- **CLI repo**: The repository you're working in

## Process

1. Read INPUT-CONVENTIONS.md to understand the criterion
2. Read the command's current implementation
3. Identify what's missing (positional? stdin? short flag? conflict check?)
4. Implement the minimum change
5. Add tests covering each input path and the conflict case
6. Verify the change doesn't break the surface test

## Criterion Reference

| ID | What to implement |
|----|-------------------|
| I1 | Named flag with semantic name (`--title`, `--message`, `--content`) |
| I2 | Short flag (`-t`, `-m`, `-c`) — use `StringVarP` instead of `StringVar` |
| I3 | Positional shorthand — add `Args: cobra.MaximumNArgs(1)`, update `Use:`, resolve in `run()` |
| I4 | Stdin — check `!stdinIsTerminal()`, call `readStdin()` |
| I5 | `$EDITOR` fallback — call `editor.Open("")` when stdin is a terminal and no text given |
| I6 | Flag/positional conflict — error when both flag and positional provide the same field |
| I7 | Missing-text hint — `ErrUsageHint` showing both positional and flag forms |

## Implementation Pattern

See the "Implementation Template" section in INPUT-CONVENTIONS.md for the standard `run()` pattern.

## Test Pattern

For each command, test:

- Positional arg works: `app foo "text" --json`
- Short flag works: `app foo -t "text" --json`
- Long flag works: `app foo --title "text" --json`
- Conflict errors: `app foo --title "X" "Y"` → "mutually exclusive"
- Empty errors: `app foo` → "required" with hint
- Stdin works: pipe content, verify it's used

Use `httptest.NewServer` to mock the API. Set `HEY_TOKEN` / `APP_TOKEN` env var for auth. Use `--base-url` to point at the test server.
