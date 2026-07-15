import { createContext, useContext, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { useSearchParams } from "react-router-dom";
import { api } from "../api";
import type { ProjectRecord } from "../types";

type ProjectContextValue = {
  projects: ProjectRecord[];
  selectedProject: ProjectRecord | null;
  selectedProjectKey: string;
  setSelectedProjectKey: (key: string) => void;
  loading: boolean;
  error: string;
  retryProjects: () => void;
};

const ProjectContext = createContext<ProjectContextValue | null>(null);

export function ProjectProvider({ children }: { children: ReactNode }) {
  const [searchParams, setSearchParams] = useSearchParams();
  const [projects, setProjects] = useState<ProjectRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [attempt, setAttempt] = useState(0);
  const requestedKey = searchParams.get("project") ?? "";

  useEffect(() => {
    let active = true;
    setLoading(true);
    api.projects()
      .then((result) => {
        if (!active) return;
        setProjects(result.projects);
        setError("");
      })
      .catch((nextError: unknown) => {
        if (active) setError(errorMessage(nextError));
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [attempt]);

  useEffect(() => {
    if (loading || !projects[0] || projects.some((project) => project.key === requestedKey)) return;
    const next = new URLSearchParams(searchParams);
    next.set("project", projects[0].key);
    setSearchParams(next, { replace: true });
  }, [loading, projects, requestedKey, searchParams, setSearchParams]);

  const selectedProject = projects.find((project) => project.key === requestedKey) ?? projects[0] ?? null;
  const selectedProjectKey = selectedProject?.key ?? requestedKey;

  const value = useMemo<ProjectContextValue>(
    () => ({
      projects,
      selectedProject,
      selectedProjectKey,
      setSelectedProjectKey: (key) => {
        const next = new URLSearchParams(searchParams);
        next.set("project", key);
        setSearchParams(next);
      },
      loading,
      error,
      retryProjects: () => setAttempt((current) => current + 1),
    }),
    [projects, selectedProject, selectedProjectKey, searchParams, setSearchParams, loading, error],
  );

  return <ProjectContext.Provider value={value}>{children}</ProjectContext.Provider>;
}

export function useProject() {
  const value = useContext(ProjectContext);
  if (!value) throw new Error("useProject must be used inside ProjectProvider");
  return value;
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}
