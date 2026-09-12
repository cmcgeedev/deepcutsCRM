import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import OrderDetail from "./OrderDetail";

const order = {
  id: 7, customer: { id: 1, name: "Blue Plate", billingAddress: "", deliveryAddress: "1 Main St", contactName: "", phone: "", email: "", deliveryNotes: "", deliveryDays: [], active: true },
  requestedDeliveryDate: "2026-09-12", status: "draft", notes: "", needsReview: false, totalCents: 65880, createdAt: "2026-09-10T15:00:00Z",
  lines: [{ id: 3, productId: 2, sku: "BRIS", productName: "Brisket", sellUnit: "case", catchWeight: true, orderedQty: 200, unitPriceCents: 549, priceOverridden: false, estWeight: 12000, shortageNote: "", amountCents: 65880, amountSource: "estimated" }],
};

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("OrderDetail", () => {
  afterEach(() => vi.restoreAllMocks());
  it("renders lines with amounts and the estimated source", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url = typeof input === "string" ? input : (input as Request).url;
      if (url.includes("/products")) return jsonResponse([]);
      return jsonResponse(order);
    });
    render(<MemoryRouter initialEntries={["/office/orders/7"]}><Routes><Route path="/office/orders/:id" element={<OrderDetail />} /></Routes></MemoryRouter>);
    expect(await screen.findByText("Blue Plate")).toBeInTheDocument();
    expect(screen.getByText("Brisket")).toBeInTheDocument();
    expect(screen.getAllByText("$658.80").length).toBeGreaterThan(0); // line amount and total
    expect(screen.getByText(/estimated/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /confirm/i })).toBeInTheDocument();
    expect(screen.getByPlaceholderText("shortage note")).toBeEnabled();
  });
});
