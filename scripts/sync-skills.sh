#!/usr/bin/env bash
# sync-skills.sh — Publish this repo's skills (skills/rubric-audit) to basecamp/skills.
#
# The implementation is the seed's, seed/scripts/sync-skills.sh, run as it is: one
# script, exercised here before any CLI inherits it. Only the identity differs —
# this repo is basecamp/cli, so the publishing source is `cli`, not `<name>-cli`.
# Knobs and env vars are documented in the seed script's header.

set -euo pipefail

export SYNC_SOURCE="${SYNC_SOURCE:-cli}"
exec "$(dirname "${BASH_SOURCE[0]}")/../seed/scripts/sync-skills.sh" "$@"
