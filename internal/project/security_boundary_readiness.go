package project

import (
	"context"
	"time"
)

type SecurityBoundaryReadinessOptions struct {
	GeneratedAt time.Time
}

type SecurityBoundaryReadinessItem struct {
	Key              string
	Category         string
	Status           string
	Message          string
	RequiredEvidence []string
	BlockedBy        []string
	Metadata         map[string]any
}

type SecurityBoundaryReadiness struct {
	Status                        string
	Mode                          string
	Items                         []SecurityBoundaryReadinessItem
	Capabilities                  []string
	ForbiddenActions              []string
	GeneratedAt                   time.Time
	AuthEnforcementOpen           bool
	TeamPermissionEnforcementOpen bool
	APITokenIssuanceOpen          bool
	APITokenEnforcementOpen       bool
	SecretResolveOpen             bool
	RemoteWorkerCredentialsOpen   bool
	BudgetEnforcementOpen         bool
	QuotaDecrementOpen            bool
	UsageChargeWritten            bool
	WebhookDeliveryOpen           bool
	InboundCallbackOpen           bool
	ExternalAPICallOpen           bool
	AuthorizationChanged          bool
	SecretPlaintextRead           bool
	RemoteWorkerDirectPGAllowed   bool
	TeamConsoleCommandOpen        bool
	RemoteOpsControlOpen          bool
	ManagedUpgradeOpen            bool
	SupportBundleExportOpen       bool
	DefaultRemoteTelemetryOpen    bool
}

func (s Store) SecurityBoundaryReadiness(ctx context.Context, options SecurityBoundaryReadinessOptions) (SecurityBoundaryReadiness, error) {
	return BuildSecurityBoundaryReadiness(options), nil
}

func BuildSecurityBoundaryReadiness(options SecurityBoundaryReadinessOptions) SecurityBoundaryReadiness {
	options = normalizeSecurityBoundaryReadinessOptions(options)
	readiness := SecurityBoundaryReadiness{
		Status:      "ready",
		Mode:        "read_only_security_boundary_readiness",
		Items:       []SecurityBoundaryReadinessItem{},
		GeneratedAt: options.GeneratedAt,
		Capabilities: []string{
			"read_security_boundary",
			"read_actor_model",
			"read_team_readiness",
			"read_api_token_readiness",
			"read_secret_readiness",
			"read_remote_worker_readiness",
			"read_budget_quota_readiness",
			"read_integration_readiness",
			"read_remote_ops_readiness",
		},
		ForbiddenActions: []string{
			"change_api_authorization",
			"create_api_token",
			"rotate_api_token",
			"revoke_api_token",
			"enforce_team_permission",
			"resolve_secret_plaintext",
			"inject_secret_into_worker",
			"issue_remote_worker_credential",
			"allow_remote_worker_direct_postgres",
			"decrement_quota",
			"write_usage_charge",
			"deliver_webhook",
			"accept_inbound_callback_as_business_fact",
			"call_external_api",
			"open_team_command_console",
			"open_remote_ops_control",
			"run_managed_upgrade",
			"export_full_support_bundle",
			"enable_remote_telemetry_by_default",
		},
	}

	for _, item := range defaultSecurityBoundaryReadinessItems() {
		readiness.addItem(item)
	}
	return readiness
}

func normalizeSecurityBoundaryReadinessOptions(options SecurityBoundaryReadinessOptions) SecurityBoundaryReadinessOptions {
	if options.GeneratedAt.IsZero() {
		options.GeneratedAt = time.Now().UTC()
	}
	return options
}

func (r *SecurityBoundaryReadiness) addItem(item SecurityBoundaryReadinessItem) {
	if item.Metadata == nil {
		item.Metadata = map[string]any{}
	}
	r.Items = append(r.Items, item)
	if worseSecurityBoundaryStatus(item.Status, r.Status) {
		r.Status = item.Status
	}
}

