import { useCallback, useEffect, useState } from "react";

export type AsyncValue<T> = {
  data: T | null;
  loading: boolean;
  error: string;
  retry: () => void;
};

export function useAsyncValue<T>(loader: () => Promise<T>, dependencies: readonly unknown[]): AsyncValue<T> {
  const [value, setValue] = useState<Omit<AsyncValue<T>, "retry">>({ data: null, loading: true, error: "" });
  const [attempt, setAttempt] = useState(0);
  const retry = useCallback(() => setAttempt((current) => current + 1), []);

  useEffect(() => {
    let active = true;
    setValue((current) => ({ ...current, loading: true, error: "" }));
    loader()
      .then((data) => {
        if (active) setValue({ data, loading: false, error: "" });
      })
      .catch((error: unknown) => {
        if (active) setValue({ data: null, loading: false, error: errorMessage(error) });
      });
    return () => {
      active = false;
    };
  }, [...dependencies, attempt]);

  return { ...value, retry };
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}
