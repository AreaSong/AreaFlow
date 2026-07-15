import { Link } from "react-router-dom";
import { ArrowRight } from "lucide-react";
import { api } from "../api";
import { ErrorState, LoadingState, Metric, PageHeader, Section, StatusBadge } from "../components/ui";
import { useProject } from "../context/ProjectContext";
import { useAsyncValue } from "../hooks/useAsyncValue";
import { formatDate } from "../lib/format";

export function OverviewPage() {
  const { selectedProject, selectedProjectKey } = useProject();
  const overview = useAsyncValue(async () => {
    if (!selectedProjectKey) throw new Error("No project is available");
    const [summary, readiness, events, runs, workers] = await Promise.all([
      api.projectSummary(selectedProjectKey),
      api.projectReadiness(selectedProjectKey),
      api.projectEvents(selectedProjectKey),
      api.runs(selectedProjectKey),
      api.workers(selectedProjectKey),
    ]);
    return { summary, readiness, events, runs, workers };
  }, [selectedProjectKey]);

  return <div className="page"><PageHeader eyebrow="Control plane" title="Overview" description="Current project health, execution capacity, and operational signals." />
    {overview.loading ? <LoadingState label="Loading overview" /> : overview.error ? <ErrorState message={overview.error} onRetry={overview.retry} /> : overview.data ? <>
      <div className="summary-grid">
        <Metric label="Workflow versions" value={overview.data.summary.inventory.versions} detail={selectedProject?.name} />
        <Metric label="Artifacts" value={overview.data.summary.inventory.artifacts} detail="Indexed metadata" />
        <Metric label="Active runs" value={overview.data.runs.runs.filter((item) => ["queued", "running", "cancelling"].includes(item.run.status)).length} detail={`${overview.data.workers.workers.filter((item) => item.worker.status === "online").length} workers online`} />
        <Metric label="Readiness" value={overview.data.readiness.status} detail={`${overview.data.readiness.items.length} checks`} />
      </div>
      <div className="page-grid two-columns">
        <Section title="Project health" description="Configuration, import, and readiness state.">
          <div className="health-list">{overview.data.readiness.items.length ? overview.data.readiness.items.slice(0, 6).map((item) => <div key={item.key}><div><strong>{item.key}</strong><p>{item.message}</p></div><StatusBadge value={item.status} /></div>) : <p className="inline-empty">No readiness checks</p>}</div>
          <Link className="text-link" to={`/projects/${selectedProjectKey}?project=${selectedProjectKey}`}>Open project <ArrowRight size={15} /></Link>
        </Section>
        <Section title="Recent events" description="Latest domain activity for the selected project.">
          <div className="timeline-list">{overview.data.events.events.length ? overview.data.events.events.slice(0, 7).map((event) => <div key={event.id}><span className={`event-dot ${event.severity}`} /><div><strong>{event.type}</strong><p>{event.message}</p><small>{formatDate(event.created_at)}</small></div></div>) : <p className="inline-empty">No recent events</p>}</div>
          <Link className="text-link" to={`/audit?project=${selectedProjectKey}`}>Open audit <ArrowRight size={15} /></Link>
        </Section>
      </div>
    </> : null}
  </div>;
}
