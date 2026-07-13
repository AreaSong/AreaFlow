import "./styles.css";

type ComponentStatus = {
  status: string;
  message: string;
};

type WorkerPoolStatus = ComponentStatus & {
  total_projects: number;
  total_workers: number;
  total_online_workers: number;
  total_active_leases: number;
  total_queued_tasks: number;
  total_needs_recovery: number;
};

type DashboardStatus = ComponentStatus & {
  url: string;
  api_url: string;
};

type ServiceStatus = {
  status: string;
  mode: string;
  api: ComponentStatus;
  database: ComponentStatus;
  worker_pool: WorkerPoolStatus;
  dashboard: DashboardStatus;
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

type ServiceControlAction = {
  key: string;
  label: string;
  category: string;
  status: string;
  default_ui_state: string;
  risk_level: string;
  blockers: string[];
};

type ServiceControlGate = {
  status: string;
  mode: string;
  actions: ServiceControlAction[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
  db_write_attempted: boolean;
  project_write_attempted: boolean;
  process_control_attempted: boolean;
  command_created: boolean;
  approval_created: boolean;
  audit_event_written: boolean;
  worker_scheduled: boolean;
  workflow_execution_started: boolean;
  secrets_resolved: boolean;
  network_used: boolean;
};

type NotificationAction = ServiceControlAction;

type NotificationGate = {
  status: string;
  mode: string;
  actions: NotificationAction[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
  db_write_attempted: boolean;
  project_write_attempted: boolean;
  event_stream_opened: boolean;
  notification_requested: boolean;
  command_created: boolean;
  approval_created: boolean;
  audit_event_written: boolean;
  worker_scheduled: boolean;
  workflow_execution_started: boolean;
  secrets_resolved: boolean;
  network_used: boolean;
};

type TrayMenuAction = ServiceControlAction;

type TrayMenuGate = {
  status: string;
  mode: string;
  actions: TrayMenuAction[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
  db_write_attempted: boolean;
  project_write_attempted: boolean;
  tray_menu_created: boolean;
  os_integration_requested: boolean;
  command_created: boolean;
  approval_created: boolean;
  audit_event_written: boolean;
  service_control_attempted: boolean;
  notification_requested: boolean;
  worker_scheduled: boolean;
  workflow_execution_started: boolean;
  secrets_resolved: boolean;
  network_used: boolean;
};

type OperationsReadinessItem = {
  key: string;
  category: string;
  status: string;
  message: string;
  blocked_by: string[];
  next_command: string;
};

type SupportBundlePreview = {
  status: string;
  mode: string;
  included_metadata: string[];
  excluded_sensitive_content: string[];
  path_references: unknown[];
  hashes: unknown[];
  safety_facts: Record<string, boolean>;
};

type MigrationLedgerReadiness = {
  status: string;
  mode: string;
  entries: unknown[];
  applied_count: number;
  pending_count: number;
  schema_migrations_table_present: boolean;
  full_ledger_table_present: boolean;
  preflight_apply_verify_remediation_ready: boolean;
  safety_facts: Record<string, boolean>;
};

type OperationsReadiness = {
  status: string;
  mode: string;
  items: OperationsReadinessItem[];
  service_status: ServiceStatus;
  support_bundle: SupportBundlePreview;
  migration_ledger: MigrationLedgerReadiness;
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  telemetry_default: string;
  managed_ops_status: string;
  support_export_status: string;
  generated_at: string;
};

type ShimFilePlan = {
  path: string;
  action: string;
  required: boolean;
  reason: string;
  boundary: string;
};

type ShimAuthorizationPacket = {
  project: {
    key: string;
    name: string;
  };
  status: string;
  mode: string;
  readiness_status: string;
  allowed_files: ShimFilePlan[];
  required_preflight: string[];
  post_edit_verification: string[];
  rollback_scope: string[];
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  next_required_approval: string;
};

type ShimApplyGateItem = {
  key: string;
  category: string;
  status: string;
  message: string;
  expected: string;
  actual: string;
  required_evidence: string[];
  blocked_by: string[];
};

type ShimApplyGate = {
  project: {
    key: string;
    name: string;
  };
  status: string;
  mode: string;
  decision: string;
  message: string;
  items: ShimApplyGateItem[];
  required_packet_fields: string[];
  required_capabilities: string[];
  allowed_files: string[];
  forbidden_actions: string[];
  required_proof_facts: string[];
  safety_facts: Record<string, boolean>;
  approval_status: string;
  apply_command_eligible: boolean;
  apply_open: boolean;
  command_request_created: boolean;
  project_write_attempted: boolean;
  execution_write_attempted: boolean;
  engine_call_attempted: boolean;
  task_loop_run_forwarded: boolean;
  status_projection_written: boolean;
  area_matrix_files_modified: boolean;
  generated_at: string;
};

type ShimApplyPacket = {
  command_type: string;
  project_key: string;
  allowed_files: string[];
  approval_scope: string;
  authorization_snapshot_hash: string;
  expected_authorization_mode: string;
  failure_mode: string;
  explicit_approval: boolean;
};

type ShimApplyPacketPreview = {
  project: {
    key: string;
    name: string;
  };
  status: string;
  mode: string;
  decision: string;
  message: string;
  gate: ShimApplyGate;
  packet: ShimApplyPacket;
  apply_gate_command: string[];
  future_apply_command: string[];
  required_human_review: string[];
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  would_create_command_request_after_apply_command: boolean;
  would_write_area_matrix_shim_files_after_apply_command: boolean;
  would_write_status_projection_after_apply_command: boolean;
  command_request_created: boolean;
  project_write_attempted: boolean;
  execution_write_attempted: boolean;
  engine_call_attempted: boolean;
  task_loop_run_forwarded: boolean;
  status_projection_written: boolean;
  area_matrix_files_modified: boolean;
  generated_at: string;
};

type ExecutionCutoverReadinessItem = {
  key: string;
  category: string;
  status: string;
  message: string;
  required_evidence: string[];
  next_command: string;
  metadata: Record<string, unknown>;
};

type ExecutionCutoverReadiness = {
  project: {
    key: string;
    name: string;
  };
  status: string;
  mode: string;
  items: ExecutionCutoverReadinessItem[];
  migration_path: string[];
  command_evidence: Record<string, number>;
  capabilities: string[];
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  generated_at: string;
};

type ExecutionForwardingV1Readiness = {
  project: {
    key: string;
    name: string;
  };
  status: string;
  mode: string;
  items: ExecutionCutoverReadinessItem[];
  allowed_task_types: string[];
  command_evidence: Record<string, number>;
  capabilities: string[];
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  next_steps: unknown[];
  generated_at: string;
};

type ExecutionForwardingV1ApplyPreviewItem = {
  key: string;
  category: string;
  status: string;
  approval_status?: string;
  message: string;
  owner: string;
  required_evidence: string[];
  next_command: string;
  metadata: Record<string, unknown>;
};

type ExecutionForwardingV1ForwardingTarget = {
  task_type: string;
  target_command_type: string;
  target_status: string;
  required_capabilities: string[];
  required_packet_fields: string[];
  creates_command_request: boolean;
  creates_run: boolean;
  creates_run_task: boolean;
  creates_run_attempt: boolean;
  creates_artifact: boolean;
  creates_audit_event: boolean;
  project_write_allowed: boolean;
  execution_write_allowed: boolean;
  legacy_fallback_allowed: boolean;
  failure_mode: string;
};

type ExecutionForwardingV1BlockedTarget = {
  task_type: string;
  forbidden_action: string;
  reason: string;
  failure_mode: string;
  safety_facts: Record<string, boolean>;
};

type ExecutionForwardingV1ApplyPreview = {
  project: {
    key: string;
    name: string;
  };
  status: string;
  mode: string;
  readiness: ExecutionForwardingV1Readiness;
  items: ExecutionForwardingV1ApplyPreviewItem[];
  allowed_task_types: string[];
  forwarding_targets: ExecutionForwardingV1ForwardingTarget[];
  blocked_targets: ExecutionForwardingV1BlockedTarget[];
  required_capabilities: string[];
  apply_packet_fields: string[];
  fail_closed_fields: string[];
  required_proof_facts: string[];
  required_evidence: string[];
  forbidden_actions: string[];
  approval_required: boolean;
  approval_status: string;
  apply_open: boolean;
  rollback_target: string;
  safety_facts: Record<string, boolean>;
  generated_at: string;
};

type ExecutionForwardingV1ApplyGateItem = {
  key: string;
  category: string;
  status: string;
  message: string;
  expected: string;
  actual: string;
  required_evidence: string[];
  blocked_by: string[];
};

type ExecutionForwardingV1ApplyGate = {
  project: {
    key: string;
    name: string;
  };
  status: string;
  mode: string;
  decision: string;
  message: string;
  items: ExecutionForwardingV1ApplyGateItem[];
  required_packet_fields: string[];
  required_capabilities: string[];
  allowed_task_types: string[];
  target_command_types: string[];
  blocked_task_types: string[];
  forbidden_actions: string[];
  fail_closed_fields: string[];
  required_proof_facts: string[];
  safety_facts: Record<string, boolean>;
  approval_required: boolean;
  approval_status: string;
  apply_command_eligible: boolean;
  apply_open: boolean;
  command_request_created: boolean;
  area_flow_run_created: boolean;
  task_loop_run_forwarded: boolean;
  project_write_attempted: boolean;
  execution_write_attempted: boolean;
  engine_call_attempted: boolean;
  generated_at: string;
};

type ExecutionForwardingV1ApplyPacket = {
  command_type: string;
  project_key: string;
  allowed_task_types: string[];
  target_command_types: string[];
  approval_id: string;
  approval_scope: string;
  readiness_snapshot_hash: string;
  expected_shim_lifecycle_state: string;
  legacy_non_write_proof_id: string;
  rollback_plan_id: string;
  protected_path_fingerprint_id: string;
  idempotency_key: string;
  audit_correlation_id: string;
  failure_mode: string;
  explicit_approval: boolean;
  approval_actor: string;
  approval_reason: string;
};

type ExecutionForwardingV1ApplyPacketPreview = {
  project: {
    key: string;
    name: string;
  };
  status: string;
  mode: string;
  decision: string;
  message: string;
  apply_preview: ExecutionForwardingV1ApplyPreview;
  gate: ExecutionForwardingV1ApplyGate;
  packet: ExecutionForwardingV1ApplyPacket;
  apply_gate_command: string[];
  future_apply_command: string[];
  required_human_review: string[];
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  would_create_command_request_after_apply_command: boolean;
  would_create_run_after_apply_command: boolean;
  would_create_run_task_after_apply_command: boolean;
  would_create_audit_event_after_apply_command: boolean;
  command_request_created: boolean;
  area_flow_run_created: boolean;
  task_loop_run_forwarded: boolean;
  project_write_attempted: boolean;
  execution_write_attempted: boolean;
  engine_call_attempted: boolean;
  generated_at: string;
};

type ExecutionForwardingV1CommandPreview = {
  project: {
    key: string;
    name: string;
  };
  status: string;
  mode: string;
  decision: string;
  message: string;
  task_type: string;
  target_command_type: string;
  target_status: string;
  failure_mode: string;
  allowed_task_type: boolean;
  blocked_task_type: boolean;
  apply_open: boolean;
  would_create_command_request_after_approval: boolean;
  would_create_run_after_approval: boolean;
  would_create_run_task_after_approval: boolean;
  would_create_run_attempt_after_approval: boolean;
  would_create_artifact_after_approval: boolean;
  would_create_audit_event_after_approval: boolean;
  project_write_allowed: boolean;
  execution_write_allowed: boolean;
  legacy_fallback_allowed: boolean;
  required_packet_fields: string[];
  required_capabilities: string[];
  fail_closed_fields: string[];
  blocked_by: string[];
  allowed_task_types: string[];
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  generated_at: string;
};

type ExecutionForwardingV1RollbackPreviewItem = {
  key: string;
  category: string;
  status: string;
  message: string;
  owner: string;
  required_evidence: string[];
  next_command: string;
  metadata: Record<string, unknown>;
};

type ExecutionForwardingV1RollbackPreview = {
  project: {
    key: string;
    name: string;
  };
  status: string;
  mode: string;
  apply_preview: ExecutionForwardingV1ApplyPreview;
  items: ExecutionForwardingV1RollbackPreviewItem[];
  rollback_target: string;
  fail_closed_steps: string[];
  reopen_conditions: string[];
  required_proof_facts: string[];
  required_evidence: string[];
  forbidden_actions: string[];
  rollback_apply_open: boolean;
  safety_facts: Record<string, boolean>;
  generated_at: string;
};

type ReleaseFinalGateItem = {
  key: string;
  category: string;
  status: string;
  message: string;
  owner: string;
  required_evidence: string[];
  next_command: string;
  metadata: Record<string, unknown>;
};

type ReleaseFinalGate = {
  status: string;
  mode: string;
  scope: string;
  project_key?: string;
  items: ReleaseFinalGateItem[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

type ReleaseEvidenceBundleItem = {
  key: string;
  category: string;
  status: string;
  source: string;
  description: string;
  metadata: Record<string, unknown>;
};

type ReleaseEvidenceBundle = {
  status: string;
  mode: string;
  scope: string;
  project_key?: string;
  bundle_hash: string;
  items: ReleaseEvidenceBundleItem[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

type ReleasePackagePreviewItem = {
  key: string;
  category: string;
  status: string;
  package_path: string;
  source: string;
  description: string;
  metadata: Record<string, unknown>;
};

type ReleasePackagePreview = {
  status: string;
  mode: string;
  scope: string;
  project_key?: string;
  evidence_bundle: ReleaseEvidenceBundle;
  package_name: string;
  items: ReleasePackagePreviewItem[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

type ReleasePublishGateItem = {
  key: string;
  category: string;
  status: string;
  channel: string;
  message: string;
  owner: string;
  required_evidence: string[];
  next_command: string;
  metadata: Record<string, unknown>;
};

type ReleasePublishGate = {
  status: string;
  mode: string;
  scope: string;
  project_key?: string;
  items: ReleasePublishGateItem[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

type ReleaseDistributionPreviewItem = {
  key: string;
  category: string;
  status: string;
  channel: string;
  action: string;
  message: string;
  owner: string;
  required_evidence: string[];
  next_command: string;
  metadata: Record<string, unknown>;
};

type ReleaseDistributionPreview = {
  status: string;
  mode: string;
  scope: string;
  project_key?: string;
  items: ReleaseDistributionPreviewItem[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

type ReleasePublishApprovalPreviewItem = {
  key: string;
  category: string;
  status: string;
  approval_status: string;
  channel: string;
  message: string;
  owner: string;
  required_evidence: string[];
  next_command: string;
  metadata: Record<string, unknown>;
};

type ReleasePublishApprovalPreview = {
  status: string;
  mode: string;
  scope: string;
  project_key?: string;
  items: ReleasePublishApprovalPreviewItem[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

type ReleaseRolloutPlanPreviewItem = {
  key: string;
  category: string;
  status: string;
  stage: string;
  action: string;
  message: string;
  owner: string;
  required_evidence: string[];
  next_command: string;
  metadata: Record<string, unknown>;
};

type ReleaseRolloutPlanPreviewStep = {
  order: number;
  stage: string;
  action: string;
  description: string;
  blocked_by: string[];
};

type ReleaseRolloutPlanPreview = {
  status: string;
  mode: string;
  scope: string;
  project_key?: string;
  items: ReleaseRolloutPlanPreviewItem[];
  rollout_steps: ReleaseRolloutPlanPreviewStep[];
  verification_checkpoints: ReleaseRolloutPlanPreviewStep[];
  rollback_steps: ReleaseRolloutPlanPreviewStep[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

const API_BASE_KEY = "areaflow.desktop.apiBase";
const PROJECT_KEY = "areaflow.desktop.project";
const defaultAPIBase = "http://127.0.0.1:3847";
const defaultProject = "areamatrix";
const appRoot = document.querySelector<HTMLDivElement>("#app");

if (!appRoot) {
  throw new Error("missing #app root");
}

const app = appRoot;

let lastStatus: ServiceStatus | null = null;
let lastGate: ServiceControlGate | null = null;
let lastNotificationGate: NotificationGate | null = null;
let lastTrayMenuGate: TrayMenuGate | null = null;
let lastOperationsReadiness: OperationsReadiness | null = null;
let lastShimAuthorization: ShimAuthorizationPacket | null = null;
let lastShimApplyGate: ShimApplyGate | null = null;
let lastShimApplyPacket: ShimApplyPacketPreview | null = null;
let lastExecutionCutover: ExecutionCutoverReadiness | null = null;
let lastExecutionForwardingV1Readiness: ExecutionForwardingV1Readiness | null = null;
let lastExecutionForwardingV1ApplyPreview: ExecutionForwardingV1ApplyPreview | null = null;
let lastExecutionForwardingV1ApplyPacket: ExecutionForwardingV1ApplyPacketPreview | null = null;
let lastExecutionForwardingV1CommandPreviewAllowed: ExecutionForwardingV1CommandPreview | null = null;
let lastExecutionForwardingV1CommandPreviewBlocked: ExecutionForwardingV1CommandPreview | null = null;
let lastExecutionForwardingV1RollbackPreview: ExecutionForwardingV1RollbackPreview | null = null;
let lastReleaseFinalGate: ReleaseFinalGate | null = null;
let lastReleaseEvidenceBundle: ReleaseEvidenceBundle | null = null;
let lastReleasePackagePreview: ReleasePackagePreview | null = null;
let lastReleaseDistributionPreview: ReleaseDistributionPreview | null = null;
let lastReleasePublishGate: ReleasePublishGate | null = null;
let lastReleasePublishApproval: ReleasePublishApprovalPreview | null = null;
let lastReleaseRolloutPlan: ReleaseRolloutPlanPreview | null = null;
let lastError = "";

function configuredAPIBase(): string {
  const params = new URLSearchParams(window.location.search);
  const fromQuery = params.get("api");
  if (fromQuery) {
    localStorage.setItem(API_BASE_KEY, fromQuery);
    return fromQuery;
  }
  return localStorage.getItem(API_BASE_KEY) ?? defaultAPIBase;
}

function configuredProject(): string {
  const params = new URLSearchParams(window.location.search);
  const fromQuery = params.get("project");
  if (fromQuery) {
    localStorage.setItem(PROJECT_KEY, fromQuery);
    return fromQuery;
  }
  return localStorage.getItem(PROJECT_KEY) ?? defaultProject;
}

function configuredProjectQuery(): string {
  return `project_key=${encodeURIComponent(configuredProject())}`;
}

function serviceStatusURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/service/status`;
}

function serviceControlGateURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/desktop/service-control-gate`;
}

function notificationGateURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/desktop/notification-gate`;
}

function trayMenuGateURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/desktop/tray-menu-gate`;
}

function operationsReadinessURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/ops/readiness`;
}

function shimAuthorizationURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/projects/${configuredProject()}/shim-authorization`;
}

function shimApplyGateURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/projects/${configuredProject()}/shim-apply-gate`;
}

function shimApplyPacketURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/projects/${configuredProject()}/shim-apply-packet`;
}

function executionCutoverURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/projects/${configuredProject()}/execution-cutover-readiness`;
}

function executionForwardingV1ReadinessURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/projects/${configuredProject()}/execution-forwarding-v1-readiness`;
}

function executionForwardingV1ApplyPreviewURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/projects/${configuredProject()}/execution-forwarding-v1-apply-preview`;
}

function executionForwardingV1ApplyPacketURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/projects/${configuredProject()}/execution-forwarding-v1-apply-packet`;
}

function executionForwardingV1CommandPreviewURL(taskType: string): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/projects/${configuredProject()}/execution-forwarding-v1-command-preview?task_type=${encodeURIComponent(taskType)}`;
}

function executionForwardingV1RollbackPreviewURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/projects/${configuredProject()}/execution-forwarding-v1-rollback-preview`;
}

function releaseFinalGateURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/release/final-gate?${configuredProjectQuery()}`;
}

function releaseEvidenceBundleURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/release/evidence-bundle?${configuredProjectQuery()}`;
}

function releasePackagePreviewURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/release/package-preview?${configuredProjectQuery()}`;
}

function releaseDistributionPreviewURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/release/distribution-preview?${configuredProjectQuery()}`;
}

function releasePublishGateURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/release/publish-gate?${configuredProjectQuery()}`;
}

function releasePublishApprovalURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/release/publish-approval-preview?${configuredProjectQuery()}`;
}

function releaseRolloutPlanURL(): string {
  return `${configuredAPIBase().replace(/\/$/, "")}/api/v1/release/rollout-plan-preview?${configuredProjectQuery()}`;
}

async function loadDesktopState(): Promise<void> {
  lastError = "";
  try {
    const [statusResponse, gateResponse, notificationResponse, trayResponse, opsResponse] = await Promise.all([
      fetch(serviceStatusURL(), { headers: { Accept: "application/json" } }),
      fetch(serviceControlGateURL(), { headers: { Accept: "application/json" } }),
      fetch(notificationGateURL(), { headers: { Accept: "application/json" } }),
      fetch(trayMenuGateURL(), { headers: { Accept: "application/json" } }),
      fetch(operationsReadinessURL(), { headers: { Accept: "application/json" } }),
    ]);
    if (!statusResponse.ok) {
      throw new Error(`service status returned ${statusResponse.status}`);
    }
    if (!gateResponse.ok) {
      throw new Error(`service control gate returned ${gateResponse.status}`);
    }
    if (!notificationResponse.ok) {
      throw new Error(`notification gate returned ${notificationResponse.status}`);
    }
    if (!trayResponse.ok) {
      throw new Error(`tray menu gate returned ${trayResponse.status}`);
    }
    if (!opsResponse.ok) {
      throw new Error(`operations readiness returned ${opsResponse.status}`);
    }
    lastStatus = (await statusResponse.json()) as ServiceStatus;
    lastGate = (await gateResponse.json()) as ServiceControlGate;
    lastNotificationGate = (await notificationResponse.json()) as NotificationGate;
    lastTrayMenuGate = (await trayResponse.json()) as TrayMenuGate;
    lastOperationsReadiness = (await opsResponse.json()) as OperationsReadiness;
    lastShimAuthorization = await loadShimAuthorization();
    lastShimApplyGate = await loadShimApplyGate();
    lastShimApplyPacket = await loadShimApplyPacket();
    lastExecutionCutover = await loadExecutionCutover();
    lastExecutionForwardingV1Readiness = await loadExecutionForwardingV1Readiness();
    lastExecutionForwardingV1ApplyPreview = await loadExecutionForwardingV1ApplyPreview();
    lastExecutionForwardingV1ApplyPacket = await loadExecutionForwardingV1ApplyPacket();
    lastExecutionForwardingV1CommandPreviewAllowed = await loadExecutionForwardingV1CommandPreview("read_only_verify");
    lastExecutionForwardingV1CommandPreviewBlocked = await loadExecutionForwardingV1CommandPreview("engine_execution");
    lastExecutionForwardingV1RollbackPreview = await loadExecutionForwardingV1RollbackPreview();
    lastReleaseFinalGate = await loadReleaseFinalGate();
    lastReleaseEvidenceBundle = await loadReleaseEvidenceBundle();
    lastReleasePackagePreview = await loadReleasePackagePreview();
    lastReleaseDistributionPreview = await loadReleaseDistributionPreview();
    lastReleasePublishGate = await loadReleasePublishGate();
    lastReleasePublishApproval = await loadReleasePublishApproval();
    lastReleaseRolloutPlan = await loadReleaseRolloutPlan();
  } catch (error) {
    lastStatus = null;
    lastGate = null;
    lastNotificationGate = null;
    lastTrayMenuGate = null;
    lastOperationsReadiness = null;
    lastShimAuthorization = null;
    lastShimApplyGate = null;
    lastShimApplyPacket = null;
    lastExecutionCutover = null;
    lastExecutionForwardingV1Readiness = null;
    lastExecutionForwardingV1ApplyPreview = null;
    lastExecutionForwardingV1ApplyPacket = null;
    lastExecutionForwardingV1CommandPreviewAllowed = null;
    lastExecutionForwardingV1CommandPreviewBlocked = null;
    lastExecutionForwardingV1RollbackPreview = null;
    lastReleaseFinalGate = null;
    lastReleaseEvidenceBundle = null;
    lastReleasePackagePreview = null;
    lastReleaseDistributionPreview = null;
    lastReleasePublishGate = null;
    lastReleasePublishApproval = null;
    lastReleaseRolloutPlan = null;
    lastError = error instanceof Error ? error.message : String(error);
  }
  render();
}

async function loadShimAuthorization(): Promise<ShimAuthorizationPacket | null> {
  const response = await fetch(shimAuthorizationURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`shim authorization returned ${response.status}`);
  }
  return (await response.json()) as ShimAuthorizationPacket;
}

async function loadShimApplyPacket(): Promise<ShimApplyPacketPreview | null> {
  const response = await fetch(shimApplyPacketURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`shim apply packet returned ${response.status}`);
  }
  return (await response.json()) as ShimApplyPacketPreview;
}

async function loadShimApplyGate(): Promise<ShimApplyGate | null> {
  const response = await fetch(shimApplyGateURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`shim apply gate returned ${response.status}`);
  }
  return (await response.json()) as ShimApplyGate;
}

async function loadExecutionCutover(): Promise<ExecutionCutoverReadiness | null> {
  const response = await fetch(executionCutoverURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`execution cutover readiness returned ${response.status}`);
  }
  return (await response.json()) as ExecutionCutoverReadiness;
}

async function loadExecutionForwardingV1Readiness(): Promise<ExecutionForwardingV1Readiness | null> {
  const response = await fetch(executionForwardingV1ReadinessURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`execution forwarding v1 readiness returned ${response.status}`);
  }
  return (await response.json()) as ExecutionForwardingV1Readiness;
}

async function loadExecutionForwardingV1ApplyPreview(): Promise<ExecutionForwardingV1ApplyPreview | null> {
  const response = await fetch(executionForwardingV1ApplyPreviewURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`execution forwarding v1 apply preview returned ${response.status}`);
  }
  return (await response.json()) as ExecutionForwardingV1ApplyPreview;
}

async function loadExecutionForwardingV1ApplyPacket(): Promise<ExecutionForwardingV1ApplyPacketPreview | null> {
  const response = await fetch(executionForwardingV1ApplyPacketURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`execution forwarding v1 apply packet returned ${response.status}`);
  }
  return (await response.json()) as ExecutionForwardingV1ApplyPacketPreview;
}

async function loadExecutionForwardingV1CommandPreview(taskType: string): Promise<ExecutionForwardingV1CommandPreview | null> {
  const response = await fetch(executionForwardingV1CommandPreviewURL(taskType), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`execution forwarding v1 command preview returned ${response.status}`);
  }
  return (await response.json()) as ExecutionForwardingV1CommandPreview;
}

