import type {
  ApprovalRecordsResponse,
  ArtifactListResponse,
  AuditEventsResponse,
  CompletionAuditSnapshotReadinessResponse,
  ExecutionCutoverReadinessResponse,
  ExecutionForwardingV1ApplyPacketPreviewResponse,
  ExecutionForwardingV1ApplyPreviewResponse,
  ExecutionForwardingV1CommandPreviewResponse,
  ExecutionForwardingV1ReadinessResponse,
  ExecutionForwardingV1RollbackPreviewResponse,
  OperationsReadinessResponse,
  ProjectEventsResponse,
  ProjectListResponse,
  ProjectReadiness,
  ProjectSummary,
  ReleaseDistributionPreviewResponse,
  ReleaseEvidenceBundleResponse,
  ReleaseFinalGateResponse,
  ReleasePackagePreviewResponse,
  ReleasePublishApprovalPreviewResponse,
  ReleasePublishGateResponse,
  ReleaseRolloutPlanPreviewResponse,
  ResidualListResponse,
  RunDetailResponse,
  ShimApplyGateResponse,
  ShimApplyPacketPreviewResponse,
  ShimAuthorizationPacketResponse,
  StatusProjectionApplyGateResponse,
  StatusProjectionApplyPacketPreviewResponse,
  StatusProjectionAuthorizationPreviewResponse,
  WorkerListResponse,
  WebWriteActionGateResponse,
  WorkerPoolSchedulePreviewResponse,
  WorkerPoolSummaryResponse,
  WorkflowStagesResponse,
  WorkflowVersionListResponse,
  WorkflowVersionRunsResponse,
} from "./types";

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(path, {
    headers: {
      Accept: "application/json",
    },
  });

  if (!response.ok) {
    throw new Error(`${path} returned ${response.status}`);
  }

  return (await response.json()) as T;
}

const API_BASE = "/api/v1";

function projectQuery(projectKey: string): string {
  return `project_key=${encodeURIComponent(projectKey)}`;
}

