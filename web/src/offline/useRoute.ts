import { useCallback, useEffect, useMemo, useState } from "react";
import { api, errorOf, type ApiError } from "../api/client";
import { clearRoute, loadRouteMeta, saveRoute } from "./db";
import { applyLocally, enqueue, flush, startAutoFlush, subscribe } from "./queue";
import type { DriverAction, DriverRoute, QueueItem } from "./types";

const CACHE_TTL_MS = 20 * 60 * 60 * 1000;

/**
 * Cached route, but only if it was saved recently. The server (not the browser)
 * picks the route date in the business timezone, so comparing the cached
 * route's routeDate against the browser-local date rejects perfectly good
 * cache near midnight/DST boundaries; routeDate is only a tiebreak/debug aid
 * now. A stale (>20h old) cache is cleared and ignored.
 */
async function loadFreshRoute(): Promise<DriverRoute | null> {
  const cached = await loadRouteMeta();
  if (!cached) return null;
  if (Date.now() - cached.savedAt > CACHE_TTL_MS) { await clearRoute(); return null; }
  return cached.route;
}

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
    // Reload the cached route on every queue change (enqueue, and after each flush)
    // so a server-confirmed merge (see queue.ts's mergeStop) is picked up without
    // waiting for the next GET -- a successful send must never regress the display.
    // Await the cache read before either setState so React batches route+queue into
    // one render -- otherwise the queue empties a render ahead of the merged route
    // landing, and the stop flickers back to "pending" for a frame.
    const unsub = subscribe(async (items) => {
      const r = await loadFreshRoute();
      if (r) setServer(r);
      setQueue(items);
    });
    return () => { unsub(); window.removeEventListener("online", on); window.removeEventListener("offline", off); };
  }, []);

  useEffect(() => {
    let alive = true;
    // Resetting to true at the start of every fetch (tick bump = reload()) is the
    // point of this effect, not state derived from props -- not a set-state-in-effect footgun.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true);
    (async () => {
      const fresh = await loadFreshRoute();
      if (fresh && alive) setServer(fresh);
      // Don't block the first paint on the sync queue: read the cache immediately
      // and let the flush (which has its own 20s request timeout) run in the background.
      flush(true).catch(() => {});
      try {
        const r = await api.GET("/api/driver/route");
        if (!alive) return;
        const e = errorOf(r);
        if (r.data) { setServer(r.data); await saveRoute(r.data); setError(null); }
        else if (e?.code === "not_found") { setServer(null); setError(null); await clearRoute(); }
        else if (r.response.status === 401) { setError(e); }
        else if (!fresh) setError(e);
      } catch {
        if (!fresh && alive) setError({ code: "offline", message: "No connection and no cached route" });
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
      api.GET("/api/driver/route").then((r) => { if (r.data) { setServer(r.data); saveRoute(r.data); } }).catch(() => {});
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

/** Count of not-yet-delivered driver actions, for gating things like sign-out from any driver screen. */
export function useUnsyncedCount(): number {
  const [count, setCount] = useState(0);
  useEffect(() => subscribe((items) => setCount(items.filter((q) => q.status === "pending").length)), []);
  return count;
}
