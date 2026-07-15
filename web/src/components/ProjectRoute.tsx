import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { useProject } from "../context/ProjectContext";
import { EmptyState, ErrorState, LoadingState } from "./ui";

export function ProjectRoute({ children }: { children: ReactNode }) {
  const { selectedProjectKey, loading, error, retryProjects } = useProject();

  if (loading) return <div className="page"><LoadingState label="Loading project context" /></div>;
  if (error) return <div className="page"><ErrorState message={error} onRetry={retryProjects} /></div>;
  if (!selectedProjectKey) {
    return <div className="page"><EmptyState message="No projects are registered" action={<Link className="secondary-button" to="/projects">Open Projects</Link>} /></div>;
  }

  return children;
}