async function loadExecutionForwardingV1RollbackPreview(): Promise<ExecutionForwardingV1RollbackPreview | null> {
  const response = await fetch(executionForwardingV1RollbackPreviewURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`execution forwarding v1 rollback preview returned ${response.status}`);
  }
  return (await response.json()) as ExecutionForwardingV1RollbackPreview;
}

async function loadReleaseFinalGate(): Promise<ReleaseFinalGate | null> {
  const response = await fetch(releaseFinalGateURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`release final gate returned ${response.status}`);
  }
  return (await response.json()) as ReleaseFinalGate;
}

async function loadReleaseEvidenceBundle(): Promise<ReleaseEvidenceBundle | null> {
  const response = await fetch(releaseEvidenceBundleURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`release evidence bundle returned ${response.status}`);
  }
  return (await response.json()) as ReleaseEvidenceBundle;
}

async function loadReleasePackagePreview(): Promise<ReleasePackagePreview | null> {
  const response = await fetch(releasePackagePreviewURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`release package preview returned ${response.status}`);
  }
  return (await response.json()) as ReleasePackagePreview;
}

async function loadReleaseDistributionPreview(): Promise<ReleaseDistributionPreview | null> {
  const response = await fetch(releaseDistributionPreviewURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`release distribution preview returned ${response.status}`);
  }
  return (await response.json()) as ReleaseDistributionPreview;
}

