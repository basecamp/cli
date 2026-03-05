#!/usr/bin/env bash
# session-start.sh - {{.Name}} plugin liveness check
#
# Lightweight: one subprocess call. Context priming happens on first
# use via the /{{.Name}} skill, not here.

set -euo pipefail

CLI="{{.Name}}"

if ! command -v "$CLI" &>/dev/null; then
  cat << EOF
<hook-output>
$CLI plugin active — CLI not found on PATH.
Install: https://github.com/basecamp/${CLI}-cli#installation
</hook-output>
EOF
  exit 0
fi

if ! command -v jq &>/dev/null; then
  cat << EOF
<hook-output>
$CLI plugin active.
</hook-output>
EOF
  exit 0
fi

auth_json=$("$CLI" auth status --json 2>/dev/null || echo '{}')

is_auth=false
if parsed_auth=$(echo "$auth_json" | jq -er '.data.authenticated' 2>/dev/null); then
  is_auth="$parsed_auth"
fi

if [[ "$is_auth" == "true" ]]; then
  cat << EOF
<hook-output>
$CLI plugin active.
</hook-output>
EOF
else
  cat << EOF
<hook-output>
$CLI plugin active — not authenticated.
Run: $CLI auth login
</hook-output>
EOF
fi
