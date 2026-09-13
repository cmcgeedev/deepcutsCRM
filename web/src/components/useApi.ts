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
    // Resetting to true at the start of every fetch (deps change or reload()) is the
    // point of this effect, not state derived from props -- not a set-state-in-effect footgun.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true);
    fn().then((r) => {
      if (!alive) return;
      const e = errorOf(r);
      setError(e);
      setData(e ? undefined : r.data);
    }).catch(() => {
      // openapi-fetch rejects (rather than resolving) on a network failure.
      if (!alive) return;
      setError({ code: "offline", message: "No connection" });
      setData(undefined);
    }).finally(() => alive && setLoading(false));
    return () => { alive = false; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, tick]);
  const reload = useCallback(() => setTick((t) => t + 1), []);
  return { data, error, loading, reload, setData };
}
