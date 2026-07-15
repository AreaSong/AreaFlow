.PHONY: test build fmt fmt-check check docs-check governance-check contract-check web-install web-build desktop-install desktop-build brand-export brand-validate package-a-readiness package-a-dirty-review package-a-source-hash package-a-authorization-packet package-b-readiness package-b-dirty-review package-b-authorization-packet smoke-package-a smoke-docker-package-a smoke-package-b-readiness smoke-docker-package-b-readiness smoke-package-a-fingerprint-parity smoke-docker-package-a-fingerprint-parity smoke-status-projection-schema smoke-fixture smoke-docker-fixture smoke-compatibility-fixture smoke-docker-compatibility-fixture smoke-areamatrix-readonly smoke-docker-areamatrix-readonly smoke-shim-authorization-preflight smoke-docker-shim-authorization-preflight smoke-local smoke-docker smoke-approved-artifact-write smoke-docker-approved-artifact-write smoke-managed-generated-write smoke-docker-managed-generated-write smoke-execution-plan smoke-docker-execution-plan smoke-execution-forwarding-v1-readiness smoke-docker-execution-forwarding-v1-readiness smoke-completion-proof smoke-docker-completion-proof smoke-completion-audit-full-proof smoke-docker-completion-audit-full-proof smoke-completion-audit-release-candidate-snapshot smoke-docker-completion-audit-release-candidate-snapshot smoke-completion-audit-real-identity-readiness smoke-docker-completion-audit-real-identity-readiness smoke-completion-audit-real-identity-protected-path-proof smoke-docker-completion-audit-real-identity-protected-path-proof smoke-completion-audit-real-identity-fixture-snapshot smoke-docker-completion-audit-real-identity-fixture-snapshot smoke-execution-cutover-proof smoke-docker-execution-cutover-proof smoke-validation-proof smoke-docker-validation-proof smoke-source-alignment-proof smoke-docker-source-alignment-proof smoke-task-matrix-proof smoke-docker-task-matrix-proof smoke-security-closure-proof smoke-docker-security-closure-proof smoke-operations-proof smoke-docker-operations-proof smoke-backup-restore-proof smoke-docker-backup-restore-proof smoke-release-packaging-proof smoke-docker-release-packaging-proof smoke-v1-stable-fixture smoke-docker-v1-stable-fixture smoke-project-isolation smoke-docker-project-isolation smoke-web smoke-docker-web smoke-web-areamatrix-readonly smoke-docker-web-areamatrix-readonly smoke-graceful-shutdown smoke-docker-graceful-shutdown
.PHONY: security-check release-check smoke-append-only smoke-docker-append-only smoke-s3-minio smoke-s3-artifact smoke-ha-local smoke-production-ha smoke-production-capacity smoke-docker-ha-local smoke-auth-postgres smoke-oidc-rbac smoke-docker-auth-postgres smoke-openapi-contract smoke-upgrade-rollback
.PHONY: production-smoke load-check

fmt:
	go fmt ./...

fmt-check:
	@unformatted="$$(gofmt -l $$(git ls-files --cached --others --exclude-standard -- '*.go'))"; \
	if [ -n "$$unformatted" ]; then echo "Go files require gofmt:"; echo "$$unformatted"; exit 1; fi

test:
	go test ./...

build:
	go build ./cmd/areaflow

web-install:
	cd web && npm install

web-build:
	cd web && npm run build

desktop-install:
	cd desktop && npm install

desktop-build:
	cd desktop && npm run build

docs-check:
	node scripts/check-doc-links.mjs
	node scripts/check-doc-governance.mjs

governance-check:
	node scripts/check-asw-governance.mjs

contract-check:
	node scripts/check-openapi-contract.mjs
	node scripts/check-control-plane-boundary.mjs
	node scripts/check-production-observability.mjs

security-check:
	go vet ./...
	go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
	npm audit --audit-level=high
	npm audit --audit-level=high --prefix web
	npm audit --audit-level=high --prefix desktop
	node scripts/check-license-policy.mjs

release-check: check
	node scripts/check-release-contract.mjs v1.0.0
	docker build --build-arg VERSION=1.0.0 --build-arg COMMIT=local --build-arg BUILD_DATE=local -t areaflow:1.0.0-rc .

smoke-append-only:
	bash scripts/smoke-append-only.sh

