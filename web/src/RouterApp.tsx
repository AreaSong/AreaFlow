import { BrowserRouter, Route, Routes } from "react-router-dom";
import { App as CompatibilityWorkspace } from "./App";
import { AppShell } from "./components/AppShell";
import { ProjectRoute } from "./components/ProjectRoute";
import { ProjectProvider } from "./context/ProjectContext";
import { AuthGate } from "./context/AuthContext";
import { ArtifactsPage } from "./pages/ArtifactsPage";
import { AccessPage } from "./pages/AccessPage";
import { AuditPage } from "./pages/AuditPage";
import { OperationsPage } from "./pages/OperationsPage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { OverviewPage } from "./pages/OverviewPage";
import { ProjectsPage } from "./pages/ProjectsPage";
import { RunsPage } from "./pages/RunsPage";
import { WorkersPage } from "./pages/WorkersPage";
import { WorkflowsPage } from "./pages/WorkflowsPage";

export function RouterApp() {
  return (
    <BrowserRouter>
      <AuthGate>
        <ProjectProvider>
          <Routes>
          <Route element={<AppShell />}>
            <Route index element={<ProjectRoute><OverviewPage /></ProjectRoute>} />
            <Route path="projects" element={<ProjectsPage />} />
            <Route path="projects/:projectKey" element={<ProjectsPage />} />
            <Route path="workflows" element={<ProjectRoute><WorkflowsPage /></ProjectRoute>} />
            <Route path="projects/:projectKey/workflows/:version" element={<ProjectRoute><WorkflowsPage /></ProjectRoute>} />
            <Route path="runs" element={<ProjectRoute><RunsPage /></ProjectRoute>} />
            <Route path="runs/:runId" element={<ProjectRoute><RunsPage /></ProjectRoute>} />
            <Route path="workers" element={<ProjectRoute><WorkersPage /></ProjectRoute>} />
            <Route path="workers/:workerKey" element={<ProjectRoute><WorkersPage /></ProjectRoute>} />
            <Route path="artifacts" element={<ProjectRoute><ArtifactsPage /></ProjectRoute>} />
            <Route path="artifacts/:artifactId" element={<ProjectRoute><ArtifactsPage /></ProjectRoute>} />
            <Route path="audit" element={<ProjectRoute><AuditPage /></ProjectRoute>} />
            <Route path="access" element={<ProjectRoute><AccessPage /></ProjectRoute>} />
            <Route path="operations" element={<ProjectRoute><OperationsPage /></ProjectRoute>} />
            <Route path="*" element={<NotFoundPage />} />
          </Route>
          <Route path="operations/compatibility" element={<CompatibilityWorkspace />} />
          </Routes>
        </ProjectProvider>
      </AuthGate>
    </BrowserRouter>
  );
}
