import { useCallback, useEffect, useState } from "react";
import { errorOf, type ApiError } from "../api/client";

type Result<T> = { data?: T; error?: unknown; response: Response };

export function useApi<T>(fn: () => Promise<Result<T>>, deps: unknown[]) {
  const [data, setData] = useState<T | undefined>();
  const [error, setError] = useState<ApiError | null>(null);
  const [loading, setLoading] = useState(true);
  const [tick, setTick] = useState(0);
  useEffect(() => {
    let alive = true;
    setLoading(true);
    fn().then((r) => {
      if (!alive) return;
      const e = errorOf(r);
      setError(e);
      setData(e ? undefined : r.data);
    }).finally(() => alive && setLoading(false));
    return () => { alive = false; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, tick]);
  const reload = useCallback(() => setTick((t) => t + 1), []);
  return { data, error, loading, reload, setData };
}
