import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SessionProvider } from "../../auth/Session";
import DriverLogin from "./Login";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function renderLogin() {
  return render(
    <MemoryRouter initialEntries={["/driver/login"]}>
      <SessionProvider realm="driver">
        <Routes>
          <Route path="/driver/login" element={<DriverLogin />} />
          <Route path="/driver/route" element={<p>route page</p>} />
        </Routes>
      </SessionProvider>
    </MemoryRouter>,
  );
}

async function fillAndSubmit(user: ReturnType<typeof userEvent.setup>) {
  await screen.findByText("Sam");
  await user.selectOptions(screen.getByLabelText(/who are you/i), "Sam");
  await user.type(screen.getByLabelText(/pin/i), "123456");
  await user.click(screen.getByRole("button", { name: /start/i }));
}

describe("DriverLogin", () => {
  afterEach(() => vi.restoreAllMocks());

  it("shows a reachability message and re-enables Start when the server is unreachable", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const url = typeof input === "string" ? input : (input as Request).url;
      const method = init?.method ?? (input as Request).method;
      if (url.includes("/api/driver/me")) return jsonResponse({ code: "unauthorized", message: "not signed in" }, 401);
      if (url.includes("/api/driver/drivers")) return jsonResponse([{ id: 2, displayName: "Sam" }]);
      if (url.includes("/api/driver/login") && method === "POST") throw new TypeError("Failed to fetch");
      return jsonResponse({}, 404);
    });
    renderLogin();
    const user = userEvent.setup();
    await fillAndSubmit(user);
    expect(await screen.findByRole("alert")).toHaveTextContent(/Can't reach the server/i);
    expect(screen.getByRole("button", { name: /start/i })).toBeEnabled();
  });

  it("shows Wrong PIN and re-enables Start on invalid credentials", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const url = typeof input === "string" ? input : (input as Request).url;
      const method = init?.method ?? (input as Request).method;
      if (url.includes("/api/driver/me")) return jsonResponse({ code: "unauthorized", message: "not signed in" }, 401);
      if (url.includes("/api/driver/drivers")) return jsonResponse([{ id: 2, displayName: "Sam" }]);
      if (url.includes("/api/driver/login") && method === "POST") return jsonResponse({ code: "unauthorized", message: "invalid credentials" }, 401);
      return jsonResponse({}, 404);
    });
    renderLogin();
    const user = userEvent.setup();
    await fillAndSubmit(user);
    expect(await screen.findByRole("alert")).toHaveTextContent(/Wrong PIN/i);
    expect(screen.getByRole("button", { name: /start/i })).toBeEnabled();
  });
});
