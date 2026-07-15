#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-production-ha: skipped; AREAFLOW_DATABASE_URL is not set"
  exit 0
fi

fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-ha.XXXXXX")"
binary="${fixture_dir}/areaflow"
pid_a=""
pid_b=""
proxy_pid=""

cleanup() {
  [[ -z "${pid_a}" ]] || kill "${pid_a}" >/dev/null 2>&1 || true
  [[ -z "${pid_b}" ]] || kill "${pid_b}" >/dev/null 2>&1 || true
  [[ -z "${proxy_pid}" ]] || kill "${proxy_pid}" >/dev/null 2>&1 || true
  rm -rf "${fixture_dir}"
}
trap cleanup EXIT

go build -o "${binary}" ./cmd/areaflow
"${binary}" migrate up >/dev/null

start_replica() {
  local port="$1"
  local metrics_port="$2"
  local artifact_root="${fixture_dir}/artifacts"
  AREAFLOW_ENV=development AREAFLOW_AUTH_MODE=disabled AREAFLOW_HOST=127.0.0.1 \
    AREAFLOW_PORT="${port}" AREAFLOW_METRICS_PORT="${metrics_port}" AREAFLOW_ARTIFACT_ROOT="${artifact_root}" \
    "${binary}" server >"${fixture_dir}/${port}.log" 2>&1 &
  echo $!
}

pid_a="$(start_replica 3857 9107)"
pid_b="$(start_replica 3858 9108)"
AREAFLOW_HA_PROXY_PORT=3860 node scripts/ha-health-proxy.mjs >"${fixture_dir}/proxy.log" 2>&1 &
proxy_pid=$!

deadline=$((SECONDS + 30))
until curl -fsS http://127.0.0.1:3860/api/v1/ready >/dev/null; do
  if (( SECONDS >= deadline )); then
    echo "smoke-production-ha: proxy readiness timed out" >&2
    cat "${fixture_dir}"/*.log >&2 || true
    exit 1
  fi
  sleep 1
done

AREAFLOW_LOAD_URL=http://127.0.0.1:3860 AREAFLOW_LOAD_SUSTAINED_SECONDS=2 AREAFLOW_LOAD_PEAK_SECONDS=1 node scripts/load-production-baseline.js
kill "${pid_a}"
wait "${pid_a}" 2>/dev/null || true
pid_a=""

deadline=$((SECONDS + 15))
until curl -fsS http://127.0.0.1:3860/api/v1/ready >/dev/null; do
  if (( SECONDS >= deadline )); then
    echo "smoke-production-ha: automatic failover timed out" >&2
    exit 1
  fi
  sleep 1
done
AREAFLOW_LOAD_URL=http://127.0.0.1:3860 AREAFLOW_LOAD_SUSTAINED_SECONDS=2 AREAFLOW_LOAD_PEAK_SECONDS=1 node scripts/load-production-baseline.js

echo "smoke-production-ha: ok"
