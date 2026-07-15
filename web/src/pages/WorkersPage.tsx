import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { api } from "../api";
import { DefinitionList, EmptyState, ErrorState, ListControls, LoadingState, Metric, PageHeader, Section, StatusBadge } from "../components/ui";
import { useProject } from "../context/ProjectContext";
import { useAsyncValue } from "../hooks/useAsyncValue";
import { useListView } from "../hooks/useListView";
import { formatDate } from "../lib/format";

export function WorkersPage() {
  const { workerKey } = useParams();
  const navigate = useNavigate();
  const { selectedProjectKey } = useProject();
  const registry = useAsyncValue(async () => {
    const [workerCollection, routeWorkerCollection] = await Promise.all([
      api.workers(selectedProjectKey),
      workerKey ? api.workers(selectedProjectKey, { worker_key: workerKey, limit: 1 }) : Promise.resolve(null),
    ]);
    return { workers: workerCollection.workers.map((item) => item.worker), routeWorker: routeWorkerCollection?.workers[0]?.worker };
  }, [selectedProjectKey, workerKey]);
  const [selectedWorker, setSelectedWorker] = useState(workerKey ?? "");
  const [sort, setSort] = useState("recent");
  useEffect(() => {
    if (workerKey) setSelectedWorker(workerKey);
  }, [workerKey]);
  const sortedWorkers = useMemo(() => [...(registry.data?.workers ?? [])].sort((left, right) => {
    if (sort === "key") return left.worker_key.localeCompare(right.worker_key);
    if (sort === "status") return left.status.localeCompare(right.status) || left.worker_key.localeCompare(right.worker_key);
    return right.updated_at.localeCompare(left.updated_at) || right.id - left.id;
  }), [registry.data, sort]);
  const workerList = useListView(sortedWorkers, useCallback((item, query) => `${item.worker_key} ${item.worker_type} ${item.hostname} ${item.status} ${item.capabilities.join(" ")}`.toLowerCase().includes(query), []), 10);
  const worker = selectedWorker
    ? registry.data?.workers.find((item) => item.worker_key === selectedWorker) ?? registry.data?.routeWorker
    : registry.data?.workers[0];
  const onlineWorkers = registry.data?.workers.filter((item) => item.status === "online").length ?? 0;
  const capabilityCount = new Set(registry.data?.workers.flatMap((item) => item.capabilities) ?? []).size;
  const detail = useAsyncValue(
    () => worker ? api.workerDetail(selectedProjectKey, worker.id) : Promise.resolve(null),
    [selectedProjectKey, worker?.id],
  );

  return <div className="page"><PageHeader eyebrow="Execution capacity" title="Workers" description="Worker registration, heartbeat health, capabilities, pool capacity, and scheduling decisions." />
    {registry.loading ? <LoadingState label="Loading workers" /> : registry.error ? <ErrorState message={registry.error} onRetry={registry.retry} /> : registry.data ? <>
      <div className="summary-grid"><Metric label="Workers" value={registry.data.workers.length} /><Metric label="Online" value={onlineWorkers} /><Metric label="Capabilities" value={capabilityCount} /><Metric label="Selected" value={worker?.worker_key || "-"} /></div>
      <div className="resource-layout"><Section title="Worker registry" className="resource-index"><ListControls query={workerList.query} onQueryChange={workerList.setQuery} page={workerList.page} pageCount={workerList.pageCount} total={workerList.total} onPageChange={workerList.setPage} placeholder="Search workers" sortValue={sort} onSortChange={setSort} sortOptions={[{ value: "recent", label: "Recently updated" }, { value: "key", label: "Worker key A-Z" }, { value: "status", label: "Status A-Z" }]} /><div className="resource-list dense">{workerList.items.length ? workerList.items.map((item) => <button key={item.worker_key} className={item.worker_key === worker?.worker_key ? "active" : ""} onClick={() => { setSelectedWorker(item.worker_key); navigate(`/workers/${encodeURIComponent(item.worker_key)}?project=${encodeURIComponent(selectedProjectKey)}`); }}><div><strong>{item.worker_key}</strong><small>{item.worker_type} / {item.hostname}</small></div><StatusBadge value={item.status} /></button>) : <EmptyState message={registry.data.workers.length ? "No workers match this search" : "No workers registered"} />}</div></Section>
        <div className="resource-detail">{worker ? detail.loading ? <LoadingState label="Loading worker detail" /> : detail.error ? <ErrorState message={detail.error} onRetry={detail.retry} /> : detail.data ? <><Section title={detail.data.worker.worker_key} actions={<StatusBadge value={detail.data.worker.status} />}><DefinitionList rows={[["Worker type", detail.data.worker.worker_type], ["Hostname", detail.data.worker.hostname], ["PID", detail.data.worker.pid ?? "-"], ["Capabilities", detail.data.worker.capabilities.join(", ") || "-"], ["Registered", formatDate(detail.data.worker.registered_at)], ["Last heartbeat", formatDate(detail.data.worker.last_heartbeat_at)], ["Lease timeout", `${detail.data.worker.lease_timeout_seconds}s`]]} /></Section><div className="page-grid two-columns"><Section title="Heartbeat history">{detail.data.heartbeats.length ? <div className="table-list">{detail.data.heartbeats.slice(0, 10).map((heartbeat) => <div key={heartbeat.id}><div><strong>{formatDate(heartbeat.observed_at)}</strong><small>Heartbeat #{heartbeat.id}</small></div><StatusBadge value={heartbeat.status} /></div>)}</div> : <EmptyState message="No heartbeat history" />}</Section><Section title="Lease history">{detail.data.leases.length ? <div className="table-list">{detail.data.leases.slice(0, 10).map((lease) => <div key={lease.id}><div><strong>{lease.lease_kind}</strong><small>Run {lease.run_id || "-"} / expires {formatDate(lease.expires_at)}</small></div><StatusBadge value={lease.status} /></div>)}</div> : <EmptyState message="No lease history" />}</Section></div></> : null : <EmptyState message="Select a worker" />}</div>
      </div>
    </> : null}
  </div>;
}
