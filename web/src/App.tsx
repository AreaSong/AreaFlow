import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";
import { api } from "./api";
import type {
  ApprovalRecord,
  ArtifactRecord,
  AuditEventRecord,
  CompletionAuditSnapshotReadinessResponse,
  EventRecord,
  ExecutionCutoverReadinessResponse,
  ExecutionForwardingV1ApplyPacketPreviewResponse,
  ExecutionForwardingV1ApplyPreviewResponse,
  ExecutionForwardingV1CommandPreviewResponse,
  ExecutionForwardingV1ReadinessResponse,
  ExecutionForwardingV1RollbackPreviewResponse,
  OperationsReadinessResponse,
  ProjectRecord,
  ProjectReadiness,
  ProjectSummary,
  Real100Guardrail,
  ReleaseDistributionPreviewResponse,
  ReleaseEvidenceBundleResponse,
  ReleaseFinalGateResponse,
  ReleasePackagePreviewResponse,
  ReleasePublishApprovalPreviewResponse,
  ReleasePublishGateResponse,
  ReleaseRolloutPlanPreviewResponse,
  ResidualRecord,
  RunDetailResponse,
  RunRecord,
  ShimApplyGateResponse,
  ShimApplyPacketPreviewResponse,
  ShimAuthorizationPacketResponse,
  StatusProjectionApplyGateResponse,
  StatusProjectionApplyPacketPreviewResponse,
  StatusProjectionAuthorizationPreviewResponse,
  WebWriteActionGateResponse,
  WorkerRecord,
  WorkerPoolSchedulePreviewResponse,
  WorkerPoolSummaryResponse,
  WorkflowItem,
  WorkflowVersion,
} from "./types";

type LoadState = "idle" | "loading" | "ready" | "error";

type DashboardSelection =
  | { kind: "stage"; stage: string }
  | { kind: "item"; id: number }
  | { kind: "artifact"; id: number }
  | { kind: "residual"; key: string }
  | { kind: "approval"; id: number }
  | { kind: "run"; id: number };

type RecordListItem = {
  key: string;
  title: string;
  meta: string;
  tone: string;
  selection?: DashboardSelection;
};

const stageOrder = [
  "discussion",
  "middle_layer",
  "changes",
  "plans",
  "drafts",
  "queue",
  "promotion_preview",
  "approval",
  "execution",
  "run",
  "projection",
  "closeout",
];

