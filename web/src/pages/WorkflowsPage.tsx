import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { api } from "../api";
import { EmptyState, ErrorState, ListControls, LoadingState, Metric, PageHeader, Section, StatusBadge } from "../components/ui";
import { useProject } from "../context/ProjectContext";
import { useAsyncValue } from "../hooks/useAsyncValue";
import { useListView } from "../hooks/useListView";
import { formatDate } from "../lib/format";
import type { WorkflowItem } from "../types";

export function WorkflowsPage() {
  const { projectKey, version } = useParams();
  const { selectedProjectKey } = useProject();
  const activeProjectKey = projectKey || selectedProjectKey;
  const navigate = useNavigate();
  const versions = useAsyncValue(async () => {
    const result = await api.workflows(activeProjectKey);
    return result.workflows.map((item) => item.workflow_version);
  }, [activeProjectKey]);
  const [selectedVersion, setSelectedVersion] = useState(version ?? "");
  const [sort, setSort] = useState("sequence");
  useEffect(() => {
    if (version) setSelectedVersion(version);
    else if (versions.data?.[0]) setSelectedVersion(versions.data[0].display_label);
  }, [version, versions.data]);
  const detail = useAsyncValue(async () => {
    if (!activeProjectKey || !selectedVersion) return null;
    const [stages, approvals, residuals] = await Promise.all([
      api.workflowStages(activeProjectKey, selectedVersion),
      api.versionApprovals(activeProjectKey, selectedVersion),
      api.versionResiduals(activeProjectKey, selectedVersion),
    ]);
    return { stages, approvals, residuals };
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

  function selectVersion(nextVersion: string) {
    setSelectedVersion(nextVersion);
    if (activeProjectKey) navigate(`/projects/${encodeURIComponent(activeProjectKey)}/workflows/${encodeURIComponent(nextVersion)}?project=${encodeURIComponent(activeProjectKey)}`);
  }

  return <div className="page"><PageHeader eyebrow="Lifecycle" title="Workflows" description="Versioned stages, workflow items, approvals, and residual work." actions={<select aria-label="Workflow version" value={selectedVersion} onChange={(event) => selectVersion(event.target.value)}>{versions.data?.map((item) => <option key={item.id} value={item.display_label}>{item.display_label}</option>)}</select>} />
    {versions.loading ? <LoadingState label="Loading workflows" /> : versions.error ? <ErrorState message={versions.error} onRetry={versions.retry} /> : versions.data?.length === 0 ? <EmptyState message="No workflow versions" /> : detail.loading ? <LoadingState label="Loading workflow details" /> : detail.error ? <ErrorState message={detail.error} onRetry={detail.retry} /> : detail.data ? <>
      <div className="summary-grid"><Metric label="Versions" value={versions.data?.length ?? 0} /><Metric label="Stages" value={stageGroups.length} /><Metric label="Items" value={detail.data.stages.items.length} /><Metric label="Open residuals" value={detail.data.residuals.residuals.filter((item) => item.status !== "closed").length} /></div>
      <Section title="Stage board" description={`${selectedVersion} lifecycle state`}><ListControls query={itemList.query} onQueryChange={itemList.setQuery} page={itemList.page} pageCount={itemList.pageCount} total={itemList.total} onPageChange={itemList.setPage} placeholder="Search workflow items" sortValue={sort} onSortChange={setSort} sortOptions={[{ value: "sequence", label: "Workflow sequence" }, { value: "title", label: "Title A-Z" }, { value: "status", label: "Status A-Z" }]} />{itemList.total ? <div className="stage-board">{stageGroups.map(([stage, items]) => <div key={stage}><header><strong>{stage}</strong><span>{items?.length ?? 0}</span></header><div>{items?.map((item) => <article key={item.id}><span>{item.item_type}</span><strong>{item.title || item.external_key}</strong><StatusBadge value={item.status} /></article>)}</div></div>)}</div> : <EmptyState message="No workflow items match this search" />}</Section>
      <div className="page-grid two-columns"><Section title="Approvals">{detail.data.approvals.approval_records.length ? <div className="table-list">{detail.data.approvals.approval_records.map((item) => <div key={item.id}><div><strong>{item.approval_kind}</strong><small>{item.actor} / {formatDate(item.created_at)}</small></div><StatusBadge value={item.decision} /></div>)}</div> : <EmptyState message="No approval records" />}</Section><Section title="Residual work">{detail.data.residuals.residuals.length ? <div className="table-list">{detail.data.residuals.residuals.map((item) => <div key={item.id}><div><strong>{item.title}</strong><small>{item.current_impact}</small></div><StatusBadge value={item.status} /></div>)}</div> : <EmptyState message="No residual work" />}</Section></div>
    </> : null}
  </div>;
}