smoke-docker-append-only:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-append-only.sh bash scripts/smoke-docker.sh

smoke-s3-minio:
	bash scripts/smoke-s3-minio.sh

smoke-s3-artifact:
	bash scripts/smoke-s3-artifact.sh

smoke-ha-local:
	bash scripts/smoke-ha-local.sh

smoke-production-ha:
	bash scripts/smoke-production-ha.sh

smoke-production-capacity:
	bash scripts/smoke-production-capacity.sh

smoke-docker-ha-local:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-ha-local.sh bash scripts/smoke-docker.sh

smoke-auth-postgres:
	bash scripts/smoke-auth-postgres.sh

smoke-oidc-rbac:
	bash scripts/smoke-oidc-rbac.sh

smoke-openapi-contract:
	bash scripts/smoke-openapi-contract.sh

smoke-upgrade-rollback:
	bash scripts/smoke-upgrade-rollback.sh

production-smoke: smoke-openapi-contract smoke-append-only smoke-oidc-rbac smoke-project-isolation smoke-s3-artifact smoke-upgrade-rollback smoke-backup-restore-proof smoke-production-ha smoke-graceful-shutdown smoke-web

load-check: smoke-production-capacity

smoke-docker-auth-postgres:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-auth-postgres.sh bash scripts/smoke-docker.sh

brand-export:
	npm run brand:export

brand-validate:
	npm run brand:validate

check: fmt-check test build web-build desktop-build docs-check governance-check contract-check brand-validate

package-a-readiness:
	bash scripts/audit-package-a-readiness.sh

package-a-dirty-review:
	bash scripts/audit-package-a-dirty-review.sh

package-a-source-hash:
	bash scripts/audit-package-a-source-hash.sh

package-a-authorization-packet:
	bash scripts/audit-package-a-authorization-packet.sh

package-b-readiness:
	bash scripts/audit-package-b-readiness.sh

package-b-dirty-review:
	bash scripts/audit-package-b-dirty-review.sh

package-b-authorization-packet:
	bash scripts/audit-package-b-authorization-packet.sh

smoke-package-a:
	bash scripts/smoke-package-a.sh

smoke-docker-package-a:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-package-a.sh bash scripts/smoke-docker.sh

smoke-package-b-readiness:
	bash scripts/smoke-package-b-readiness.sh

smoke-docker-package-b-readiness:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-package-b-readiness.sh bash scripts/smoke-docker.sh

smoke-package-a-fingerprint-parity:
	bash scripts/smoke-package-a-fingerprint-parity.sh

smoke-docker-package-a-fingerprint-parity:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-package-a-fingerprint-parity.sh bash scripts/smoke-docker.sh

smoke-local:
	bash scripts/smoke-local.sh

smoke-fixture:
	bash scripts/smoke-fixture.sh

smoke-status-projection-schema:
	bash scripts/smoke-status-projection-schema.sh

smoke-docker-fixture:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-fixture.sh bash scripts/smoke-docker.sh

smoke-compatibility-fixture:
	bash scripts/smoke-compatibility-fixture.sh

smoke-docker-compatibility-fixture:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-compatibility-fixture.sh bash scripts/smoke-docker.sh

smoke-areamatrix-readonly:
	bash scripts/smoke-areamatrix-readonly.sh

smoke-docker-areamatrix-readonly:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-areamatrix-readonly.sh bash scripts/smoke-docker.sh

smoke-shim-authorization-preflight:
	bash scripts/smoke-shim-authorization-preflight.sh

smoke-docker-shim-authorization-preflight:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-shim-authorization-preflight.sh bash scripts/smoke-docker.sh

smoke-docker:
	bash scripts/smoke-docker.sh

smoke-approved-artifact-write:
	bash scripts/smoke-approved-artifact-write.sh

smoke-docker-approved-artifact-write:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-approved-artifact-write.sh bash scripts/smoke-docker.sh

smoke-managed-generated-write:
	bash scripts/smoke-managed-generated-write.sh

smoke-docker-managed-generated-write:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-managed-generated-write.sh bash scripts/smoke-docker.sh

smoke-execution-plan:
	bash scripts/smoke-execution-plan.sh

smoke-docker-execution-plan:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-execution-plan.sh bash scripts/smoke-docker.sh

smoke-execution-forwarding-v1-readiness:
	bash scripts/smoke-execution-forwarding-v1-readiness.sh

