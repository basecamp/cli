#!/usr/bin/env bash
# session-start.sh - {{.Name}} plugin liveness check
#
# Lightweight: one subprocess call. Context priming happens on first
# use via the /{{.Name}} skill, not here.

set -euo pipefail

if ! command -v {{.Name}} &>/dev/null; then
  cat << 'EOF'
<hook-output>
{{.Name}} plugin active — CLI not found on PATH.
Install: https://github.com/basecamp/{{.Name}}-cli#installation
</hook-output>
EOF
  exit 0
fi

auth_json=$({{.Name}} auth status --json 2>/dev/null || echo '{}')

if ! command -v jq &>/dev/null; then
  cat << 'EOF'
<hook-output>
{{.Name}} plugin active.
</hook-output>
EOF
  exit 0
fi

is_auth=false
if parsed_auth=$(echo "$auth_json" | jq -er '.data.authenticated' 2>/dev/null); then
  is_auth="$parsed_auth"
fi

if [[ "$is_auth" == "true" ]]; then
  cat << 'EOF'
<hook-output>
{{.Name}} plugin active.
</hook-output>
EOF
else
  cat << 'EOF'
<hook-output>
{{.Name}} plugin active — not authenticated.
Run: {{.Name}} auth login
</hook-output>
EOF
fi
