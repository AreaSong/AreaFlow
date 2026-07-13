import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { App as CompatibilityWorkspace } from "./App";
import { AppShell } from "./components/AppShell";
import { ProjectProvider } from "./context/ProjectContext";
import { ArtifactsPage } from "./pages/ArtifactsPage";
import { AuditPage } from "./pages/AuditPage";
import { OperationsPage } from "./pages/OperationsPage";
import { OverviewPage } from "./pages/OverviewPage";
import { ProjectsPage } from "./pages/ProjectsPage";
import { RunsPage } from "./pages/RunsPage";
import { WorkersPage } from "./pages/WorkersPage";
import { WorkflowsPage } from "./pages/WorkflowsPage";

export function RouterApp() {
  return (
    <BrowserRouter>
      <ProjectProvider>
        <Routes>
          <Route element={<AppShell />}>
            <Route index element={<OverviewPage />} />
            <Route path="projects" element={<ProjectsPage />} />
            <Route path="projects/:projectKey" element={<ProjectsPage />} />
            <Route path="workflows" element={<WorkflowsPage />} />
            <Route path="projects/:projectKey/workflows/:version" element={<WorkflowsPage />} />
            <Route path="runs" element={<RunsPage />} />
            <Route path="runs/:runId" element={<RunsPage />} />
            <Route path="workers" element={<WorkersPage />} />
            <Route path="workers/:workerKey" element={<WorkersPage />} />
            <Route path="artifacts" element={<ArtifactsPage />} />
            <Route path="artifacts/:artifactId" element={<ArtifactsPage />} />
            <Route path="audit" element={<AuditPage />} />
            <Route path="operations" element={<OperationsPage />} />
          </Route>
          <Route path="operations/compatibility" element={<CompatibilityWorkspace />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </ProjectProvider>
    </BrowserRouter>
  );
}