smoke-docker-execution-forwarding-v1-readiness:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-execution-forwarding-v1-readiness.sh bash scripts/smoke-docker.sh

smoke-completion-proof:
	bash scripts/smoke-completion-proof.sh

smoke-docker-completion-proof:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-completion-proof.sh bash scripts/smoke-docker.sh

smoke-completion-audit-full-proof:
	bash scripts/smoke-completion-audit-full-proof.sh

smoke-docker-completion-audit-full-proof:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-completion-audit-full-proof.sh bash scripts/smoke-docker.sh

smoke-completion-audit-release-candidate-snapshot:
	bash scripts/smoke-completion-audit-release-candidate-snapshot.sh

smoke-docker-completion-audit-release-candidate-snapshot:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-completion-audit-release-candidate-snapshot.sh bash scripts/smoke-docker.sh

smoke-completion-audit-real-identity-readiness:
	bash scripts/smoke-completion-audit-real-identity-readiness.sh

smoke-docker-completion-audit-real-identity-readiness:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-completion-audit-real-identity-readiness.sh bash scripts/smoke-docker.sh

smoke-completion-audit-real-identity-protected-path-proof:
	bash scripts/smoke-completion-audit-real-identity-protected-path-proof.sh

smoke-docker-completion-audit-real-identity-protected-path-proof:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-completion-audit-real-identity-protected-path-proof.sh bash scripts/smoke-docker.sh

smoke-completion-audit-real-identity-fixture-snapshot:
	bash scripts/smoke-completion-audit-real-identity-fixture-snapshot.sh

smoke-docker-completion-audit-real-identity-fixture-snapshot:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-completion-audit-real-identity-fixture-snapshot.sh bash scripts/smoke-docker.sh

smoke-execution-cutover-proof:
	bash scripts/smoke-execution-cutover-proof.sh

smoke-docker-execution-cutover-proof:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-execution-cutover-proof.sh bash scripts/smoke-docker.sh

smoke-validation-proof:
	bash scripts/smoke-validation-proof.sh

smoke-docker-validation-proof:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-validation-proof.sh bash scripts/smoke-docker.sh

smoke-source-alignment-proof:
	bash scripts/smoke-source-alignment-proof.sh

smoke-docker-source-alignment-proof:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-source-alignment-proof.sh bash scripts/smoke-docker.sh

smoke-task-matrix-proof:
	bash scripts/smoke-task-matrix-proof.sh

smoke-docker-task-matrix-proof:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-task-matrix-proof.sh bash scripts/smoke-docker.sh

smoke-security-closure-proof:
	bash scripts/smoke-security-closure-proof.sh

smoke-docker-security-closure-proof:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-security-closure-proof.sh bash scripts/smoke-docker.sh

smoke-operations-proof:
	bash scripts/smoke-operations-proof.sh

smoke-docker-operations-proof:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-operations-proof.sh bash scripts/smoke-docker.sh

smoke-backup-restore-proof:
	bash scripts/smoke-backup-restore-proof.sh

smoke-docker-backup-restore-proof:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-backup-restore-proof.sh bash scripts/smoke-docker.sh

smoke-release-packaging-proof:
	bash scripts/smoke-release-packaging-proof.sh

smoke-docker-release-packaging-proof:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-release-packaging-proof.sh bash scripts/smoke-docker.sh

smoke-v1-stable-fixture:
	bash scripts/smoke-v1-stable-fixture.sh

smoke-docker-v1-stable-fixture:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-v1-stable-fixture.sh bash scripts/smoke-docker.sh

smoke-project-isolation:
	bash scripts/smoke-project-isolation.sh

smoke-docker-project-isolation:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-project-isolation.sh bash scripts/smoke-docker.sh

smoke-web:
	bash scripts/smoke-web.sh

smoke-docker-web:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-web.sh bash scripts/smoke-docker.sh

smoke-graceful-shutdown:
	bash scripts/smoke-graceful-shutdown.sh

smoke-docker-graceful-shutdown:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-graceful-shutdown.sh bash scripts/smoke-docker.sh

smoke-web-areamatrix-readonly:
	bash scripts/smoke-web-areamatrix-readonly.sh

smoke-docker-web-areamatrix-readonly:
	AREAFLOW_SMOKE_SCRIPT=scripts/smoke-web-areamatrix-readonly.sh bash scripts/smoke-docker.sh
