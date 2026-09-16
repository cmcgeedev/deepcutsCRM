import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SessionProvider } from "../../auth/Session";
import OfficeLogin from "./Login";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

function renderLogin() {
  return render(
    <MemoryRouter initialEntries={["/office/login"]}>
      <SessionProvider realm="office">
        <Routes>
          <Route path="/office/login" element={<OfficeLogin />} />
          <Route path="/office/day" element={<p>day page</p>} />
        </Routes>
      </SessionProvider>
    </MemoryRouter>,
  );
}

async function fillAndSubmit(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/email/i), "manager@deepcuts.test");
  await user.type(screen.getByLabelText(/password/i), "hunter2");
  await user.click(screen.getByRole("button", { name: /sign in/i }));
}

describe("OfficeLogin", () => {
  afterEach(() => vi.restoreAllMocks());

  it("shows a reachability message and re-enables Sign in when the server is unreachable", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const url = typeof input === "string" ? input : (input as Request).url;
      const method = init?.method ?? (input as Request).method;
      if (url.includes("/api/office/me")) return jsonResponse({ code: "unauthorized", message: "not signed in" }, 401);
      if (url.includes("/api/office/login") && method === "POST") throw new TypeError("Failed to fetch");
      return jsonResponse({}, 404);
    });
    renderLogin();
    const user = userEvent.setup();
    await fillAndSubmit(user);
    expect(await screen.findByRole("alert")).toHaveTextContent(/Can't reach the server/i);
    expect(screen.getByRole("button", { name: /sign in/i })).toBeEnabled();
  });

  it("shows the server's error message and re-enables Sign in on invalid credentials", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const url = typeof input === "string" ? input : (input as Request).url;
      const method = init?.method ?? (input as Request).method;
      if (url.includes("/api/office/me")) return jsonResponse({ code: "unauthorized", message: "not signed in" }, 401);
      if (url.includes("/api/office/login") && method === "POST") return jsonResponse({ code: "unauthorized", message: "invalid credentials" }, 401);
      return jsonResponse({}, 404);
    });
    renderLogin();
    const user = userEvent.setup();
    await fillAndSubmit(user);
    expect(await screen.findByRole("alert")).toHaveTextContent(/invalid credentials/i);
    expect(screen.getByRole("button", { name: /sign in/i })).toBeEnabled();
  });
});