async function loadReleasePublishGate(): Promise<ReleasePublishGate | null> {
  const response = await fetch(releasePublishGateURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`release publish gate returned ${response.status}`);
  }
  return (await response.json()) as ReleasePublishGate;
}

async function loadReleasePublishApproval(): Promise<ReleasePublishApprovalPreview | null> {
  const response = await fetch(releasePublishApprovalURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`release publish approval preview returned ${response.status}`);
  }
  return (await response.json()) as ReleasePublishApprovalPreview;
}

async function loadReleaseRolloutPlan(): Promise<ReleaseRolloutPlanPreview | null> {
  const response = await fetch(releaseRolloutPlanURL(), { headers: { Accept: "application/json" } });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`release rollout plan preview returned ${response.status}`);
  }
  return (await response.json()) as ReleaseRolloutPlanPreview;
}

function render(): void {
  app.innerHTML = "";
  const shell = document.createElement("section");
  shell.className = "shell";
  shell.append(header());
  shell.append(statusPanel());
  if (lastStatus) {
    shell.append(componentGrid(lastStatus));
    shell.append(guardrailPanel(lastStatus));
  }
  if (lastOperationsReadiness) {
    shell.append(operationsReadinessPanel(lastOperationsReadiness));
  }
  if (lastGate) {
    shell.append(serviceControlPanel(lastGate));
  }
  if (lastNotificationGate) {
    shell.append(notificationPanel(lastNotificationGate));
  }
  if (lastTrayMenuGate) {
    shell.append(trayMenuPanel(lastTrayMenuGate));
  }
  if (lastShimAuthorization) {
    shell.append(shimAuthorizationPanel(lastShimAuthorization));
  }
  if (lastShimApplyGate) {
    shell.append(shimApplyGatePanel(lastShimApplyGate));
  }
  if (lastShimApplyPacket) {
    shell.append(shimApplyPacketPanel(lastShimApplyPacket));
  }
  if (lastExecutionCutover) {
    shell.append(executionCutoverPanel(lastExecutionCutover));
  }
  if (lastExecutionForwardingV1Readiness) {
    shell.append(executionForwardingV1ReadinessPanel(lastExecutionForwardingV1Readiness));
  }
  if (lastExecutionForwardingV1ApplyPreview) {
    shell.append(executionForwardingV1ApplyPreviewPanel(lastExecutionForwardingV1ApplyPreview));
  }
  if (lastExecutionForwardingV1ApplyPacket) {
    shell.append(executionForwardingV1ApplyPacketPanel(lastExecutionForwardingV1ApplyPacket));
  }
  if (lastExecutionForwardingV1CommandPreviewAllowed || lastExecutionForwardingV1CommandPreviewBlocked) {
    shell.append(
      executionForwardingV1CommandPreviewPanel(
        lastExecutionForwardingV1CommandPreviewAllowed,
        lastExecutionForwardingV1CommandPreviewBlocked,
      ),
    );
  }
  if (lastExecutionForwardingV1RollbackPreview) {
    shell.append(executionForwardingV1RollbackPreviewPanel(lastExecutionForwardingV1RollbackPreview));
  }
  if (lastReleaseFinalGate) {
    shell.append(releaseFinalGatePanel(lastReleaseFinalGate));
  }
  if (lastReleaseEvidenceBundle) {
    shell.append(releaseEvidenceBundlePanel(lastReleaseEvidenceBundle));
  }
  if (lastReleasePackagePreview) {
    shell.append(releasePackagePreviewPanel(lastReleasePackagePreview));
  }
  if (lastReleaseDistributionPreview) {
    shell.append(releaseDistributionPreviewPanel(lastReleaseDistributionPreview));
  }
  if (lastReleasePublishGate) {
    shell.append(releasePublishGatePanel(lastReleasePublishGate));
  }
  if (lastReleasePublishApproval) {
    shell.append(releasePublishApprovalPanel(lastReleasePublishApproval));
  }
  if (lastReleaseRolloutPlan) {
    shell.append(releaseRolloutPlanPanel(lastReleaseRolloutPlan));
  }
  app.append(shell);
}

