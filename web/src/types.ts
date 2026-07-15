export type ProjectRecord = {
  key: string;
  name: string;
  kind: string;
  adapter: string;
  workflow_profile: string;
  default_branch: string;
  root: string;
};

export type ProjectListResponse = {
  projects: ProjectRecord[];
};

export type ProjectSummary = {
  project: ProjectRecord;
  config?: {
    protocol_version: number;
    config_path: string;
    config_hash: string;
    ownership: Record<string, unknown>;
    status_export: Record<string, unknown>;
    migration: Record<string, unknown>;
    loaded_at: string;
  };
  inventory: {
    versions: number;
    residuals: number;
    artifacts: number;
    import_snapshots: number;
    mirror_exports: number;
  };
  import?: {
    source_hash: string;
    created_at: string;
    history_ready_for_diff: boolean;
    source_hash_changed_since_previous: boolean;
  };
  doctor?: {
    status: string;
    drift_status: string;
    config_drift_status: string;
    stage_coverage_status: string;
    native_doctor_status: string;
    severity: string;
    created_at: string;
  };
};

export type ProjectReadinessItem = {
  key: string;
  status: string;
  message: string;
  metadata: Record<string, unknown>;
};

export type ProjectReadiness = {
  project: ProjectRecord;
  status: string;
  items: ProjectReadinessItem[];
  summary: ProjectSummary;
};

export type WorkflowVersion = {
  id: number;
  project_id?: number;
  display_label: string;
  version_kind: string;
  lifecycle_status: string;
  import_mode: string;
  immutable: boolean;
  status_summary: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  imported_at?: string;
};

export type WorkflowVersionListResponse = {
  project: ProjectRecord;
  workflow_versions: WorkflowVersion[];
};

export type WorkflowCollectionResponse = {
  workflows: Array<{ project: ProjectRecord; workflow_version: WorkflowVersion }>;
  count: number;
  next_cursor?: string;
};

export type WorkflowItem = {
  id: number;
  workflow_version_id: number;
  stage: string;
  item_type: string;
  external_key: string;
  title: string;
  status: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  imported_at?: string;
};

export type WorkflowItemLink = {
  id: number;
  workflow_version_id: number;
  from_item_id: number;
  to_item_id: number;
  relation_type: string;
  metadata: Record<string, unknown>;
  created_at: string;
};

export type WorkflowStagesResponse = {
  project: ProjectRecord;
  workflow_version: WorkflowVersion;
  items: WorkflowItem[];
  links: WorkflowItemLink[];
};

export type ArtifactRecord = {
  id: number;
  project_id?: number;
  workflow_version_id: number;
  run_id?: number;
  workflow_item_id?: number;
  artifact_type: string;
  storage_backend: string;
  uri: string;
  source_path: string;
  sha256: string;
  size_bytes: number;
  content_type: string;
  metadata: Record<string, unknown>;
  created_at: string;
};

export type ArtifactListResponse = {
  project: ProjectRecord;
  workflow_version?: WorkflowVersion;
  artifacts: ArtifactRecord[];
};

export type ResidualRecord = {
  id: number;
  workflow_version_id?: number;
  residual_key: string;
  status: string;
  type: string;
  title: string;
  source_path: string;
  current_impact: string;
  executable_task: boolean;
  promotion_required: boolean;
  close_condition: string;
  metadata: Record<string, unknown>;
  immutable: boolean;
  created_at: string;
  updated_at: string;
  imported_at?: string;
};

export type ResidualListResponse = {
  project: ProjectRecord;
  workflow_version?: WorkflowVersion;
  residuals: ResidualRecord[];
};

export type ApprovalRecord = {
  id: number;
  workflow_version_id: number;
  transition_preview_id?: number;
  approval_kind: string;
  decision: string;
  scope_type: string;
  scope_id: string;
  actor: string;
  reason: string;
  risk_level: string;
  metadata: Record<string, unknown>;
  created_at: string;
};

export type ApprovalRecordsResponse = {
  project: ProjectRecord;
  workflow_version: WorkflowVersion;
  approval_records: ApprovalRecord[];
};

