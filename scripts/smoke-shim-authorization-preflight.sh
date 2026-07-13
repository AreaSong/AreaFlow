#!/usr/bin/env bash
set -euo pipefail

echo "smoke-shim-authorization-preflight: verifying real AreaMatrix read-only shim/status authorization surface"
if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-shim-authorization-preflight: blocked; AREAFLOW_DATABASE_URL is required" >&2
  echo "smoke-shim-authorization-preflight: use make smoke-docker-shim-authorization-preflight for an isolated DB-backed run" >&2
  exit 1
fi

readonly_output="$(bash scripts/smoke-areamatrix-readonly.sh 2>&1)"
printf "%s\n" "${readonly_output}"
if grep -Fq "skipped; AREAFLOW_DATABASE_URL is not set" <<<"${readonly_output}"; then
  echo "smoke-shim-authorization-preflight: readonly smoke skipped unexpectedly" >&2
  exit 1
fi
if ! grep -Fq "smoke-areamatrix-readonly: pass " <<<"${readonly_output}"; then
  echo "smoke-shim-authorization-preflight: readonly smoke did not report pass" >&2
  exit 1
fi
echo "smoke-shim-authorization-preflight: ok"
