import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { deleteDB } from "idb";
import { loadRoute, saveRoute } from "./db";
import { applyLocally, enqueue, flush, listQueue, resetForTests } from "./queue";
import type { DriverRoute } from "./types";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

const stopBody = { clientId: "11111111-1111-4111-8111-111111111111", type: "deliver" as const, proof: { type: "name" as const, name: "Pat" } };

describe("sync queue", () => {
  beforeEach(async () => { await deleteDB("deepcuts-driver"); resetForTests(); });
  afterEach(() => vi.restoreAllMocks());

  it("retries after a network failure, then completes", async () => {
    const f = vi.spyOn(globalThis, "fetch");
    f.mockRejectedValueOnce(new TypeError("Failed to fetch"));
    await enqueue(11, stopBody);
    await flush();
    let items = await listQueue();
    expect(items).toHaveLength(1);
    expect(items[0].attempts).toBe(1);
    expect(items[0].status).toBe("pending");
    f.mockResolvedValueOnce(jsonResponse({ applied: true, stop: {} }));
    await flush(true);
    items = await listQueue();
    expect(items).toHaveLength(0);
    expect(f).toHaveBeenCalledTimes(2);
  });

  it("marks a 409 client_id_reused conflict as stuck", async () => {
    const f = vi.spyOn(globalThis, "fetch");
    f.mockResolvedValueOnce(jsonResponse({ code: "client_id_reused", message: "already used" }, 409));
    await enqueue(11, stopBody);
    await flush();
    const items = await listQueue();
    expect(items[0].status).toBe("stuck");
    expect(items[0].error).toBe("already used");
  });

  it("marks a 401 as stuck asking the driver to sign in again", async () => {
    const f = vi.spyOn(globalThis, "fetch");
    f.mockResolvedValueOnce(jsonResponse({ code: "unauthorized", message: "no session" }, 401));
    await enqueue(11, stopBody);
    await flush();
    const items = await listQueue();
    expect(items[0].status).toBe("stuck");
    expect(items[0].error).toBe("please sign in again");
  });

  it("keeps strict oldest-first order: a later item is not sent while an earlier one is in backoff", async () => {
    const f = vi.spyOn(globalThis, "fetch");
    f.mockResolvedValueOnce(jsonResponse({ code: "server_error", message: "boom" }, 500));
    await enqueue(11, stopBody);
    await new Promise((r) => setTimeout(r, 2));
    await enqueue(11, { ...stopBody, clientId: "33333333-3333-4333-8333-333333333333", type: "skip", skipReason: "closed" });
    await flush();
    expect(f).toHaveBeenCalledTimes(1);
    const afterFirst = await listQueue();
    expect(afterFirst).toHaveLength(2);
    expect(afterFirst.every((i) => i.status === "pending")).toBe(true);
    // The earlier item is still backed off; a non-forced flush must not skip ahead to the newer one.
    await flush();
    expect(f).toHaveBeenCalledTimes(1);
  });

  it("treats a replay (applied:false) as done and a 422 as stuck", async () => {
    const f = vi.spyOn(globalThis, "fetch");
    f.mockResolvedValueOnce(jsonResponse({ applied: false, stop: {} }));
    await enqueue(11, stopBody);
    await flush();
    expect(await listQueue()).toHaveLength(0);
    f.mockResolvedValueOnce(jsonResponse({ code: "invalid", message: "proof required" }, 422));
    await enqueue(12, { ...stopBody, clientId: "22222222-2222-4222-8222-222222222222" });
    await flush();
    const items = await listQueue();
    expect(items[0].status).toBe("stuck");
    expect(items[0].error).toBe("proof required");
  });

  it("applies deliver and adjust optimistically", () => {
    const route = { route: { id: 1, routeDate: "", driverUserId: 1, driverName: "", truckLabel: "", status: "out", stops: [] }, stops: [{
      stop: { id: 11, routeId: 1, orderId: 3, sequence: 1, status: "pending", hasProofImage: false, skipReason: "", driverNote: "", customerName: "", deliveryAddress: "", phone: "", contactName: "", deliveryNotes: "", orderStatus: "scheduled", needsReview: false, lineCount: 1 },
      order: { id: 3, customer: { id: 1, name: "", billingAddress: "", deliveryAddress: "", contactName: "", phone: "", email: "", deliveryNotes: "", deliveryDays: [], active: true }, requestedDeliveryDate: "", status: "scheduled", notes: "", needsReview: false, totalCents: 0, createdAt: "", lines: [{ id: 8, productId: 1, sku: "", productName: "", sellUnit: "case", catchWeight: true, orderedQty: 200, unitPriceCents: 1, priceOverridden: false, shortageNote: "", amountCents: 0, amountSource: "shipped" }] },
    }] } as unknown as DriverRoute;
    const adjusted = applyLocally(route, { clientId: "x", stopId: 11, status: "pending", attempts: 0, createdAt: 0, body: { clientId: "x", type: "adjust", lines: [{ lineId: 8, deliveredWeight: 5000, shortageNote: "short" }] } });
    expect(adjusted.stops[0].order.lines[0].deliveredWeight).toBe(5000);
    expect(adjusted.stops[0].order.lines[0].shortageNote).toBe("short");
    const delivered = applyLocally(adjusted, { clientId: "y", stopId: 11, status: "pending", attempts: 0, createdAt: 0, body: { clientId: "y", type: "deliver", proof: { type: "name", name: "Pat" } } });
    expect(delivered.stops[0].stop.status).toBe("delivered");
    expect(delivered.stops[0].stop.proofName).toBe("Pat");
  });

  it("merges the returned stop into the cached route on success", async () => {
    const cached = { route: { id: 1, routeDate: "2026-09-10", driverUserId: 1, driverName: "", truckLabel: "", status: "out", stops: [] }, stops: [{
      stop: { id: 11, routeId: 1, orderId: 3, sequence: 1, status: "pending", hasProofImage: false, skipReason: "", driverNote: "", customerName: "Blue Plate", deliveryAddress: "", phone: "", contactName: "", deliveryNotes: "", orderStatus: "scheduled", needsReview: false, lineCount: 1 },
      order: { id: 3, customer: { id: 1, name: "", billingAddress: "", deliveryAddress: "", contactName: "", phone: "", email: "", deliveryNotes: "", deliveryDays: [], active: true }, requestedDeliveryDate: "", status: "scheduled", notes: "", needsReview: false, totalCents: 0, createdAt: "", lines: [] },
    }] } as unknown as DriverRoute;
    await saveRoute(cached);
    const updatedStop = { ...cached.stops[0], stop: { ...cached.stops[0].stop, status: "delivered", proofType: "name", proofName: "Pat" } };
    const f = vi.spyOn(globalThis, "fetch");
    f.mockResolvedValueOnce(jsonResponse({ applied: true, stop: updatedStop }));
    await enqueue(11, stopBody);
    await flush();
    const merged = await loadRoute();
    expect(merged?.stops[0].stop.status).toBe("delivered");
    expect(merged?.stops[0].stop.proofName).toBe("Pat");
  });
});