function header(): HTMLElement {
  const element = document.createElement("header");
  element.className = "header";
  element.innerHTML = `
    <div>
      <p class="eyebrow">AreaFlow Desktop</p>
      <h1>Local Service Shell</h1>
    </div>
    <button type="button" id="refresh">Refresh</button>
  `;
  element.querySelector("button")?.addEventListener("click", () => {
    void loadDesktopState();
  });
  return element;
}

function statusPanel(): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel primary";
  if (!lastStatus) {
    panel.innerHTML = `
      <div>
        <p class="label">Service</p>
        <strong class="status blocked">blocked</strong>
      </div>
      <p>${lastError || "Loading service status..."}</p>
      <p class="mono">${serviceStatusURL()}</p>
    `;
    return panel;
  }

  panel.innerHTML = `
    <div>
      <p class="label">Service</p>
      <strong class="status ${lastStatus.status}">${lastStatus.status}</strong>
    </div>
    <p>${lastStatus.mode}</p>
    <p class="mono">${serviceStatusURL()}</p>
  `;
  const link = document.createElement("a");
  link.className = "buttonLink";
  link.href = lastStatus.dashboard.url;
  link.textContent = "Open dashboard";
  panel.append(link);
  return panel;
}

function componentGrid(status: ServiceStatus): HTMLElement {
  const grid = document.createElement("section");
  grid.className = "grid";
  grid.append(componentCard("API", status.api.status, status.api.message));
  grid.append(componentCard("PostgreSQL", status.database.status, status.database.message));
  grid.append(
    componentCard(
      "Worker pool",
      status.worker_pool.status,
      `${status.worker_pool.total_online_workers}/${status.worker_pool.total_workers} online, ${status.worker_pool.total_queued_tasks} queued, ${status.worker_pool.total_needs_recovery} recovery`,
    ),
  );
  grid.append(componentCard("Dashboard", status.dashboard.status, status.dashboard.url));
  return grid;
}

