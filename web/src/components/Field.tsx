import type { ReactNode } from "react";

export function Field({ label, error, children }: { label: string; error?: string; children: ReactNode }) {
  return (
    <label>
      {label}
      {children}
      {error && <span className="error">{error}</span>}
    </label>
  );
}

export function ErrorBanner({ error }: { error: { message: string } | null | undefined }) {
  return error ? <p className="error" role="alert">{error.message}</p> : null;
}
