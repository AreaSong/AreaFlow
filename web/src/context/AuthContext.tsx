import { createContext, useContext, useEffect, useMemo, useState } from "react";
import type { FormEvent, ReactNode } from "react";
import { api, authSession } from "../api";
import type { AuthPrincipal, AuthStatus } from "../types";
import { LoginPage } from "../pages/LoginPage";

type AuthContextValue = {
  status: AuthStatus;
  principal: AuthPrincipal;
  allowsCapability: (capability: string) => boolean;
  signOut: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthGate({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<AuthStatus | null>(null);
  const [principal, setPrincipal] = useState<AuthPrincipal | null>(null);
  const [token, setToken] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  async function loadPrincipal(nextStatus: AuthStatus) {
    if (nextStatus.requires_token && !authSession.hasToken()) {
      setPrincipal(null);
      setLoading(false);
      return;
    }
    try {
      setPrincipal(await api.authMe());
      setError("");
    } catch (nextError) {
      setPrincipal(null);
      setError(message(nextError));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    let active = true;
    api.authStatus()
      .then(async (nextStatus) => {
        if (!active) return;
        setStatus(nextStatus);
        await loadPrincipal(nextStatus);
      })
      .catch((nextError) => {
        if (active) {
          setError(message(nextError));
          setLoading(false);
        }
      });
    const invalidate = () => {
      setPrincipal(null);
      setError("会话已失效，请重新登录。");
    };
    window.addEventListener(authSession.eventName, invalidate);
    return () => {
      active = false;
      window.removeEventListener(authSession.eventName, invalidate);
    };
  }, []);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!status || !token.trim()) return;
    setLoading(true);
    authSession.setToken(token);
    setToken("");
    await loadPrincipal(status);
  }

  const value = useMemo<AuthContextValue | null>(() => status && principal ? {
    status,
    principal,
    allowsCapability: (capability) => principal.capabilities.includes("*") || principal.capabilities.includes(capability),
    signOut: async () => {
      if (status.mode === "oidc") await api.logout();
      authSession.clearToken();
      setPrincipal(null);
      setError("");
    },
  } : null, [status, principal]);

  if (loading) return <div className="auth-screen"><div className="auth-panel"><strong>正在验证 AreaFlow 访问凭据</strong></div></div>;
  if (!status) return <div className="auth-screen"><div className="auth-panel"><strong>无法读取认证状态</strong><p>{error}</p></div></div>;
  if (!value) return <LoginPage status={status} token={token} error={error} onTokenChange={setToken} onTokenSubmit={submit} />;
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthGate");
  return value;
}

function message(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}
