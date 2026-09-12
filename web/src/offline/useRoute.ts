import { useCallback, useEffect, useMemo, useState } from "react";
import { api, errorOf, type ApiError } from "../api/client";
import { loadRoute, saveRoute } from "./db";
import { applyLocally, enqueue, flush, startAutoFlush, subscribe } from "./queue";
import type { DriverAction, DriverRoute, QueueItem } from "./types";

export function useRoute() {
  const [server, setServer] = useState<DriverRoute | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ApiError | null>(null);
  const [queue, setQueue] = useState<QueueItem[]>([]);
  const [online, setOnline] = useState(typeof navigator === "undefined" ? true : navigator.onLine);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    startAutoFlush();
    const on = () => setOnline(true), off = () => setOnline(false);
    window.addEventListener("online", on); window.addEventListener("offline", off);
    const unsub = subscribe(setQueue);
    return () => { unsub(); window.removeEventListener("online", on); window.removeEventListener("offline", off); };
  }, []);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    (async () => {
      await flush(true);
      const cached = await loadRoute();
      if (cached && alive) setServer(cached);
      try {
        const r = await api.GET("/api/driver/route");
        if (!alive) return;
        const e = errorOf(r);
        if (r.data) { setServer(r.data); await saveRoute(r.data); setError(null); }
        else if (e?.code === "not_found") { setServer(null); setError(null); }
        else if (!cached) setError(e);
      } catch {
        if (!cached && alive) setError({ code: "offline", message: "No connection and no cached route" });
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => { alive = false; };
  }, [tick]);

  const route = useMemo(() => {
    if (!server) return null;
    return queue.filter((q) => q.status === "pending").reduce((r, q) => applyLocally(r, q), server);
  }, [server, queue]);

  const pending = useMemo(() => {
    const m: Record<number, number> = {};
    for (const q of queue) if (q.status === "pending") m[q.stopId] = (m[q.stopId] ?? 0) + 1;
    return m;
  }, [queue]);

  const reload = useCallback(() => setTick((t) => t + 1), []);

  const act = useCallback(async (stopId: number, body: DriverAction): Promise<ApiError | null> => {
    await enqueue(stopId, body);
    await flush(true);
    const items = await new Promise<QueueItem[]>((res) => { const un = subscribe((i) => { un(); res(i); }); });
    const mine = items.find((i) => i.clientId === body.clientId);
    if (mine?.status === "stuck") return { code: "stuck", message: mine.error ?? "could not send" };
    if (!mine) {
      // delivered to the server: refresh the cached copy in the background
      api.GET("/api/driver/route").then((r) => { if (r.data) { setServer(r.data); saveRoute(r.data); } });
    }
    return null;
  }, []);

  const complete = useCallback(async (routeId: number) => {
    const res = await api.POST("/api/driver/routes/{routeId}/complete", { params: { path: { routeId } } });
    const e = errorOf(res);
    if (res.data) { setServer(res.data); await saveRoute(res.data); }
    return e;
  }, []);

  return { route, loading, error, reload, pending, stuck: queue.filter((q) => q.status === "stuck"), act, complete, online };
}
