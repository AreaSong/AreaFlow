#!/usr/bin/env bash
set -euo pipefail

export AREAFLOW_COMPLETION_PROOF_PROJECT_KEY="${AREAFLOW_EXECUTION_CUTOVER_PROOF_PROJECT_KEY:-areamatrix}"

bash scripts/smoke-completion-proof.sh
