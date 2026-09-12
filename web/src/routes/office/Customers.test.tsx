import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import Customers from "./Customers";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("Customers", () => {
  afterEach(() => vi.restoreAllMocks());
  it("lists customers from the API", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async () => jsonResponse([
      { id: 1, name: "Blue Plate", billingAddress: "", deliveryAddress: "1 Main St", contactName: "", phone: "", email: "", deliveryNotes: "", deliveryDays: ["mon"], active: true },
    ]));
    render(<MemoryRouter><Customers /></MemoryRouter>);
    expect(await screen.findByText("Blue Plate")).toBeInTheDocument();
    expect(screen.getByText("1 Main St")).toBeInTheDocument();
  });
  it("shows a field error from a 422", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch");
    fetchMock.mockResolvedValueOnce(jsonResponse([]));
    fetchMock.mockResolvedValueOnce(jsonResponse({ code: "invalid", message: "validation failed", fields: { name: "required" } }, 422));
    render(<MemoryRouter><Customers /></MemoryRouter>);
    await screen.findByText(/no customers/i);
    (await screen.findByRole("button", { name: /add customer/i })).click();
    expect(await screen.findByText("required")).toBeInTheDocument();
  });
});
