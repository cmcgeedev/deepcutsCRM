import { useState } from "react";
import { Link } from "react-router";
import type { ApiError } from "../../api/client";
import { ErrorBanner } from "../../components/Field";
import { StatusBadge } from "../../components/StatusBadge";
import { useRoute } from "../../offline/useRoute";

export default function DriverRoute() {
  const { route, loading, error, reload, pending, stuck, complete, online } = useRoute();
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<ApiError | null>(null);
  if (loading && !route) return <p className="muted">Loading…</p>;
  if (error) return <><ErrorBanner error={error} /><button onClick={reload}>Retry</button></>;
  if (!route) return <><h1>No route today</h1><p className="muted">Nothing is assigned to you for today.</p><button className="secondary" onClick={reload}>Refresh</button></>;
  const done = route.stops.every((s) => s.stop.status !== "pending");

  async function finish() {
    if (busy) return;
    setBusy(true);
    setErr(null);
    const e = await complete(route.route.id);
    setErr(e);
    setBusy(false);
  }
  return (
    <>
      <div className="row">
        <h1>Today · {route.route.truckLabel || "Route"} <StatusBadge status={route.route.status} /></h1>
        <span className="spacer" />
        {!online && <span className="badge warn">offline</span>}
        <button className="secondary" onClick={reload}>Refresh</button>
      </div>
      {stuck.length > 0 && <p className="error">{stuck.length} update(s) could not be sent: {stuck[0].error}</p>}
      {route.stops.map((s) => (
        <Link key={s.stop.id} to={`/driver/stops/${s.stop.id}`} className="stop card">
          <h3>{s.stop.sequence}. {s.stop.customerName} <StatusBadge status={s.stop.status} /> {pending[s.stop.id] > 0 && <span className="badge pending">pending sync</span>}</h3>
          <div className="muted">{s.stop.deliveryAddress}</div>
          <div className="muted">{s.order.lines.length} lines{s.stop.deliveryNotes ? ` · ${s.stop.deliveryNotes}` : ""}</div>
        </Link>
      ))}
      {route.route.status === "out" && (
        <div className="actions">
          <ErrorBanner error={err} />
          <button disabled={!done || busy} onClick={finish}>{done ? "Finish route" : "Finish route (stops remaining)"}</button>
        </div>
      )}
    </>
  );
}
