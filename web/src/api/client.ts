import createClient from "openapi-fetch";
import type { components, paths } from "./schema";

// An absolute baseUrl (rather than "/") is required so the underlying Request
// construction resolves correctly under Node's fetch implementation (used by
// Vitest/jsdom in tests), which -- unlike browsers -- has no document to
// resolve a relative URL against. `location.origin` is the same origin the
// app is served from in both dev (via the Vite proxy) and production.
//
// `fetch` is wrapped instead of passed directly so it resolves `globalThis.fetch`
// dynamically on every call rather than capturing it once here at module-import
// time -- otherwise `vi.spyOn(globalThis, "fetch")` in tests would have no effect,
// since this module (and its baked-in fetch reference) is imported before any
// test runs.
export const api = createClient<paths>({
  baseUrl: location.origin,
  credentials: "include",
  fetch: (...args: Parameters<typeof fetch>) => globalThis.fetch(...args),
});

export type Schemas = components["schemas"];
export type ApiError = { code: string; message: string; fields?: Record<string, string> };

/** Normalizes an openapi-fetch result into an ApiError, or null when the call succeeded. */
export function errorOf(res: { error?: unknown; response: Response }): ApiError | null {
  if (res.response.ok) return null;
  const e = res.error as Partial<ApiError> | undefined;
  return { code: e?.code ?? "http_" + res.response.status, message: e?.message ?? res.response.statusText, fields: e?.fields };
}
