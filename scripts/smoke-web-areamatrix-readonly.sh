#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${AREAFLOW_DATABASE_URL:-}" ]]; then
  echo "smoke-web-areamatrix-readonly: blocked; AREAFLOW_DATABASE_URL is required" >&2
  exit 1
fi

AREAFLOW_WEB_SMOKE_MODE=real-areamatrix \
  AREAFLOW_WEB_SMOKE_PROJECT="${AREAFLOW_WEB_SMOKE_PROJECT:-areamatrix}" \
  AREAFLOW_WEB_SMOKE_PROJECT_ROOT="${AREAFLOW_WEB_SMOKE_PROJECT_ROOT:-/Users/as/Ai-Project/project/AreaMatrix}" \
  AREAFLOW_WEB_SMOKE_CONFIG="${AREAFLOW_WEB_SMOKE_CONFIG:-examples/areamatrix/areaflow.yaml}" \
  bash scripts/smoke-web.sh
