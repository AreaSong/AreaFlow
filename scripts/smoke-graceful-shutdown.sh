#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-graceful-shutdown: AREAFLOW_DATABASE_URL is required" >&2
  exit 1
fi

host="127.0.0.1"
port="${AREAFLOW_GRACEFUL_SHUTDOWN_PORT:-3861}"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-graceful-shutdown.XXXXXX")"
binary="${tmp_dir}/areaflow"
log_file="${tmp_dir}/server.log"
server_pid=""

cleanup() {
  if [[ -n "${server_pid}" ]] && kill -0 "${server_pid}" >/dev/null 2>&1; then
    kill -TERM "${server_pid}" >/dev/null 2>&1 || true
    wait "${server_pid}" >/dev/null 2>&1 || true
  fi
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

go build -o "${binary}" ./cmd/areaflow
AREAFLOW_HOST="${host}" AREAFLOW_PORT="${port}" "${binary}" server >"${log_file}" 2>&1 &
server_pid=$!

deadline=$((SECONDS + 30))
while ! curl -fsS "http://${host}:${port}/api/v1/ready" >/dev/null 2>&1; do
  if ! kill -0 "${server_pid}" >/dev/null 2>&1; then
    cat "${log_file}" >&2
    echo "smoke-graceful-shutdown: server exited before readiness" >&2
    exit 1
  fi
  if (( SECONDS >= deadline )); then
    cat "${log_file}" >&2
    echo "smoke-graceful-shutdown: readiness timeout" >&2
    exit 1
  fi
  sleep 1
done

kill -TERM "${server_pid}"
set +e
wait "${server_pid}"
exit_code=$?
set -e
server_pid=""

if [[ ${exit_code} -ne 0 ]]; then
  cat "${log_file}" >&2
  echo "smoke-graceful-shutdown: exit code ${exit_code}, want 0" >&2
  exit 1
fi
if ! grep -Fq "AreaFlow API stopped" "${log_file}"; then
  cat "${log_file}" >&2
  echo "smoke-graceful-shutdown: missing stopped lifecycle log" >&2
  exit 1
fi

echo "smoke-graceful-shutdown: ok"
