import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { ArrowRight, FolderKanban } from "lucide-react";
import { api } from "../api";
import { DefinitionList, EmptyState, ErrorState, ListControls, LoadingState, Metric, PageHeader, Section, StatusBadge } from "../components/ui";
import { useProject } from "../context/ProjectContext";
import { useAsyncValue } from "../hooks/useAsyncValue";
import { useListView } from "../hooks/useListView";
import { compactHash, formatDate } from "../lib/format";

export function ProjectsPage() {
  const { projectKey } = useParams();
  const { projects, selectedProjectKey, setSelectedProjectKey } = useProject();
  const activeKey = projectKey || selectedProjectKey;
  useEffect(() => {
    if (projectKey && projectKey !== selectedProjectKey && projects.some((project) => project.key === projectKey)) {
      setSelectedProjectKey(projectKey);
    }
  }, [projectKey, projects, selectedProjectKey, setSelectedProjectKey]);
  const [sort, setSort] = useState("name");
  const sortedProjects = useMemo(() => [...projects].sort((left, right) => sort === "key" ? left.key.localeCompare(right.key) : (left.name || left.key).localeCompare(right.name || right.key)), [projects, sort]);
  const projectList = useListView(sortedProjects, useCallback((project, query) => `${project.key} ${project.name} ${project.adapter} ${project.workflow_profile}`.toLowerCase().includes(query), []), 8);
  const detail = useAsyncValue(async () => {
    if (!activeKey) return null;
    const [summary, readiness] = await Promise.all([api.projectSummary(activeKey), api.projectReadiness(activeKey)]);
    return { summary, readiness };
  }, [activeKey]);

  return <div className="page"><PageHeader eyebrow="Managed resources" title="Projects" description="Project connections, configuration identity, readiness, and managed boundaries." />
    <div className="resource-layout">
      <Section title="Project registry" description={`${projects.length} managed projects`} className="resource-index">
        <ListControls query={projectList.query} onQueryChange={projectList.setQuery} page={projectList.page} pageCount={projectList.pageCount} total={projectList.total} onPageChange={projectList.setPage} placeholder="Search projects" sortValue={sort} onSortChange={setSort} sortOptions={[{ value: "name", label: "Name A-Z" }, { value: "key", label: "Project key A-Z" }]} />
        <div className="resource-list">{projectList.items.map((project) => <Link key={project.key} className={project.key === activeKey ? "active" : ""} to={`/projects/${project.key}?project=${project.key}`} onClick={() => setSelectedProjectKey(project.key)}><FolderKanban size={18} /><div><strong>{project.name || project.key}</strong><small>{project.adapter} / {project.workflow_profile}</small></div><ArrowRight size={16} /></Link>)}</div>
        {projectList.total === 0 ? <EmptyState message={projects.length ? "No projects match this search" : "No projects are registered"} /> : null}
      </Section>
      <div className="resource-detail">{activeKey && detail.loading ? <LoadingState label="Loading project" /> : detail.error ? <ErrorState message={detail.error} onRetry={detail.retry} /> : detail.data ? <>
        <div className="summary-grid compact">
          <Metric label="Versions" value={detail.data.summary.inventory.versions} />
          <Metric label="Artifacts" value={detail.data.summary.inventory.artifacts} />
          <Metric label="Residuals" value={detail.data.summary.inventory.residuals} />
          <Metric label="Readiness" value={detail.data.readiness.status} />
        </div>
        <Section title={detail.data.summary.project.name || detail.data.summary.project.key} description={detail.data.summary.project.root} actions={<StatusBadge value={detail.data.readiness.status} />}>
          <DefinitionList rows={[
            ["Project key", detail.data.summary.project.key], ["Kind", detail.data.summary.project.kind], ["Adapter", detail.data.summary.project.adapter], ["Workflow profile", detail.data.summary.project.workflow_profile], ["Default branch", detail.data.summary.project.default_branch], ["Config hash", compactHash(detail.data.summary.config?.config_hash ?? "")], ["Config loaded", formatDate(detail.data.summary.config?.loaded_at)],
          ]} />
        </Section>
        <Section title="Readiness checks"><div className="health-list">{detail.data.readiness.items.map((item) => <div key={item.key}><div><strong>{item.key}</strong><p>{item.message}</p></div><StatusBadge value={item.status} /></div>)}</div></Section>
      </> : <EmptyState message="Select a project" />}</div>
    </div>
  </div>;
}
