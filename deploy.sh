#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cd "$ROOT_DIR"

echo "[deploy] git pull"
git pull

echo "[deploy] docker compose up --build"
docker compose -f docker/compose.central.yml up -d --build

echo "[deploy] restart nginx"
docker compose -f docker/compose.central.yml restart nginx

echo "[deploy] done"
