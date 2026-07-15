#!/usr/bin/env bash
set -euo pipefail

container="areaflow-minio-smoke-$$"
port="${AREAFLOW_MINIO_SMOKE_PORT:-19000}"
server_pid=""
fixture_dir="$(mktemp -d "${TMPDIR:-/tmp}/areaflow-s3.XXXXXX")"

cleanup() {
	[[ -z "${server_pid}" ]] || kill "${server_pid}" >/dev/null 2>&1 || true
	docker unpause "${container}" >/dev/null 2>&1 || true
  docker rm -f "${container}" >/dev/null 2>&1 || true
	rm -rf "${fixture_dir}"
}
trap cleanup EXIT

docker run -d --rm --name "${container}" -p "127.0.0.1:${port}:9000" \
  -e MINIO_ROOT_USER=areaflow \
  -e MINIO_ROOT_PASSWORD=areaflow-secret \
  -e MINIO_KMS_SECRET_KEY=areaflow-default-key:MDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTIzNDU2Nzg5MDE= \
  -e MINIO_KMS_AUTO_ENCRYPTION=on \
  minio/minio:RELEASE.2025-04-22T22-12-26Z server /data >/dev/null

deadline=$((SECONDS + 60))
until curl -fsS "http://127.0.0.1:${port}/minio/health/ready" >/dev/null; do
  if (( SECONDS >= deadline )); then
    echo "smoke-s3-minio: MinIO readiness timed out" >&2
    docker logs "${container}" >&2 || true
    exit 1
  fi
  sleep 1
done

AREAFLOW_S3_SMOKE=1 \
AREAFLOW_S3_ENDPOINT="http://127.0.0.1:${port}" \
AREAFLOW_S3_REGION=us-east-1 \
AREAFLOW_S3_BUCKET=areaflow-smoke \
AREAFLOW_S3_USE_PATH_STYLE=true \
AWS_ACCESS_KEY_ID=areaflow \
AWS_SECRET_ACCESS_KEY=areaflow-secret \
AWS_EC2_METADATA_DISABLED=true \
go test ./internal/artifact -run TestS3BackendMinIOSmoke -count=1

if [[ -n "${AREAFLOW_DATABASE_URL:-}" ]]; then
  AREAFLOW_ARTIFACT_MIGRATION_SMOKE=1 \
  AREAFLOW_S3_ENDPOINT="http://127.0.0.1:${port}" \
  AREAFLOW_S3_REGION=us-east-1 \
  AREAFLOW_S3_BUCKET=areaflow-smoke \
  AREAFLOW_S3_USE_PATH_STYLE=true \
  AWS_ACCESS_KEY_ID=areaflow \
  AWS_SECRET_ACCESS_KEY=areaflow-secret \
  AWS_EC2_METADATA_DISABLED=true \
  go test ./internal/project -run TestArtifactMigrationPostgresS3Smoke -count=1

	go build -o "${fixture_dir}/areaflow" ./cmd/areaflow
	"${fixture_dir}/areaflow" migrate up >/dev/null
	AREAFLOW_ENV=development AREAFLOW_AUTH_MODE=disabled AREAFLOW_HOST=127.0.0.1 AREAFLOW_PORT=3880 AREAFLOW_METRICS_PORT=9130 \
	AREAFLOW_ARTIFACT_BACKEND=s3 AREAFLOW_S3_ENDPOINT="http://127.0.0.1:${port}" AREAFLOW_S3_REGION=us-east-1 \
	AREAFLOW_S3_BUCKET=areaflow-smoke AREAFLOW_S3_USE_PATH_STYLE=true AWS_ACCESS_KEY_ID=areaflow \
	AWS_SECRET_ACCESS_KEY=areaflow-secret AWS_EC2_METADATA_DISABLED=true \
	"${fixture_dir}/areaflow" server >"${fixture_dir}/server.log" 2>&1 &
	server_pid=$!
	deadline=$((SECONDS + 30))
	until curl -fsS http://127.0.0.1:3880/api/v1/ready >/dev/null; do
		if (( SECONDS >= deadline )); then cat "${fixture_dir}/server.log" >&2; exit 1; fi
		sleep 1
	done
	docker pause "${container}" >/dev/null
	if curl -fsS --max-time 5 http://127.0.0.1:3880/api/v1/ready >/dev/null 2>&1; then
		echo "smoke-s3-minio: readiness stayed ready while S3 was unavailable" >&2
		exit 1
	fi
	docker unpause "${container}" >/dev/null
	deadline=$((SECONDS + 30))
	until curl -fsS http://127.0.0.1:3880/api/v1/ready >/dev/null; do
		if (( SECONDS >= deadline )); then echo "smoke-s3-minio: readiness did not recover" >&2; exit 1; fi
		sleep 1
	done
fi

echo "smoke-s3-minio: ok"
