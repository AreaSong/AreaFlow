import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { api } from "../api";
import { DefinitionList, EmptyState, ErrorState, ListControls, LoadingState, Metric, PageHeader, Section, StatusBadge } from "../components/ui";
import { useProject } from "../context/ProjectContext";
import { useAsyncValue } from "../hooks/useAsyncValue";
import { useListView } from "../hooks/useListView";
import { formatBytes, formatDate } from "../lib/format";
import type { RunRecord } from "../types";

type ProjectRun = RunRecord & { versionLabel: string };

export function RunsPage() {
  const params = useParams();
  const navigate = useNavigate();
  const { selectedProjectKey } = useProject();
  const runs = useAsyncValue(async () => {
    const result = await api.runs(selectedProjectKey);
    return result.runs.map((item): ProjectRun => ({ ...item.run, versionLabel: item.workflow_version.display_label }));
  }, [selectedProjectKey]);
  const [selectedVersion, setSelectedVersion] = useState("all");
  const [sort, setSort] = useState("newest");
  const [selectedRunID, setSelectedRunID] = useState<number | null>(params.runId ? Number(params.runId) : null);
  const versionLabels = useMemo(() => Array.from(new Set(runs.data?.map((run) => run.versionLabel) ?? [])), [runs.data]);
  const filteredRuns = useMemo(() => (runs.data?.filter((run) => selectedVersion === "all" || run.versionLabel === selectedVersion) ?? []).sort((left, right) => {
    if (sort === "oldest") return left.started_at.localeCompare(right.started_at) || left.id - right.id;
    if (sort === "status") return left.status.localeCompare(right.status) || right.id - left.id;
    return right.started_at.localeCompare(left.started_at) || right.id - left.id;
  }), [runs.data, selectedVersion, sort]);
  const runList = useListView(filteredRuns, useCallback((run, query) => `run ${run.id} ${run.versionLabel} ${run.run_type} ${run.run_kind} ${run.status} ${run.risk_level}`.toLowerCase().includes(query), []), 10);
  useEffect(() => {
    if (params.runId) return;
    setSelectedRunID(runList.items[0]?.id ?? null);
  }, [runList.items, params.runId]);
  const detail = useAsyncValue(async () => {
    if (!selectedRunID) return null;
    const [run, tasks, attempts] = await Promise.all([
      api.runDetail(selectedProjectKey, selectedRunID),
      api.runTasks(selectedProjectKey, selectedRunID),
      api.runAttempts(selectedProjectKey, selectedRunID),
    ]);
    return { ...run, tasks: tasks.tasks, attempts: attempts.attempts };
  }, [selectedProjectKey, selectedRunID]);
  const counts = useMemo(() => ({ active: filteredRuns.filter((run) => ["running", "queued"].includes(run.status)).length, failed: filteredRuns.filter((run) => run.status === "failed").length }), [filteredRuns]);

  return <div className="page"><PageHeader eyebrow="Execution" title="Runs" description="Runs, tasks, attempts, and execution evidence for the selected project." actions={<select aria-label="Run workflow version" value={selectedVersion} onChange={(event) => setSelectedVersion(event.target.value)}><option value="all">All versions</option>{versionLabels.map((version) => <option key={version} value={version}>{version}</option>)}</select>} />
    {runs.loading ? <LoadingState label="Loading runs" /> : runs.error ? <ErrorState message={runs.error} onRetry={runs.retry} /> : filteredRuns.length === 0 ? <EmptyState message="No runs have been recorded" /> : <>
      <div className="summary-grid"><Metric label="Runs" value={filteredRuns.length} /><Metric label="Active" value={counts.active} /><Metric label="Failed" value={counts.failed} /><Metric label="Selected" value={selectedRunID ?? "-"} /></div>
      <div className="resource-layout"><Section title="Run timeline" className="resource-index"><ListControls query={runList.query} onQueryChange={runList.setQuery} page={runList.page} pageCount={runList.pageCount} total={runList.total} onPageChange={runList.setPage} placeholder="Search runs" sortValue={sort} onSortChange={setSort} sortOptions={[{ value: "newest", label: "Newest first" }, { value: "oldest", label: "Oldest first" }, { value: "status", label: "Status A-Z" }]} /><div className="resource-list dense">{runList.items.map((run) => <button key={run.id} className={run.id === selectedRunID ? "active" : ""} onClick={() => { setSelectedRunID(run.id); navigate(`/runs/${run.id}?project=${encodeURIComponent(selectedProjectKey)}`); }}><div><strong>Run #{run.id}</strong><small>{run.versionLabel} / {run.run_kind}</small></div><StatusBadge value={run.status} /></button>)}</div>{runList.total === 0 ? <EmptyState message="No runs match this search" /> : null}</Section>
        <div className="resource-detail">{detail.loading ? <LoadingState label="Loading run detail" /> : detail.error ? <ErrorState message={detail.error} onRetry={detail.retry} /> : detail.data ? <>
          <Section title={`Run #${detail.data.run.id}`} actions={<StatusBadge value={detail.data.run.status} />}><DefinitionList rows={[["Type", detail.data.run.run_type], ["Kind", detail.data.run.run_kind], ["Risk", detail.data.run.risk_level], ["Dry run", String(detail.data.run.dry_run)], ["Started", formatDate(detail.data.run.started_at)], ["Finished", formatDate(detail.data.run.finished_at)]]} /></Section>
          <div className="page-grid two-columns"><Section title="Tasks">{detail.data.tasks.length ? <div className="table-list">{detail.data.tasks.map((task) => <div key={task.id}><div><strong>{task.task_key}</strong><small>{task.task_kind}</small></div><StatusBadge value={task.status} /></div>)}</div> : <EmptyState message="No tasks recorded for this run" />}</Section><Section title="Attempts">{detail.data.attempts.length ? <div className="table-list">{detail.data.attempts.map((attempt) => <div key={attempt.id}><div><strong>{attempt.attempt_kind}</strong><small>{formatDate(attempt.started_at)}</small></div><StatusBadge value={attempt.status} /></div>)}</div> : <EmptyState message="No attempts recorded for this run" />}</Section></div>
          <Section title="Execution artifacts">{detail.data.artifacts.length ? <div className="table-list">{detail.data.artifacts.map((artifact) => <div key={artifact.id}><div><strong>{artifact.source_path || artifact.uri}</strong><small>{artifact.artifact_type} / {formatBytes(artifact.size_bytes)}</small></div><StatusBadge value={artifact.storage_backend} /></div>)}</div> : <EmptyState message="No artifacts linked to this run" />}</Section>
        </> : null}</div>
      </div>
    </>}
  </div>;
}
