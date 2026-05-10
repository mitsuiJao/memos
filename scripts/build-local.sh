#!/usr/bin/env sh
# Build the memos Docker image from source without buildx.
# Usage: ./scripts/build-local.sh [--tag TAG]
set -e

TAG="${1:-memos-local}"

DOCKER_BUILDKIT=0 docker build \
  --network=host \
  -f "$(dirname "$0")/../Dockerfile.standalone" \
  -t "$TAG" \
  "$(dirname "$0")/.."

echo ""
echo "Image built: $TAG"
echo "Start with:  docker compose up -d"
