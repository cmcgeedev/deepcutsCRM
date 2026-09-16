const cls: Record<string, string> = { confirmed: "pending", scheduled: "pending", out: "pending", delivered: "warn", finalized: "ok", complete: "ok", cancelled: "bad", skipped: "bad" };
export function StatusBadge({ status }: { status: string }) {
  return <span className={`badge ${cls[status] ?? ""}`}>{status}</span>;
}
