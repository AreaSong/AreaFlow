import type {
  ApprovalRecordsResponse,
  ApprovalRecord,
  AuthPrincipal,
  AuthStatus,
  ArtifactListResponse,
  ArtifactRecord,
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
  RunAttemptsResponse,
  RunTasksResponse,
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
  WorkflowCollectionResponse,
  WorkflowVersionListResponse,
  WorkflowVersionRunsResponse,
  TransitionPreviewsResponse,
  RunCollectionResponse,
  WorkerCollectionResponse,
  ArtifactCollectionResponse,
  WorkerDetailResponse,
} from "./types";

const TOKEN_STORAGE_KEY = "areaflow.api_token";
const AUTH_INVALID_EVENT = "areaflow:auth-invalid";

function authHeaders(): Record<string, string> {
  const token = sessionStorage.getItem(TOKEN_STORAGE_KEY);
  return token ? { Authorization: `Bearer ${token}` } : {};
}

function handleUnauthorized(response: Response) {
  if (response.status !== 401 || !sessionStorage.getItem(TOKEN_STORAGE_KEY)) return;
  sessionStorage.removeItem(TOKEN_STORAGE_KEY);
  window.dispatchEvent(new Event(AUTH_INVALID_EVENT));
}

async function getJSON<T>(path: string): Promise<T> {
  const response = await fetch(path, {
    headers: {
      Accept: "application/json",
      ...authHeaders(),
    },
  });

  if (!response.ok) {
    handleUnauthorized(response);
    throw new Error(await responseError(response, path));
  }

  return (await response.json()) as T;
}

async function getText(path: string) {
  const response = await fetch(path, { headers: { Accept: "*/*", ...authHeaders() } });
  handleUnauthorized(response);
  if (!response.ok) throw new Error(await responseError(response, path));
  return {
    content: await response.text(),
    contentType: response.headers.get("content-type") ?? "application/octet-stream",
    sha256: response.headers.get("x-areaflow-artifact-sha256") ?? "",
  };
}

async function postJSON<T>(path: string, body: unknown, idempotencyKey: string): Promise<T> {
  const response = await fetch(path, {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
      "Idempotency-Key": idempotencyKey,
      ...authHeaders(),
    },
    body: JSON.stringify(body),
  });
  if (!response.ok) {
    handleUnauthorized(response);
    throw new Error(await responseError(response, path));
  }
  return (await response.json()) as T;
}

async function responseError(response: Response, path: string) {
  const fallback = `${path} returned ${response.status}`;
  try {
    const body = await response.json() as { error?: string };
    return body.error ? `${body.error} (${response.status})` : fallback;
  } catch {
    return fallback;
  }
}

const API_BASE = "/api/v1";

function projectQuery(projectKey: string): string {
  return `project_key=${encodeURIComponent(projectKey)}`;
}

type CollectionOptions = Record<string, string | number | boolean | undefined>;

function collectionPath(resource: string, options: CollectionOptions = {}) {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(options)) {
    if (value !== undefined && value !== "") query.set(key, String(value));
  }
  return `${API_BASE}/${resource}${query.size ? `?${query}` : ""}`;
}

async function getAllCollection<T extends { next_cursor?: string }, K extends keyof T>(
  resource: string,
  options: CollectionOptions,
  itemsKey: K,
): Promise<T> {
  let cursor = typeof options.cursor === "string" ? options.cursor : "";
  let firstPage: T | null = null;
  const items: unknown[] = [];
  const seenCursors = new Set<string>();

  do {
    const page = await getJSON<T>(collectionPath(resource, { limit: 200, ...options, cursor }));
    firstPage ??= page;
    const pageItems = page[itemsKey];
    if (!Array.isArray(pageItems)) throw new Error(`${resource} returned an invalid collection`);
    items.push(...pageItems);

    const nextCursor = page.next_cursor ?? "";
    if (nextCursor && seenCursors.has(nextCursor)) throw new Error(`${resource} returned a repeated cursor`);
    if (nextCursor) seenCursors.add(nextCursor);
    cursor = nextCursor;
  } while (cursor);

  if (!firstPage) throw new Error(`${resource} returned no collection page`);
  return { ...firstPage, [itemsKey]: items, count: items.length, next_cursor: undefined } as T;
}

