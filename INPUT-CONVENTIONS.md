# CLI Text Input Conventions

Rules for how 37signals CLI commands accept their primary text input. Companion to [RUBRIC.md](RUBRIC.md) — the rubric covers structural contract (output envelope, exit codes, discovery); this document covers how commands receive content from humans and agents.

These conventions apply to all **content-creation commands** — commands whose primary purpose is to create or send text: adding a todo, writing a journal entry, replying to a thread, composing a message.

---

## The Resolution Chain

When a command needs text input (a title, message body, content), resolve from these sources in order. First non-empty value wins:

1. **Named flag** (`--title`, `--content`, `--message` / `-m`)
2. **Positional argument** (trailing arg after any required positional IDs)
3. **Stdin** (when piped — i.e., stdin is not a terminal)
4. **$EDITOR** (when interactive and the command supports multi-line input)

If both a flag and a positional arg provide the same value, **error** — the command must not silently pick one over the other:

```
Error: --title and positional argument are mutually exclusive
```

---

## Convention 1: Positional Shorthand

If a creation/send command has exactly one "primary text" input, accept it as a trailing positional arg. The flag form remains canonical; the positional form is a shorthand.

```bash
# Fluent (positional)
app todo add "Buy milk"
app journal write "Today was great"

# Canonical (flag)
app todo add --title "Buy milk"
app journal write --content "Today was great"
```

### When to offer positional shorthand

- The command has at most one "text" arg
- No ambiguity with other positional args (or disambiguation is trivial — e.g., YYYY-MM-DD is a date, anything else is content)

### When NOT to offer positional shorthand

- The command already uses its positional slot(s) for required identifiers AND adding text creates parsing ambiguity
- The command requires multiple text inputs (e.g., `compose` needs both `--subject` and `--message`)
- The command's required positional (like a topic ID) and the text arg can't be disambiguated by format

---

## Convention 2: Stdin as Implicit Content

All content-creation commands read stdin when it's a pipe and no explicit text was given via flag or positional. This enables Unix pipeline composition:

```bash
echo "Buy milk" | app todo add
cat notes.md | app journal write
pbpaste | app reply 123
```

Stdin resolution sits at position 3 in the chain — after flags and positional args, before `$EDITOR`.

### When to offer stdin

Always, for any command that accepts a text body or content. Even short-label commands like `todo add` benefit — it enables scripting.

### When NOT to offer stdin

Only if the command has no text input at all (e.g., `todo complete <id>`).

---

## Convention 3: Short Flags

Every primary text flag gets a one-letter shorthand. Pick the letter that matches the semantic:

| Semantic | Long flag | Short | Mnemonic |
|----------|-----------|-------|----------|
| Short label/title | `--title` | `-t` | **t**itle |
| Message body | `--message` | `-m` | **m**essage |
| General content | `--content` | `-c` | **c**ontent |

Don't normalize everything to `--message` — a todo title is not a message. Pick the name that matches the role.

---

## Disambiguation Patterns

When a positional arg could be either a date or content (e.g., `journal write`), disambiguate by format:

```go
func isDateArg(s string) bool {
    _, err := time.Parse("2006-01-02", s)
    return err == nil
}
```

YYYY-MM-DD parses as a date; anything else is content. These formats are disjoint — no ambiguity.

For two-positional commands (`journal write 2024-01-15 "Content"`), accept `MaximumNArgs(2)` and slot the first as date-if-parseable, second as content.

---

## Error Messages

### Missing text

When no text arrives from any source, hint both forms:

```
Error: title is required
Hint:  app todo add "Buy milk"  or  app todo add --title "Buy milk"
```

### Flag/positional conflict

When both a flag and positional supply the same field:

```
Error: --title and positional argument are mutually exclusive
```

### Empty stdin

When stdin is a pipe but empty:

```
Error: no content provided (use --content to provide inline, or pipe to stdin)
```

---

## Implementation Template

Standard pattern for a command with positional + flag + stdin text input:

```go
func newFooCommand() *fooCommand {
    c := &fooCommand{}
    c.cmd = &cobra.Command{
        Use:  "foo [text]",
        RunE: c.run,
        Args: cobra.MaximumNArgs(1),
    }
    c.cmd.Flags().StringVarP(&c.text, "text", "t", "", "The text")
    return c
}

func (c *fooCommand) run(cmd *cobra.Command, args []string) error {
    text := c.text

    // 1. Conflict check
    if text != "" && len(args) > 0 {
        return ErrUsage("--text and positional argument are mutually exclusive")
    }

    // 2. Positional
    if text == "" && len(args) > 0 {
        text = args[0]
    }

    // 3. Stdin
    if text == "" && !stdinIsTerminal() {
        var err error
        text, err = readStdin()
        if err != nil {
            return err
        }
    }

    // 4. $EDITOR (optional, for multi-line content)
    if text == "" && stdinIsTerminal() {
        var err error
        text, err = editor.Open("")
        if err != nil {
            return err
        }
    }

    // 5. Nothing
    if text == "" {
        return ErrUsageHint("text is required",
            "app foo \"hello\"  or  app foo --text \"hello\"")
    }

    // ... proceed with text
}
```

---

## Conformance Audit

Use this table to audit content-creation commands across all 37signals CLIs. Each command should support all applicable input sources.

### Audit criteria

| ID | Criterion | Applies to |
|----|-----------|-----------|
| I1 | Named flag with semantic name (`--title`, `--message`, `--content`) | All content commands |
| I2 | Short flag (`-t`, `-m`, `-c`) | All content commands |
| I3 | Positional shorthand (when unambiguous) | Commands with a single text input |
| I4 | Stdin | All content commands |
| I5 | `$EDITOR` fallback | Commands accepting multi-line input |
| I6 | Flag/positional conflict error | Commands offering positional shorthand |
| I7 | Missing-text error with hint showing both forms | All content commands |

### Current status

#### hey-cli

| Command | Text field | I1 | I2 | I3 | I4 | I5 | I6 | I7 |
|---------|-----------|----|----|----|----|----|----|-----|
| `todo add` | title | `--title` | `-t` | `"text"` | pipe | — | yes | yes |
| `journal write` | content | `--content` | `-c` | `"text"` | pipe | `$EDITOR` | yes | yes |
| `reply` | message | `--message` | `-m` | — (slot used by topic-id) | pipe | `$EDITOR` | — | yes |
| `compose` | message | `--message` | `-m` | — (multiple required flags) | pipe | `$EDITOR` | — | yes |

#### basecamp-cli

_Audit pending._

#### fizzy-cli

_Audit pending._

---

## Adding to the Rubric

These conventions are candidates for a future rubric criterion under Tier 1 (Agent Contract) or Tier 4 (Developer Experience). The audit table above tracks conformance until then. Proposed criterion:

> **1A.11 Text input resolution chain**: Content-creation commands accept their primary text via named flag, positional shorthand (when unambiguous), stdin, and `$EDITOR` (when applicable), in that priority order. Flag and positional conflict is an error.

---

## References

- [RUBRIC.md](RUBRIC.md) — structural contract (output, exit codes, discovery)
- [MAKEFILE-CONVENTION.md](MAKEFILE-CONVENTION.md) — build targets
- `prompts/close-gap.md` — agent prompt for closing rubric gaps