export type TransitionPreview = {
  id: number;
  workflow_version_id: number;
  from_stage: string;
  to_stage: string;
  status: string;
  required_gate_name: string;
  gate_result_id?: number;
  blockers: string[];
  warnings: string[];
  metadata: Record<string, unknown>;
  created_at: string;
};

export type TransitionPreviewsResponse = {
  project: ProjectRecord;
  workflow_version: WorkflowVersion;
  transition_previews: TransitionPreview[];
};

export type AuthStatus = {
  mode: "disabled" | "token" | "oidc";
  requires_token: boolean;
  requires_login: boolean;
  login_url: string;
};

export type AuthPrincipal = {
  actor: string;
  token_key: string;
  user_id: number;
  auth_mode: "" | "token" | "oidc";
  roles: string[];
  projects: string[];
  capabilities: string[];
  scope_hash: string;
};

export type RoleBinding = {
  id: number;
  project_id?: number;
  project_key?: string;
  user_id?: number;
  team_id?: number;
  role: string;
  status: string;
  reason: string;
  expires_at?: string;
  created_at: string;
};

export type RoleBindingsResponse = {
  project_key: string;
  role_bindings: RoleBinding[];
};

export type RunRecord = {
  id: number;
  project_id?: number;
  workflow_version_id: number;
  run_type: string;
  run_kind: string;
  status: string;
  risk_level: string;
  risk_policy: string;
  dry_run: boolean;
  summary: Record<string, unknown>;
  metadata: Record<string, unknown>;
  started_at: string;
  finished_at?: string;
};

