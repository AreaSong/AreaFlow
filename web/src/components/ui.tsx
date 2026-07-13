import type { ReactNode } from "react";
import { AlertTriangle, Inbox, RefreshCw, Search } from "lucide-react";
import { statusTone } from "../lib/format";

export function PageHeader({ eyebrow, title, description, actions }: { eyebrow: string; title: string; description: string; actions?: ReactNode }) {
  return <header className="page-header"><div><span>{eyebrow}</span><h1>{title}</h1><p>{description}</p></div>{actions ? <div className="page-actions">{actions}</div> : null}</header>;
}

export function Section({ title, description, actions, children, className = "" }: { title: string; description?: string; actions?: ReactNode; children: ReactNode; className?: string }) {
  return <section className={`content-section ${className}`.trim()}><div className="section-header"><div><h2>{title}</h2>{description ? <p>{description}</p> : null}</div>{actions}</div>{children}</section>;
}

export function StatusBadge({ value }: { value?: string }) {
  return <span className={`status-badge ${statusTone(value)}`}>{value || "unknown"}</span>;
}

export function Metric({ label, value, detail }: { label: string; value: string | number; detail?: string }) {
  return <div className="summary-metric"><span>{label}</span><strong>{value}</strong>{detail ? <small>{detail}</small> : null}</div>;
}

export function LoadingState({ label = "Loading" }: { label?: string }) {
  return <div className="empty-state"><span className="loading-indicator" /> <p>{label}</p></div>;
}

export function ErrorState({ message, onRetry }: { message: string; onRetry?: () => void }) {
  return <div className="message-state error"><AlertTriangle size={18} /><p>{message}</p>{onRetry ? <button className="retry-button" type="button" onClick={onRetry}><RefreshCw size={15} />Retry</button> : null}</div>;
}

export function EmptyState({ message }: { message: string }) {
  return <div className="empty-state"><Inbox size={20} /><p>{message}</p></div>;
}

export function DefinitionList({ rows }: { rows: Array<[string, ReactNode]> }) {
  return <dl className="definition-list">{rows.map(([label, value]) => <div key={label}><dt>{label}</dt><dd>{value}</dd></div>)}</dl>;
}

type SortOption = { value: string; label: string };

export function ListControls({ query, onQueryChange, page, pageCount, total, onPageChange, placeholder = "Search", sortValue, onSortChange, sortOptions }: { query: string; onQueryChange: (value: string) => void; page: number; pageCount: number; total: number; onPageChange: (page: number) => void; placeholder?: string; sortValue?: string; onSortChange?: (value: string) => void; sortOptions?: SortOption[] }) {
  return <div className="list-controls"><label><Search size={15} /><input type="search" value={query} onChange={(event) => onQueryChange(event.target.value)} placeholder={placeholder} /></label>{sortOptions?.length && onSortChange ? <select aria-label="Sort results" value={sortValue} onChange={(event) => onSortChange(event.target.value)}>{sortOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select> : null}<div className="pagination"><span>{total} results</span><button type="button" aria-label="Previous page" disabled={page <= 1} onClick={() => onPageChange(page - 1)}>Previous</button><strong>{page} / {pageCount}</strong><button type="button" aria-label="Next page" disabled={page >= pageCount} onClick={() => onPageChange(page + 1)}>Next</button></div></div>;
}
