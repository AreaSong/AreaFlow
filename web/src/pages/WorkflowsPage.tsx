import { useCallback, useEffect, useMemo, useState } from "react";
import { Check, ShieldCheck, X } from "lucide-react";
import { useNavigate, useParams } from "react-router-dom";
import { api } from "../api";
import { EmptyState, ErrorState, ListControls, LoadingState, Metric, PageHeader, Section, StatusBadge } from "../components/ui";
import { useAuth } from "../context/AuthContext";
import { useProject } from "../context/ProjectContext";
import { useAsyncValue } from "../hooks/useAsyncValue";
import { useListView } from "../hooks/useListView";
import { formatDate } from "../lib/format";
import type { TransitionPreview, WorkflowItem } from "../types";

type ApprovalDecision = "approved" | "rejected";

export function WorkflowsPage() {
  const { projectKey, version } = useParams();
  const { projects, selectedProjectKey, setSelectedProjectKey } = useProject();
  const { principal, allowsCapability } = useAuth();
  const activeProjectKey = projectKey || selectedProjectKey;
  const navigate = useNavigate();
  const versions = useAsyncValue(async () => {
    const result = await api.workflows(activeProjectKey);
    return result.workflows.map((item) => item.workflow_version);
  }, [activeProjectKey]);
  const [selectedVersion, setSelectedVersion] = useState(version ?? "");
  const [sort, setSort] = useState("sequence");
  const [approval, setApproval] = useState<{ preview: TransitionPreview; decision: ApprovalDecision } | null>(null);
  const [reason, setReason] = useState("");
  const [submitError, setSubmitError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (projectKey && projectKey !== selectedProjectKey && projects.some((project) => project.key === projectKey)) {
      setSelectedProjectKey(projectKey);
    }
  }, [projectKey, projects, selectedProjectKey, setSelectedProjectKey]);
  useEffect(() => {
    if (version) setSelectedVersion(version);
    else if (versions.data?.[0]) setSelectedVersion(versions.data[0].display_label);
  }, [version, versions.data]);

  const detail = useAsyncValue(async () => {
    if (!activeProjectKey || !selectedVersion) return null;
    const [stages, approvals, residuals, previews] = await Promise.all([
      api.workflowStages(activeProjectKey, selectedVersion),
      api.versionApprovals(activeProjectKey, selectedVersion),
      api.versionResiduals(activeProjectKey, selectedVersion),
      api.versionTransitionPreviews(activeProjectKey, selectedVersion),
    ]);
    return { stages, approvals, residuals, previews };
  }, [activeProjectKey, selectedVersion]);

  const sortedItems = useMemo(() => [...(detail.data?.stages.items ?? [])].sort((left, right) => {
    if (sort === "title") return (left.title || left.external_key).localeCompare(right.title || right.external_key);
    if (sort === "status") return left.status.localeCompare(right.status) || left.id - right.id;
    return left.id - right.id;
  }), [detail.data, sort]);
  const itemList = useListView(sortedItems, useCallback((item, query) => `${item.stage} ${item.item_type} ${item.external_key} ${item.title} ${item.status}`.toLowerCase().includes(query), []), 12);
  const stageGroups = useMemo(() => {
    const groups = itemList.items.reduce<Record<string, WorkflowItem[]>>((result, item) => {
      (result[item.stage] ??= []).push(item);
      return result;
    }, {});
    return Object.entries(groups);
  }, [itemList.items]);
  const readyPreviews = detail.data?.previews.transition_previews.filter((item) => item.status === "ready") ?? [];
  const canApprove = allowsCapability("workflow.approval.record");

  function selectVersion(nextVersion: string) {
    setSelectedVersion(nextVersion);
    if (activeProjectKey) navigate(`/projects/${encodeURIComponent(activeProjectKey)}/workflows/${encodeURIComponent(nextVersion)}?project=${encodeURIComponent(activeProjectKey)}`);
  }

  function openApproval(preview: TransitionPreview, decision: ApprovalDecision) {
    setApproval({ preview, decision });
    setReason("");
    setSubmitError("");
  }

  async function submitApproval() {
    if (!approval || !activeProjectKey || !reason.trim()) return;
    setSubmitting(true);
    setSubmitError("");
    try {
      await api.createApproval(activeProjectKey, selectedVersion, {
        decision: approval.decision,
        reason: reason.trim(),
        transitionPreviewID: approval.preview.id,
        actor: principal.actor,
      }, crypto.randomUUID());
      setApproval(null);
      setReason("");
      detail.retry();
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : String(error));
    } finally {
      setSubmitting(false);
    }
  }

  return <div className="page">
    <PageHeader eyebrow="Lifecycle" title="Workflows" description="Versioned stages, workflow items, approvals, and residual work." actions={<select aria-label="Workflow version" value={selectedVersion} onChange={(event) => selectVersion(event.target.value)}>{versions.data?.map((item) => <option key={item.id} value={item.display_label}>{item.display_label}</option>)}</select>} />
    {versions.loading ? <LoadingState label="Loading workflows" /> : versions.error ? <ErrorState message={versions.error} onRetry={versions.retry} /> : versions.data?.length === 0 ? <EmptyState message="No workflow versions" /> : detail.loading ? <LoadingState label="Loading workflow details" /> : detail.error ? <ErrorState message={detail.error} onRetry={detail.retry} /> : detail.data ? <>
      <div className="summary-grid"><Metric label="Versions" value={versions.data?.length ?? 0} /><Metric label="Stages" value={stageGroups.length} /><Metric label="Items" value={detail.data.stages.items.length} /><Metric label="Ready decisions" value={readyPreviews.length} /></div>
      <Section title="Stage board" description={`${selectedVersion} lifecycle state`}>
        <ListControls query={itemList.query} onQueryChange={itemList.setQuery} page={itemList.page} pageCount={itemList.pageCount} total={itemList.total} onPageChange={itemList.setPage} placeholder="Search workflow items" sortValue={sort} onSortChange={setSort} sortOptions={[{ value: "sequence", label: "Workflow sequence" }, { value: "title", label: "Title A-Z" }, { value: "status", label: "Status A-Z" }]} />
        {itemList.total ? <div className="stage-board">{stageGroups.map(([stage, items]) => <div key={stage}><header><strong>{stage}</strong><span>{items.length}</span></header><div>{items.map((item) => <article key={item.id}><span>{item.item_type}</span><strong>{item.title || item.external_key}</strong><StatusBadge value={item.status} /></article>)}</div></div>)}</div> : <EmptyState message="No workflow items match this search" />}
      </Section>
      <Section title="Transition decisions" description="Only ready previews can be approved; every decision is audited.">
        {readyPreviews.length ? <div className="decision-list">{readyPreviews.map((preview) => <div key={preview.id}><div><strong>{preview.from_stage} → {preview.to_stage}</strong><small>Preview #{preview.id} · {preview.required_gate_name} · {formatDate(preview.created_at)}</small></div><StatusBadge value={preview.status} />{canApprove ? <div className="decision-actions"><button type="button" className="icon-command approve" onClick={() => openApproval(preview, "approved")} title="批准" aria-label={`批准 preview ${preview.id}`}><Check size={16} /></button><button type="button" className="icon-command reject" onClick={() => openApproval(preview, "rejected")} title="拒绝" aria-label={`拒绝 preview ${preview.id}`}><X size={16} /></button></div> : null}</div>)}</div> : <EmptyState message="No ready transition previews" />}
      </Section>
      <div className="page-grid two-columns"><Section title="Approvals">{detail.data.approvals.approval_records.length ? <div className="table-list">{detail.data.approvals.approval_records.map((item) => <div key={item.id}><div><strong>{item.approval_kind}</strong><small>{item.actor} · {item.reason} · {formatDate(item.created_at)}</small></div><StatusBadge value={item.decision} /></div>)}</div> : <EmptyState message="No approval records" />}</Section><Section title="Residual work">{detail.data.residuals.residuals.length ? <div className="table-list">{detail.data.residuals.residuals.map((item) => <div key={item.id}><div><strong>{item.title}</strong><small>{item.current_impact}</small></div><StatusBadge value={item.status} /></div>)}</div> : <EmptyState message="No residual work" />}</Section></div>
    </> : null}
    {approval ? <div className="modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget && !submitting) setApproval(null); }}><div className="command-modal" role="dialog" aria-modal="true" aria-labelledby="approval-title"><header><div><ShieldCheck size={20} /><div><h2 id="approval-title">{approval.decision === "approved" ? "批准 transition" : "拒绝 transition"}</h2><p>Preview #{approval.preview.id} · {approval.preview.from_stage} → {approval.preview.to_stage}</p></div></div><button type="button" className="icon-command" onClick={() => setApproval(null)} disabled={submitting} aria-label="关闭"><X size={16} /></button></header><label><span>原因</span><textarea value={reason} onChange={(event) => setReason(event.target.value)} rows={4} autoFocus placeholder="记录可长期审计的决策原因" /></label>{submitError ? <p className="form-error">{submitError}</p> : null}<footer><button type="button" className="secondary-button" onClick={() => setApproval(null)} disabled={submitting}>取消</button><button type="button" className={approval.decision === "approved" ? "primary-button" : "danger-button"} onClick={submitApproval} disabled={submitting || !reason.trim()}>{approval.decision === "approved" ? <Check size={16} /> : <X size={16} />}{submitting ? "提交中" : approval.decision === "approved" ? "确认批准" : "确认拒绝"}</button></footer></div></div> : null}
  </div>;
}
