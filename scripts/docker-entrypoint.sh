#!/bin/sh
set -eu

UPLOADS_DIR="${UPLOADS_DIR:-/data/uploads}"
SEED_DIR="/app/seed-uploads"

mkdir -p "$UPLOADS_DIR/seed"

if [ -d "$SEED_DIR" ]; then
  for file in "$SEED_DIR"/*; do
    [ -f "$file" ] || continue
    target="$UPLOADS_DIR/seed/$(basename "$file")"
    if [ ! -f "$target" ]; then
      cp "$file" "$target"
    fi
  done
fi

exec "$@"
