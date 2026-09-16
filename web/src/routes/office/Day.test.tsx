import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import Day from "./Day";

const day = {
  date: "2026-09-10",
  unscheduled: [{ id: 5, customerId: 1, customerName: "Corner Tavern", requestedDeliveryDate: "2026-09-10", status: "confirmed", needsReview: false, lineCount: 2, notes: "" }],
  routes: [{
    id: 2, routeDate: "2026-09-10", driverUserId: 9, driverName: "Sam", truckLabel: "Reefer 1", status: "out",
    stops: [
      { id: 11, routeId: 2, orderId: 3, sequence: 1, status: "delivered", hasProofImage: true, proofType: "signature", skipReason: "", driverNote: "", customerName: "Blue Plate", deliveryAddress: "1 Main St", phone: "", contactName: "", deliveryNotes: "", orderStatus: "delivered", needsReview: true, lineCount: 3 },
      { id: 12, routeId: 2, orderId: 4, sequence: 2, status: "pending", hasProofImage: false, skipReason: "", driverNote: "", customerName: "Red Barn", deliveryAddress: "4400 County Rd", phone: "", contactName: "", deliveryNotes: "", orderStatus: "scheduled", needsReview: false, lineCount: 1 },
    ],
  }],
};

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("Day", () => {
  afterEach(() => vi.restoreAllMocks());
  it("renders unscheduled orders and routes with stops", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url = typeof input === "string" ? input : (input as Request).url;
      if (url.includes("/drivers")) return jsonResponse([{ id: 9, displayName: "Sam" }]);
      return jsonResponse(day);
    });
    render(<MemoryRouter initialEntries={["/office/day?date=2026-09-10"]}><Day /></MemoryRouter>);
    expect(await screen.findByText("Corner Tavern")).toBeInTheDocument();
    expect(screen.getAllByText(/Sam/).length).toBeGreaterThan(0); // route header and driver select
    expect(screen.getByText("Blue Plate")).toBeInTheDocument();
    expect(screen.getByText("needs review")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /proof/i })).toHaveAttribute("href", "/api/office/stops/11/proof");
    expect(screen.getByRole("button", { name: /complete/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /mark out/i })).not.toBeInTheDocument();
  });
});
