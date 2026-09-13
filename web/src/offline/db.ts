import { openDB as open, type DBSchema, type IDBPDatabase } from "idb";
import type { DriverRoute, QueueItem } from "./types";

interface Schema extends DBSchema {
  route: { key: string; value: { route: DriverRoute; savedAt: number } };
  queue: { key: string; value: QueueItem; indexes: { byStop: number; byCreated: number } };
}

let dbp: Promise<IDBPDatabase<Schema>> | null = null;

export function openDB() {
  if (!dbp) {
    dbp = open<Schema>("deepcuts-driver", 1, {
      upgrade(db) {
        db.createObjectStore("route");
        const q = db.createObjectStore("queue", { keyPath: "clientId" });
        q.createIndex("byStop", "stopId");
        q.createIndex("byCreated", "createdAt");
      },
      // A pending deleteDB()/version upgrade elsewhere (e.g. between tests) blocks
      // on this open connection until it closes; close it and forget the cached
      // promise so the next openDB() call reopens fresh.
      blocking() {
        dbp?.then((db) => db.close());
        dbp = null;
      },
    });
  }
  return dbp;
}

export async function saveRoute(route: DriverRoute) { await (await openDB()).put("route", { route, savedAt: Date.now() }, "current"); }
export async function loadRoute(): Promise<DriverRoute | null> { return (await (await openDB()).get("route", "current"))?.route ?? null; }
export async function clearRoute(): Promise<void> { await (await openDB()).delete("route", "current"); }
export async function putQueue(item: QueueItem) { await (await openDB()).put("queue", item); }
export async function listQueue(): Promise<QueueItem[]> { return (await openDB()).getAllFromIndex("queue", "byCreated"); }
export async function deleteQueue(clientId: string) { await (await openDB()).delete("queue", clientId); }
