import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import Stop from "./Stop";

const route = {
  route: { id: 2, routeDate: "2026-09-10", driverUserId: 9, driverName: "Sam", truckLabel: "", status: "out", stops: [] },
  stops: [{
    stop: { id: 11, routeId: 2, orderId: 3, sequence: 1, status: "pending", hasProofImage: false, skipReason: "", driverNote: "", customerName: "Blue Plate", deliveryAddress: "1 Main St", phone: "555-0101", contactName: "Marcy", deliveryNotes: "Back door", orderStatus: "scheduled", needsReview: false, lineCount: 1 },
    order: {
      id: 3, customer: { id: 1, name: "Blue Plate", billingAddress: "", deliveryAddress: "1 Main St", contactName: "Marcy", phone: "555-0101", email: "", deliveryNotes: "Back door", deliveryDays: [], active: true },
      requestedDeliveryDate: "2026-09-10", status: "scheduled", notes: "", needsReview: false, totalCents: 70982, createdAt: "2026-09-10T15:00:00Z",
      lines: [{ id: 8, productId: 2, sku: "BRIS", productName: "Brisket", sellUnit: "case", catchWeight: true, orderedQty: 200, unitPriceCents: 599, priceOverridden: false, estWeight: 12000, shippedWeight: 11850, shortageNote: "", amountCents: 70982, amountSource: "shipped" }],
    },
  }],
};

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("Stop", () => {
  afterEach(() => vi.restoreAllMocks());
  it("shows the stop, lines with shipped weight, and posts a skip with a reason", async () => {
    const calls: { url: string; body: unknown }[] = [];
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const url = typeof input === "string" ? input : (input as Request).url;
      const method = init?.method ?? (input as Request).method;
      if (method === "POST") {
        const bodyText = init?.body ?? (await (input as Request).clone().text());
        calls.push({ url, body: JSON.parse(String(bodyText)) });
        return jsonResponse({ applied: true, stop: { ...route.stops[0], stop: { ...route.stops[0].stop, status: "skipped", skipReason: "closed" } } });
      }
      return jsonResponse(route);
    });
    render(<MemoryRouter initialEntries={["/driver/stops/11"]}><Routes><Route path="/driver/stops/:id" element={<Stop />} /></Routes></MemoryRouter>);
    expect(await screen.findByText("Blue Plate")).toBeInTheDocument();
    expect(screen.getByText(/118\.50 lb/)).toBeInTheDocument();
    expect(screen.getByText("Back door")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /skip/i }));
    await userEvent.type(screen.getByLabelText(/reason/i), "closed");
    await userEvent.click(screen.getByRole("button", { name: /confirm skip/i }));
    expect(calls).toHaveLength(1);
    expect(calls[0].url).toContain("/api/driver/stops/11/actions");
    const body = calls[0].body as { type: string; skipReason: string; clientId: string };
    expect(body.type).toBe("skip");
    expect(body.skipReason).toBe("closed");
    expect(body.clientId).toMatch(/^[0-9a-f-]{36}$/);
  });
});