export const api = {
  projects: () => getJSON<ProjectListResponse>(`${API_BASE}/projects`),
  projectSummary: (projectKey: string) =>
    getJSON<ProjectSummary>(`${API_BASE}/projects/${projectKey}/summary`),
  projectReadiness: (projectKey: string) =>
    getJSON<ProjectReadiness>(`${API_BASE}/projects/${projectKey}/readiness`),
  workflowVersions: (projectKey: string) =>
    getJSON<WorkflowVersionListResponse>(
      `${API_BASE}/projects/${projectKey}/workflow-versions`,
    ),
  workflowStages: (projectKey: string, version: string) =>
    getJSON<WorkflowStagesResponse>(
      `${API_BASE}/projects/${projectKey}/workflow-versions/${version}/stages`,
    ),
  versionArtifacts: (projectKey: string, version: string) =>
    getJSON<ArtifactListResponse>(
      `${API_BASE}/projects/${projectKey}/workflow-versions/${version}/artifacts?limit=20`,
    ),
  versionResiduals: (projectKey: string, version: string) =>
    getJSON<ResidualListResponse>(
      `${API_BASE}/projects/${projectKey}/workflow-versions/${version}/residuals?limit=20`,
    ),
  versionApprovals: (projectKey: string, version: string) =>
    getJSON<ApprovalRecordsResponse>(
      `${API_BASE}/projects/${projectKey}/workflow-versions/${version}/approvals?limit=20`,
    ),
  versionRuns: (projectKey: string, version: string) =>
    getJSON<WorkflowVersionRunsResponse>(
      `${API_BASE}/projects/${projectKey}/workflow-versions/${version}/runs?limit=20`,
    ),
  runDetail: (projectKey: string, runID: number) =>
    getJSON<RunDetailResponse>(
      `${API_BASE}/runs/${runID}?project_key=${encodeURIComponent(projectKey)}`,
    ),
  projectArtifacts: (projectKey: string) =>
    getJSON<ArtifactListResponse>(`${API_BASE}/projects/${projectKey}/artifacts?limit=8`),
  projectResiduals: (projectKey: string) =>
    getJSON<ResidualListResponse>(`${API_BASE}/projects/${projectKey}/residuals?limit=8`),
  projectWorkers: (projectKey: string) =>
    getJSON<WorkerListResponse>(`${API_BASE}/projects/${projectKey}/workers?limit=20`),
  workerPoolSummary: () => getJSON<WorkerPoolSummaryResponse>(`${API_BASE}/worker-pool/summary`),
  workerPoolSchedulePreview: () =>
    getJSON<WorkerPoolSchedulePreviewResponse>(`${API_BASE}/worker-pool/schedule-preview`),
  webWriteActionGate: () =>
    getJSON<WebWriteActionGateResponse>(`${API_BASE}/web/write-action-gate`),
  completionAuditSnapshotReadiness: (projectKey: string) =>
    getJSON<CompletionAuditSnapshotReadinessResponse>(
      `${API_BASE}/completion-audit/snapshot-readiness?project_key=${encodeURIComponent(projectKey)}`,
    ),
  operationsReadiness: () => getJSON<OperationsReadinessResponse>(`${API_BASE}/ops/readiness`),
  releaseFinalGate: (projectKey: string) =>
    getJSON<ReleaseFinalGateResponse>(`${API_BASE}/release/final-gate?${projectQuery(projectKey)}`),
  releaseEvidenceBundle: (projectKey: string) =>
    getJSON<ReleaseEvidenceBundleResponse>(
      `${API_BASE}/release/evidence-bundle?${projectQuery(projectKey)}`,
    ),
  releasePackagePreview: (projectKey: string) =>
    getJSON<ReleasePackagePreviewResponse>(
      `${API_BASE}/release/package-preview?${projectQuery(projectKey)}`,
    ),
  releaseDistributionPreview: (projectKey: string) =>
    getJSON<ReleaseDistributionPreviewResponse>(
      `${API_BASE}/release/distribution-preview?${projectQuery(projectKey)}`,
    ),
  releasePublishGate: (projectKey: string) =>
    getJSON<ReleasePublishGateResponse>(`${API_BASE}/release/publish-gate?${projectQuery(projectKey)}`),
  releasePublishApprovalPreview: (projectKey: string) =>
    getJSON<ReleasePublishApprovalPreviewResponse>(
      `${API_BASE}/release/publish-approval-preview?${projectQuery(projectKey)}`,
    ),
  releaseRolloutPlanPreview: (projectKey: string) =>
    getJSON<ReleaseRolloutPlanPreviewResponse>(
      `${API_BASE}/release/rollout-plan-preview?${projectQuery(projectKey)}`,
    ),
  projectShimAuthorization: (projectKey: string) =>
    getJSON<ShimAuthorizationPacketResponse>(
      `${API_BASE}/projects/${projectKey}/shim-authorization`,
    ),
  projectShimApplyPacket: (projectKey: string) =>
    getJSON<ShimApplyPacketPreviewResponse>(
      `${API_BASE}/projects/${projectKey}/shim-apply-packet`,
    ),
  projectShimApplyGate: (projectKey: string) =>
    getJSON<ShimApplyGateResponse>(
      `${API_BASE}/projects/${projectKey}/shim-apply-gate`,
    ),
  projectStatusProjectionAuthorization: (projectKey: string) =>
    getJSON<StatusProjectionAuthorizationPreviewResponse>(
      `${API_BASE}/projects/${projectKey}/status-projections/authorization`,
    ),
  projectStatusProjectionApplyPacket: (projectKey: string) =>
    getJSON<StatusProjectionApplyPacketPreviewResponse>(
      `${API_BASE}/projects/${projectKey}/status-projections/apply-packet`,
    ),
  projectStatusProjectionApplyGate: (projectKey: string) =>
    getJSON<StatusProjectionApplyGateResponse>(
      `${API_BASE}/projects/${projectKey}/status-projections/apply-gate`,
    ),
  projectExecutionCutoverReadiness: (projectKey: string) =>
    getJSON<ExecutionCutoverReadinessResponse>(
      `${API_BASE}/projects/${projectKey}/execution-cutover-readiness`,
    ),
  projectExecutionForwardingV1Readiness: (projectKey: string) =>
    getJSON<ExecutionForwardingV1ReadinessResponse>(
      `${API_BASE}/projects/${projectKey}/execution-forwarding-v1-readiness`,
    ),
  projectExecutionForwardingV1ApplyPreview: (projectKey: string) =>
    getJSON<ExecutionForwardingV1ApplyPreviewResponse>(
      `${API_BASE}/projects/${projectKey}/execution-forwarding-v1-apply-preview`,
    ),
  projectExecutionForwardingV1ApplyPacket: (projectKey: string) =>
    getJSON<ExecutionForwardingV1ApplyPacketPreviewResponse>(
      `${API_BASE}/projects/${projectKey}/execution-forwarding-v1-apply-packet`,
    ),
  projectExecutionForwardingV1CommandPreview: (projectKey: string, taskType: string) =>
    getJSON<ExecutionForwardingV1CommandPreviewResponse>(
      `${API_BASE}/projects/${projectKey}/execution-forwarding-v1-command-preview?task_type=${encodeURIComponent(taskType)}`,
    ),
  projectExecutionForwardingV1RollbackPreview: (projectKey: string) =>
    getJSON<ExecutionForwardingV1RollbackPreviewResponse>(
      `${API_BASE}/projects/${projectKey}/execution-forwarding-v1-rollback-preview`,
    ),
  projectAuditEvents: (projectKey: string) =>
    getJSON<AuditEventsResponse>(`${API_BASE}/audit-events?project_key=${projectKey}&limit=20`),
  projectEvents: (projectKey: string) =>
    getJSON<ProjectEventsResponse>(`${API_BASE}/projects/${projectKey}/events?limit=12`),
};
