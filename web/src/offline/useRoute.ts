import { useCallback, useEffect, useState } from "react";
import { api, errorOf, type ApiError } from "../api/client";
import type { DriverAction, DriverRoute, QueueItem } from "./types";

export function useRoute() {
  const [route, setRoute] = useState<DriverRoute | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ApiError | null>(null);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    api.GET("/api/driver/route").then((r) => {
      if (!alive) return;
      const e = errorOf(r);
      setError(e && e.code !== "not_found" ? e : null);
      setRoute(r.data ?? null);
    }).finally(() => alive && setLoading(false));
    return () => { alive = false; };
  }, [tick]);

  const reload = useCallback(() => setTick((t) => t + 1), []);

  const act = useCallback(async (stopId: number, body: DriverAction) => {
    const res = await api.POST("/api/driver/stops/{stopId}/actions", { params: { path: { stopId } }, body });
    const e = errorOf(res);
    if (!e && res.data) {
      setRoute((r) => r && { ...r, stops: r.stops.map((s) => (s.stop.id === stopId ? res.data!.stop : s)) });
    }
    return e;
  }, []);

  const complete = useCallback(async (routeId: number) => {
    const res = await api.POST("/api/driver/routes/{routeId}/complete", { params: { path: { routeId } } });
    const e = errorOf(res);
    if (res.data) setRoute(res.data);
    return e;
  }, []);

  const pending: Record<number, number> = {};
  const stuck: QueueItem[] = [];
  return { route, loading, error, reload, pending, stuck, act, complete, online: true };
}
