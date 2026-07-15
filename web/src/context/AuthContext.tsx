import { createContext, useContext, useEffect, useMemo, useState } from "react";
import type { FormEvent, ReactNode } from "react";
import { KeyRound, LogIn } from "lucide-react";
import { api, authSession } from "../api";
import type { AuthPrincipal, AuthStatus } from "../types";

type AuthContextValue = {
  status: AuthStatus;
  principal: AuthPrincipal;
  allowsCapability: (capability: string) => boolean;
  signOut: () => void;
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
      setError("Token 已失效，请重新输入。");
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
    signOut: () => {
      authSession.clearToken();
      setPrincipal(null);
      setError("");
    },
  } : null, [status, principal]);

  if (loading) return <div className="auth-screen"><div className="auth-panel"><KeyRound size={24} /><strong>正在验证 AreaFlow 访问凭据</strong></div></div>;
  if (!status) return <div className="auth-screen"><div className="auth-panel"><strong>无法读取认证状态</strong><p>{error}</p></div></div>;
  if (!value) return <div className="auth-screen"><form className="auth-panel" onSubmit={submit}><KeyRound size={24} /><div><strong>连接 AreaFlow</strong><p>输入由 AreaFlow CLI 签发的本机会话 token。</p></div><label><span>API token</span><input type="password" value={token} onChange={(event) => setToken(event.target.value)} autoComplete="off" autoFocus /></label>{error ? <p className="form-error">{error}</p> : null}<button className="primary-button" type="submit" disabled={!token.trim()}><LogIn size={16} />登录</button></form></div>;
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
