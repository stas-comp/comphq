#!/usr/bin/env bash
# Layer 4 container tests (SPEC B7, PLAN.md P1-06). Linux + Docker only;
# runs in CI, not on the Windows build machine.
#
# Proves gates 1.38 (Compose file validates, healthy with only the
# documented edits), 1.39 (restart and full down/up, healthy <= 60s, data
# intact) and 1.40 (remove container and image, recreate with the same
# folder, data intact).
#
# There's no HTTP endpoint that writes data yet (the People section lands
# in P1-13), so "data intact" here means: a marker file written directly
# into the bind-mounted /data folder, plus a whole-folder checksum, both
# survive every restart/recreate. A later task can extend this script to
# also create a person over HTTP once that's possible (docs/decisions.md).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WORK="$(mktemp -d)"
DATA_DIR="$WORK/data"
COMPOSE_FILE="$WORK/truenas.yaml"
PROJECT="comphq-container-test"
IMAGE_REF="ghcr.io/stas-comp/comphq:0.0.1"
MARKER="$DATA_DIR/container-test-marker.txt"

log() { echo "[container-test] $*"; }

cleanup() {
  log "cleaning up"
  docker compose -f "$COMPOSE_FILE" -p "$PROJECT" down --remove-orphans >/dev/null 2>&1 || true
  rm -rf "$WORK"
}
trap cleanup EXIT

mkdir -p "$DATA_DIR"
sudo chown -R 568:568 "$DATA_DIR"

# Substitute the three owner-editable lines with test values.
sed \
  -e "s#/mnt/YOUR-POOL/comphq:/data#${DATA_DIR}:/data#" \
  -e 's#TZ: "Europe/London"#TZ: "Europe/London"#' \
  "$ROOT/deploy/truenas.yaml" > "$COMPOSE_FILE"

log "building the image and tagging it as the Compose file's reference"
docker build --build-arg VERSION=0.0.1-container-test -t "$IMAGE_REF" "$ROOT"

wait_healthy() {
  local deadline=$((SECONDS + 60))
  while [ "$SECONDS" -lt "$deadline" ]; do
    status="$(docker inspect --format '{{.State.Health.Status}}' "${PROJECT}-comphq-1" 2>/dev/null || echo "starting")"
    if [ "$status" = "healthy" ]; then
      return 0
    fi
    sleep 2
  done
  log "container never became healthy; last status: ${status:-unknown}"
  docker compose -f "$COMPOSE_FILE" -p "$PROJECT" logs || true
  return 1
}

checksum_data_dir() {
  find "$DATA_DIR" -type f -print0 | sort -z | xargs -0 sha256sum | sha256sum | awk '{print $1}'
}

log "1. docker compose config validates"
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" config >/dev/null

log "2. up -d becomes healthy within 60s"
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" up -d
wait_healthy

log "3. restart: writing a marker and checksumming the data folder"
echo "comphq container test $(date -u +%FT%TZ)" > "$MARKER"
sudo chown 568:568 "$MARKER"
CHECKSUM_BEFORE="$(checksum_data_dir)"

docker compose -f "$COMPOSE_FILE" -p "$PROJECT" restart
wait_healthy
if [ ! -f "$MARKER" ] || [ "$(checksum_data_dir)" != "$CHECKSUM_BEFORE" ]; then
  log "data changed after 'restart'"
  exit 1
fi

docker compose -f "$COMPOSE_FILE" -p "$PROJECT" down
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" up -d
wait_healthy
if [ ! -f "$MARKER" ] || [ "$(checksum_data_dir)" != "$CHECKSUM_BEFORE" ]; then
  log "data changed after 'down' + 'up'"
  exit 1
fi

log "4. recreate: remove the container and the image, then bring it back up"
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" down
docker image rm "$IMAGE_REF"
docker build --build-arg VERSION=0.0.1-container-test -t "$IMAGE_REF" "$ROOT"
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" up -d
wait_healthy
if [ ! -f "$MARKER" ] || [ "$(checksum_data_dir)" != "$CHECKSUM_BEFORE" ]; then
  log "data changed after remove-and-recreate"
  exit 1
fi

log "all container tests passed"
