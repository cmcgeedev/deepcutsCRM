import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import CustomerDetail from "./CustomerDetail";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

const customer = {
  id: 1,
  name: "Blue Plate",
  billingAddress: "",
  deliveryAddress: "1 Main St",
  contactName: "",
  phone: "",
  email: "",
  deliveryNotes: "",
  deliveryDays: ["mon"],
  active: true,
};

function mockFetch() {
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = typeof input === "string" ? input : (input as Request).url;
    if (url.includes("/prices")) return jsonResponse([]);
    if (url.includes("/products")) return jsonResponse([]);
    if (url.includes("/customers/1")) return jsonResponse(customer);
    return jsonResponse({ code: "not_found", message: "not found" }, 404);
  });
}

describe("CustomerDetail", () => {
  afterEach(() => vi.restoreAllMocks());

  it("keeps unsaved name edits when unrelated form state changes", async () => {
    mockFetch();
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={["/office/customers/1"]}>
        <Routes>
          <Route path="/office/customers/:id" element={<CustomerDetail />} />
        </Routes>
      </MemoryRouter>,
    );

    const name = (await screen.findByLabelText(/^name$/i)) as HTMLInputElement;
    await user.clear(name);
    await user.type(name, "New Name");
    expect(name.value).toBe("New Name");

    // Trigger a CustomerDetail re-render via unrelated state (the
    // negotiated-price form) that must not reset the customer form.
    const priceInput = screen.getByPlaceholderText("5.49");
    await user.type(priceInput, "9.99");

    expect(name.value).toBe("New Name");
  });
});
