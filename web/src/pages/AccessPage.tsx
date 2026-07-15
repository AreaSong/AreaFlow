import { useEffect, useState } from "react";
import { ShieldCheck, Trash2 } from "lucide-react";
import { api } from "../api";
import { EmptyState, ErrorState, LoadingState, PageHeader, Section, StatusBadge } from "../components/ui";
import { useAuth } from "../context/AuthContext";
import { useProject } from "../context/ProjectContext";
import type { RoleBinding } from "../types";

const roles = ["project_admin", "operator", "approver", "auditor", "viewer"];

export function AccessPage() {
  const { selectedProjectKey } = useProject();
  const { allowsCapability } = useAuth();
  const [bindings, setBindings] = useState<RoleBinding[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [userID, setUserID] = useState("");
  const [role, setRole] = useState("viewer");
  const [reason, setReason] = useState("");

  async function load() {
    if (!selectedProjectKey || !allowsCapability("auth.role.manage")) return;
    setLoading(true);
    try {
      const response = await api.roleBindings(selectedProjectKey);
      setBindings(response.role_bindings);
      setError("");
    } catch (nextError) {
      setError(message(nextError));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { void load(); }, [selectedProjectKey]);

  if (!allowsCapability("auth.role.manage")) {
    return <div className="page"><PageHeader eyebrow="Security" title="Access" description="Project role bindings and authorization evidence." /><EmptyState message="You do not have permission to manage project access" /></div>;
  }

  async function grant(event: React.FormEvent) {
    event.preventDefault();
    const parsedUserID = Number(userID);
    if (!Number.isInteger(parsedUserID) || parsedUserID <= 0 || !reason.trim()) return;
    try {
      await api.grantRole(selectedProjectKey, { user_id: parsedUserID, team_id: 0, role, reason: reason.trim() });
      setUserID("");
      setReason("");
      await load();
    } catch (nextError) {
      setError(message(nextError));
    }
  }

  async function revoke(binding: RoleBinding) {
    const revokeReason = window.prompt("Reason for revoking this role")?.trim();
    if (!revokeReason) return;
    try {
      await api.revokeRole(selectedProjectKey, binding.id, revokeReason);
      await load();
    } catch (nextError) {
      setError(message(nextError));
    }
  }

  return <div className="page">
    <PageHeader eyebrow="Security" title="Access" description="Project-scoped role bindings. Every change requires an authenticated actor, reason, capability and audit event." />
    <Section title="Grant role" description={selectedProjectKey}>
      <form className="access-form" onSubmit={grant}>
        <label><span>User ID</span><input inputMode="numeric" value={userID} onChange={(event) => setUserID(event.target.value)} /></label>
        <label><span>Role</span><select value={role} onChange={(event) => setRole(event.target.value)}>{roles.map((item) => <option key={item}>{item}</option>)}</select></label>
        <label><span>Reason</span><input value={reason} onChange={(event) => setReason(event.target.value)} /></label>
        <button className="primary-button" type="submit" disabled={!userID || !reason.trim()}><ShieldCheck size={16} />Grant</button>
      </form>
    </Section>
    {loading ? <LoadingState label="Loading role bindings" /> : error ? <ErrorState message={error} onRetry={() => void load()} /> :
      <Section title="Active bindings" description={`${bindings.length} records`}>
        <div className="health-list">{bindings.map((binding) => <div key={binding.id}><div><strong>{binding.role}</strong><p>User {binding.user_id || `team ${binding.team_id}`} · {binding.reason}</p></div><div className="inline-actions"><StatusBadge value={binding.status} /><button type="button" className="icon-button" title="Revoke role" aria-label="Revoke role" onClick={() => void revoke(binding)}><Trash2 size={15} /></button></div></div>)}</div>
        {bindings.length === 0 ? <EmptyState message="No project role bindings" /> : null}
      </Section>}
  </div>;
}

function message(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}
