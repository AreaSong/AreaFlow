import { KeyRound, LogIn } from "lucide-react";
import type { FormEvent } from "react";
import type { AuthStatus } from "../types";

type LoginPageProps = {
  status: AuthStatus;
  token: string;
  error: string;
  onTokenChange: (value: string) => void;
  onTokenSubmit: (event: FormEvent) => void;
};

export function LoginPage({ status, token, error, onTokenChange, onTokenSubmit }: LoginPageProps) {
  if (status.mode === "oidc") {
    const returnTo = `${window.location.pathname}${window.location.search}`;
    const loginURL = `${status.login_url}?return_to=${encodeURIComponent(returnTo)}`;
    return (
      <div className="auth-screen">
        <div className="auth-panel">
          <KeyRound size={24} />
          <div><strong>登录 AreaFlow</strong><p>使用组织身份进入受项目权限控制的生产控制面。</p></div>
          {error ? <p className="form-error">{error}</p> : null}
          <a className="primary-button" href={loginURL}><LogIn size={16} />使用组织身份登录</a>
        </div>
      </div>
    );
  }

  return (
    <div className="auth-screen">
      <form className="auth-panel" onSubmit={onTokenSubmit}>
        <KeyRound size={24} />
        <div><strong>连接 AreaFlow</strong><p>输入由 AreaFlow CLI 签发的短期 service token。</p></div>
        <label><span>API token</span><input type="password" value={token} onChange={(event) => onTokenChange(event.target.value)} autoComplete="off" autoFocus /></label>
        {error ? <p className="form-error">{error}</p> : null}
        <button className="primary-button" type="submit" disabled={!token.trim()}><LogIn size={16} />登录</button>
      </form>
    </div>
  );
}
