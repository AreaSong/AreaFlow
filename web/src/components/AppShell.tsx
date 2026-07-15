import {
  Activity,
  Boxes,
  FileArchive,
  FolderKanban,
  Gauge,
  History,
  Network,
  ServerCog,
  Workflow,
  RefreshCw,
  LogOut,
  ShieldCheck,
} from "lucide-react";
import { NavLink, Outlet } from "react-router-dom";
import { useProject } from "../context/ProjectContext";
import { useAuth } from "../context/AuthContext";

const navigation = [
  { to: "/", label: "Overview", icon: Gauge, end: true },
  { to: "/projects", label: "Projects", icon: FolderKanban },
  { to: "/workflows", label: "Workflows", icon: Workflow },
  { to: "/runs", label: "Runs", icon: Activity },
  { to: "/workers", label: "Workers", icon: Network },
  { to: "/artifacts", label: "Artifacts", icon: FileArchive },
  { to: "/audit", label: "Audit", icon: History },
  { to: "/access", label: "Access", icon: ShieldCheck, capability: "auth.role.manage" },
  { to: "/operations", label: "Operations", icon: ServerCog },
];

export function AppShell() {
  const { projects, selectedProjectKey, setSelectedProjectKey, loading, error, retryProjects } = useProject();
  const { status, principal, signOut, allowsCapability } = useAuth();
  const visibleNavigation = navigation.filter((item) =>
    (item.to !== "/operations" || principal.projects.includes("*")) && (!item.capability || allowsCapability(item.capability)),
  );

  return (
    <div className="app-shell">
      <aside className="app-sidebar">
        <div className="app-brand">
          <span className="app-brand-mark"><Boxes size={20} /></span>
          <div><strong>AreaFlow</strong><small>Development control plane</small></div>
        </div>

        <label className="project-switcher">
          <span>Project context</span>
          <select
            aria-label="Project context"
            value={selectedProjectKey}
            onChange={(event) => setSelectedProjectKey(event.target.value)}
            disabled={loading || projects.length === 0}
          >
            {projects.map((project) => <option key={project.key} value={project.key}>{project.name || project.key}</option>)}
          </select>
        </label>
        {error ? <div className="sidebar-error"><span>{error}</span><button type="button" onClick={retryProjects} aria-label="Retry project list"><RefreshCw size={14} /></button></div> : null}

        <nav className="primary-nav" aria-label="Primary navigation">
          {visibleNavigation.map(({ to, label, icon: Icon, end }) => (
            <NavLink key={to} to={withProject(to, selectedProjectKey)} end={end} aria-label={label}>
              <Icon size={18} />
              <span>{label}</span>
            </NavLink>
          ))}
        </nav>
        <div className="auth-summary"><div><strong>{principal.actor}</strong><small>{status.mode === "token" ? principal.token_key : status.mode === "oidc" ? principal.roles.join(", ") || "no role" : "local mode"}</small></div>{status.mode !== "disabled" ? <button type="button" onClick={() => void signOut()} aria-label="退出登录" title="退出登录"><LogOut size={16} /></button> : null}</div>
      </aside>
      <main className="app-main"><Outlet /></main>
    </div>
  );
}

function withProject(path: string, projectKey: string) {
  return projectKey ? `${path}?project=${encodeURIComponent(projectKey)}` : path;
}
