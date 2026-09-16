import type { Schemas } from "../api/client";

export type DriverRoute = Schemas["DriverRoute"];
export type DriverStop = Schemas["DriverStop"];
export type DriverAction = Schemas["DriverAction"];
export type QueueItem = { clientId: string; stopId: number; body: DriverAction; status: "pending" | "stuck"; attempts: number; error?: string; createdAt: number };

export function newClientId(): string {
  return crypto.randomUUID();
}
