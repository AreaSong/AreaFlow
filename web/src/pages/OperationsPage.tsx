import { ExternalLink } from "lucide-react";
import { Link } from "react-router-dom";
import { api } from "../api";
import { ErrorState, LoadingState, Metric, PageHeader, Section, StatusBadge } from "../components/ui";
import { useProject } from "../context/ProjectContext";
import { useAsyncValue } from "../hooks/useAsyncValue";

export function OperationsPage() {
  const { selectedProjectKey } = useProject();
  const data = useAsyncValue(async () => {
    const [operations, release, completion, webGate] = await Promise.all([
      api.operationsReadiness(), api.releaseFinalGate(selectedProjectKey), api.completionAuditSnapshotReadiness(selectedProjectKey), api.webWriteActionGate(),
    ]);
    return { operations, release, completion, webGate };
  }, [selectedProjectKey]);

  return <div className="page"><PageHeader eyebrow="Platform administration" title="Operations" description="Service readiness, migrations, support metadata, release controls, and guarded actions." actions={<Link className="secondary-button" to={`/operations/compatibility?project=${selectedProjectKey}`}>Compatibility workspace <ExternalLink size={15} /></Link>} />
    {data.loading ? <LoadingState label="Loading operational state" /> : data.error ? <ErrorState message={data.error} onRetry={data.retry} /> : data.data ? <>
      <div className="summary-grid"><Metric label="Operations" value={data.data.operations.status} /><Metric label="Service" value={data.data.operations.service_status.status} /><Metric label="Release gate" value={data.data.release.status} /><Metric label="Web actions" value={data.data.webGate.status} /></div>
      <div className="page-grid two-columns"><Section title="Operational readiness"><div className="health-list">{data.data.operations.items.map((item) => <div key={item.key}><div><strong>{item.key}</strong><p>{item.message}</p></div><StatusBadge value={item.status} /></div>)}</div></Section><Section title="Migration ledger"><div className="health-list">{data.data.operations.migration_ledger.entries.map((entry) => <div key={entry.name}><div><strong>{entry.name}</strong><p>{entry.applied ? "Applied" : "Pending evidence"}</p></div><StatusBadge value={entry.status} /></div>)}</div></Section></div>
      <div className="page-grid two-columns"><Section title="Release readiness"><div className="health-list">{data.data.release.items.slice(0, 10).map((item) => <div key={item.key}><div><strong>{item.key}</strong><p>{item.message}</p></div><StatusBadge value={item.status} /></div>)}</div></Section><Section title="Web command boundary" description={data.data.webGate.mode}><div className="health-list">{data.data.webGate.actions.map((action) => <div key={action.key}><div><strong>{action.label}</strong><p>{action.blockers.join(", ") || action.command_api}</p></div><StatusBadge value={action.status} /></div>)}</div></Section></div>
    </> : null}
  </div>;
}
