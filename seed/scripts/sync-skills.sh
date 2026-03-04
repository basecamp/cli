#!/usr/bin/env bash
# sync-skills.sh — Sync embedded skills to basecamp/skills distribution repo.
#
# Run from CI on release (tag push). Copies skills/*/SKILL.md to the
# basecamp/skills repo, commits, and pushes.
#
# Required env vars:
#   SKILLS_TOKEN   — GitHub token with push access to basecamp/skills
#   RELEASE_TAG    — The release tag (e.g., v1.2.3)
#   SOURCE_SHA     — The source commit SHA
#
# Optional env vars:
#   DRY_RUN        — "local" to skip push, "remote" to skip commit+push
#
# TODO: Replace CLI_NAME with your CLI name.

set -euo pipefail

CLI_NAME="${CLI_NAME:-mycli}"
SKILLS_REPO="basecamp/skills"
SKILLS_DIR="skills"
MANAGED_MANIFEST=".managed-skills"

: "${SKILLS_TOKEN:?SKILLS_TOKEN is required}"
: "${RELEASE_TAG:?RELEASE_TAG is required}"
: "${SOURCE_SHA:?SOURCE_SHA is required}"

DRY_RUN="${DRY_RUN:-}"

# Clone the skills repo
WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

echo "Cloning ${SKILLS_REPO}..."
git clone "https://x-access-token:${SKILLS_TOKEN}@github.com/${SKILLS_REPO}.git" "$WORK_DIR/skills-repo" 2>/dev/null

TARGET_DIR="${WORK_DIR}/skills-repo"

# Collect skill directories from source
SKILL_DIRS=()
for skill_dir in ${SKILLS_DIR}/*/; do
  if [[ -f "${skill_dir}/SKILL.md" ]]; then
    SKILL_DIRS+=("$skill_dir")
  fi
done

if [[ ${#SKILL_DIRS[@]} -eq 0 ]]; then
  echo "No skills found in ${SKILLS_DIR}/"
  exit 0
fi

echo "Found ${#SKILL_DIRS[@]} skill(s) to sync"

# Copy skills to target repo
MANAGED_SKILLS=()
for skill_dir in "${SKILL_DIRS[@]}"; do
  skill_name=$(basename "$skill_dir")
  dest="${TARGET_DIR}/${CLI_NAME}/${skill_name}"

  echo "  Syncing ${skill_name}..."
  mkdir -p "$dest"

  # Copy non-Go, non-dotfiles preserving subdirectory structure
  (cd "$skill_dir" && find . -type f ! -name '*.go' ! -name '.*' | while read -r f; do
    mkdir -p "$dest/$(dirname "$f")"
    cp "$f" "$dest/$f"
  done)

  MANAGED_SKILLS+=("${CLI_NAME}/${skill_name}")
done

# Update managed manifest
MANIFEST_PATH="${TARGET_DIR}/${MANAGED_MANIFEST}"
if [[ -f "$MANIFEST_PATH" ]]; then
  # Remove stale entries for this CLI
  grep -v "^${CLI_NAME}/" "$MANIFEST_PATH" > "${MANIFEST_PATH}.tmp" || true
  mv "${MANIFEST_PATH}.tmp" "$MANIFEST_PATH"
fi

# Append current skills
for skill in "${MANAGED_SKILLS[@]}"; do
  echo "$skill" >> "$MANIFEST_PATH"
done
sort -u -o "$MANIFEST_PATH" "$MANIFEST_PATH"

# Check for stale skills to remove
if [[ -d "${TARGET_DIR}/${CLI_NAME}" ]]; then
  for existing in "${TARGET_DIR}/${CLI_NAME}"/*/; do
    existing_name=$(basename "$existing")
    found=false
    for skill_dir in "${SKILL_DIRS[@]}"; do
      if [[ "$(basename "$skill_dir")" == "$existing_name" ]]; then
        found=true
        break
      fi
    done
    if [[ "$found" == "false" ]]; then
      echo "  Removing stale skill: ${existing_name}"
      rm -rf "$existing"
    fi
  done
fi

if [[ "$DRY_RUN" == "remote" ]]; then
  echo "DRY_RUN=remote: skipping commit and push"
  exit 0
fi

# Commit and push
cd "$TARGET_DIR"
git add -A

if git diff --cached --quiet; then
  echo "No changes to commit"
  exit 0
fi

git config user.name "${CLI_NAME}-cli[bot]"
git config user.email "${CLI_NAME}-cli[bot]@users.noreply.github.com"

git commit -m "$(cat <<EOF
Sync ${CLI_NAME} skills from ${RELEASE_TAG}

Source: ${SOURCE_SHA}
EOF
)"

if [[ "$DRY_RUN" == "local" ]]; then
  echo "DRY_RUN=local: skipping push"
  exit 0
fi

echo "Pushing to ${SKILLS_REPO}..."
if ! git push origin main; then
  echo "Push failed, retrying after pull..."
  git pull --rebase origin main
  git push origin main
fi

echo "Skills synced successfully"
