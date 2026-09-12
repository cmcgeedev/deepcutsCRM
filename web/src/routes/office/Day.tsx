import { useState, type FormEvent } from "react";
import { Link, useSearchParams } from "react-router";
import { api, errorOf, type ApiError, type Schemas } from "../../api/client";
import { ErrorBanner, Field } from "../../components/Field";
import { StatusBadge } from "../../components/StatusBadge";
import { useApi } from "../../components/useApi";

type Route = Schemas["Route"];

function today(): string {
  const d = new Date();
  return new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 10);
}

export default function Day() {
  const [params, setParams] = useSearchParams();
  const date = params.get("date") || today();
  const day = useApi(() => api.GET("/api/office/day", { params: { query: { date } } }), [date]);
  const drivers = useApi(() => api.GET("/api/office/drivers"), []);
  const [err, setErr] = useState<ApiError | null>(null);
  const [driverId, setDriverId] = useState(0);
  const [truck, setTruck] = useState("");

  const run = async (p: Promise<{ error?: unknown; response: Response }>) => {
    const res = await p;
    setErr(errorOf(res));
    day.reload();
  };
  const rp = (routeId: number) => ({ params: { path: { routeId } } });

  async function newRoute(e: FormEvent) {
    e.preventDefault();
    await run(api.POST("/api/office/routes", { body: { routeDate: date, driverUserId: driverId, truckLabel: truck } }));
    setTruck("");
  }

  function move(r: Route, idx: number, dir: -1 | 1) {
    const ids = r.stops.map((s) => s.id);
    const j = idx + dir;
    if (j < 0 || j >= ids.length) return;
    [ids[idx], ids[j]] = [ids[j], ids[idx]];
    run(api.PUT("/api/office/routes/{routeId}/sequence", { ...rp(r.id), body: { stopIds: ids } }));
  }

  return (
    <>
      <div className="row">
        <h1>Day</h1>
        <input type="date" value={date} onChange={(e) => setParams({ date: e.target.value })} />
        <button className="secondary" onClick={() => setParams({ date: shift(date, -1) })}>‹ prev</button>
        <button className="secondary" onClick={() => setParams({ date: shift(date, 1) })}>next ›</button>
      </div>
      <ErrorBanner error={err} />
      <ErrorBanner error={day.error} />
      <div className="cols">
        <section>
          <h2>Unscheduled ({day.data?.unscheduled.length ?? 0})</h2>
          {day.data?.unscheduled.length === 0 && <p className="muted">Every confirmed order for this date is on a route.</p>}
          {day.data?.unscheduled.map((o) => (
            <div key={o.id} className="card row">
              <div style={{ flex: 1 }}>
                <Link to={`/office/orders/${o.id}`}>#{o.id}</Link> <strong>{o.customerName}</strong> <span className="muted">{o.lineCount} lines</span>
                {o.needsReview && <> <span className="badge warn">skipped earlier</span></>}
              </div>
              {day.data && day.data.routes.filter((r) => r.status === "planned").length > 0 && (
                <select defaultValue="" onChange={(e) => e.target.value && run(api.POST("/api/office/routes/{routeId}/stops", { ...rp(Number(e.target.value)), body: { orderId: o.id } }))}>
                  <option value="">add to route…</option>
                  {day.data.routes.filter((r) => r.status === "planned").map((r) => <option key={r.id} value={r.id}>#{r.id} {r.driverName} {r.truckLabel}</option>)}
                </select>
              )}
            </div>
          ))}
          <h2>New route</h2>
          <form onSubmit={newRoute} className="row card">
            <Field label="Driver"><select value={driverId} onChange={(e) => setDriverId(Number(e.target.value))}><option value={0}>— choose —</option>{drivers.data?.map((d) => <option key={d.id} value={d.id}>{d.displayName}</option>)}</select></Field>
            <Field label="Truck"><input className="short" value={truck} onChange={(e) => setTruck(e.target.value)} placeholder="Reefer 1" /></Field>
            <button disabled={!driverId}>Create route</button>
          </form>
        </section>
        <section>
          <h2>Routes ({day.data?.routes.length ?? 0})</h2>
          {day.data?.routes.map((r) => (
            <div key={r.id} className="card" style={{ marginBottom: "1rem" }}>
              <div className="row">
                <strong>Route #{r.id} · {r.driverName}</strong> <span className="muted">{r.truckLabel}</span> <StatusBadge status={r.status} />
                <span className="spacer" />
                {r.status === "planned" && <button onClick={() => run(api.POST("/api/office/routes/{routeId}/out", rp(r.id)))}>Mark out</button>}
                {r.status === "out" && <button className="secondary" onClick={() => run(api.POST("/api/office/routes/{routeId}/complete", rp(r.id)))}>Complete</button>}
              </div>
              {r.stops.length === 0 && <p className="muted">No stops yet.</p>}
              <table>
                <tbody>
                  {r.stops.map((s, i) => (
                    <tr key={s.id}>
                      <td className="num">{s.sequence}</td>
                      <td>
                        <Link to={`/office/orders/${s.orderId}`}>{s.customerName}</Link><br />
                        <span className="muted">{s.deliveryAddress}</span>
                        {s.deliveryNotes && <div className="muted">{s.deliveryNotes}</div>}
                        {s.skipReason && <div className="error">skipped: {s.skipReason}</div>}
                        {s.driverNote && <div className="muted">driver: {s.driverNote}</div>}
                      </td>
                      <td>
                        <StatusBadge status={s.status} />
                        {s.needsReview && <> <Link to={`/office/orders/${s.orderId}`} className="badge warn">needs review</Link></>}
                        {s.hasProofImage && <> <a href={`/api/office/stops/${s.id}/proof`} target="_blank" rel="noreferrer">proof</a></>}
                        {s.proofName && <div className="muted">received by {s.proofName}</div>}
                      </td>
                      <td>
                        {r.status === "planned" && (
                          <span className="row">
                            <button className="secondary" onClick={() => move(r, i, -1)} disabled={i === 0}>↑</button>
                            <button className="secondary" onClick={() => move(r, i, 1)} disabled={i === r.stops.length - 1}>↓</button>
                            <button className="link" onClick={() => run(api.DELETE("/api/office/routes/{routeId}/stops/{stopId}", { params: { path: { routeId: r.id, stopId: s.id } } }))}>remove</button>
                          </span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
        </section>
      </div>
    </>
  );
}

function shift(date: string, days: number): string {
  const d = new Date(date + "T12:00:00");
  d.setDate(d.getDate() + days);
  return d.toISOString().slice(0, 10);
}
