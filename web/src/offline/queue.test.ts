import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { deleteDB } from "idb";
import { applyLocally, enqueue, flush, listQueue } from "./queue";
import type { DriverRoute } from "./types";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

const stopBody = { clientId: "11111111-1111-4111-8111-111111111111", type: "deliver" as const, proof: { type: "name" as const, name: "Pat" } };

describe("sync queue", () => {
  beforeEach(async () => { await deleteDB("deepcuts-driver"); });
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
});