func defaultSecurityBoundaryReadinessItems() []SecurityBoundaryReadinessItem {
	return []SecurityBoundaryReadinessItem{
		{
			Key:      "actor_audit_subjects",
			Category: "actors",
			Status:   "ready",
			Message:  "stable actor kinds are reserved for single-user and future team modes",
			RequiredEvidence: []string{
				"actors schema",
				"audit subject mapping",
			},
			Metadata: map[string]any{
				"actor_kinds": []string{
					"system",
					"local-user",
					"human",
					"service",
					"worker",
					"api-token",
					"cli-token",
					"agent",
					"areamatrix-shim",
				},
			},
		},
		{
			Key:      "auth_enforcement",
			Category: "auth",
			Status:   "ready",
			Message:  "auth enforcement remains closed in v1.0 and cannot change API visibility",
			RequiredEvidence: []string{
				"auth-team-secret boundary",
				"authorization_changed=false",
			},
			Metadata: map[string]any{
				"auth_enforcement_open": false,
				"authorization_changed": false,
				"v1_scope":              "schema/readiness/preview only",
				"next_rung":             "v1.x team/auth enforcement",
				"current_mode":          "local single-user readiness",
			},
		},
		{
			Key:      "team_permission_enforcement",
			Category: "team",
			Status:   "ready",
			Message:  "team membership and role matrix are readiness-only until a v1.x apply packet opens enforcement",
			RequiredEvidence: []string{
				"team remote control boundary",
				"project scope matrix",
				"token/session revoke plan",
			},
			Metadata: map[string]any{
				"team_permission_enforcement_open": false,
				"team_console_command_open":        false,
				"remote_ops_control_open":          false,
				"blocked_reason":                   "team_permission_enforcement_deferred",
			},
		},
		{
			Key:      "api_token_lifecycle",
			Category: "token",
			Status:   "ready",
			Message:  "API token issuance and enforcement are closed; v1.0 only keeps token metadata/readiness shape",
			RequiredEvidence: []string{
				"api token hash/scope/status design",
				"issuance gate",
				"revocation audit plan",
			},
			Metadata: map[string]any{
				"api_token_issuance_open":    false,
				"api_token_enforcement_open": false,
				"token_plaintext_read":       false,
				"blocked_reason":             "api_token_lifecycle_deferred",
			},
		},
		{
			Key:      "secret_resolve",
			Category: "secret",
			Status:   "ready",
			Message:  "secret refs may be named but plaintext resolve and injection stay closed",
			RequiredEvidence: []string{
				"secret_ref metadata",
				"redaction plan",
				"scoped binding approval plan",
			},
			Metadata: map[string]any{
				"secret_resolve_open":    false,
				"secret_plaintext_read":  false,
				"engine_secret_injected": false,
				"blocked_reason":         "secret_resolve_deferred",
			},
		},
		{
			Key:      "remote_worker_credentials",
			Category: "worker",
			Status:   "ready",
			Message:  "remote worker remains a readiness/blocker label and cannot receive credentials",
			RequiredEvidence: []string{
				"remote worker credential scope design",
				"lease-scoped credential plan",
				"revocation audit plan",
			},
			Metadata: map[string]any{
				"remote_worker_credentials_open":  false,
				"remote_worker_direct_pg_allowed": false,
				"remote_worker_dispatch_open":     false,
				"blocked_reason":                  "remote_worker_credentials_deferred",
			},
		},
		{
			Key:      "budget_quota_metering",
			Category: "budget",
			Status:   "ready",
			Message:  "budget and quota remain estimate/readiness only; no quota decrement or charge is written",
			RequiredEvidence: []string{
				"budget quota boundary",
				"estimate preview",
				"charge idempotency plan",
			},
			Metadata: map[string]any{
				"budget_enforcement_open": false,
				"quota_decrement_open":    false,
				"usage_charge_written":    false,
				"blocked_reason":          "budget_quota_enforcement_deferred",
			},
		},
		{
			Key:      "integration_webhook_boundary",
			Category: "integration",
			Status:   "ready",
			Message:  "integrations stay catalog/readiness/preview only; no webhook or external API side effect is open",
			RequiredEvidence: []string{
				"integration webhook boundary",
				"delivery plan preview",
				"inbound callback policy",
			},
			Metadata: map[string]any{
				"webhook_delivery_open":  false,
				"inbound_callback_open":  false,
				"external_api_call_open": false,
				"blocked_reason":         "integration_apply_deferred",
			},
		},
		{
			Key:      "managed_ops_boundary",
			Category: "ops",
			Status:   "ready",
			Message:  "remote ops, managed upgrade, full support export and remote telemetry stay closed",
			RequiredEvidence: []string{
				"operations deployment observability boundary",
				"metadata-only support bundle preview",
				"local-only telemetry default",
			},
			Metadata: map[string]any{
				"remote_ops_control_open":       false,
				"managed_upgrade_open":          false,
				"support_bundle_export_open":    false,
				"default_remote_telemetry_open": false,
				"blocked_reason":                "managed_ops_deferred",
			},
		},
	}
}

func worseSecurityBoundaryStatus(candidate string, current string) bool {
	rank := map[string]int{
		"ready":   0,
		"warn":    1,
		"blocked": 2,
	}
	return rank[candidate] > rank[current]
}