export type RunTaskRecord = {
  id: number;
  project_id?: number;
  workflow_version_id: number;
  workflow_item_id?: number;
  run_id: number;
  task_key: string;
  task_kind: string;
  status: string;
  risk_level: string;
  sequence: number;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type RunAttemptRecord = {
  id: number;
  project_id?: number;
  workflow_version_id: number;
  workflow_item_id?: number;
  run_id: number;
  run_task_id?: number;
  attempt_kind: string;
  status: string;
  dry_run: boolean;
  metadata: Record<string, unknown>;
  started_at: string;
  finished_at?: string;
};

export type WorkflowVersionRunsResponse = {
  project: ProjectRecord;
  workflow_version: WorkflowVersion;
  runs: RunRecord[];
};

export type RunCollectionResponse = {
  runs: Array<{ project: ProjectRecord; workflow_version: WorkflowVersion; run: RunRecord }>;
  count: number;
  next_cursor?: string;
};

export type RunTasksResponse = { run_id: number; tasks: RunTaskRecord[] };
export type RunAttemptsResponse = { run_id: number; attempts: RunAttemptRecord[] };

export type RunDetailResponse = {
  run: RunRecord;
  tasks: RunTaskRecord[];
  attempts: RunAttemptRecord[];
  artifacts: ArtifactRecord[];
};

export type WorkerRecord = {
  id: number;
  project_id: number;
  actor_id?: number;
  worker_key: string;
  worker_type: string;
  status: string;
  hostname: string;
  pid?: number;
  capabilities: string[];
  metadata: Record<string, unknown>;
  registered_at: string;
  last_heartbeat_at?: string;
  heartbeat_interval_seconds: number;
  lease_timeout_seconds: number;
  updated_at: string;
};

export type WorkerListResponse = {
  project: ProjectRecord;
  workers: WorkerRecord[];
};

export type WorkerCollectionResponse = {
  workers: Array<{ project: ProjectRecord; worker: WorkerRecord }>;
  count: number;
  next_cursor?: string;
};

export type ArtifactCollectionResponse = {
  artifacts: Array<{ project: ProjectRecord; artifact: ArtifactRecord }>;
  count: number;
  next_cursor?: string;
};

export type WorkerHeartbeatRecord = {
  id: number;
  project_id: number;
  worker_id: number;
  status: string;
  observed_at: string;
  metadata: Record<string, unknown>;
};

export type LeaseRecord = {
  id: number;
  project_id: number;
  run_id?: number;
  run_task_id?: number;
  workflow_item_id?: number;
  worker_id?: number;
  lease_kind: string;
  status: string;
  acquired_at: string;
  expires_at: string;
  heartbeat_at?: string;
  released_at?: string;
  allowed_capabilities: string[];
  scope: Record<string, unknown>;
  metadata: Record<string, unknown>;
};

export type WorkerDetailResponse = {
  worker: WorkerRecord;
  heartbeats: WorkerHeartbeatRecord[];
  leases: LeaseRecord[];
};

export type SchedulingPolicy = {
  priority: number;
  max_parallel_tasks: number;
  agent_role: string;
  required_capabilities: string[];
  engine_profile?: string;
};

export type RoleReadiness = {
  required_role: string;
  matched: boolean;
  matched_types: string[];
  status: string;
  blocked_reasons: string[];
};

export type EngineReadiness = {
  profile_id: string;
  provider?: string;
  enabled: boolean;
  secret_ref: string;
  secret_required: boolean;
  secret_ready: boolean;
  resource_limits: Record<string, unknown>;
  status: string;
  blocked_reasons: string[];
};

export type ResourceReadiness = {
  max_active_leases: number;
  max_queued_tasks: number;
  status: string;
  blocked_reasons: string[];
};

export type WorkerPoolProjectSummary = {
  project: ProjectRecord;
  workers: number;
  online_workers: number;
  offline_workers: number;
  active_leases: number;
  needs_recovery_leases: number;
  queued_tasks: number;
  needs_recovery_tasks: number;
  capabilities: string[];
  worker_types: string[];
  scheduling: SchedulingPolicy;
  role: RoleReadiness;
  engine: EngineReadiness;
  resources: ResourceReadiness;
  last_worker_heartbeat?: string;
};

export type WorkerPoolSummaryResponse = {
  projects: WorkerPoolProjectSummary[];
  total_projects: number;
  total_workers: number;
  total_online_workers: number;
  total_active_leases: number;
  total_queued_tasks: number;
  total_needs_recovery: number;
  generated_at: string;
};

export type WorkerPoolProjectSchedule = {
  project: ProjectRecord;
  priority: number;
  max_parallel: number;
  agent_role: string;
  role: RoleReadiness;
  engine_profile?: string;
  engine: EngineReadiness;
  resources: ResourceReadiness;
  queued_tasks: number;
  active_leases: number;
  online_workers: number;
  available_slots: number;
  needs_recovery: number;
  capabilities: string[];
  required_capabilities: string[];
  recommended: boolean;
  blocked_reasons: string[];
  next_action: string;
};

export type WorkerPoolSchedulePreviewResponse = {
  projects: WorkerPoolProjectSchedule[];
  policy: {
    strategy: string;
    default_project_priority: number;
    slot_strategy: string;
    dry_run_only: boolean;
  };
  generated_at: string;
  recommended: number;
  blocked: number;
  queued_tasks: number;
  available_slots: number;
};

export type WebWriteAction = {
  key: string;
  label: string;
  category: string;
  status: string;
  default_ui_state: string;
  command_api: string;
  risk_level: string;
  required_capabilities: string[];
  required_previews: string[];
  required_approvals: string[];
  required_audit_events: string[];
  required_evidence: string[];
  blockers: string[];
  forbidden_direct_actions: string[];
};

export type WebWriteActionGateResponse = {
  status: string;
  mode: string;
  actions: WebWriteAction[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
  db_write_attempted: boolean;
  project_write_attempted: boolean;
  artifact_write_attempted: boolean;
  execution_write_attempted: boolean;
  command_created: boolean;
  approval_created: boolean;
  audit_event_written: boolean;
  worker_scheduled: boolean;
  engine_call_attempted: boolean;
  commands_run: boolean;
  secrets_resolved: boolean;
  network_used: boolean;
};

export type LocalServiceStatusResponse = {
  status: string;
  mode: string;
  api: {
    status: string;
    message: string;
  };
  database: {
    status: string;
    message: string;
  };
  worker_pool: {
    status: string;
    message: string;
    total_projects: number;
    total_workers: number;
    total_online_workers: number;
    total_active_leases: number;
    total_queued_tasks: number;
    total_needs_recovery: number;
  };
  dashboard: {
    url: string;
    api_url: string;
    status: string;
    message: string;
  };
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

export type SupportBundlePreviewResponse = {
  status: string;
  mode: string;
  bundle_id: string;
  scope: string;
  projects: ProjectRecord[];
  included_metadata: string[];
  excluded_sensitive_content: string[];
  path_references: Array<{
    key: string;
    kind: string;
    uri: string;
    project_key?: string;
    description: string;
  }>;
  hashes: Array<{
    key: string;
    hash: string;
    source: string;
    description: string;
  }>;
  capabilities: string[];
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  generated_at: string;
};

export type MigrationLedgerReadinessResponse = {
  status: string;
  mode: string;
  entries: Array<{
    name: string;
    applied: boolean;
    status: string;
    required_evidence: string[];
    metadata: Record<string, unknown>;
  }>;
  applied_count: number;
  pending_count: number;
  schema_migrations_table_present: boolean;
  full_ledger_table_present: boolean;
  preflight_apply_verify_remediation_ready: boolean;
  capabilities: string[];
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  generated_at: string;
};

export type OperationsReadinessItem = {
  key: string;
  category: string;
  status: string;
  message: string;
  evidence_refs: string[];
  required_evidence: string[];
  blocked_by: string[];
  next_command: string;
  metadata: Record<string, unknown>;
};

export type OperationsReadinessResponse = {
  status: string;
  mode: string;
  items: OperationsReadinessItem[];
  service_status: LocalServiceStatusResponse;
  support_bundle: SupportBundlePreviewResponse;
  migration_ledger: MigrationLedgerReadinessResponse;
  capabilities: string[];
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  telemetry_default: string;
  managed_ops_status: string;
  support_export_status: string;
  generated_at: string;
};

export type Real100Guardrail = {
  claim_scope: string;
  not_real_100: boolean;
  evidence_only: boolean;
  status_alone_is_not_completion: boolean;
  release_candidate_decision: string;
  readiness_scope: string;
  real_100_status: string;
  real_100_blockers: string[];
  real_100_breakdown?: Real100Breakdown;
};

export type Real100Breakdown = {
  needs_exact_authorization?: Real100BreakdownItem[];
  needs_real_areamatrix_write?: Real100BreakdownItem[];
  areaflow_only_can_continue?: Real100BreakdownItem[];
  completed_evidence?: Real100BreakdownItem[];
};

export type Real100BreakdownItem = {
  key: string;
  status?: string;
  message?: string;
  required_authorization_phrase?: string;
  blockers?: string[];
  evidence_refs?: string[];
  next_command?: string;
};

export type StatusProjectionApplyGateItem = {
  key: string;
  category: string;
  status: string;
  message: string;
  expected: string;
  actual: string;
  required_evidence: string[];
  blocked_by: string[];
};

export type StatusProjectionApplyPacket = {
  target_uri: string;
  expected_before_exists: boolean;
  expected_before_sha256: string;
  expected_before_size: number;
  source_hash: string;
  schema_uri: string;
  validator_preflight: string;
  protected_path_check: string;
  protected_path_fingerprint_sha256: string;
  rollback_action: string;
  accept_preimage_schema: string;
  explicit_approval: boolean;
  approval_actor: string;
  approval_reason: string;
  required_authorization_phrase?: string;
};

export type StatusProjectionAuthorizationPreviewResponse = {
  project: ProjectRecord;
  status: string;
  mode: string;
  claim_scope: string;
  not_real_100: boolean;
  decision: string;
  message: string;
  target_kind: string;
  target_uri: string;
  target_path: string;
  schema_uri: string;
  validator_preflight: string;
  protected_path_fingerprint_sha256?: string;
  source_hash?: string;
  summary_state: string;
  required_authorization_phrase?: string;
  permission: {
    capability: string;
    target_uri: string;
    allowed: boolean;
    reason: string;
  };
  preimage: {
    exists: boolean;
    readable: boolean;
    size_bytes: number;
    sha256?: string;
    schema_status: string;
    legacy_shape: boolean;
  };
  required_preflight: string[];
  required_packet_fields: string[];
  required_capabilities: string[];
  protected_paths: string[];
  rollback_plan: string[];
  blocked_by: string[];
  warnings: string[];
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  apply_open: boolean;
  approval_required: boolean;
  approval_status: string;
  would_create_command_request_after_approval: boolean;
  would_create_status_projection_after_approval: boolean;
  would_write_project_file_after_approval: boolean;
  would_write_execution_after_approval: boolean;
  would_run_engine_after_approval: boolean;
  project_write_attempted: boolean;
  execution_write_attempted: boolean;
  engine_call_attempted: boolean;
  generated_at: string;
};

export type StatusProjectionApplyGateResponse = {
  project: ProjectRecord;
  status: string;
  mode: string;
  claim_scope: string;
  not_real_100: boolean;
  decision: string;
  message: string;
  target_uri: string;
  target_path: string;
  authorization: Record<string, unknown>;
  items: StatusProjectionApplyGateItem[];
  required_packet_fields: string[];
  required_capabilities: string[];
  required_authorization_phrase?: string;
  protected_paths: string[];
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  apply_command_eligible: boolean;
  apply_command_eligible_is_not_apply: boolean;
  requires_separate_apply_command: boolean;
  approval_required: boolean;
  approval_status: string;
  project_write_attempted: boolean;
  execution_write_attempted: boolean;
  engine_call_attempted: boolean;
  command_request_created: boolean;
  status_projection_written: boolean;
  generated_at: string;
};

export type StatusProjectionApplyPacketPreviewResponse = {
  project: ProjectRecord;
  status: string;
  mode: string;
  claim_scope: string;
  not_real_100: boolean;
  decision: string;
  message: string;
  blockers: string[];
  required_authorization_phrase?: string;
  authorization: Record<string, unknown>;
  gate: StatusProjectionApplyGateResponse;
  packet: StatusProjectionApplyPacket;
  apply_command: string[];
  api_request: StatusProjectionApplyPacket;
  required_human_review: string[];
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  would_create_command_request_after_apply_command: boolean;
  would_create_status_projection_after_apply_command: boolean;
  would_write_project_file_after_apply_command: boolean;
  apply_command_eligible_is_not_apply: boolean;
  requires_separate_apply_command: boolean;
  project_write_attempted: boolean;
  execution_write_attempted: boolean;
  engine_call_attempted: boolean;
  command_request_created: boolean;
  status_projection_written: boolean;
  generated_at: string;
};

export type CompletionAuditSnapshotReadinessItem = {
  key: string;
  status: string;
  message: string;
  metadata: Record<string, unknown>;
};

export type CompletionAuditSnapshotRecord = Real100Guardrail & {
  status: string;
  decision: string;
  message: string;
  audit_status: string;
  audit_scope: string;
  audit_hash: string;
  release_candidate_label: string;
  evidence_class: string;
  evidence_uri: string;
  proof_event_ids: Record<string, number>;
  event_id?: number;
  audit_event_id?: number;
  idempotency_key: string;
  created_at?: string;
  metadata: Record<string, unknown>;
};

export type CompletionAuditSnapshotReadinessResponse = Real100Guardrail & {
  project: ProjectRecord;
  status: string;
  message: string;
  has_snapshot: boolean;
  required_class: string;
  bundle_hash: string;
  latest: CompletionAuditSnapshotRecord;
  items: CompletionAuditSnapshotReadinessItem[];
  safety_facts: Record<string, boolean>;
};

export type ShimFilePlan = {
  path: string;
  action: string;
  required: boolean;
  reason: string;
  boundary: string;
};

export type ShimAuthorizationPacketResponse = {
  project: ProjectRecord;
  status: string;
  mode: string;
  intent: string;
  readiness_status: string;
  allowed_files: ShimFilePlan[];
  forbidden_paths: string[];
  forbidden_actions: string[];
  required_preflight: string[];
  post_edit_verification: string[];
  rollback_scope: string[];
  safety_facts: Record<string, boolean>;
  next_required_approval: string;
};

export type ShimApplyGateItem = {
  key: string;
  category: string;
  status: string;
  message: string;
  expected: string;
  actual: string;
  required_evidence: string[];
  blocked_by: string[];
};

export type ShimApplyGateResponse = {
  project: ProjectRecord;
  status: string;
  mode: string;
  decision: string;
  message: string;
  items: ShimApplyGateItem[];
  required_packet_fields: string[];
  required_capabilities: string[];
  allowed_files: string[];
  forbidden_paths: string[];
  forbidden_actions: string[];
  required_preflight: string[];
  post_edit_verification: string[];
  rollback_scope: string[];
  required_proof_facts: string[];
  safety_facts: Record<string, boolean>;
  approval_required: boolean;
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

export type ShimApplyPacket = {
  command_type: string;
  project_key: string;
  allowed_files: string[];
  approval_id: string;
  approval_scope: string;
  authorization_snapshot_hash: string;
  expected_authorization_mode: string;
  status_projection_packet_id: string;
  status_projection_gate_id: string;
  read_only_smoke_evidence_id: string;
  dirty_worktree_review_id: string;
  protected_path_fingerprint_id: string;
  rollback_plan_id: string;
  idempotency_key: string;
  audit_correlation_id: string;
  failure_mode: string;
  explicit_approval: boolean;
  approval_actor: string;
  approval_reason: string;
};

export type ShimApplyPacketPreviewResponse = {
  project: ProjectRecord;
  status: string;
  mode: string;
  decision: string;
  message: string;
  authorization: ShimAuthorizationPacketResponse;
  gate: ShimApplyGateResponse;
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

export type ExecutionCutoverReadinessItem = {
  key: string;
  category: string;
  status: string;
  message: string;
  required_evidence: string[];
  next_command: string;
  metadata: Record<string, unknown>;
};

export type ExecutionCutoverNextStep = {
  key: string;
  owner: string;
  action: string;
  risk_level: string;
  blocked_by: string[];
  next_command: string;
  metadata: Record<string, unknown>;
};

export type ExecutionCutoverReadinessResponse = {
  project: ProjectRecord;
  status: string;
  mode: string;
  items: ExecutionCutoverReadinessItem[];
  migration_path: string[];
  command_evidence: Record<string, number>;
  capabilities: string[];
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  next_steps: ExecutionCutoverNextStep[];
  generated_at: string;
};

export type ExecutionForwardingV1ReadinessResponse = {
  project: ProjectRecord;
  status: string;
  mode: string;
  items: ExecutionCutoverReadinessItem[];
  allowed_task_types: string[];
  command_evidence: Record<string, number>;
  capabilities: string[];
  forbidden_actions: string[];
  safety_facts: Record<string, boolean>;
  next_steps: ExecutionCutoverNextStep[];
  generated_at: string;
};

export type ExecutionForwardingV1ApplyPreviewItem = {
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

export type ExecutionForwardingV1ForwardingTarget = {
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

export type ExecutionForwardingV1BlockedTarget = {
  task_type: string;
  forbidden_action: string;
  reason: string;
  failure_mode: string;
  safety_facts: Record<string, boolean>;
};

export type ExecutionForwardingV1ApplyPreviewResponse = {
  project: ProjectRecord;
  status: string;
  mode: string;
  readiness: ExecutionForwardingV1ReadinessResponse;
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

export type ExecutionForwardingV1ApplyGateItem = {
  key: string;
  category: string;
  status: string;
  message: string;
  expected: string;
  actual: string;
  required_evidence: string[];
  blocked_by: string[];
};

export type ExecutionForwardingV1ApplyGateResponse = {
  project: ProjectRecord;
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

export type ExecutionForwardingV1ApplyPacket = {
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

export type ExecutionForwardingV1ApplyPacketPreviewResponse = {
  project: ProjectRecord;
  status: string;
  mode: string;
  decision: string;
  message: string;
  apply_preview: ExecutionForwardingV1ApplyPreviewResponse;
  gate: ExecutionForwardingV1ApplyGateResponse;
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

export type ExecutionForwardingV1CommandPreviewResponse = {
  project: ProjectRecord;
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

export type ExecutionForwardingV1RollbackPreviewItem = {
  key: string;
  category: string;
  status: string;
  message: string;
  owner: string;
  required_evidence: string[];
  next_command: string;
  metadata: Record<string, unknown>;
};

export type ExecutionForwardingV1RollbackPreviewResponse = {
  project: ProjectRecord;
  status: string;
  mode: string;
  apply_preview: ExecutionForwardingV1ApplyPreviewResponse;
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

export type ReleaseFinalGateItem = {
  key: string;
  category: string;
  status: string;
  message: string;
  owner: string;
  required_evidence: string[];
  next_command: string;
  metadata: Record<string, unknown>;
};

export type ReleaseFinalGateResponse = Real100Guardrail & {
  status: string;
  mode: string;
  scope: string;
  project_key?: string;
  readiness: Record<string, unknown>;
  acceptance_gate: Record<string, unknown>;
  exception_apply: Record<string, unknown>;
  items: ReleaseFinalGateItem[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

export type ReleaseEvidenceBundleItem = {
  key: string;
  category: string;
  status: string;
  source: string;
  description: string;
  metadata: Record<string, unknown>;
};

export type ReleaseEvidenceBundleResponse = Real100Guardrail & {
  status: string;
  mode: string;
  scope: string;
  project_key?: string;
  bundle_hash: string;
  final_gate: ReleaseFinalGateResponse;
  backup: Record<string, unknown>;
  audit_coverage: Record<string, unknown>;
  items: ReleaseEvidenceBundleItem[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

export type ReleasePackagePreviewItem = {
  key: string;
  category: string;
  status: string;
  package_path: string;
  source: string;
  description: string;
  metadata: Record<string, unknown>;
};

export type ReleasePackagePreviewResponse = Real100Guardrail & {
  status: string;
  mode: string;
  scope: string;
  project_key?: string;
  evidence_bundle: ReleaseEvidenceBundleResponse;
  package_name: string;
  items: ReleasePackagePreviewItem[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

export type ReleasePublishGateItem = {
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

export type ReleasePublishGateResponse = Real100Guardrail & {
  status: string;
  mode: string;
  scope: string;
  project_key?: string;
  distribution_preview: Record<string, unknown>;
  items: ReleasePublishGateItem[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

export type ReleaseDistributionPreviewItem = {
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

export type ReleaseDistributionPreviewResponse = Real100Guardrail & {
  status: string;
  mode: string;
  scope: string;
  project_key?: string;
  package_preview: Record<string, unknown>;
  items: ReleaseDistributionPreviewItem[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

export type ReleasePublishApprovalPreviewItem = {
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

export type ReleasePublishApprovalPreviewResponse = Real100Guardrail & {
  status: string;
  mode: string;
  scope: string;
  project_key?: string;
  publish_gate: Record<string, unknown>;
  items: ReleasePublishApprovalPreviewItem[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

export type ReleaseRolloutPlanPreviewItem = {
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

export type ReleaseRolloutPlanPreviewStep = {
  order: number;
  stage: string;
  action: string;
  description: string;
  blocked_by: string[];
};

export type ReleaseRolloutPlanPreviewResponse = Real100Guardrail & {
  status: string;
  mode: string;
  scope: string;
  project_key?: string;
  publish_approval_preview: Record<string, unknown>;
  items: ReleaseRolloutPlanPreviewItem[];
  rollout_steps: ReleaseRolloutPlanPreviewStep[];
  verification_checkpoints: ReleaseRolloutPlanPreviewStep[];
  rollback_steps: ReleaseRolloutPlanPreviewStep[];
  capabilities: string[];
  forbidden_actions: string[];
  generated_at: string;
};

export type EventRecord = {
  id: number;
  project_id?: number;
  run_id?: number;
  workflow_version_id?: number;
  type: string;
  severity: string;
  message: string;
  metadata: Record<string, unknown>;
  created_at: string;
};

export type ProjectEventsResponse = {
  project: string;
  events: EventRecord[];
};

export type AuditEventRecord = {
  id: number;
  project_id?: number;
  actor_id?: number;
  action: string;
  capability?: string;
  resource_type?: string;
  resource?: string;
  decision: string;
  reason?: string;
  metadata: Record<string, unknown>;
  created_at: string;
};

export type AuditEventsResponse = {
  project_key?: string;
  audit_events: AuditEventRecord[];
  count: number;
  next_cursor?: string;
};