function componentCard(title: string, status: string, message: string): HTMLElement {
  const card = document.createElement("article");
  card.className = "card";
  card.innerHTML = `
    <p class="label">${title}</p>
    <strong class="status ${status}">${status}</strong>
    <p>${message}</p>
  `;
  return card;
}

function guardrailPanel(status: ServiceStatus): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Capabilities</p>
      <p>${status.capabilities.join(", ")}</p>
    </div>
    <div>
      <p class="label">Forbidden</p>
      <p>${status.forbidden_actions.join(", ")}</p>
    </div>
    <p class="mono">generated_at=${status.generated_at}</p>
  `;
  return panel;
}

function operationsReadinessPanel(readiness: OperationsReadiness): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Operations Readiness</p>
      <strong class="status ${readiness.status}">${readiness.status}</strong>
      <p>${readiness.mode}</p>
    </div>
    <div class="grid"></div>
    <div class="actionGrid"></div>
    <p class="mono">${operationsSafetyFacts(readiness).join(" ")}</p>
    <p class="mono">generated_at=${readiness.generated_at}</p>
  `;
  const grid = panel.querySelector(".grid");
  grid?.append(componentCard("Support bundle", readiness.support_bundle.status, readiness.support_bundle.mode));
  grid?.append(
    componentCard(
      "Migration ledger",
      readiness.migration_ledger.status,
      `${readiness.migration_ledger.applied_count} applied, ${readiness.migration_ledger.pending_count} pending`,
    ),
  );
  grid?.append(componentCard("Telemetry", "ready", readiness.telemetry_default));
  grid?.append(componentCard("Managed ops", "warn", readiness.managed_ops_status));

  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of readiness.items) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.category}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.key}</p>
      <p class="mono">${item.blocked_by.join(", ") || item.next_command || item.message}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function serviceControlPanel(gate: ServiceControlGate): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Service Control Gate</p>
      <strong class="status ${gate.status}">${gate.status}</strong>
      <p>${gate.mode}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">${safetyFacts(gate).join(" ")}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const action of gate.actions) {
    actionGrid?.append(actionCard(action));
  }
  return panel;
}

