export function formatDate(value?: string) {
  if (!value) return "-";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

export function formatBytes(value: number) {
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / (1024 * 1024)).toFixed(1)} MB`;
}

export function compactHash(value: string) {
  return value ? `${value.slice(0, 10)}...${value.slice(-6)}` : "-";
}

export function statusTone(value?: string) {
  const normalized = value?.toLowerCase() ?? "";
  if (["ready", "pass", "passed", "ok", "online", "complete", "approved"].includes(normalized)) return "success";
  if (["blocked", "failed", "error", "offline", "rejected", "denied"].includes(normalized)) return "danger";
  if (["warning", "pending", "preview", "waiting", "degraded"].includes(normalized)) return "warning";
  return "neutral";
}
