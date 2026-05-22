#!/usr/bin/env sh
# Install git hooks from scripts/hooks/ into .git/hooks/.
# Run once after cloning: ./scripts/setup-hooks.sh
set -e

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
HOOKS_SRC="$REPO_ROOT/scripts/hooks"
HOOKS_DST="$REPO_ROOT/.git/hooks"

for hook in "$HOOKS_SRC"/*; do
  name="$(basename "$hook")"
  dst="$HOOKS_DST/$name"

  if [ -e "$dst" ] && [ ! -L "$dst" ]; then
    echo "Skipping $name: $dst already exists and is not a symlink"
    continue
  fi

  ln -sf "$hook" "$dst"
  chmod +x "$hook"
  echo "Installed: $name -> $dst"
done

echo "Hooks installed."
