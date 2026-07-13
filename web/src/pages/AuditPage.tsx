import { useCallback, useMemo, useState } from "react";
import { api } from "../api";
import { EmptyState, ErrorState, ListControls, LoadingState, PageHeader, Section, StatusBadge } from "../components/ui";
import { useProject } from "../context/ProjectContext";
import { useAsyncValue } from "../hooks/useAsyncValue";
import { useListView } from "../hooks/useListView";
import { useProjectEventStream } from "../hooks/useProjectEventStream";
import { formatDate } from "../lib/format";

export function AuditPage() {
  const { selectedProjectKey } = useProject();
  const [view, setView] = useState<"audit" | "events">("audit");
  const [sort, setSort] = useState("newest");
  const [filters, setFilters] = useState({ actor: "", action: "", resource: "", decision: "", from: "", to: "" });
  const data = useAsyncValue(async () => {
    const auditFilters = {
      actor_id: filters.actor,
      action: filters.action,
      resource: filters.resource,
      decision: filters.decision,
      from: toRFC3339(filters.from),
      to: toRFC3339(filters.to),
    };
    const [audit, events] = await Promise.all([api.projectAuditEvents(selectedProjectKey, auditFilters), api.projectEvents(selectedProjectKey)]);
    return { audit, events };
  }, [selectedProjectKey, filters.actor, filters.action, filters.resource, filters.decision, filters.from, filters.to]);
  const initialEvents = useMemo(() => data.data?.events.events ?? [], [data.data]);
  const stream = useProjectEventStream(selectedProjectKey, initialEvents);
  const sortedAuditEvents = useMemo(() => [...(data.data?.audit.audit_events ?? [])].sort((left, right) => sort === "action" ? left.action.localeCompare(right.action) || right.id - left.id : sort === "decision" ? left.decision.localeCompare(right.decision) || right.id - left.id : right.created_at.localeCompare(left.created_at) || right.id - left.id), [data.data, sort]);
  const sortedDomainEvents = useMemo(() => [...stream.events].sort((left, right) => sort === "action" ? left.type.localeCompare(right.type) || right.id - left.id : sort === "decision" ? left.severity.localeCompare(right.severity) || right.id - left.id : right.created_at.localeCompare(left.created_at) || right.id - left.id), [stream.events, sort]);
  const auditList = useListView(sortedAuditEvents, useCallback((event, query) => `${event.action} ${event.decision} ${event.reason} ${event.resource} ${event.resource_type} ${event.capability}`.toLowerCase().includes(query), []), 15);
  const eventList = useListView(sortedDomainEvents, useCallback((event, query) => `${event.type} ${event.message} ${event.severity}`.toLowerCase().includes(query), []), 15);
  const list = view === "audit" ? auditList : eventList;

  return <div className="page"><PageHeader eyebrow="Accountability" title="Audit" description="Security decisions and domain events, separated by intent and lifecycle." actions={<div className="audit-actions"><StatusBadge value={`SSE ${stream.status}`} /><div className="segmented-control"><button className={view === "audit" ? "active" : ""} onClick={() => setView("audit")}>Audit events</button><button className={view === "events" ? "active" : ""} onClick={() => setView("events")}>Domain events</button></div></div>} />
    {data.loading ? <LoadingState label="Loading audit timeline" /> : data.error ? <ErrorState message={data.error} onRetry={data.retry} /> : data.data ? <Section title={view === "audit" ? "Security decisions" : "Domain activity"}>{view === "audit" ? <div className="filter-grid" aria-label="Audit filters"><label><span>Actor ID</span><input type="number" min="1" value={filters.actor} onChange={(event) => setFilters((current) => ({ ...current, actor: event.target.value }))} /></label><label><span>Action</span><input value={filters.action} onChange={(event) => setFilters((current) => ({ ...current, action: event.target.value }))} /></label><label><span>Resource</span><input value={filters.resource} onChange={(event) => setFilters((current) => ({ ...current, resource: event.target.value }))} /></label><label><span>Decision</span><select value={filters.decision} onChange={(event) => setFilters((current) => ({ ...current, decision: event.target.value }))}><option value="">All</option><option value="allowed">Allowed</option><option value="denied">Denied</option><option value="blocked">Blocked</option></select></label><label><span>From</span><input type="datetime-local" value={filters.from} onChange={(event) => setFilters((current) => ({ ...current, from: event.target.value }))} /></label><label><span>To</span><input type="datetime-local" value={filters.to} onChange={(event) => setFilters((current) => ({ ...current, to: event.target.value }))} /></label></div> : null}<ListControls query={list.query} onQueryChange={list.setQuery} page={list.page} pageCount={list.pageCount} total={list.total} onPageChange={list.setPage} placeholder={view === "audit" ? "Search audit events" : "Search domain events"} sortValue={sort} onSortChange={setSort} sortOptions={[{ value: "newest", label: "Newest first" }, { value: "action", label: view === "audit" ? "Action A-Z" : "Event type A-Z" }, { value: "decision", label: view === "audit" ? "Decision A-Z" : "Severity A-Z" }]} />{view === "audit" ? <div className="audit-table">{auditList.items.length ? auditList.items.map((event) => <article key={event.id}><div><strong>{event.action}</strong><p>{event.reason || event.resource || "No reason recorded"}</p><small>{event.resource_type || "resource"} / {formatDate(event.created_at)}</small></div><div><StatusBadge value={event.decision} />{event.capability ? <code>{event.capability}</code> : null}</div></article>) : <EmptyState message="No audit events match these filters" />}</div> : <div className="timeline-list large">{eventList.items.length ? eventList.items.map((event) => <div key={event.id}><span className={`event-dot ${event.severity}`} /><div><strong>{event.type}</strong><p>{event.message}</p><small>{formatDate(event.created_at)}</small></div><StatusBadge value={event.severity} /></div>) : <EmptyState message="No domain events match this search" />}</div>}</Section> : null}
  </div>;
}

function toRFC3339(value: string) {
  return value ? new Date(value).toISOString() : undefined;
}