export function App() {
  const [projects, setProjects] = useState<ProjectRecord[]>([]);
  const [selectedProjectKey, setSelectedProjectKey] = useState("");
  const [selectedVersionLabel, setSelectedVersionLabel] = useState("");
  const [summary, setSummary] = useState<ProjectSummary | null>(null);
  const [readiness, setReadiness] = useState<ProjectReadiness | null>(null);
  const [versions, setVersions] = useState<WorkflowVersion[]>([]);
  const [items, setItems] = useState<WorkflowItem[]>([]);
  const [artifacts, setArtifacts] = useState<ArtifactRecord[]>([]);
  const [residuals, setResiduals] = useState<ResidualRecord[]>([]);
  const [approvals, setApprovals] = useState<ApprovalRecord[]>([]);
  const [runs, setRuns] = useState<RunRecord[]>([]);
  const [runDetail, setRunDetail] = useState<RunDetailResponse | null>(null);
  const [workers, setWorkers] = useState<WorkerRecord[]>([]);
  const [workerPool, setWorkerPool] = useState<WorkerPoolSummaryResponse | null>(null);
  const [schedulePreview, setSchedulePreview] = useState<WorkerPoolSchedulePreviewResponse | null>(null);
  const [webWriteGate, setWebWriteGate] = useState<WebWriteActionGateResponse | null>(null);
  const [completionSnapshotReadiness, setCompletionSnapshotReadiness] =
    useState<CompletionAuditSnapshotReadinessResponse | null>(null);
  const [operationsReadiness, setOperationsReadiness] = useState<OperationsReadinessResponse | null>(null);
  const [releaseFinalGate, setReleaseFinalGate] = useState<ReleaseFinalGateResponse | null>(null);
  const [releaseEvidenceBundle, setReleaseEvidenceBundle] = useState<ReleaseEvidenceBundleResponse | null>(null);
  const [releasePackagePreview, setReleasePackagePreview] = useState<ReleasePackagePreviewResponse | null>(null);
  const [releaseDistributionPreview, setReleaseDistributionPreview] =
    useState<ReleaseDistributionPreviewResponse | null>(null);
  const [releasePublishGate, setReleasePublishGate] = useState<ReleasePublishGateResponse | null>(null);
  const [releasePublishApproval, setReleasePublishApproval] =
    useState<ReleasePublishApprovalPreviewResponse | null>(null);
  const [releaseRolloutPlan, setReleaseRolloutPlan] = useState<ReleaseRolloutPlanPreviewResponse | null>(null);
  const [shimAuthorization, setShimAuthorization] = useState<ShimAuthorizationPacketResponse | null>(null);
  const [shimApplyPacket, setShimApplyPacket] = useState<ShimApplyPacketPreviewResponse | null>(null);
  const [shimApplyGate, setShimApplyGate] = useState<ShimApplyGateResponse | null>(null);
  const [statusProjectionAuthorization, setStatusProjectionAuthorization] =
    useState<StatusProjectionAuthorizationPreviewResponse | null>(null);
  const [statusProjectionApplyPacket, setStatusProjectionApplyPacket] =
    useState<StatusProjectionApplyPacketPreviewResponse | null>(null);
  const [statusProjectionApplyGate, setStatusProjectionApplyGate] =
    useState<StatusProjectionApplyGateResponse | null>(null);
  const [executionCutover, setExecutionCutover] = useState<ExecutionCutoverReadinessResponse | null>(null);
  const [executionForwardingV1Readiness, setExecutionForwardingV1Readiness] =
    useState<ExecutionForwardingV1ReadinessResponse | null>(null);
  const [executionForwardingV1ApplyPreview, setExecutionForwardingV1ApplyPreview] =
    useState<ExecutionForwardingV1ApplyPreviewResponse | null>(null);
  const [executionForwardingV1ApplyPacket, setExecutionForwardingV1ApplyPacket] =
    useState<ExecutionForwardingV1ApplyPacketPreviewResponse | null>(null);
  const [executionForwardingV1CommandPreviewAllowed, setExecutionForwardingV1CommandPreviewAllowed] =
    useState<ExecutionForwardingV1CommandPreviewResponse | null>(null);
  const [executionForwardingV1CommandPreviewBlocked, setExecutionForwardingV1CommandPreviewBlocked] =
    useState<ExecutionForwardingV1CommandPreviewResponse | null>(null);
  const [executionForwardingV1RollbackPreview, setExecutionForwardingV1RollbackPreview] =
    useState<ExecutionForwardingV1RollbackPreviewResponse | null>(null);
  const [auditEvents, setAuditEvents] = useState<AuditEventRecord[]>([]);
  const [events, setEvents] = useState<EventRecord[]>([]);
  const [selection, setSelection] = useState<DashboardSelection | null>(null);
  const [state, setState] = useState<LoadState>("idle");
  const [error, setError] = useState("");
  const [auxiliaryError, setAuxiliaryError] = useState("");

  useEffect(() => {
    let cancelled = false;

    async function loadProjects() {
      setState("loading");
      setError("");
      try {
        const result = await api.projects();
        if (cancelled) {
          return;
        }
        setProjects(result.projects);
        setSelectedProjectKey((current) => current || result.projects[0]?.key || "");
        setState("ready");
      } catch (err) {
        if (!cancelled) {
          setError(errorMessage(err));
          setState("error");
        }
      }
    }

    void loadProjects();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!selectedProjectKey) {
      return;
    }
    let cancelled = false;

    async function loadProject() {
      setState("loading");
      setError("");
      setAuxiliaryError("");
      try {
        const [nextSummary, nextReadiness] = await Promise.all([
          api.projectSummary(selectedProjectKey),
          api.projectReadiness(selectedProjectKey),
        ]);
        if (cancelled) {
          return;
        }
        setSummary(nextSummary);
        setReadiness(nextReadiness);
        setState("ready");
      } catch (err) {
        if (!cancelled) {
          setError(errorMessage(err));
          setVersions([]);
          setWorkers([]);
          setAuditEvents([]);
          setWorkerPool(null);
          setSchedulePreview(null);
          setWebWriteGate(null);
          setCompletionSnapshotReadiness(null);
          setOperationsReadiness(null);
          setReleaseFinalGate(null);
          setReleaseEvidenceBundle(null);
          setReleasePackagePreview(null);
          setReleaseDistributionPreview(null);
          setReleasePublishGate(null);
          setReleasePublishApproval(null);
          setReleaseRolloutPlan(null);
          setShimAuthorization(null);
          setShimApplyPacket(null);
          setShimApplyGate(null);
          setStatusProjectionAuthorization(null);
          setStatusProjectionApplyPacket(null);
          setStatusProjectionApplyGate(null);
          setExecutionCutover(null);
          setExecutionForwardingV1Readiness(null);
          setExecutionForwardingV1ApplyPreview(null);
          setExecutionForwardingV1ApplyPacket(null);
          setExecutionForwardingV1CommandPreviewAllowed(null);
          setExecutionForwardingV1CommandPreviewBlocked(null);
          setExecutionForwardingV1RollbackPreview(null);
          setReadiness(null);
          setState("error");
        }
      }
    }

    void loadProject();
    return () => {
      cancelled = true;
    };
  }, [selectedProjectKey]);

  useEffect(() => {
    if (!selectedProjectKey) {
      setVersions([]);
      setSelectedVersionLabel("");
      return;
    }

    let cancelled = false;

    async function loadVersions() {
      setVersions([]);
      setSelectedVersionLabel("");
      setError("");
      try {
        const nextVersions = await api.workflowVersions(selectedProjectKey);
        if (cancelled) {
          return;
        }
        setVersions(nextVersions.workflow_versions);
        setSelectedVersionLabel(nextVersions.workflow_versions[0]?.display_label || "");
      } catch (err) {
        if (!cancelled) {
          setAuxiliaryError(`Version timeline unavailable: ${errorMessage(err)}`);
          setVersions([]);
          setSelectedVersionLabel("");
        }
      }
    }

    void loadVersions();
    return () => {
      cancelled = true;
    };
  }, [selectedProjectKey]);

  useEffect(() => {
    if (!selectedProjectKey) {
      setEvents([]);
      setWorkers([]);
      setAuditEvents([]);
      setWorkerPool(null);
      setSchedulePreview(null);
      setWebWriteGate(null);
      setCompletionSnapshotReadiness(null);
      setOperationsReadiness(null);
      setReleaseFinalGate(null);
      setReleaseEvidenceBundle(null);
      setReleasePackagePreview(null);
      setReleaseDistributionPreview(null);
      setReleasePublishGate(null);
      setReleasePublishApproval(null);
      setReleaseRolloutPlan(null);
      setShimAuthorization(null);
      setShimApplyPacket(null);
      setShimApplyGate(null);
      setStatusProjectionAuthorization(null);
      setStatusProjectionApplyPacket(null);
      setStatusProjectionApplyGate(null);
      setExecutionCutover(null);
      setExecutionForwardingV1Readiness(null);
      setExecutionForwardingV1ApplyPreview(null);
      setExecutionForwardingV1ApplyPacket(null);
      setExecutionForwardingV1CommandPreviewAllowed(null);
      setExecutionForwardingV1CommandPreviewBlocked(null);
      setExecutionForwardingV1RollbackPreview(null);
      return;
    }

    let cancelled = false;

    async function loadAuxiliaryPanels() {
      try {
        const [
          nextEvents,
          nextWorkers,
          nextAuditEvents,
          nextWorkerPool,
          nextSchedulePreview,
          nextWebWriteGate,
          nextCompletionSnapshotReadiness,
          nextOperationsReadiness,
          nextReleaseFinalGate,
          nextReleaseEvidenceBundle,
          nextReleasePackagePreview,
          nextReleaseDistributionPreview,
          nextReleasePublishGate,
          nextReleasePublishApproval,
          nextReleaseRolloutPlan,
          nextShimAuthorization,
          nextShimApplyPacket,
          nextShimApplyGate,
          nextStatusProjectionAuthorization,
          nextStatusProjectionApplyPacket,
          nextStatusProjectionApplyGate,
          nextExecutionCutover,
          nextExecutionForwardingV1Readiness,
          nextExecutionForwardingV1ApplyPreview,
          nextExecutionForwardingV1ApplyPacket,
          nextExecutionForwardingV1CommandPreviewAllowed,
          nextExecutionForwardingV1CommandPreviewBlocked,
          nextExecutionForwardingV1RollbackPreview,
        ] = await Promise.all([
          api.projectEvents(selectedProjectKey),
          api.projectWorkers(selectedProjectKey),
          api.projectAuditEvents(selectedProjectKey),
          api.workerPoolSummary(),
          api.workerPoolSchedulePreview(),
          api.webWriteActionGate(),
          api.completionAuditSnapshotReadiness(selectedProjectKey),
          api.operationsReadiness(),
          api.releaseFinalGate(selectedProjectKey),
          api.releaseEvidenceBundle(selectedProjectKey),
          api.releasePackagePreview(selectedProjectKey),
          api.releaseDistributionPreview(selectedProjectKey),
          api.releasePublishGate(selectedProjectKey),
          api.releasePublishApprovalPreview(selectedProjectKey),
          api.releaseRolloutPlanPreview(selectedProjectKey),
          api.projectShimAuthorization(selectedProjectKey),
          api.projectShimApplyPacket(selectedProjectKey),
          api.projectShimApplyGate(selectedProjectKey),
          api.projectStatusProjectionAuthorization(selectedProjectKey),
          api.projectStatusProjectionApplyPacket(selectedProjectKey),
          api.projectStatusProjectionApplyGate(selectedProjectKey),
          api.projectExecutionCutoverReadiness(selectedProjectKey),
          api.projectExecutionForwardingV1Readiness(selectedProjectKey),
          api.projectExecutionForwardingV1ApplyPreview(selectedProjectKey),
          api.projectExecutionForwardingV1ApplyPacket(selectedProjectKey),
          api.projectExecutionForwardingV1CommandPreview(selectedProjectKey, "read_only_verify"),
          api.projectExecutionForwardingV1CommandPreview(selectedProjectKey, "engine_execution"),
          api.projectExecutionForwardingV1RollbackPreview(selectedProjectKey),
        ]);
        if (cancelled) {
          return;
        }
        setEvents(nextEvents.events);
        setWorkers(nextWorkers.workers);
        setAuditEvents(nextAuditEvents.audit_events);
        setWorkerPool(nextWorkerPool);
        setSchedulePreview(nextSchedulePreview);
        setWebWriteGate(nextWebWriteGate);
        setCompletionSnapshotReadiness(nextCompletionSnapshotReadiness);
        setOperationsReadiness(nextOperationsReadiness);
        setReleaseFinalGate(nextReleaseFinalGate);
        setReleaseEvidenceBundle(nextReleaseEvidenceBundle);
        setReleasePackagePreview(nextReleasePackagePreview);
        setReleaseDistributionPreview(nextReleaseDistributionPreview);
        setReleasePublishGate(nextReleasePublishGate);
        setReleasePublishApproval(nextReleasePublishApproval);
        setReleaseRolloutPlan(nextReleaseRolloutPlan);
        setShimAuthorization(nextShimAuthorization);
        setShimApplyPacket(nextShimApplyPacket);
        setShimApplyGate(nextShimApplyGate);
        setStatusProjectionAuthorization(nextStatusProjectionAuthorization);
        setStatusProjectionApplyPacket(nextStatusProjectionApplyPacket);
        setStatusProjectionApplyGate(nextStatusProjectionApplyGate);
        setExecutionCutover(nextExecutionCutover);
        setExecutionForwardingV1Readiness(nextExecutionForwardingV1Readiness);
        setExecutionForwardingV1ApplyPreview(nextExecutionForwardingV1ApplyPreview);
        setExecutionForwardingV1ApplyPacket(nextExecutionForwardingV1ApplyPacket);
        setExecutionForwardingV1CommandPreviewAllowed(nextExecutionForwardingV1CommandPreviewAllowed);
        setExecutionForwardingV1CommandPreviewBlocked(nextExecutionForwardingV1CommandPreviewBlocked);
        setExecutionForwardingV1RollbackPreview(nextExecutionForwardingV1RollbackPreview);
        setAuxiliaryError("");
      } catch (err) {
        if (!cancelled) {
          setAuxiliaryError(`Auxiliary panels unavailable: ${errorMessage(err)}`);
          setEvents([]);
          setWorkers([]);
          setAuditEvents([]);
          setWorkerPool(null);
          setSchedulePreview(null);
          setWebWriteGate(null);
          setCompletionSnapshotReadiness(null);
          setOperationsReadiness(null);
          setReleaseFinalGate(null);
          setReleaseEvidenceBundle(null);
          setReleasePackagePreview(null);
          setReleaseDistributionPreview(null);
          setReleasePublishGate(null);
          setReleasePublishApproval(null);
          setReleaseRolloutPlan(null);
          setShimAuthorization(null);
          setShimApplyPacket(null);
          setShimApplyGate(null);
          setStatusProjectionAuthorization(null);
          setStatusProjectionApplyPacket(null);
          setStatusProjectionApplyGate(null);
          setExecutionCutover(null);
          setExecutionForwardingV1Readiness(null);
          setExecutionForwardingV1ApplyPreview(null);
          setExecutionForwardingV1ApplyPacket(null);
          setExecutionForwardingV1CommandPreviewAllowed(null);
          setExecutionForwardingV1CommandPreviewBlocked(null);
          setExecutionForwardingV1RollbackPreview(null);
        }
      }
    }

    void loadAuxiliaryPanels();
    return () => {
      cancelled = true;
    };
  }, [selectedProjectKey]);

  useEffect(() => {
    setSelection(null);
  }, [selectedProjectKey, selectedVersionLabel]);

  useEffect(() => {
    if (!selectedProjectKey || !selectedVersionLabel) {
      setItems([]);
      setArtifacts([]);
      setResiduals([]);
      setApprovals([]);
      setRuns([]);
      setRunDetail(null);
      return;
    }

    let cancelled = false;

    async function loadVersion() {
      try {
        const [nextStages, nextArtifacts, nextResiduals, nextApprovals, nextRuns] = await Promise.all([
          api.workflowStages(selectedProjectKey, selectedVersionLabel),
          api.versionArtifacts(selectedProjectKey, selectedVersionLabel),
          api.versionResiduals(selectedProjectKey, selectedVersionLabel),
          api.versionApprovals(selectedProjectKey, selectedVersionLabel),
          api.versionRuns(selectedProjectKey, selectedVersionLabel),
        ]);
        if (cancelled) {
          return;
        }
        setItems(nextStages.items);
        setArtifacts(nextArtifacts.artifacts);
        setResiduals(nextResiduals.residuals);
        setApprovals(nextApprovals.approval_records);
        setRuns(nextRuns.runs);
        setRunDetail(null);
      } catch (err) {
        if (!cancelled) {
          setAuxiliaryError(`Version data unavailable: ${errorMessage(err)}`);
          setItems([]);
          setArtifacts([]);
          setResiduals([]);
          setApprovals([]);
          setRuns([]);
          setRunDetail(null);
        }
      }
    }

    void loadVersion();
    return () => {
      cancelled = true;
    };
  }, [selectedProjectKey, selectedVersionLabel]);

  useEffect(() => {
    if (!selectedProjectKey || !selection || selection.kind !== "run") {
      setRunDetail(null);
      return;
    }

    let cancelled = false;
    const runID = selection.id;

    async function loadRunDetail() {
      try {
        const nextRunDetail = await api.runDetail(selectedProjectKey, runID);
        if (!cancelled) {
          setRunDetail(nextRunDetail);
        }
      } catch (err) {
        if (!cancelled) {
          setError(errorMessage(err));
          setRunDetail(null);
        }
      }
    }

    void loadRunDetail();
    return () => {
      cancelled = true;
    };
  }, [selectedProjectKey, selection]);

  useEffect(() => {
    if (!selectedProjectKey) {
      return;
    }

    const stream = new EventSource(`/api/v1/projects/${selectedProjectKey}/events/stream`);
    stream.onmessage = (event) => {
      pushEvent(event.data);
    };
    stream.addEventListener("project.import.completed", (event) => {
      pushEvent((event as MessageEvent<string>).data);
    });
    stream.addEventListener("project.doctor.completed", (event) => {
      pushEvent((event as MessageEvent<string>).data);
    });

    function pushEvent(data: string) {
      try {
        const next = JSON.parse(data) as EventRecord;
        setEvents((current) => {
          if (current.some((item) => item.id === next.id)) {
            return current;
          }
          return [next, ...current].slice(0, 12);
        });
      } catch {
        setError("event stream returned invalid JSON");
      }
    }

    return () => {
      stream.close();
    };
  }, [selectedProjectKey]);

  const selectedVersion = useMemo(
    () => versions.find((item) => item.display_label === selectedVersionLabel),
    [selectedVersionLabel, versions],
  );

  const stageGroups = useMemo(() => {
    const grouped = new Map<string, WorkflowItem[]>();
    for (const item of items) {
      grouped.set(item.stage, [...(grouped.get(item.stage) ?? []), item]);
    }
    return stageOrder.map((stage) => ({
      stage,
      items: grouped.get(stage) ?? [],
    }));
  }, [items]);

  const artifactsByItemId = useMemo(() => {
    const grouped = new Map<number, ArtifactRecord[]>();
    for (const artifact of artifacts) {
      if (!artifact.workflow_item_id) {
        continue;
      }
      grouped.set(artifact.workflow_item_id, [
        ...(grouped.get(artifact.workflow_item_id) ?? []),
        artifact,
      ]);
    }
    return grouped;
  }, [artifacts]);

  const selectedKey = selection ? selectionKey(selection) : "";
  const project = projects.find((item) => item.key === selectedProjectKey);

  return (
    <main className="shell">
      <aside className="sidebar" aria-label="Projects">
        <div className="brand">
          <span className="brand-mark">AF</span>
          <div>
            <h1>AreaFlow</h1>
            <p>Workflow control plane</p>
          </div>
        </div>
        <div className="section-label">Projects</div>
        <div className="project-list">
          {projects.map((item) => (
            <button
              className={item.key === selectedProjectKey ? "project-button active" : "project-button"}
              data-project-key={item.key}
              key={item.key}
              type="button"
              onClick={() => setSelectedProjectKey(item.key)}
            >
              <span>{item.name || item.key}</span>
              <small>{item.workflow_profile}</small>
            </button>
          ))}
          {projects.length === 0 ? <p className="muted">No projects registered</p> : null}
        </div>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <p className="eyebrow">Dashboard</p>
            <h2>{project?.name ?? (selectedProjectKey || "AreaFlow")}</h2>
          </div>
          <div className="status-strip">
            <StatusPill label="API" value={state === "error" ? "error" : state} />
            <StatusPill label="Readiness" value={readiness?.status ?? "unknown"} />
            <StatusPill label="Profile" value={project?.workflow_profile ?? "none"} />
            <StatusPill label="Adapter" value={project?.adapter ?? "none"} />
          </div>
        </header>

        {error ? <div className="notice">{error}</div> : null}
        {auxiliaryError ? <div className="notice subtle">{auxiliaryError}</div> : null}

        <section className="metric-grid">
          <Metric label="Versions" value={summary?.inventory.versions ?? 0} />
          <Metric label="Artifacts" value={summary?.inventory.artifacts ?? 0} />
          <Metric label="Residuals" value={summary?.inventory.residuals ?? 0} />
          <Metric label="Mirror Exports" value={summary?.inventory.mirror_exports ?? 0} />
        </section>

        <section className="layout-grid">
          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Readiness</p>
                <h3>Project Gate</h3>
              </div>
              <StatusPill label="Status" value={readiness?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No readiness checks available"
              items={(readiness?.items ?? []).slice(0, 8).map((item) => ({
                key: item.key,
                title: item.key,
                meta: item.message,
                tone: item.status,
              }))}
            />
          </div>

          <div className="panel version-panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Workflow Versions</p>
                <h3>Version Timeline</h3>
              </div>
              <select
                aria-label="Workflow version"
                value={selectedVersionLabel}
                onChange={(event) => setSelectedVersionLabel(event.target.value)}
              >
                {versions.map((item) => (
                  <option key={item.display_label} value={item.display_label}>
                    {item.display_label}
                  </option>
                ))}
              </select>
            </div>
            <div className="version-list">
              {versions.map((item) => (
                <button
                  className={item.display_label === selectedVersionLabel ? "version-row active" : "version-row"}
                  key={item.display_label}
                  type="button"
                  onClick={() => setSelectedVersionLabel(item.display_label)}
                >
                  <span>{item.display_label}</span>
                  <small>{item.lifecycle_status}</small>
                  <em>{item.import_mode}</em>
                </button>
              ))}
            </div>
          </div>

          <div className="panel stage-panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">{selectedVersion?.display_label ?? "Version"}</p>
                <h3>Stage Board</h3>
              </div>
              <StatusPill label="Mode" value={selectedVersion?.import_mode ?? "none"} />
            </div>
            <div className="stage-grid">
              {stageGroups.map((group) => (
                <button
                  className={
                    selectedKey === selectionKey({ kind: "stage", stage: group.stage })
                      ? "stage active"
                      : "stage"
                  }
                  key={group.stage}
                  type="button"
                  onClick={() => setSelection({ kind: "stage", stage: group.stage })}
                >
                  <span>{group.stage}</span>
                  <strong>{group.items.length}</strong>
                  {group.items.slice(0, 2).map((item) => (
                    <small key={item.id}>
                      {(item.title || item.item_type).slice(0, 42)}
                      {artifactsByItemId.get(item.id)?.length
                        ? ` · ${artifactsByItemId.get(item.id)?.length} artifacts`
                        : ""}
                    </small>
                  ))}
                  {group.items.length > 2 ? <small>+{group.items.length - 2} more items</small> : null}
                </button>
              ))}
            </div>
          </div>

          <TracePanel
            approvals={approvals}
            artifacts={artifacts}
            items={items}
            onSelect={setSelection}
            residuals={residuals}
            runDetail={runDetail}
            runs={runs}
            selection={selection}
          />

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Runs</p>
                <h3>Run Timeline</h3>
              </div>
              <span className="count">{runs.length}</span>
            </div>
            <RecordList
              empty="No runs for this version"
              onSelect={setSelection}
              selectedKey={selectedKey}
              items={runs.map((item) => ({
                key: selectionKey({ kind: "run", id: item.id }),
                title: `${item.run_type} · ${item.status}`,
                meta: `${item.run_kind || "run"} · ${item.dry_run ? "dry-run" : "live"} · ${formatDate(item.started_at)}`,
                tone: item.risk_level || "low",
                selection: { kind: "run", id: item.id },
              }))}
            />
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Forwarding v1</p>
                <h3>Packet Gate</h3>
              </div>
              <StatusPill label="Packet" value={executionForwardingV1ApplyPacket?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No execution forwarding v1 apply packet loaded"
              items={(executionForwardingV1ApplyPacket?.gate.items ?? [])
                .filter((item) =>
                  [
                    "readiness_snapshot_hash",
                    "legacy_non_write_proof_id",
                    "rollback_plan_id",
                    "protected_path_fingerprint_id",
                    "explicit_approval",
                    "read_only_shim",
                  ].includes(item.key),
                )
                .map((item) => ({
                  key: item.key,
                  title: `${item.key} · ${item.status}`,
                  meta: `${item.category} · expected=${item.expected || "n/a"} · actual=${item.actual || "missing"} · blockers=${item.blocked_by.join(",") || "none"}`,
                  tone: item.status,
                }))}
            />
            {executionForwardingV1ApplyPacket ? (
              <div className="guardrail-strip">
                <span>{executionForwardingV1ApplyPacket.mode}</span>
                <span>decision={executionForwardingV1ApplyPacket.decision}</span>
                <span>hash={formatHash(executionForwardingV1ApplyPacket.packet.readiness_snapshot_hash)}</span>
                <span>
                  legacy_ref={formatShortValue(executionForwardingV1ApplyPacket.packet.legacy_non_write_proof_id)}
                </span>
                <span>rollback_ref={formatShortValue(executionForwardingV1ApplyPacket.packet.rollback_plan_id)}</span>
                <span>
                  fingerprint_ref=
                  {formatShortValue(executionForwardingV1ApplyPacket.packet.protected_path_fingerprint_id)}
                </span>
                <span>eligible={String(executionForwardingV1ApplyPacket.gate.apply_command_eligible)}</span>
                <span>command_created={String(executionForwardingV1ApplyPacket.command_request_created)}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Shim Apply Review</p>
                <h3>Packet Gate</h3>
              </div>
              <StatusPill label="Gate" value={shimApplyGate?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No shim apply gate loaded"
              items={(shimApplyGate?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.status}`,
                meta: `${item.category} · ${item.message} · ${item.blocked_by.slice(0, 2).join(", ") || "no blockers"}`,
                tone: item.status,
              }))}
            />
            {shimApplyPacket && shimApplyGate ? (
              <>
                <RecordList
                  empty="No required proof facts"
                  items={shimApplyGate.required_proof_facts.slice(0, 6).map((item, index) => ({
                    key: `proof-${index}`,
                    title: item,
                    meta: `proof fact ${index + 1}/${shimApplyGate.required_proof_facts.length}`,
                    tone: shimApplyGate.approval_status,
                  }))}
                />
                <div className="guardrail-strip">
                  <span>{shimApplyPacket.mode}</span>
                  <span>decision={shimApplyPacket.decision}</span>
                  <span>command_type={shimApplyPacket.packet.command_type}</span>
                  <span>allowed_files={shimApplyPacket.packet.allowed_files.length}</span>
                  <span>eligible={String(shimApplyGate.apply_command_eligible)}</span>
                  <span>apply_open={String(shimApplyGate.apply_open)}</span>
                  <span>command_created={String(shimApplyGate.command_request_created)}</span>
                  <span>project_write={String(shimApplyGate.project_write_attempted)}</span>
                  <span>execution_write={String(shimApplyGate.execution_write_attempted)}</span>
                  <span>task_loop={String(shimApplyGate.task_loop_run_forwarded)}</span>
                  <span>status_projection={String(shimApplyGate.status_projection_written)}</span>
                  <span>area_matrix_files={String(shimApplyGate.area_matrix_files_modified)}</span>
                  <span>engine_call={String(shimApplyGate.engine_call_attempted)}</span>
                </div>
              </>
            ) : null}
          </div>

          <div className="panel" data-panel="status-projection-authorization">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Status Projection</p>
                <h3>Authorization Preview</h3>
              </div>
              <StatusPill label="Authorization" value={statusProjectionAuthorization?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No status projection authorization loaded"
              items={(statusProjectionAuthorization?.required_preflight ?? []).slice(0, 4).map((item, index) => ({
                key: `status-auth-preflight-${index}`,
                title: item,
                meta: `preflight ${index + 1}/${statusProjectionAuthorization?.required_preflight.length ?? 0}`,
                tone: statusProjectionAuthorization?.approval_status ?? "missing",
              }))}
            />
            {statusProjectionAuthorization ? (
              <div className="guardrail-strip">
                <span>{statusProjectionAuthorization.mode}</span>
                <span>scope={statusProjectionAuthorization.claim_scope}</span>
                <span>not_real_100={String(statusProjectionAuthorization.not_real_100)}</span>
                <span>decision={statusProjectionAuthorization.decision}</span>
                <span>target={statusProjectionAuthorization.target_uri}</span>
                <span>schema={statusProjectionAuthorization.preimage.schema_status}</span>
                <span>preview_only=true</span>
                <span>required={statusProjectionAuthorization.required_authorization_phrase || "none"}</span>
                <span>approval={statusProjectionAuthorization.approval_status}</span>
                <span>apply_open={String(statusProjectionAuthorization.apply_open)}</span>
                <span>
                  command_after_approval=
                  {String(statusProjectionAuthorization.would_create_command_request_after_approval)}
                </span>
                <span>project_write={String(statusProjectionAuthorization.project_write_attempted)}</span>
                <span>execution_write={String(statusProjectionAuthorization.execution_write_attempted)}</span>
                <span>engine_call={String(statusProjectionAuthorization.engine_call_attempted)}</span>
                <span>protected={String(Boolean(statusProjectionAuthorization.protected_path_fingerprint_sha256))}</span>
              </div>
            ) : null}
          </div>

          <div className="panel" data-panel="status-projection-gate">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Status Projection</p>
                <h3>Package A Gate</h3>
              </div>
              <StatusPill label="Gate" value={statusProjectionApplyGate?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No status projection apply gate loaded"
              items={(statusProjectionApplyGate?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.status}`,
                meta: `${item.category} · ${item.message} · ${item.blocked_by.slice(0, 2).join(", ") || "no blockers"}`,
                tone: item.status,
              }))}
            />
            {statusProjectionApplyPacket && statusProjectionApplyGate ? (
              <div className="guardrail-strip">
                <span>{statusProjectionApplyPacket.mode}</span>
                <span>scope={statusProjectionApplyPacket.claim_scope}</span>
                <span>not_real_100={String(statusProjectionApplyPacket.not_real_100)}</span>
                <span>decision={statusProjectionApplyPacket.decision}</span>
                <span>target={statusProjectionApplyPacket.packet.target_uri}</span>
                <span>preview_only=true</span>
                <span>apply_run={String(statusProjectionApplyPacket.safety_facts.apply_command_executed ?? false)}</span>
                <span>
                  applied=
                  {String(
                    statusProjectionApplyGate.status_projection_written ||
                      statusProjectionApplyGate.project_write_attempted,
                  )}
                </span>
                <span>required={statusProjectionApplyPacket.required_authorization_phrase || "none"}</span>
                <span>blockers={formatReal100Blockers(statusProjectionApplyPacket.blockers)}</span>
                <span>eligible={String(statusProjectionApplyGate.apply_command_eligible)}</span>
                <span>eligible_is_not_apply={String(statusProjectionApplyGate.apply_command_eligible_is_not_apply)}</span>
                <span>separate_apply={String(statusProjectionApplyGate.requires_separate_apply_command)}</span>
                <span>approval={statusProjectionApplyGate.approval_status}</span>
                <span>command_created={String(statusProjectionApplyGate.command_request_created)}</span>
                <span>status_projection={String(statusProjectionApplyGate.status_projection_written)}</span>
                <span>project_write={String(statusProjectionApplyGate.project_write_attempted)}</span>
                <span>execution_write={String(statusProjectionApplyGate.execution_write_attempted)}</span>
                <span>engine_call={String(statusProjectionApplyGate.engine_call_attempted)}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Forwarding v1</p>
                <h3>Command Preview</h3>
              </div>
              <StatusPill
                label="Preview"
                value={executionForwardingV1CommandPreviewAllowed?.status ?? "unknown"}
              />
            </div>
            <RecordList
              empty="No execution forwarding v1 command preview loaded"
              items={[executionForwardingV1CommandPreviewAllowed, executionForwardingV1CommandPreviewBlocked]
                .filter((item): item is ExecutionForwardingV1CommandPreviewResponse => Boolean(item))
                .map((item) => ({
                  key: item.task_type,
                  title: `${item.task_type} · ${item.decision}`,
                  meta: `${item.failure_mode} · ${item.target_command_type || item.blocked_by.join(",")}`,
                  tone: item.decision,
                }))}
            />
            {executionForwardingV1CommandPreviewAllowed || executionForwardingV1CommandPreviewBlocked ? (
              <div className="guardrail-strip">
                {executionForwardingV1CommandPreviewAllowed ? (
                  <span>allowed={executionForwardingV1CommandPreviewAllowed.decision}</span>
                ) : null}
                {executionForwardingV1CommandPreviewBlocked ? (
                  <span>blocked={executionForwardingV1CommandPreviewBlocked.decision}</span>
                ) : null}
                <span>
                  command_created={String(
                    executionForwardingV1CommandPreviewAllowed?.safety_facts.area_flow_command_created ?? false,
                  )}
                </span>
                <span>
                  task_loop={String(
                    executionForwardingV1CommandPreviewAllowed?.safety_facts.task_loop_run_forwarded ?? false,
                  )}
                </span>
                <span>
                  engine={String(
                    executionForwardingV1CommandPreviewBlocked?.safety_facts.engine_call_attempted ?? false,
                  )}
                </span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Artifacts</p>
                <h3>Version Files</h3>
              </div>
              <span className="count">{artifacts.length}</span>
            </div>
            <RecordList
              empty="No artifacts for this version"
              onSelect={setSelection}
              selectedKey={selectedKey}
              items={artifacts.map((item) => ({
                key: selectionKey({ kind: "artifact", id: item.id }),
                title: item.source_path || item.uri,
                meta: `${linkedItemLabel(item, items)} · ${item.artifact_type} · ${formatBytes(item.size_bytes)}`,
                tone: item.storage_backend,
                selection: { kind: "artifact", id: item.id },
              }))}
            />
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Residuals</p>
                <h3>Blockers</h3>
              </div>
              <span className="count">{residuals.length}</span>
            </div>
            <RecordList
              empty="No residuals for this version"
              onSelect={setSelection}
              selectedKey={selectedKey}
              items={residuals.map((item) => ({
                key: selectionKey({ kind: "residual", key: item.residual_key }),
                title: item.title || item.residual_key,
                meta: item.current_impact || item.close_condition || item.type,
                tone: item.status,
                selection: { kind: "residual", key: item.residual_key },
              }))}
            />
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Approvals</p>
                <h3>Approval Records</h3>
              </div>
              <span className="count">{approvals.length}</span>
            </div>
            <RecordList
              empty="No approval records for this version"
              onSelect={setSelection}
              selectedKey={selectedKey}
              items={approvals.map((item) => ({
                key: selectionKey({ kind: "approval", id: item.id }),
                title: `${item.decision || "pending"} · ${item.approval_kind}`,
                meta: `${item.actor || "unknown"} · ${item.reason || item.scope_id || item.scope_type} · ${formatDate(item.created_at)}`,
                tone: item.risk_level || "normal",
                selection: { kind: "approval", id: item.id },
              }))}
            />
          </div>

          <div className="panel event-panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Live Events</p>
                <h3>Timeline</h3>
              </div>
              <StatusPill label="SSE" value={events.length > 0 ? "connected" : "waiting"} />
            </div>
            <RecordList
              empty="No events yet"
              items={events.map((item) => ({
                key: String(item.id),
                title: item.message,
                meta: `${item.type} · ${formatDate(item.created_at)}`,
                tone: item.severity,
              }))}
            />
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Audit Trail</p>
                <h3>Access Decisions</h3>
              </div>
              <span className="count">{auditEvents.length}</span>
            </div>
            <RecordList
              empty="No audit events yet"
              items={auditEvents.map((item) => ({
                key: String(item.id),
                title: `${item.action} · ${item.decision}`,
                meta: `${auditResourceLabel(item)} · ${item.reason || "no reason"} · ${formatDate(item.created_at)}`,
                tone: item.capability || "audit",
              }))}
            />
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Workers</p>
                <h3>Worker Status</h3>
              </div>
              <span className="count">{workers.length}</span>
            </div>
            <RecordList
              empty="No workers registered for this project"
              items={workers.map((item) => ({
                key: String(item.id),
                title: `${item.worker_key} · ${item.status}`,
                meta: `${item.worker_type} · ${item.hostname || "host unknown"} · ${workerHeartbeatLabel(item)}`,
                tone: `${item.capabilities.length} caps`,
              }))}
            />
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Worker Pool</p>
                <h3>Multi-project</h3>
              </div>
              <StatusPill label="Queued" value={String(workerPool?.total_queued_tasks ?? 0)} />
            </div>
            <RecordList
              empty="No projects in worker pool"
              items={(workerPool?.projects ?? []).map((item) => ({
                key: item.project.key,
                title: `${item.project.key} · ${item.online_workers}/${item.workers} online · ${item.scheduling.agent_role}`,
                meta: `leases ${item.active_leases} · queued ${item.queued_tasks} · recovery ${item.needs_recovery_leases + item.needs_recovery_tasks} · workers ${item.worker_types.join(",") || "none"}`,
                tone: `p${item.scheduling.priority} · ${item.role.status}/${item.engine.status}/${item.resources.status}`,
              }))}
            />
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Schedule Preview</p>
                <h3>Dry-run Order</h3>
              </div>
              <StatusPill label="Ready" value={String(schedulePreview?.recommended ?? 0)} />
            </div>
            <RecordList
              empty="No schedulable projects"
              items={(schedulePreview?.projects ?? []).map((item) => ({
                key: item.project.key,
                title: `${item.project.key} · ${item.recommended ? "recommended" : "blocked"}`,
                meta: `queued ${item.queued_tasks} · slots ${item.available_slots}/${item.max_parallel} · role ${item.role.status} · ${engineLabel(item.engine.status, item.engine.profile_id)} · engine blockers ${blockedReasonsLabel(item.engine.blocked_reasons)} · resources ${item.resources.status} · ${item.next_action}`,
                tone: item.recommended
                  ? `p${item.priority} · ${capabilityCountLabel(item.required_capabilities)}`
                  : blockedReasonsLabel(item.blocked_reasons),
              }))}
            />
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Operations</p>
                <h3>Readiness</h3>
              </div>
              <StatusPill label="Ops" value={operationsReadiness?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No operations readiness loaded"
              items={(operationsReadiness?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.status}`,
                meta: `${item.category} · ${item.message} · ${item.next_command || "no command"}`,
                tone: item.blocked_by.length > 0 ? item.blocked_by.slice(0, 2).join(", ") : item.status,
              }))}
            />
            {operationsReadiness ? (
              <div className="guardrail-strip">
                <span>{operationsReadiness.mode}</span>
                <span>service={operationsReadiness.service_status.status}</span>
                <span>support={operationsReadiness.support_bundle.status}</span>
                <span>migration={operationsReadiness.migration_ledger.status}</span>
                <span>telemetry={operationsReadiness.telemetry_default}</span>
                <span>managed_ops={operationsReadiness.managed_ops_status}</span>
                <span>support_export={operationsReadiness.support_export_status}</span>
                <span>db_write={String(operationsReadiness.safety_facts.database_write_attempted)}</span>
                <span>project_write={String(operationsReadiness.safety_facts.project_write_attempted)}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Shim Authorization</p>
                <h3>AreaMatrix Shim</h3>
              </div>
              <StatusPill label="Gate" value={shimAuthorization?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No shim authorization packet loaded"
              items={(shimAuthorization?.allowed_files ?? []).map((item) => ({
                key: item.path,
                title: `${item.path} · ${item.action}`,
                meta: `${item.required ? "required" : "optional"} · ${item.boundary}`,
                tone: shimAuthorization?.readiness_status ?? "unknown",
              }))}
            />
            {shimAuthorization ? (
              <>
                <RecordList
                  empty="No required preflight"
                  items={shimAuthorization.required_preflight.slice(0, 3).map((item, index) => ({
                    key: `preflight-${index}`,
                    title: item,
                    meta: `required preflight ${index + 1}/${shimAuthorization.required_preflight.length}`,
                    tone: "preflight",
                  }))}
                />
                <RecordList
                  empty="No post-edit verification"
                  items={shimAuthorization.post_edit_verification.slice(0, 3).map((item, index) => ({
                    key: `post-edit-${index}`,
                    title: item,
                    meta: `post-edit verification ${index + 1}/${shimAuthorization.post_edit_verification.length}`,
                    tone: "verify",
                  }))}
                />
                <RecordList
                  empty="No rollback scope"
                  items={shimAuthorization.rollback_scope.slice(0, 3).map((item, index) => ({
                    key: `rollback-${index}`,
                    title: item,
                    meta: `rollback scope ${index + 1}/${shimAuthorization.rollback_scope.length}`,
                    tone: "rollback",
                  }))}
                />
                <div className="guardrail-strip">
                  <span>preflight={shimAuthorization.required_preflight.length}</span>
                  <span>post_edit={shimAuthorization.post_edit_verification.length}</span>
                  <span>rollback={shimAuthorization.rollback_scope.length}</span>
                  <span>project_write={String(shimAuthorization.safety_facts.project_write_attempted)}</span>
                  <span>execution_write={String(shimAuthorization.safety_facts.execution_write_attempted)}</span>
                  <span>task_loop_run={String(shimAuthorization.safety_facts.task_loop_run_forwarded)}</span>
                  <span>engine_call={String(shimAuthorization.safety_facts.engine_call_attempted)}</span>
                </div>
              </>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Execution Cutover</p>
                <h3>AreaMatrix Readiness</h3>
              </div>
              <StatusPill label="Gate" value={executionCutover?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No execution cutover readiness loaded"
              items={(executionCutover?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.status}`,
                meta: `${item.category} · ${item.message} · ${item.next_command}`,
                tone: item.status,
              }))}
            />
            {executionCutover ? (
              <div className="guardrail-strip">
                <span>
                  execution_cutover_apply={String(
                    executionCutover.safety_facts.execution_cutover_apply_open,
                  )}
                </span>
                <span>project_write={String(executionCutover.safety_facts.project_write_attempted)}</span>
                <span>execution_write={String(executionCutover.safety_facts.execution_write_attempted)}</span>
                <span>task_loop_run_forwarded={String(executionCutover.safety_facts.task_loop_run_forwarded)}</span>
                <span>worker_scheduled={String(executionCutover.safety_facts.worker_scheduled)}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Forwarding v1</p>
                <h3>Read-only Scope</h3>
              </div>
              <StatusPill label="Gate" value={executionForwardingV1Readiness?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No execution forwarding v1 readiness loaded"
              items={(executionForwardingV1Readiness?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.status}`,
                meta: `${item.category} · ${item.message} · ${item.next_command}`,
                tone: item.status,
              }))}
            />
            {executionForwardingV1Readiness ? (
              <div className="guardrail-strip">
                <span>{executionForwardingV1Readiness.mode}</span>
                <span>tasks={executionForwardingV1Readiness.allowed_task_types.join(",")}</span>
                <span>apply_open={String(executionForwardingV1Readiness.safety_facts.apply_open ?? false)}</span>
                <span>project_write={String(executionForwardingV1Readiness.safety_facts.project_write_attempted)}</span>
                <span>engine_call={String(executionForwardingV1Readiness.safety_facts.engine_call_attempted)}</span>
                <span>network={String(executionForwardingV1Readiness.safety_facts.network_used)}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Forwarding v1</p>
                <h3>Apply Preview</h3>
              </div>
              <StatusPill label="Apply" value={executionForwardingV1ApplyPreview?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No execution forwarding v1 apply preview loaded"
              items={(executionForwardingV1ApplyPreview?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.status}`,
                meta: `${item.category} · ${item.message} · ${item.next_command}`,
                tone: item.approval_status || item.status,
              }))}
            />
            {executionForwardingV1ApplyPreview ? (
              <div className="guardrail-strip">
                <span>{executionForwardingV1ApplyPreview.mode}</span>
                <span>approval={executionForwardingV1ApplyPreview.approval_status}</span>
                <span>apply_open={String(executionForwardingV1ApplyPreview.apply_open)}</span>
                <span>rollback={executionForwardingV1ApplyPreview.rollback_target}</span>
                <span>targets={executionForwardingV1ApplyPreview.forwarding_targets.length}</span>
                <span>blocked={executionForwardingV1ApplyPreview.blocked_targets.length}</span>
                <span>
                  forwarding_apply={String(
                    executionForwardingV1ApplyPreview.safety_facts.forwarding_v1_apply_open,
                  )}
                </span>
                <span>project_write={String(executionForwardingV1ApplyPreview.safety_facts.project_write_attempted)}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Forwarding v1</p>
                <h3>Rollback Preview</h3>
              </div>
              <StatusPill label="Rollback" value={executionForwardingV1RollbackPreview?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No execution forwarding v1 rollback preview loaded"
              items={(executionForwardingV1RollbackPreview?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.status}`,
                meta: `${item.category} · ${item.message} · ${item.next_command}`,
                tone: item.owner || item.status,
              }))}
            />
            {executionForwardingV1RollbackPreview ? (
              <div className="guardrail-strip">
                <span>{executionForwardingV1RollbackPreview.mode}</span>
                <span>rollback_open={String(executionForwardingV1RollbackPreview.rollback_apply_open)}</span>
                <span>target={executionForwardingV1RollbackPreview.rollback_target}</span>
                <span>fail_closed={executionForwardingV1RollbackPreview.fail_closed_steps.length}</span>
                <span>reopen={executionForwardingV1RollbackPreview.reopen_conditions.length}</span>
                <span>commands={String(executionForwardingV1RollbackPreview.safety_facts.commands_run)}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Completion Snapshot</p>
                <h3>Release Candidate Gate</h3>
              </div>
              <StatusPill label="Gate" value={completionSnapshotReadiness?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No completion snapshot readiness loaded"
              items={(completionSnapshotReadiness?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.status}`,
                meta: item.message,
                tone: item.status,
              }))}
            />
            {completionSnapshotReadiness ? (
              <div className="guardrail-strip">
                <Real100GuardrailStrip value={completionSnapshotReadiness} />
                <span>required={completionSnapshotReadiness.required_class}</span>
                <span>has_snapshot={String(completionSnapshotReadiness.has_snapshot)}</span>
                <span>bundle={formatHash(completionSnapshotReadiness.bundle_hash)}</span>
                <span>latest_class={completionSnapshotReadiness.latest.evidence_class || "none"}</span>
                <span>rc={completionSnapshotReadiness.latest.release_candidate_label || "none"}</span>
                <span>project_write={String(completionSnapshotReadiness.safety_facts.project_write_attempted)}</span>
                <span>execution_write={String(completionSnapshotReadiness.safety_facts.execution_write_attempted)}</span>
                <span>smoke_run={String(completionSnapshotReadiness.safety_facts.smoke_run_attempted)}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Release Final Gate</p>
                <h3>Release Readiness</h3>
              </div>
              <StatusPill label="Gate" value={releaseFinalGate?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No release final gate loaded"
              items={(releaseFinalGate?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.status}`,
                meta: `${item.category} · ${item.message} · ${item.next_command}`,
                tone: item.owner || item.status,
              }))}
            />
            {releaseFinalGate ? (
              <div className="guardrail-strip">
                <span>{releaseFinalGate.mode}</span>
                <Real100GuardrailStrip value={releaseFinalGate} />
                <span>forbidden={releaseFinalGate.forbidden_actions.join(",")}</span>
                <span>generated_at={formatDate(releaseFinalGate.generated_at)}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Release Evidence</p>
                <h3>Evidence Bundle</h3>
              </div>
              <StatusPill label="Gate" value={releaseEvidenceBundle?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No release evidence bundle loaded"
              items={(releaseEvidenceBundle?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.status}`,
                meta: `${item.category} · ${item.source} · ${item.description}`,
                tone: item.status,
              }))}
            />
            {releaseEvidenceBundle ? (
              <div className="guardrail-strip">
                <span>{releaseEvidenceBundle.mode}</span>
                <Real100GuardrailStrip value={releaseEvidenceBundle} />
                <span>bundle={formatHash(releaseEvidenceBundle.bundle_hash)}</span>
                <span>forbidden={releaseEvidenceBundle.forbidden_actions.join(",")}</span>
                <span>generated_at={formatDate(releaseEvidenceBundle.generated_at)}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Release Package</p>
                <h3>Package Preview</h3>
              </div>
              <StatusPill label="Gate" value={releasePackagePreview?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No release package preview loaded"
              items={(releasePackagePreview?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.status}`,
                meta: `${item.package_path || "no package path"} · ${item.source} · ${item.description}`,
                tone: item.category,
              }))}
            />
            {releasePackagePreview ? (
              <div className="guardrail-strip">
                <span>{releasePackagePreview.mode}</span>
                <Real100GuardrailStrip value={releasePackagePreview} />
                <span>{releasePackagePreview.package_name}</span>
                <span>bundle={formatHash(releasePackagePreview.evidence_bundle.bundle_hash)}</span>
                <span>forbidden={releasePackagePreview.forbidden_actions.join(",")}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Release Publish</p>
                <h3>Publish Gate</h3>
              </div>
              <StatusPill label="Gate" value={releasePublishGate?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No release publish gate loaded"
              items={(releasePublishGate?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.status}`,
                meta: `${item.channel} · ${item.message} · ${item.next_command}`,
                tone: item.owner || item.status,
              }))}
            />
            {releasePublishGate ? (
              <div className="guardrail-strip">
                <span>{releasePublishGate.mode}</span>
                <Real100GuardrailStrip value={releasePublishGate} />
                <span>forbidden={releasePublishGate.forbidden_actions.join(",")}</span>
                <span>generated_at={formatDate(releasePublishGate.generated_at)}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Release Distribution</p>
                <h3>Distribution Preview</h3>
              </div>
              <StatusPill label="Gate" value={releaseDistributionPreview?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No release distribution preview loaded"
              items={(releaseDistributionPreview?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.status}`,
                meta: `${item.channel} · ${item.action} · ${item.message}`,
                tone: item.owner || item.category,
              }))}
            />
            {releaseDistributionPreview ? (
              <div className="guardrail-strip">
                <span>{releaseDistributionPreview.mode}</span>
                <Real100GuardrailStrip value={releaseDistributionPreview} />
                <span>forbidden={releaseDistributionPreview.forbidden_actions.join(",")}</span>
                <span>generated_at={formatDate(releaseDistributionPreview.generated_at)}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Release Approval</p>
                <h3>Publish Approval</h3>
              </div>
              <StatusPill label="Gate" value={releasePublishApproval?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No release publish approval preview loaded"
              items={(releasePublishApproval?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.approval_status}`,
                meta: `${item.channel} · ${item.message} · ${item.next_command}`,
                tone: item.status,
              }))}
            />
            {releasePublishApproval ? (
              <div className="guardrail-strip">
                <span>{releasePublishApproval.mode}</span>
                <Real100GuardrailStrip value={releasePublishApproval} />
                <span>forbidden={releasePublishApproval.forbidden_actions.join(",")}</span>
                <span>generated_at={formatDate(releasePublishApproval.generated_at)}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Release Rollout</p>
                <h3>Rollout Plan</h3>
              </div>
              <StatusPill label="Gate" value={releaseRolloutPlan?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No release rollout plan preview loaded"
              items={(releaseRolloutPlan?.items ?? []).map((item) => ({
                key: item.key,
                title: `${item.key} · ${item.status}`,
                meta: `${item.stage} · ${item.action} · ${item.message}`,
                tone: item.owner || item.category,
              }))}
            />
            {releaseRolloutPlan ? (
              <div className="guardrail-strip">
                <span>{releaseRolloutPlan.mode}</span>
                <Real100GuardrailStrip value={releaseRolloutPlan} />
                <span>rollout_steps={releaseRolloutPlan.rollout_steps.length}</span>
                <span>verify={releaseRolloutPlan.verification_checkpoints.length}</span>
                <span>rollback={releaseRolloutPlan.rollback_steps.length}</span>
                <span>forbidden={releaseRolloutPlan.forbidden_actions.join(",")}</span>
              </div>
            ) : null}
          </div>

          <div className="panel">
            <div className="panel-header">
              <div>
                <p className="eyebrow">Web Action Gate</p>
                <h3>Disabled Writes</h3>
              </div>
              <StatusPill label="Gate" value={webWriteGate?.status ?? "unknown"} />
            </div>
            <RecordList
              empty="No web write action gate loaded"
              items={(webWriteGate?.actions ?? []).map((item) => ({
                key: item.key,
                title: `${item.label} · ${item.default_ui_state}`,
                meta: `${item.key} · ${item.command_api} · ${item.risk_level} · ${item.blockers.slice(0, 2).join(", ") || "no blockers"}`,
                tone: item.status,
              }))}
            />
            {webWriteGate ? (
              <div className="guardrail-strip">
                <span>db_write={String(webWriteGate.db_write_attempted)}</span>
                <span>command_created={String(webWriteGate.command_created)}</span>
                <span>worker_scheduled={String(webWriteGate.worker_scheduled)}</span>
              </div>
            ) : null}
          </div>
        </section>
      </section>
    </main>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function StatusPill({ label, value }: { label: string; value: string }) {
  return (
    <span className="pill">
      <small>{label}</small>
      {value}
    </span>
  );
}

function Real100GuardrailStrip({ value }: { value: Real100Guardrail }) {
  return (
    <>
      <span>real_100={value.real_100_status}</span>
      <span>claim_scope={value.claim_scope}</span>
      <span>not_real_100={String(value.not_real_100)}</span>
      <span>evidence_only={String(value.evidence_only)}</span>
      <span>status_alone_is_not_completion={String(value.status_alone_is_not_completion)}</span>
      <span>release_candidate={value.release_candidate_decision}</span>
      <span>scope={value.readiness_scope}</span>
      <span>blockers={formatReal100Blockers(value.real_100_blockers)}</span>
      <span>breakdown={formatReal100Breakdown(value.real_100_breakdown)}</span>
    </>
  );
}

function RecordList({
  empty,
  items,
  onSelect,
  selectedKey,
}: {
  empty: string;
  items: RecordListItem[];
  onSelect?: (selection: DashboardSelection) => void;
  selectedKey?: string;
}) {
  if (items.length === 0) {
    return <p className="muted">{empty}</p>;
  }

  return (
    <div className="records">
      {items.map((item) => {
        const content = (
          <>
            <div>
              <strong>{item.title}</strong>
              <small>{item.meta}</small>
            </div>
            <span>{item.tone}</span>
          </>
        );
        if (onSelect && item.selection) {
          const nextSelection = item.selection;
          return (
            <button
              className={selectedKey === item.key ? "record active" : "record"}
              key={item.key}
              type="button"
              onClick={() => onSelect(nextSelection)}
            >
              {content}
            </button>
          );
        }
        return (
          <div className="record" key={item.key}>
            {content}
          </div>
        );
      })}
    </div>
  );
}

function TracePanel({
  approvals,
  artifacts,
  items,
  onSelect,
  residuals,
  runDetail,
  runs,
  selection,
}: {
  approvals: ApprovalRecord[];
  artifacts: ArtifactRecord[];
  items: WorkflowItem[];
  onSelect: (selection: DashboardSelection) => void;
  residuals: ResidualRecord[];
  runDetail: RunDetailResponse | null;
  runs: RunRecord[];
  selection: DashboardSelection | null;
}) {
  const itemById = new Map(items.map((item) => [item.id, item]));

  if (!selection) {
    return (
      <div className="panel trace-panel">
        <div className="panel-header">
          <div>
            <p className="eyebrow">Trace</p>
            <h3>Detail</h3>
          </div>
        </div>
        <p className="muted">Select a stage, artifact, residual, or approval to inspect its workflow context.</p>
      </div>
    );
  }

  if (selection.kind === "stage") {
    const stageItems = items.filter((item) => item.stage === selection.stage);
    const stageItemIds = new Set(stageItems.map((item) => item.id));
    const stageArtifacts = artifacts.filter((artifact) => (
      artifact.workflow_item_id ? stageItemIds.has(artifact.workflow_item_id) : false
    ));

    return (
      <TraceFrame eyebrow="Trace" title={selection.stage}>
        <div className="detail-grid">
          <DetailRow label="Items" value={String(stageItems.length)} />
          <DetailRow label="Artifacts" value={String(stageArtifacts.length)} />
          <DetailRow label="Stage" value={selection.stage} />
        </div>
        <TraceLinks
          empty="No workflow items in this stage"
          items={stageItems.map((item) => ({
            key: selectionKey({ kind: "item", id: item.id }),
            label: item.title || item.external_key,
            meta: `${item.item_type} · ${item.status || "unknown"}`,
            selection: { kind: "item", id: item.id },
          }))}
          onSelect={onSelect}
        />
      </TraceFrame>
    );
  }

  if (selection.kind === "item") {
    const item = itemById.get(selection.id);
    if (!item) {
      return <MissingTrace />;
    }
    const relatedArtifacts = artifacts.filter((artifact) => artifact.workflow_item_id === item.id);

    return (
      <TraceFrame eyebrow={item.stage} title={item.title || item.external_key}>
        <div className="detail-grid">
          <DetailRow label="Type" value={item.item_type} />
          <DetailRow label="Status" value={item.status || "unknown"} />
          <DetailRow label="External Key" value={item.external_key} />
          <DetailRow label="Metadata" value={metadataSummary(item.metadata)} />
        </div>
        <TraceLinks
          empty="No artifacts linked to this item"
          items={relatedArtifacts.map((artifact) => ({
            key: selectionKey({ kind: "artifact", id: artifact.id }),
            label: artifact.source_path || artifact.uri,
            meta: `${artifact.artifact_type} · ${artifact.storage_backend}`,
            selection: { kind: "artifact", id: artifact.id },
          }))}
          onSelect={onSelect}
        />
      </TraceFrame>
    );
  }

  if (selection.kind === "artifact") {
    const artifact = artifacts.find((item) => item.id === selection.id);
    if (!artifact) {
      return <MissingTrace />;
    }
    const item = artifact.workflow_item_id ? itemById.get(artifact.workflow_item_id) : undefined;

    return (
      <TraceFrame eyebrow="Artifact" title={artifact.source_path || artifact.uri}>
        <div className="detail-grid">
          <DetailRow label="Type" value={artifact.artifact_type} />
          <DetailRow label="Backend" value={artifact.storage_backend} />
          <DetailRow label="Size" value={formatBytes(artifact.size_bytes)} />
          <DetailRow label="Workflow Item" value={item ? `${item.stage} / ${item.item_type}` : "run or project level"} />
          <DetailRow label="SHA-256" value={artifact.sha256 || "missing"} />
          <DetailRow label="Created" value={formatDate(artifact.created_at)} />
          <DetailRow label="URI" value={artifact.uri} />
          <DetailRow label="Metadata" value={metadataSummary(artifact.metadata)} />
        </div>
        {item ? (
          <button className="mini-link single" type="button" onClick={() => onSelect({ kind: "item", id: item.id })}>
            Open workflow item
          </button>
        ) : null}
      </TraceFrame>
    );
  }

  if (selection.kind === "residual") {
    const residual = residuals.find((item) => item.residual_key === selection.key);
    if (!residual) {
      return <MissingTrace />;
    }
    const stage = inferStageFromPath(residual.source_path);

    return (
      <TraceFrame eyebrow="Residual" title={residual.title || residual.residual_key}>
        <div className="detail-grid">
          <DetailRow label="Status" value={residual.status} />
          <DetailRow label="Type" value={residual.type} />
          <DetailRow label="Source" value={residual.source_path || "unknown"} />
          <DetailRow label="Impact" value={residual.current_impact || "none"} />
          <DetailRow label="Close Condition" value={residual.close_condition || "not set"} />
          <DetailRow label="Promotion Required" value={residual.promotion_required ? "yes" : "no"} />
          <DetailRow label="Metadata" value={metadataSummary(residual.metadata)} />
        </div>
        {stage ? (
          <button className="mini-link single" type="button" onClick={() => onSelect({ kind: "stage", stage })}>
            Open inferred stage
          </button>
        ) : null}
      </TraceFrame>
    );
  }

  if (selection.kind === "run") {
    const run = runs.find((item) => item.id === selection.id) ?? runDetail?.run;
    if (!run) {
      return <MissingTrace />;
    }
    const detailArtifacts = runDetail?.run.id === run.id ? runDetail.artifacts : [];
    const detailTasks = runDetail?.run.id === run.id ? runDetail.tasks : [];
    const detailAttempts = runDetail?.run.id === run.id ? runDetail.attempts : [];

    return (
      <TraceFrame eyebrow="Run" title={`${run.run_type} / ${run.status}`}>
        <div className="detail-grid">
          <DetailRow label="Kind" value={run.run_kind || "unknown"} />
          <DetailRow label="Risk" value={`${run.risk_level || "low"} / ${run.risk_policy || "pause"}`} />
          <DetailRow label="Mode" value={run.dry_run ? "dry-run" : "live"} />
          <DetailRow label="Started" value={formatDate(run.started_at)} />
          <DetailRow label="Finished" value={run.finished_at ? formatDate(run.finished_at) : "pending"} />
          <DetailRow label="Tasks" value={String(detailTasks.length)} />
          <DetailRow label="Attempts" value={String(detailAttempts.length)} />
          <DetailRow label="Artifacts" value={String(detailArtifacts.length)} />
          <DetailRow label="Summary" value={metadataSummary(run.summary)} />
          <DetailRow label="Metadata" value={metadataSummary(run.metadata)} />
        </div>
        <TraceLinks
          empty="Run detail is loading or has no tasks"
          items={detailTasks.map((task) => ({
            key: `run-task:${task.id}`,
            label: task.task_key,
            meta: `${task.task_kind} · ${task.status} · seq ${task.sequence}`,
          }))}
        />
        <TraceLinks
          empty="No attempts recorded for this run"
          items={detailAttempts.map((attempt) => ({
            key: `run-attempt:${attempt.id}`,
            label: attempt.attempt_kind,
            meta: `${attempt.status} · ${attempt.dry_run ? "dry-run" : "live"} · ${formatDate(attempt.started_at)}`,
          }))}
        />
        <TraceLinks
          empty="No artifacts recorded for this run"
          items={detailArtifacts.map((artifact) => ({
            key: selectionKey({ kind: "artifact", id: artifact.id }),
            label: artifact.source_path || artifact.uri,
            meta: `${artifact.artifact_type} · ${artifact.storage_backend}`,
            selection: { kind: "artifact", id: artifact.id },
          }))}
          onSelect={onSelect}
        />
      </TraceFrame>
    );
  }

  const approval = approvals.find((item) => item.id === selection.id);
  if (!approval) {
    return <MissingTrace />;
  }
  const scopedItemID = Number(approval.scope_id);
  const scopedItem = Number.isFinite(scopedItemID) ? itemById.get(scopedItemID) : undefined;
  const transitionStage =
    metadataString(approval.metadata, "to_stage") || metadataString(approval.metadata, "from_stage");

  return (
    <TraceFrame eyebrow="Approval" title={`${approval.decision} / ${approval.approval_kind}`}>
      <div className="detail-grid">
        <DetailRow label="Risk" value={approval.risk_level || "normal"} />
        <DetailRow label="Actor" value={approval.actor || "unknown"} />
        <DetailRow label="Reason" value={approval.reason || "not provided"} />
        <DetailRow label="Scope" value={`${approval.scope_type} / ${approval.scope_id}`} />
        <DetailRow
          label="Transition"
          value={approval.transition_preview_id ? String(approval.transition_preview_id) : "none"}
        />
        <DetailRow label="Created" value={formatDate(approval.created_at)} />
        <DetailRow label="Metadata" value={metadataSummary(approval.metadata)} />
      </div>
      {scopedItem ? (
        <button
          className="mini-link single"
          type="button"
          onClick={() => onSelect({ kind: "item", id: scopedItem.id })}
        >
          Open scoped item
        </button>
      ) : null}
      {transitionStage ? (
        <button
          className="mini-link single"
          type="button"
          onClick={() => onSelect({ kind: "stage", stage: transitionStage })}
        >
          Open transition stage
        </button>
      ) : null}
    </TraceFrame>
  );
}

function TraceFrame({
  children,
  eyebrow,
  title,
}: {
  children: ReactNode;
  eyebrow: string;
  title: string;
}) {
  return (
    <div className="panel trace-panel">
      <div className="panel-header">
        <div>
          <p className="eyebrow">{eyebrow}</p>
          <h3>{title}</h3>
        </div>
      </div>
      {children}
    </div>
  );
}

function MissingTrace() {
  return (
    <div className="panel trace-panel">
      <div className="panel-header">
        <div>
          <p className="eyebrow">Trace</p>
          <h3>Missing Record</h3>
        </div>
      </div>
      <p className="muted">The selected record is no longer present in the current version data.</p>
    </div>
  );
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="detail-row">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function TraceLinks({
  empty,
  items,
  onSelect,
}: {
  empty: string;
  items: Array<{ key: string; label: string; meta: string; selection?: DashboardSelection }>;
  onSelect?: (selection: DashboardSelection) => void;
}) {
  if (items.length === 0) {
    return <p className="muted">{empty}</p>;
  }

  return (
    <div className="trace-links">
      {items.map((item) => {
        const content = (
          <>
            <span>{item.label}</span>
            <small>{item.meta}</small>
          </>
        );
        if (onSelect && item.selection) {
          const nextSelection = item.selection;
          return (
            <button className="mini-link" key={item.key} type="button" onClick={() => onSelect(nextSelection)}>
              {content}
            </button>
          );
        }
        return (
          <div className="mini-link" key={item.key}>
            {content}
          </div>
        );
      })}
    </div>
  );
}

function errorMessage(err: unknown) {
  return err instanceof Error ? err.message : "Unknown dashboard error";
}

function formatBytes(value: number) {
  if (!Number.isFinite(value) || value <= 0) {
    return "0 B";
  }
  if (value < 1024) {
    return `${value} B`;
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB`;
  }
  return `${(value / (1024 * 1024)).toFixed(1)} MB`;
}

function formatDate(value: string) {
  if (!value) {
    return "pending";
  }
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function formatHash(value: string) {
  if (!value) {
    return "pending";
  }
  return value.slice(0, 12);
}

function formatShortValue(value: string) {
  if (!value) {
    return "missing";
  }
  if (value.length <= 28) {
    return value;
  }
  return `${value.slice(0, 14)}...${value.slice(-10)}`;
}

function formatReal100Blockers(blockers: string[]) {
  if (blockers.length === 0) {
    return "none";
  }
  const visible = blockers.slice(0, 2).join(",");
  if (blockers.length <= 2) {
    return visible;
  }
  return `${visible},+${blockers.length - 2}`;
}

function formatReal100Breakdown(breakdown?: Real100Guardrail["real_100_breakdown"]) {
  if (!breakdown) {
    return "pending";
  }
  const exact = breakdown.needs_exact_authorization?.length ?? 0;
  const write = breakdown.needs_real_areamatrix_write?.length ?? 0;
  const flow = breakdown.areaflow_only_can_continue?.length ?? 0;
  const done = breakdown.completed_evidence?.length ?? 0;
  return `auth:${exact},write:${write},flow:${flow},done:${done}`;
}

function workerHeartbeatLabel(worker: WorkerRecord) {
  if (worker.last_heartbeat_at) {
    return `heartbeat ${formatDate(worker.last_heartbeat_at)}`;
  }
  return "heartbeat never";
}

function auditResourceLabel(event: AuditEventRecord) {
  const resourceType = event.resource_type || "resource";
  if (!event.resource) {
    return resourceType;
  }
  return `${resourceType} ${event.resource}`;
}

function blockedReasonsLabel(reasons: string[]) {
  if (reasons.length === 0) {
    return "none";
  }
  return reasons.slice(0, 2).join(", ");
}

function capabilityCountLabel(capabilities: string[]) {
  if (capabilities.length === 0) {
    return "no required caps";
  }
  return `${capabilities.length} required caps`;
}

function engineLabel(status: string, profileID: string) {
  if (!profileID) {
    return `engine ${status || "unknown"}`;
  }
  return `${profileID} ${status || "unknown"}`;
}

function selectionKey(selection: DashboardSelection) {
  switch (selection.kind) {
    case "stage":
      return `stage:${selection.stage}`;
    case "item":
      return `item:${selection.id}`;
    case "artifact":
      return `artifact:${selection.id}`;
    case "residual":
      return `residual:${selection.key}`;
    case "approval":
      return `approval:${selection.id}`;
    case "run":
      return `run:${selection.id}`;
  }
}

function linkedItemLabel(artifact: ArtifactRecord, items: WorkflowItem[]) {
  if (!artifact.workflow_item_id) {
    return "project/run level";
  }
  const item = items.find((candidate) => candidate.id === artifact.workflow_item_id);
  if (!item) {
    return `item ${artifact.workflow_item_id}`;
  }
  return `${item.stage} / ${item.item_type}`;
}

function metadataSummary(metadata: Record<string, unknown>) {
  const entries = Object.entries(metadata).filter(([, value]) => value !== undefined && value !== null && value !== "");
  if (entries.length === 0) {
    return "empty";
  }
  return entries
    .slice(0, 4)
    .map(([key, value]) => `${key}=${formatMetadataValue(value)}`)
    .join(" · ");
}

function formatMetadataValue(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  if (Array.isArray(value)) {
    return `${value.length} items`;
  }
  if (value && typeof value === "object") {
    return `${Object.keys(value).length} fields`;
  }
  return "null";
}

function metadataString(metadata: Record<string, unknown>, key: string) {
  const value = metadata[key];
  return typeof value === "string" ? value : "";
}

function inferStageFromPath(path: string) {
  if (!path) {
    return "";
  }
  return stageOrder.find((stage) => path.includes(`/${stage}/`) || path.includes(`${stage}/`)) ?? "";
}