function actionCard(action: ServiceControlAction): HTMLElement {
  const card = document.createElement("article");
  card.className = "card";
  card.innerHTML = `
    <p class="label">${action.category}</p>
    <strong class="status ${action.status}">${action.status}</strong>
    <p>${action.label}</p>
    <p class="mono">${action.default_ui_state} ${action.risk_level}</p>
  `;
  if (action.blockers.length > 0) {
    const blockers = document.createElement("p");
    blockers.className = "mono";
    blockers.textContent = action.blockers.join(", ");
    card.append(blockers);
  }
  return card;
}

function notificationPanel(gate: NotificationGate): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Notification Gate</p>
      <strong class="status ${gate.status}">${gate.status}</strong>
      <p>${gate.mode}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">${notificationSafetyFacts(gate).join(" ")}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const action of gate.actions) {
    actionGrid?.append(actionCard(action));
  }
  return panel;
}

function trayMenuPanel(gate: TrayMenuGate): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Tray Menu Gate</p>
      <strong class="status ${gate.status}">${gate.status}</strong>
      <p>${gate.mode}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">${trayMenuSafetyFacts(gate).join(" ")}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const action of gate.actions) {
    actionGrid?.append(actionCard(action));
  }
  return panel;
}

function shimAuthorizationPanel(packet: ShimAuthorizationPacket): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Shim Authorization</p>
      <strong class="status ${packet.status}">${packet.status}</strong>
      <p>${packet.project.key} ${packet.mode}</p>
    </div>
    <div class="actionGrid"></div>
    <div class="actionGrid shimGateGrid"></div>
    <p class="mono">${shimAuthorizationSafetyFacts(packet).join(" ")}</p>
    <p class="mono">${packet.next_required_approval}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const file of packet.allowed_files) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${file.required ? "Required" : "Optional"}</p>
      <strong class="status ${packet.readiness_status}">${packet.readiness_status}</strong>
      <p>${file.path}</p>
      <p class="mono">${file.action} ${file.boundary}</p>
    `;
    actionGrid?.append(card);
  }
  const gateGrid = panel.querySelector(".shimGateGrid");
  gateGrid?.append(shimAuthorizationGateCard("Required Preflight", packet.required_preflight));
  gateGrid?.append(shimAuthorizationGateCard("Post-edit Verification", packet.post_edit_verification));
  gateGrid?.append(shimAuthorizationGateCard("Rollback Scope", packet.rollback_scope));
  return panel;
}

function shimAuthorizationGateCard(title: string, items: string[]): HTMLElement {
  const card = document.createElement("article");
  card.className = "card";
  card.innerHTML = `
    <p class="label">${title}</p>
    <strong class="status blocked">${items.length}</strong>
    <p class="mono">${items.slice(0, 3).join(" | ")}</p>
  `;
  return card;
}

function shimApplyPacketPanel(preview: ShimApplyPacketPreview): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Shim Apply Review</p>
      <strong class="status ${preview.status}">${preview.status}</strong>
      <p>${preview.project.key} ${preview.decision}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">command=${preview.packet.command_type} gate=${preview.gate.status} eligible=${preview.gate.apply_command_eligible} files=${preview.packet.allowed_files.length}</p>
    <p class="mono">proof=${preview.gate.required_proof_facts.slice(0, 6).join(" ")}</p>
    <p class="mono">${shimApplySafetyFacts(preview).join(" ")}</p>
    <p class="mono">generated_at=${preview.generated_at}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of preview.gate.items.slice(0, 8)) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.category}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.key}</p>
      <p class="mono">${item.blocked_by.join(",") || item.expected}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function shimApplyGatePanel(gate: ShimApplyGate): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Shim Apply Gate</p>
      <strong class="status ${gate.status}">${gate.status}</strong>
      <p>${gate.project.key} ${gate.decision}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">mode=${gate.mode} approval=${gate.approval_status} eligible=${gate.apply_command_eligible} open=${gate.apply_open}</p>
    <p class="mono">fields=${gate.required_packet_fields.slice(0, 6).join(" ")} capabilities=${gate.required_capabilities.slice(0, 4).join(" ")}</p>
    <p class="mono">${shimApplyGateSafetyFacts(gate).join(" ")}</p>
    <p class="mono">generated_at=${gate.generated_at}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of gate.items.slice(0, 8)) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.category}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.key}</p>
      <p class="mono">${item.blocked_by.join(",") || item.expected || item.message}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function executionCutoverPanel(readiness: ExecutionCutoverReadiness): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Execution Cutover</p>
      <strong class="status ${readiness.status}">${readiness.status}</strong>
      <p>${readiness.project.key} ${readiness.mode}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">${executionCutoverSafetyFacts(readiness).join(" ")}</p>
    <p class="mono">generated_at=${readiness.generated_at}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of readiness.items.slice(0, 8)) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.category}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.key}</p>
      <p class="mono">${item.next_command}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function executionForwardingV1ReadinessPanel(readiness: ExecutionForwardingV1Readiness): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Forwarding v1 Readiness</p>
      <strong class="status ${readiness.status}">${readiness.status}</strong>
      <p>${readiness.project.key} ${readiness.mode}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">tasks=${readiness.allowed_task_types.join(",")}</p>
    <p class="mono">${executionForwardingV1ReadinessSafetyFacts(readiness).join(" ")}</p>
    <p class="mono">generated_at=${readiness.generated_at}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of readiness.items.slice(0, 8)) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.category}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.key}</p>
      <p class="mono">${item.next_command}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function executionForwardingV1ApplyPreviewPanel(preview: ExecutionForwardingV1ApplyPreview): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Forwarding v1 Apply Preview</p>
      <strong class="status ${preview.status}">${preview.status}</strong>
      <p>${preview.mode} approval=${preview.approval_status}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">apply_open=${preview.apply_open} rollback=${preview.rollback_target} targets=${preview.forwarding_targets.length} blocked=${preview.blocked_targets.length}</p>
    <p class="mono">${executionForwardingV1ApplySafetyFacts(preview).join(" ")}</p>
    <p class="mono">generated_at=${preview.generated_at}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of preview.items.slice(0, 8)) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.category}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.key}</p>
      <p class="mono">${item.approval_status || item.next_command}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function executionForwardingV1ApplyPacketPanel(preview: ExecutionForwardingV1ApplyPacketPreview): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Forwarding v1 Packet Gate</p>
      <strong class="status ${preview.status}">${preview.status}</strong>
      <p>${preview.mode} decision=${preview.decision}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">hash=${shortValue(preview.packet.readiness_snapshot_hash)} eligible=${preview.gate.apply_command_eligible} command_created=${preview.command_request_created}</p>
    <p class="mono">legacy_ref=${shortValue(preview.packet.legacy_non_write_proof_id)} rollback_ref=${shortValue(preview.packet.rollback_plan_id)}</p>
    <p class="mono">fingerprint_ref=${shortValue(preview.packet.protected_path_fingerprint_id)}</p>
  `;
  const reviewKeys = new Set([
    "readiness_snapshot_hash",
    "legacy_non_write_proof_id",
    "rollback_plan_id",
    "protected_path_fingerprint_id",
    "explicit_approval",
    "read_only_shim",
  ]);
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of preview.gate.items.filter((gateItem) => reviewKeys.has(gateItem.key))) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.category}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.key}</p>
      <p class="mono">expected=${shortValue(item.expected)} actual=${shortValue(item.actual)}</p>
      <p class="mono">blockers=${item.blocked_by.join(",") || "none"}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function executionForwardingV1CommandPreviewPanel(
  allowed: ExecutionForwardingV1CommandPreview | null,
  blocked: ExecutionForwardingV1CommandPreview | null,
): HTMLElement {
  const status = allowed?.status ?? blocked?.status ?? "unknown";
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Forwarding v1 Command Preview</p>
      <strong class="status ${status}">${status}</strong>
      <p>${allowed?.mode ?? blocked?.mode ?? "read_only_execution_forwarding_v1_command_preview"}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">allowed=${allowed?.decision ?? "missing"} blocked=${blocked?.decision ?? "missing"}</p>
    <p class="mono">command_created=${allowed?.safety_facts.area_flow_command_created ?? false} task_loop=${allowed?.safety_facts.task_loop_run_forwarded ?? false} engine=${blocked?.safety_facts.engine_call_attempted ?? false}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of [allowed, blocked]) {
    if (!item) {
      continue;
    }
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.task_type}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.decision}</p>
      <p class="mono">${item.target_command_type || item.blocked_by.join(",")}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function executionForwardingV1RollbackPreviewPanel(preview: ExecutionForwardingV1RollbackPreview): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Forwarding v1 Rollback Preview</p>
      <strong class="status ${preview.status}">${preview.status}</strong>
      <p>${preview.mode} target=${preview.rollback_target}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">rollback_apply_open=${preview.rollback_apply_open}</p>
    <p class="mono">fail_closed=${preview.fail_closed_steps.length} reopen=${preview.reopen_conditions.length}</p>
    <p class="mono">${executionForwardingV1RollbackSafetyFacts(preview).join(" ")}</p>
    <p class="mono">generated_at=${preview.generated_at}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of preview.items.slice(0, 8)) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.category}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.key}</p>
      <p class="mono">${item.next_command}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function releaseFinalGatePanel(gate: ReleaseFinalGate): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Release Final Gate</p>
      <strong class="status ${gate.status}">${gate.status}</strong>
      <p>${gate.mode}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">forbidden=${gate.forbidden_actions.join(",")}</p>
    <p class="mono">generated_at=${gate.generated_at}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of gate.items) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.category}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.key}</p>
      <p class="mono">${item.next_command}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function releaseEvidenceBundlePanel(bundle: ReleaseEvidenceBundle): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Release Evidence</p>
      <strong class="status ${bundle.status}">${bundle.status}</strong>
      <p>${bundle.mode}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">bundle_hash=${bundle.bundle_hash}</p>
    <p class="mono">forbidden=${bundle.forbidden_actions.join(",")}</p>
    <p class="mono">generated_at=${bundle.generated_at}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of bundle.items.slice(0, 6)) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.category}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.key}</p>
      <p class="mono">${item.source}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function releasePackagePreviewPanel(preview: ReleasePackagePreview): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Release Package</p>
      <strong class="status ${preview.status}">${preview.status}</strong>
      <p>${preview.mode}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">${preview.package_name}</p>
    <p class="mono">bundle_hash=${preview.evidence_bundle.bundle_hash}</p>
    <p class="mono">forbidden=${preview.forbidden_actions.join(",")}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of preview.items.slice(0, 6)) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.category}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.key}</p>
      <p class="mono">${item.package_path}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function releaseDistributionPreviewPanel(preview: ReleaseDistributionPreview): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Release Distribution</p>
      <strong class="status ${preview.status}">${preview.status}</strong>
      <p>${preview.mode}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">forbidden=${preview.forbidden_actions.join(",")}</p>
    <p class="mono">generated_at=${preview.generated_at}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of preview.items) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.channel}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.key}</p>
      <p class="mono">${item.action}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function releasePublishGatePanel(gate: ReleasePublishGate): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Release Publish</p>
      <strong class="status ${gate.status}">${gate.status}</strong>
      <p>${gate.mode}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">forbidden=${gate.forbidden_actions.join(",")}</p>
    <p class="mono">generated_at=${gate.generated_at}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of gate.items) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.category}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.key}</p>
      <p class="mono">${item.channel} ${item.next_command}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function releasePublishApprovalPanel(preview: ReleasePublishApprovalPreview): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Release Approval</p>
      <strong class="status ${preview.status}">${preview.status}</strong>
      <p>${preview.mode}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">forbidden=${preview.forbidden_actions.join(",")}</p>
    <p class="mono">generated_at=${preview.generated_at}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of preview.items) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.channel}</p>
      <strong class="status ${item.status}">${item.approval_status}</strong>
      <p>${item.key}</p>
      <p class="mono">${item.next_command}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function releaseRolloutPlanPanel(preview: ReleaseRolloutPlanPreview): HTMLElement {
  const panel = document.createElement("section");
  panel.className = "panel";
  panel.innerHTML = `
    <div>
      <p class="label">Release Rollout</p>
      <strong class="status ${preview.status}">${preview.status}</strong>
      <p>${preview.mode}</p>
    </div>
    <div class="actionGrid"></div>
    <p class="mono">steps=${preview.rollout_steps.length} verify=${preview.verification_checkpoints.length} rollback=${preview.rollback_steps.length}</p>
    <p class="mono">forbidden=${preview.forbidden_actions.join(",")}</p>
  `;
  const actionGrid = panel.querySelector(".actionGrid");
  for (const item of preview.items) {
    const card = document.createElement("article");
    card.className = "card";
    card.innerHTML = `
      <p class="label">${item.stage}</p>
      <strong class="status ${item.status}">${item.status}</strong>
      <p>${item.key}</p>
      <p class="mono">${item.action}</p>
    `;
    actionGrid?.append(card);
  }
  return panel;
}

