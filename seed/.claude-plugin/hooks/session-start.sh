#!/usr/bin/env bash
# session-start.sh — Load CLI context at session start for Claude Code.
#
# Emits config, auth status, and active profile so the agent knows
# what environment it's operating in.
#
# TODO: Replace CLI_NAME, config paths, and commands for your CLI.

set -euo pipefail

CLI_NAME="${CLI_NAME:-mycli}"

# Require jq for JSON parsing
if ! command -v jq &>/dev/null; then
  exit 0
fi

# Find the CLI binary — prefer PATH, fall back to plugin's bin directory
if command -v "$CLI_NAME" &>/dev/null; then
  CLI_BIN="$CLI_NAME"
else
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  CLI_BIN="${SCRIPT_DIR}/../../bin/${CLI_NAME}"
  if [[ ! -x "$CLI_BIN" ]]; then
    cat << EOF
<hook-output>
${CLI_NAME} plugin: CLI not found.
Install: https://github.com/basecamp/${CLI_NAME}-cli#installation
</hook-output>
EOF
    exit 0
  fi
fi

# Get CLI version
cli_version=$("$CLI_BIN" --version 2>/dev/null | awk '{print $NF}' || true)

# Check if we have configuration
config_output=$("$CLI_BIN" config show --json 2>/dev/null || echo '{}')
has_config=$(echo "$config_output" | jq -r '.data // empty' 2>/dev/null)

if [[ -z "$has_config" ]] || [[ "$has_config" == "{}" ]]; then
  exit 0
fi

# Build context message
context="${CLI_NAME} context loaded:"

if [[ -n "$cli_version" ]]; then
  context+="\n  CLI: v${cli_version}"
fi

# Show active profile if using named profiles
active_profile=$("$CLI_BIN" profile show --json 2>/dev/null | jq -r '.data.name // empty' 2>/dev/null || true)
if [[ -n "$active_profile" ]]; then
  context+="\n  Profile: $active_profile"
fi

# Check if authenticated
auth_status=$("$CLI_BIN" auth status --json 2>/dev/null || echo '{}')
is_auth=$(echo "$auth_status" | jq -r '.data.authenticated // false')

if [[ "$is_auth" != "true" ]]; then
  context+="\n  Auth: Not authenticated (run: ${CLI_NAME} auth login)"
fi

cat << EOF
<hook-output>
$(echo -e "$context")

Use \`${CLI_NAME}\` commands to interact with the API:
  ${CLI_NAME} auth login          # Authenticate
  ${CLI_NAME} auth status         # Check auth status
  ${CLI_NAME} doctor              # Diagnose configuration
</hook-output>
EOF
