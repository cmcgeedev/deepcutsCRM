import createClient from "openapi-fetch";
import type { components, paths } from "./schema";

export const api = createClient<paths>({ baseUrl: "/", credentials: "include" });

export type Schemas = components["schemas"];
export type ApiError = { code: string; message: string; fields?: Record<string, string> };

/** Normalizes an openapi-fetch result into an ApiError, or null when the call succeeded. */
export function errorOf(res: { error?: unknown; response: Response }): ApiError | null {
  if (res.response.ok) return null;
  const e = res.error as Partial<ApiError> | undefined;
  return { code: e?.code ?? "http_" + res.response.status, message: e?.message ?? res.response.statusText, fields: e?.fields };
}