function safetyFacts(gate: ServiceControlGate): string[] {
  return [
    `db_write=${gate.db_write_attempted}`,
    `project_write=${gate.project_write_attempted}`,
    `process_control=${gate.process_control_attempted}`,
    `command_created=${gate.command_created}`,
    `worker_scheduled=${gate.worker_scheduled}`,
    `workflow_started=${gate.workflow_execution_started}`,
    `secrets=${gate.secrets_resolved}`,
    `network=${gate.network_used}`,
  ];
}

function notificationSafetyFacts(gate: NotificationGate): string[] {
  return [
    `db_write=${gate.db_write_attempted}`,
    `project_write=${gate.project_write_attempted}`,
    `event_stream_opened=${gate.event_stream_opened}`,
    `notification_requested=${gate.notification_requested}`,
    `command_created=${gate.command_created}`,
    `worker_scheduled=${gate.worker_scheduled}`,
    `workflow_started=${gate.workflow_execution_started}`,
    `secrets=${gate.secrets_resolved}`,
    `network=${gate.network_used}`,
  ];
}

function trayMenuSafetyFacts(gate: TrayMenuGate): string[] {
  return [
    `db_write=${gate.db_write_attempted}`,
    `project_write=${gate.project_write_attempted}`,
    `tray_created=${gate.tray_menu_created}`,
    `os_integration=${gate.os_integration_requested}`,
    `command_created=${gate.command_created}`,
    `service_control=${gate.service_control_attempted}`,
    `notification=${gate.notification_requested}`,
    `worker_scheduled=${gate.worker_scheduled}`,
    `workflow_started=${gate.workflow_execution_started}`,
    `secrets=${gate.secrets_resolved}`,
    `network=${gate.network_used}`,
  ];
}

