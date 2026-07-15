import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Eye } from "lucide-react";
import { api } from "../api";
import { DefinitionList, EmptyState, ErrorState, ListControls, LoadingState, Metric, PageHeader, Section, StatusBadge } from "../components/ui";
import { useProject } from "../context/ProjectContext";
import { useAsyncValue } from "../hooks/useAsyncValue";
import { useListView } from "../hooks/useListView";
import { compactHash, formatBytes, formatDate } from "../lib/format";

export function ArtifactsPage() {
  const params = useParams();
  const navigate = useNavigate();
  const { selectedProjectKey } = useProject();
  const artifacts = useAsyncValue(async () => {
    const result = await api.artifacts(selectedProjectKey);
    return result.artifacts.map((item) => item.artifact);
  }, [selectedProjectKey]);
  const [selectedID, setSelectedID] = useState<number | null>(params.artifactId ? Number(params.artifactId) : null);
  const [sort, setSort] = useState("newest");
  const [previewRequested, setPreviewRequested] = useState(false);
  useEffect(() => {
    setSelectedID(params.artifactId ? Number(params.artifactId) : null);
  }, [params.artifactId]);
  const sortedArtifacts = useMemo(() => [...(artifacts.data ?? [])].sort((left, right) => {
    if (sort === "type") return left.artifact_type.localeCompare(right.artifact_type) || right.id - left.id;
    if (sort === "size") return right.size_bytes - left.size_bytes || right.id - left.id;
    return right.created_at.localeCompare(left.created_at) || right.id - left.id;
  }), [artifacts.data, sort]);
  const artifactList = useListView(sortedArtifacts, useCallback((item, query) => `${item.id} ${item.source_path} ${item.uri} ${item.artifact_type} ${item.storage_backend} ${item.content_type} ${item.sha256}`.toLowerCase().includes(query), []), 10);
  const detail = useAsyncValue(
    () => selectedID ? api.artifactDetail(selectedProjectKey, selectedID) : Promise.resolve(null),
    [selectedProjectKey, selectedID],
  );
  const selected = detail.data ?? artifacts.data?.find((item) => item.id === selectedID) ?? artifacts.data?.[0];
  useEffect(() => setPreviewRequested(false), [selected?.id]);
  const previewable = selected?.storage_backend === "local" && selected.size_bytes <= 512 * 1024 && isTextContent(selected.content_type);
  const content = useAsyncValue(
    () => previewRequested && selected ? api.artifactContent(selectedProjectKey, selected.id) : Promise.resolve(null),
    [previewRequested, selectedProjectKey, selected?.id],
  );
  const totalBytes = artifacts.data?.reduce((sum, item) => sum + item.size_bytes, 0) ?? 0;

  return <div className="page"><PageHeader eyebrow="Evidence" title="Artifacts" description="Indexed outputs, source paths, content metadata, and integrity identifiers." />
    {artifacts.loading ? <LoadingState label="Loading artifacts" /> : artifacts.error ? <ErrorState message={artifacts.error} onRetry={artifacts.retry} /> : artifacts.data?.length === 0 ? <EmptyState message="No artifacts indexed" /> : <>
      <div className="summary-grid"><Metric label="Artifacts" value={artifacts.data?.length ?? 0} /><Metric label="Indexed size" value={formatBytes(totalBytes)} /><Metric label="Backends" value={new Set(artifacts.data?.map((item) => item.storage_backend)).size} /><Metric label="Types" value={new Set(artifacts.data?.map((item) => item.artifact_type)).size} /></div>
      <div className="resource-layout"><Section title="Artifact index" className="resource-index"><ListControls query={artifactList.query} onQueryChange={artifactList.setQuery} page={artifactList.page} pageCount={artifactList.pageCount} total={artifactList.total} onPageChange={artifactList.setPage} placeholder="Search artifacts" sortValue={sort} onSortChange={setSort} sortOptions={[{ value: "newest", label: "Newest first" }, { value: "type", label: "Type A-Z" }, { value: "size", label: "Largest first" }]} /><div className="resource-list dense">{artifactList.items.map((item) => <button key={item.id} className={item.id === selected?.id ? "active" : ""} onClick={() => { setSelectedID(item.id); navigate(`/artifacts/${item.id}?project=${encodeURIComponent(selectedProjectKey)}`); }}><div><strong>{item.source_path || item.uri}</strong><small>{item.artifact_type} / {formatBytes(item.size_bytes)}</small></div><StatusBadge value={item.storage_backend} /></button>)}</div>{artifactList.total === 0 ? <EmptyState message="No artifacts match this search" /> : null}</Section>
        <div className="resource-detail">{selectedID && detail.loading ? <LoadingState label="Loading artifact detail" /> : detail.error ? <ErrorState message={detail.error} onRetry={detail.retry} /> : selected ? <><Section title={`Artifact #${selected.id}`} description={selected.uri} actions={previewable ? <button className="secondary-button" type="button" onClick={() => setPreviewRequested(true)}><Eye size={15} />Preview</button> : undefined}><DefinitionList rows={[["Type", selected.artifact_type], ["Backend", selected.storage_backend], ["Content type", selected.content_type], ["Size", formatBytes(selected.size_bytes)], ["SHA-256", compactHash(selected.sha256)], ["Workflow version", selected.workflow_version_id], ["Run", selected.run_id ?? "-"], ["Workflow item", selected.workflow_item_id ?? "-"], ["Created", formatDate(selected.created_at)]]} /></Section>{previewRequested ? content.loading ? <LoadingState label="Loading artifact content" /> : content.error ? <ErrorState message={content.error} onRetry={content.retry} /> : content.data ? <Section title="Content preview" description={`${content.data.contentType} / ${compactHash(content.data.sha256)}`}><pre className="artifact-preview">{formatPreview(content.data.content, content.data.contentType)}</pre></Section> : null : null}</> : <EmptyState message="Select an artifact" />}</div>
      </div>
    </>}
  </div>;
}

function isTextContent(contentType: string) {
  return contentType.startsWith("text/") || /(?:json|yaml|xml|javascript)/i.test(contentType);
}

function formatPreview(content: string, contentType: string) {
  if (!/json/i.test(contentType)) return content;
  try {
    return JSON.stringify(JSON.parse(content), null, 2);
  } catch {
    return content;
  }
}
