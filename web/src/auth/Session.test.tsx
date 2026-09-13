import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { RequireSession, SessionProvider } from "./Session";

const storedDriver = { id: 9, realm: "driver", displayName: "Sam" };

function renderDriverApp() {
  return render(
    <MemoryRouter initialEntries={["/driver/route"]}>
      <SessionProvider realm="driver">
        <Routes>
          <Route path="/driver/login" element={<p>login page</p>} />
          <Route path="/driver/route" element={<RequireSession><p>route page</p></RequireSession>} />
        </Routes>
      </SessionProvider>
    </MemoryRouter>,
  );
}

describe("SessionProvider offline cold-open", () => {
  beforeEach(() => localStorage.clear());
  afterEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  it("renders protected content from the stored driver user when /me rejects offline", async () => {
    localStorage.setItem("deepcuts.session.driver", JSON.stringify(storedDriver));
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new TypeError("Failed to fetch"));
    renderDriverApp();
    expect(await screen.findByText("route page")).toBeInTheDocument();
    expect(screen.queryByText("login page")).not.toBeInTheDocument();
  });

  it("shows an offline message instead of redirecting to login when there is no stored user", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new TypeError("Failed to fetch"));
    renderDriverApp();
    expect(await screen.findByText(/offline and not signed in/i)).toBeInTheDocument();
    expect(screen.queryByText("login page")).not.toBeInTheDocument();
  });
});