function operationsSafetyFacts(readiness: OperationsReadiness): string[] {
  return [
    `support_exported=${readiness.safety_facts.support_bundle_exported}`,
    `metadata_only=${readiness.safety_facts.support_bundle_metadata_only}`,
    `remote_telemetry=${readiness.safety_facts.remote_telemetry_enabled}`,
    `managed_upgrade=${readiness.safety_facts.managed_upgrade_attempted}`,
    `service_control=${readiness.safety_facts.service_process_control_attempted}`,
    `db_write=${readiness.safety_facts.database_write_attempted}`,
    `project_write=${readiness.safety_facts.project_write_attempted}`,
    `protected_paths=${readiness.safety_facts.area_matrix_protected_paths_touched}`,
  ];
}

function shimAuthorizationSafetyFacts(packet: ShimAuthorizationPacket): string[] {
  return [
    `project_write=${packet.safety_facts.project_write_attempted}`,
    `execution_write=${packet.safety_facts.execution_write_attempted}`,
    `task_loop_run=${packet.safety_facts.task_loop_run_forwarded}`,
    `engine_call=${packet.safety_facts.engine_call_attempted}`,
    `commands=${packet.safety_facts.commands_run}`,
    `network=${packet.safety_facts.network_used}`,
  ];
}

function shimApplySafetyFacts(preview: ShimApplyPacketPreview): string[] {
  return [
    `command_created=${preview.command_request_created}`,
    `project_write=${preview.project_write_attempted}`,
    `execution_write=${preview.execution_write_attempted}`,
    `task_loop=${preview.task_loop_run_forwarded}`,
    `status_projection=${preview.status_projection_written}`,
    `area_matrix_files=${preview.area_matrix_files_modified}`,
    `engine_call=${preview.engine_call_attempted}`,
  ];
}

function shimApplyGateSafetyFacts(gate: ShimApplyGate): string[] {
  return [
    `command_created=${gate.command_request_created}`,
    `project_write=${gate.project_write_attempted}`,
    `execution_write=${gate.execution_write_attempted}`,
    `task_loop=${gate.task_loop_run_forwarded}`,
    `status_projection=${gate.status_projection_written}`,
    `area_matrix_files=${gate.area_matrix_files_modified}`,
    `engine_call=${gate.engine_call_attempted}`,
  ];
}

function executionCutoverSafetyFacts(readiness: ExecutionCutoverReadiness): string[] {
  return [
    `execution_cutover_apply=${readiness.safety_facts.execution_cutover_apply_open}`,
    `project_write=${readiness.safety_facts.project_write_attempted}`,
    `execution_write=${readiness.safety_facts.execution_write_attempted}`,
    `task_loop_run_forwarded=${readiness.safety_facts.task_loop_run_forwarded}`,
    `worker_scheduled=${readiness.safety_facts.worker_scheduled}`,
    `engine_call=${readiness.safety_facts.engine_call_attempted}`,
  ];
}

function shortValue(value: string): string {
  if (!value) {
    return "missing";
  }
  if (value.length <= 28) {
    return value;
  }
  return `${value.slice(0, 14)}...${value.slice(-10)}`;
}

function executionForwardingV1ReadinessSafetyFacts(readiness: ExecutionForwardingV1Readiness): string[] {
  return [
    `apply_open=${readiness.safety_facts.apply_open ?? false}`,
    `project_write=${readiness.safety_facts.project_write_attempted}`,
    `execution_write=${readiness.safety_facts.execution_write_attempted}`,
    `engine_call=${readiness.safety_facts.engine_call_attempted}`,
    `network=${readiness.safety_facts.network_used}`,
  ];
}

function executionForwardingV1ApplySafetyFacts(preview: ExecutionForwardingV1ApplyPreview): string[] {
  return [
    `forwarding_apply=${preview.safety_facts.forwarding_v1_apply_open}`,
    `task_loop_run=${preview.safety_facts.task_loop_run_forwarded}`,
    `project_write=${preview.safety_facts.project_write_attempted}`,
    `worker_scheduled=${preview.safety_facts.worker_scheduled}`,
    `secret=${preview.safety_facts.secrets_resolved}`,
    `network=${preview.safety_facts.network_used}`,
  ];
}

function executionForwardingV1RollbackSafetyFacts(preview: ExecutionForwardingV1RollbackPreview): string[] {
  return [
    `rollback_apply=${preview.rollback_apply_open}`,
    `apply_open=${preview.safety_facts.apply_open}`,
    `forwarding_apply=${preview.safety_facts.forwarding_v1_apply_open}`,
    `commands=${preview.safety_facts.commands_run}`,
    `project_write=${preview.safety_facts.project_write_attempted}`,
    `protected_paths=${preview.safety_facts.areamatrix_protected_paths_touched}`,
  ];
}

render();
void loadDesktopState();
