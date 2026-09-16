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
  [ -f "$WORK/upgrade-prev.yaml" ] &&
    docker compose -f "$WORK/upgrade-prev.yaml" -p comphq-upgrade-test down --remove-orphans >/dev/null 2>&1 || true
  [ -f "$WORK/upgrade-current.yaml" ] &&
    docker compose -f "$WORK/upgrade-current.yaml" -p comphq-upgrade-test down --remove-orphans >/dev/null 2>&1 || true
  [ -d "$WORK/prev-worktree" ] &&
    git -C "$ROOT" worktree remove --force "$WORK/prev-worktree" >/dev/null 2>&1 || true
  # The container writes as uid 568; this shell isn't, so removing its
  # files needs the same escalation their creation did.
  sudo rm -rf "$WORK"
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
  local container="${1:-${PROJECT}-comphq-1}"
  local deadline=$((SECONDS + 60))
  while [ "$SECONDS" -lt "$deadline" ]; do
    status="$(docker inspect --format '{{.State.Health.Status}}' "$container" 2>/dev/null || echo "starting")"
    if [ "$status" = "healthy" ]; then
      return 0
    fi
    sleep 2
  done
  log "$container never became healthy; last status: ${status:-unknown}"
  docker logs "$container" 2>&1 | tail -50 || true
  return 1
}

checksum_data_dir() {
  # Exclude SQLite's WAL and shared-memory index files: they're working
  # files, not data, and their bytes aren't stable across a clean restart
  # even when the actual rows are unchanged.
  find "$DATA_DIR" -type f ! -name '*-wal' ! -name '*-shm' -print0 |
    sort -z | xargs -0 sha256sum | sha256sum | awk '{print $1}'
}

# run_upgrade_rollback_test proves gate 1.42: upgrade from the previous
# release to the current build, then roll back, without losing data.
# PREV is the latest existing release tag (SPEC P1-12: "the highest
# vX.Y.Z tag lower than the version in deploy/truenas.yaml, or the latest
# tag if HEAD is untagged" — during ordinary development HEAD is always
# untagged, so this is simply the latest tag). "Rule from here on": this
# sub-test is mandatory from P1-12 onward and never removed, even though
# it isn't meaningful until v0.0.2 (P1-28) gives it real data to carry
# across an upgrade, and gate 1.42 itself is only reported passed from
# the v0.1.0 release run.
run_upgrade_rollback_test() {
  local current_image="$1"
  local prev_tag
  prev_tag="$(git -C "$ROOT" tag -l 'v*.*.*' | sort -V | tail -1)"
  if [ -z "$prev_tag" ]; then
    log "  skipped: no released tag exists yet"
    return 0
  fi
  log "  previous release: $prev_tag"

  local prev_worktree="$WORK/prev-worktree"
  git -C "$ROOT" worktree add --detach "$prev_worktree" "$prev_tag" >/dev/null

  local prev_image="ghcr.io/stas-comp/comphq:prev-test"
  docker build --build-arg VERSION="${prev_tag#v}" -t "$prev_image" "$prev_worktree" >/dev/null

  local up_data="$WORK/upgrade-data"
  mkdir -p "$up_data"
  sudo chown -R 568:568 "$up_data"
  local up_project="comphq-upgrade-test"
  local up_compose_prev="$WORK/upgrade-prev.yaml"
  local up_compose_current="$WORK/upgrade-current.yaml"
  sed -e "s#/mnt/YOUR-POOL/comphq:/data#${up_data}:/data#" -e "s#image: .*#image: ${prev_image}#" \
    "$prev_worktree/deploy/truenas.yaml" >"$up_compose_prev"
  sed -e "s#/mnt/YOUR-POOL/comphq:/data#${up_data}:/data#" -e "s#image: .*#image: ${current_image}#" \
    "$ROOT/deploy/truenas.yaml" >"$up_compose_current"

  # PREV's own smoke tests: an older tag may not have any yet (true for
  # v0.0.1, which predates this harness) — that's expected, not a failure.
  run_prev_smoke() {
    local grep_pattern="$1"
    if ! compgen -G "$prev_worktree/e2e/smoke/*.spec.ts" >/dev/null; then
      log "  $prev_tag has no smoke tests yet; skipping smoke assertions for this phase"
      return 0
    fi
    (
      cd "$prev_worktree"
      npm ci >/dev/null 2>&1
      BASE_URL="http://127.0.0.1:8080" npx playwright test --project=smoke --grep "$grep_pattern"
    )
  }

  # "@verify" as a plain substring would also match "@verify-prev"; the
  # negative lookahead keeps the two tags independently selectable, exactly
  # as PLAN.md's "@seed, @verify, @verify-prev" scheme intends.
  local seed_and_verify='@seed|@verify(?!-prev)'
  local verify_only='@verify(?!-prev)'

  log "  phase 1: $prev_tag, seed and verify"
  docker compose -f "$up_compose_prev" -p "$up_project" up -d
  wait_healthy "${up_project}-comphq-1"
  run_prev_smoke "$seed_and_verify"
  docker compose -f "$up_compose_prev" -p "$up_project" down

  log "  phase 2: current build, verify-prev then seed and verify"
  docker compose -f "$up_compose_current" -p "$up_project" up -d
  wait_healthy "${up_project}-comphq-1"
  BASE_URL="http://127.0.0.1:8080" npx playwright test --project=smoke --grep "@verify-prev"
  BASE_URL="http://127.0.0.1:8080" npx playwright test --project=smoke --grep "$seed_and_verify"
  docker compose -f "$up_compose_current" -p "$up_project" down

  log "  phase 3: roll back to $prev_tag, verify"
  docker compose -f "$up_compose_prev" -p "$up_project" up -d
  wait_healthy "${up_project}-comphq-1"
  run_prev_smoke "$verify_only"
  docker compose -f "$up_compose_prev" -p "$up_project" down

  # TODO(P1-34): once backups exist, assert a pre-update backup file
  # appears under the data folder's backups/pre-update/ whenever the
  # upgrade actually ran migrations.

  git -C "$ROOT" worktree remove --force "$prev_worktree"
}

log "1. docker compose config validates"
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" config >/dev/null

log "2. up -d becomes healthy within 60s"
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" up -d
wait_healthy

log "3. restart: writing a marker and checksumming the data folder"
# The directory is owned by uid 568 (so the container can write to it);
# this shell runs as a different user, so writing into it needs sudo too.
echo "comphq container test $(date -u +%FT%TZ)" | sudo tee "$MARKER" >/dev/null
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

docker compose -f "$COMPOSE_FILE" -p "$PROJECT" down

log "5. offline: the app can't reach the internet, and the smoke suite passes"
OFFLINE_COMPOSE="$ROOT/deploy/test/offline.override.yaml"
export REPO_ROOT="$ROOT"
docker compose -f "$COMPOSE_FILE" -f "$OFFLINE_COMPOSE" -p "$PROJECT" up -d comphq
wait_healthy
EGRESS_STATUS="$(curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8080/__test/egress || echo 000)"
if [ "$EGRESS_STATUS" = "200" ]; then
  log "GET /__test/egress succeeded even though the network is internal-only"
  exit 1
fi
log "  egress correctly blocked (HTTP $EGRESS_STATUS)"
docker compose -f "$COMPOSE_FILE" -f "$OFFLINE_COMPOSE" -p "$PROJECT" run --rm runner
docker compose -f "$COMPOSE_FILE" -f "$OFFLINE_COMPOSE" -p "$PROJECT" down

log "6. upgrade/rollback"
run_upgrade_rollback_test "$IMAGE_REF"

log "all container tests passed"
