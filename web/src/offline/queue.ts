import { api, errorOf } from "../api/client";
import { deleteQueue, listQueue, putQueue } from "./db";
import type { DriverAction, DriverRoute, QueueItem } from "./types";

export { listQueue } from "./db";

type Listener = (items: QueueItem[]) => void;
const listeners = new Set<Listener>();
let flushing = false;
const nextTry = new Map<string, number>();

export function subscribe(fn: Listener): () => void {
  listeners.add(fn);
  listQueue().then(fn);
  return () => { listeners.delete(fn); };
}

async function notify() {
  const items = await listQueue();
  listeners.forEach((l) => l(items));
}

export async function enqueue(stopId: number, body: DriverAction): Promise<void> {
  await putQueue({ clientId: body.clientId, stopId, body, status: "pending", attempts: 0, createdAt: Date.now() });
  await notify();
}

/** Sends pending items oldest-first. Stops at the first retryable failure. `force` ignores backoff timers. */
export async function flush(force = false): Promise<void> {
  if (flushing) return;
  flushing = true;
  try {
    for (const item of await listQueue()) {
      if (item.status !== "pending") continue;
      if (!force && (nextTry.get(item.clientId) ?? 0) > Date.now()) continue;
      let res: { data?: unknown; error?: unknown; response: Response };
      try {
        res = await api.POST("/api/driver/stops/{stopId}/actions", { params: { path: { stopId: item.stopId } }, body: item.body });
      } catch {
        await retryLater(item, "network");
        break;
      }
      const err = errorOf(res);
      if (!err) { await deleteQueue(item.clientId); nextTry.delete(item.clientId); continue; }
      if (res.response.status >= 500) { await retryLater(item, err.message); break; }
      const msg = res.response.status === 401 ? "please sign in again" : err.message;
      await putQueue({ ...item, status: "stuck", attempts: item.attempts + 1, error: msg });
    }
  } finally {
    flushing = false;
    await notify();
  }
}

async function retryLater(item: QueueItem, reason: string) {
  const attempts = item.attempts + 1;
  nextTry.set(item.clientId, Date.now() + Math.min(2 ** attempts, 60) * 1000);
  await putQueue({ ...item, attempts, error: reason });
}

let started = false;
export function startAutoFlush() {
  if (started || typeof window === "undefined") return;
  started = true;
  window.addEventListener("online", () => flush(true));
  setInterval(() => { listQueue().then((items) => items.some((i) => i.status === "pending") && flush()); }, 15000);
}

/** Optimistically applies a queued action to the cached route. */
export function applyLocally(route: DriverRoute, item: QueueItem): DriverRoute {
  return {
    ...route,
    stops: route.stops.map((s) => {
      if (s.stop.id !== item.stopId) return s;
      const b = item.body;
      let stop = { ...s.stop };
      let lines = s.order.lines;
      if (b.lines) {
        lines = lines.map((l) => {
          const a = b.lines!.find((x) => x.lineId === l.id);
          return a ? { ...l, deliveredQty: a.deliveredQty ?? l.deliveredQty, deliveredWeight: a.deliveredWeight ?? l.deliveredWeight, shortageNote: a.shortageNote ?? l.shortageNote } : l;
        });
      }
      if (b.note) stop.driverNote = b.note;
      if (b.type === "deliver") {
        stop = { ...stop, status: "delivered", proofType: b.proof?.type, proofName: b.proof?.type === "name" ? b.proof.name : undefined, hasProofImage: b.proof?.type !== "name" };
      }
      if (b.type === "skip") stop = { ...stop, status: "skipped", skipReason: b.skipReason ?? "" };
      return { ...s, stop, order: { ...s.order, lines } };
    }),
  };
}
