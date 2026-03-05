#!/usr/bin/env bash
# post-commit-check.sh - Check for {{.Name}} item references after git commits
#
# This hook runs after Bash tool use and checks if a git commit was made
# that references a {{.Name}} item ({{.Upper}}-12345, item-12345, etc.)

set -euo pipefail

# Require jq for JSON parsing
if ! command -v jq &>/dev/null; then
  exit 0
fi

# Read tool input from stdin (JSON with tool_name, tool_input, tool_output)
input=$(cat)

# Extract tool input (the bash command that was run)
tool_input=$(echo "$input" | jq -r '.tool_input.command // empty' 2>/dev/null)

# Only process git commit commands
if [[ ! "$tool_input" =~ ^git\ commit ]]; then
  exit 0
fi

# Check if commit succeeded by looking for output patterns
tool_output=$(echo "$input" | jq -r '.tool_output // empty' 2>/dev/null)

# Skip if commit failed — detect error indicators before checking for success.
# Only match lines that look like git/hook errors, not commit subject lines
# (e.g. "[branch abc1234] Fix failed login" should not trigger this guard).
# We strip the "[branch hash] subject" success line before scanning for errors.
filtered_output=$(echo "$tool_output" | grep -v '^\[.*[a-f0-9]\{7,\}\]' || true)
if echo "$filtered_output" | grep -qiE '(^|[[:space:]])(error|fatal|aborted|rejected)[[:space:]:]|hook[[:space:]].*[[:space:]]failed|pre-commit[[:space:]].*[[:space:]]failed|^error:'; then
  exit 0
fi

# Verify commit actually succeeded - look for commit hash pattern or "create mode"
if [[ ! "$tool_output" =~ \[.*[a-f0-9]{7,}\] ]] && [[ ! "$tool_output" =~ "create mode" ]]; then
  exit 0
fi

# Look for item references in the commit message or branch name
branch=$(git branch --show-current 2>/dev/null || true)
last_commit_msg=$(git log -1 --format=%s 2>/dev/null || true)

# Patterns: {{.Upper}}-12345, item-12345, {{.Name}}-12345
todo_patterns='{{.Upper}}-[0-9]+|item-[0-9]+|{{.Name}}-[0-9]+'

found_in_branch=$(echo "$branch" | grep -oEi "$todo_patterns" | head -1 || true)
found_in_msg=$(echo "$last_commit_msg" | grep -oEi "$todo_patterns" | head -1 || true)

if [[ -n "$found_in_branch" ]] || [[ -n "$found_in_msg" ]]; then
  ref="${found_in_msg:-$found_in_branch}"
  # Extract just the number
  item_id=$(echo "$ref" | grep -oE '[0-9]+')

  short_sha=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
  comment="Commit ${short_sha}: ${last_commit_msg}"
  escaped_comment=$(printf '%q' "$comment")

  cat << EOF
<hook-output>
Detected {{.Name}} item reference: $ref

To link this commit to {{.Name}}:
  {{.Name}} comment ${escaped_comment} --on $item_id

Or complete the item:
  {{.Name}} done $item_id
</hook-output>
EOF
fi
