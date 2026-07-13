# Integration And Webhook Boundary

## Purpose

本文定义 AreaFlow 外部系统接入、webhook、third-party callback、GitHub / external API connector、
notification provider 和未来多 API 接入的安全边界。它补充
[`api-surface.md`](api-surface.md)、
[`security-permissions.md`](security-permissions.md)、
[`auth-team-secret-boundary.md`](auth-team-secret-boundary.md)、
[`command-approval-contract.md`](command-approval-contract.md) 和
[`plugin-marketplace-boundary.md`](plugin-marketplace-boundary.md)。

Integration 是受治理的外部效果面，不是插件、worker 或 Web UI 的旁路。任何 external API call、
webhook delivery、inbound callback、OAuth token、notification delivery 或 third-party connector 都必须
通过 AreaFlow API/service layer、project scope、secret policy、network allowlist、Command API、audit 和
redaction 规则。

v1.0 前只允许 schema/readiness/preview/blocked reason。真实 webhook delivery、inbound callback
processing、OAuth flow、external API write、provider notification 和 GitHub action 都属于 v1.x 能力。

## Non-goals Before v1.x

v1.0 之前禁止把以下能力解释为已打开：

- Outbound webhook delivery。
- Inbound webhook callback processing。
- OAuth / app installation flow。
- GitHub issue、PR、status、check、release 或 dispatch 写操作。
- Slack / email / push / remote notification delivery。
- External API connector 读取或写入第三方系统。
- Webhook signing secret resolve。
- Provider token refresh / rotation。
- Retry queue、dead-letter queue 或 background delivery worker。

v1.0 可以做：

- `webhooks` schema / metadata。
- Integration catalog / readiness / doctor / conformance。
- Delivery plan preview。
- Blocked reason 和 audit coverage gap。

## Concepts

```text
integration_connection:
  project-scoped connection metadata for GitHub, Slack, OpenAI, webhook target, etc.

outbound_webhook:
  AreaFlow event -> external endpoint delivery plan.

inbound_webhook:
  external callback -> AreaFlow command or event intake plan.

delivery_attempt:
  append-only attempt record for an outbound integration delivery.

callback_event:
  append-only raw-envelope metadata for inbound callback verification.

connector_command:
  Command API request that reads or writes an external API.
```

Raw external payloads are not automatically trusted source facts. Verified callback metadata can become event input;
business state changes still require Command API, permission, gate, approval and audit.

## Scope And Secret Model

Every integration must be scoped:

```text
project_id / project_key
connection_key
provider
direction: outbound | inbound | bidirectional
allowed_event_types
allowed_command_types
allowed_network_targets
secret_ref nullable
signing_key_ref nullable
actor_id
status
```

Secrets are references only. `secret_ref` / `signing_key_ref` do not grant resolve. Secret resolve must follow
[`auth-team-secret-boundary.md`](auth-team-secret-boundary.md), and secret values must never enter logs, events,
audit metadata, artifacts, webhook payload previews or provider error artifacts.

## Opening Ladder

### I0 Reserved Schema And Readiness

Status: current v1.0 boundary.

Allowed:

- Store / inspect integration metadata and webhook config shape.
- Return readiness / blocked reason.
- Document provider capabilities and required scopes.

Forbidden:

- Deliver webhook.
- Accept inbound callback as trusted input.
- Resolve signing secret.
- Call external API.
- Start delivery worker.

Required facts:

```text
mode = integration_readiness
outbound_delivery_open=false
inbound_callback_open=false
external_api_call_open=false
oauth_flow_open=false
signing_secret_resolved=false
delivery_worker_started=false
```

### I1 Catalog And Plan Preview

Status: future preview-only.

Allowed:

- List supported provider kinds.
- Build delivery plan preview for event -> target.
- Build inbound mapping preview for callback -> command candidate.
- Explain missing secret, missing network allowlist, missing approval, unsupported event type.

Forbidden:

- Send network request.
- Persist delivery attempt.
- Create command from callback.
- Read provider token.

### I2 Fixture Outbound Delivery

Status: future fixture-only.

Allowed:

- Deliver to local fixture endpoint or test harness.
- Verify signing, idempotency key, retry classification and redaction.
- Persist delivery attempt in fixture project only.

Forbidden:

- Real third-party endpoint.
- AreaMatrix endpoint.
- Secret-backed provider credential.

### I3 Fixture Inbound Callback

Status: future fixture-only.

Allowed:

- Verify fixture signature.
- Persist callback envelope metadata with hash, provider, event type and verified status.
- Produce command preview, not command apply.

Forbidden:

- Treat callback as approval.
- Mutate workflow, run, worker, release, secret or project file.
- Store raw payload when it contains secrets or user content.

### I4 Project-scoped Outbound Delivery Beta

Status: v1.x retained_beta.

Allowed:

- Deliver allowlisted event types from one project to one approved endpoint.
- Resolve signing secret through scoped secret binding.
- Persist delivery_attempt and audit event.
- Retry only according to explicit retry policy.

Forbidden:

- Unbounded retries.
- Unknown endpoint.
- Delivery without audit.
- Payload containing secret, prompt, user file content or protected artifact content.

Required evidence:

- Idempotent delivery key.
- Signature verification on receiver fixture.
- Redaction tests.
- Retry and dead-letter behavior.
- Endpoint allowlist.
- Disable / revoke path.

### I5 Inbound Callback Beta

Status: v1.x after I3 and auth/network design.

Allowed:

- Accept provider callback on project-scoped endpoint.
- Verify signature, timestamp, replay window and event type.
- Persist callback metadata and create command preview.
- Optionally create low-risk command request only if mapping, actor, scope and approval policy allow.

Forbidden:

- Callback directly mutates business state.
- Callback becomes approval without human or policy approval.
- Callback bypasses project visibility guard.
- Callback creates R2-R4 command without approval scope.

### I6 External API Connector Command

Status: v1.x external_effect command.

Allowed:

- Use Command API to call external API with explicit provider, method, endpoint, purpose and resource scope.
- Use scoped secret binding when needed.
- Persist request hash, response hash, redacted summary, audit and remediation plan.

Forbidden:

- Direct connector call from Web/Desktop/plugin/worker.
- Raw provider response with secrets or user content in unrestricted artifact.
- External API write without rollback/remediation or revoke path.
- Network wildcard.

### I7 Provider Automation

Status: v1.x after connector command stability.

Allowed:

- Automate provider-specific workflows such as GitHub checks, issue sync, notification routing or release status.

Forbidden:

- Provider automation that substitutes AreaFlow source of truth.
- Provider callback that changes AreaFlow state without Command API.
- Provider-side delete / destructive action without explicit R4 approval.

## Outbound Delivery Contract

Outbound delivery must use an outbox-style contract:

```text
event_id
project_id
webhook_id
delivery_id
idempotency_key
payload_hash
redaction_policy
signing_key_ref
target_url_hash
attempt_no
status
next_retry_at nullable
audit_event_id
```

Delivery payload must be minimal and project-scoped. Default payload should include IDs, links and summary metadata,
not prompt text, secret values, raw logs, user files or artifact contents.

## Inbound Callback Contract

Inbound callback handling order:

```text
receive envelope
check provider and project route
verify signature / timestamp / replay window
hash raw payload
extract allowed metadata
drop or quarantine disallowed fields
persist callback metadata
build command preview
require approval when command risk requires it
```

Inbound callback cannot directly approve, run, cancel, publish, restore, resolve secret, issue worker credential or
write project files.

## Provider And Network Rules

Provider calls require:

```text
network capability
provider allowlist
method allowlist
endpoint allowlist
purpose
secret scope when needed
budget / quota preflight when engine or paid API is involved
audit event
timeout and retry policy
disable / revoke path
```

Unknown provider returns `blocked:unknown_provider`. Missing network allowlist returns
`blocked:network_not_allowed`. Missing secret returns `blocked:secret_ref_unavailable`.

## AreaMatrix Dogfood Policy

AreaMatrix first phase:

- No outbound webhook delivery from AreaMatrix.
- No inbound callback can mutate AreaMatrix workflow, progress, execution, release evidence or user files.
- GitHub / external issue sync can start as metadata-only preview after explicit design.
- Any integration touching `workflow/versions/**/execution/**`, `progress.json`, logs, checkpoint, release evidence
  or user file paths requires separate explicit approval.

## Suspension Rule

Integration capability must be suspended if any of the following happens:

```text
delivery without audit
secret included in payload
raw prompt, raw log, user file, or protected artifact leaked
unknown endpoint delivery
signature verification bypass
callback replay accepted
callback mutates state directly
external API write bypasses Command API
provider token stored in plaintext
unbounded retry storm
cross-project event delivered
AreaMatrix protected path modified by integration
```

Recovery requires remediation evidence, revoked/rotated secret where relevant, delivery replay audit, focused
regression test and explicit approval.
