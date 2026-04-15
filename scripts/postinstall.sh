#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
SOURCE_DIR="$ROOT_DIR/.agents/skills"
TARGET_DIR="$ROOT_DIR/.claude/skills"

if [ ! -d "$SOURCE_DIR" ]; then
  echo "No .agents/skills directory found, skipping symlink."
  exit 0
fi

mkdir -p "$TARGET_DIR"

for skill in "$SOURCE_DIR"/*/; do
  skill_name="$(basename "$skill")"
  target="$TARGET_DIR/$skill_name"

  if [ -L "$target" ]; then
    rm "$target"
  fi

  ln -s "$skill" "$target"
done

echo "Linked $(ls -1d "$SOURCE_DIR"/*/ | wc -l | tr -d ' ') skills from .agents/skills → .claude/skills"