export const api = {
  authStatus: () => getJSON<AuthStatus>(`${API_BASE}/auth/status`),
  authMe: () => getJSON<AuthPrincipal>(`${API_BASE}/auth/me`),
  projects: () => getJSON<ProjectListResponse>(`${API_BASE}/projects`),
  workflows: (projectKey?: string, options: CollectionOptions = {}) =>
    getAllCollection<WorkflowCollectionResponse, "workflows">("workflows", { project_key: projectKey, ...options }, "workflows"),
  runs: (projectKey?: string, options: CollectionOptions = {}) =>
    getAllCollection<RunCollectionResponse, "runs">("runs", { project_key: projectKey, ...options }, "runs"),
  workers: (projectKey?: string, options: CollectionOptions = {}) =>
    getAllCollection<WorkerCollectionResponse, "workers">("workers", { project_key: projectKey, ...options }, "workers"),
  workerDetail: (projectKey: string, workerID: number) =>
    getJSON<WorkerDetailResponse>(collectionPath(`workers/${workerID}`, { limit: 50, project_key: projectKey })),
  artifacts: (projectKey?: string, options: CollectionOptions = {}) =>
    getAllCollection<ArtifactCollectionResponse, "artifacts">("artifacts", { project_key: projectKey, ...options }, "artifacts"),
  artifactDetail: (projectKey: string, artifactID: number) =>
    getJSON<ArtifactRecord>(collectionPath(`artifacts/${artifactID}`, { project_key: projectKey })),
  artifactContent: (projectKey: string, artifactID: number) =>
    getText(collectionPath(`artifacts/${artifactID}/content`, { project_key: projectKey })),
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
  versionTransitionPreviews: (projectKey: string, version: string) =>
    getJSON<TransitionPreviewsResponse>(
      `${API_BASE}/projects/${projectKey}/workflow-versions/${version}/transition-previews?limit=20`,
    ),
  createApproval: (
    projectKey: string,
    version: string,
    request: { decision: "approved" | "rejected"; reason: string; transitionPreviewID: number; actor: string },
    idempotencyKey: string,
  ) => postJSON<ApprovalRecord>(
    `${API_BASE}/projects/${projectKey}/workflow-versions/${version}/approvals`,
    {
      decision: request.decision,
      approval_kind: "workflow_transition",
      actor: request.actor,
      reason: request.reason,
      risk_level: "medium",
      idempotency_key: idempotencyKey,
      transition_preview_id: request.transitionPreviewID,
      metadata: { source: "areaflow_web" },
    },
    idempotencyKey,
  ),
  versionRuns: (projectKey: string, version: string) =>
    getJSON<WorkflowVersionRunsResponse>(
      `${API_BASE}/projects/${projectKey}/workflow-versions/${version}/runs?limit=20`,
    ),
  runDetail: (projectKey: string, runID: number) =>
    getJSON<RunDetailResponse>(
      `${API_BASE}/runs/${runID}?project_key=${encodeURIComponent(projectKey)}`,
    ),
  runTasks: (projectKey: string, runID: number) =>
    getJSON<RunTasksResponse>(`${API_BASE}/runs/${runID}/tasks?${projectQuery(projectKey)}`),
  runAttempts: (projectKey: string, runID: number) =>
    getJSON<RunAttemptsResponse>(`${API_BASE}/runs/${runID}/attempts?${projectQuery(projectKey)}`),
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
  projectAuditEvents: (projectKey: string, options: CollectionOptions = {}) =>
    getAllCollection<AuditEventsResponse, "audit_events">("audit-events", { project_key: projectKey, ...options }, "audit_events"),
  projectEvents: (projectKey: string) =>
    getJSON<ProjectEventsResponse>(`${API_BASE}/projects/${projectKey}/events?limit=12`),
};

export const authSession = {
  eventName: AUTH_INVALID_EVENT,
  hasToken: () => Boolean(sessionStorage.getItem(TOKEN_STORAGE_KEY)),
  setToken: (token: string) => sessionStorage.setItem(TOKEN_STORAGE_KEY, token.trim()),
  clearToken: () => sessionStorage.removeItem(TOKEN_STORAGE_KEY),
};
