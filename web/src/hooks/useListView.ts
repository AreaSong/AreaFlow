import { useEffect, useMemo, useState } from "react";

export function useListView<T>(items: T[], matcher: (item: T, normalizedQuery: string) => boolean, pageSize = 10) {
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const normalizedQuery = query.trim().toLowerCase();
  const filtered = useMemo(
    () => normalizedQuery ? items.filter((item) => matcher(item, normalizedQuery)) : items,
    [items, matcher, normalizedQuery],
  );
  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));

  useEffect(() => setPage(1), [normalizedQuery]);
  useEffect(() => setPage((current) => Math.min(current, pageCount)), [pageCount]);
  const visibleItems = useMemo(
    () => filtered.slice((page - 1) * pageSize, page * pageSize),
    [filtered, page, pageSize],
  );

  return {
    query,
    setQuery,
    page,
    setPage,
    pageCount,
    total: filtered.length,
    items: visibleItems,
  };
}
