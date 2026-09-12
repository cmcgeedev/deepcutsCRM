import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import type { ApiError, Schemas } from "../../api/client";
import { ErrorBanner } from "../../components/Field";
import { StatusBadge } from "../../components/StatusBadge";
import { hundredths, lb, qty, toHundredths } from "../../lib/format";
import { newClientId } from "../../offline/types";
import { useRoute } from "../../offline/useRoute";
import { ProofCapture } from "./Proof";

type Adj = { deliveredQty: string; deliveredWeight: string; shortageNote: string };

export default function Stop() {
  const id = Number(useParams().id);
  const nav = useNavigate();
  const { route, loading, act, pending } = useRoute();
  const [mode, setMode] = useState<"view" | "adjust" | "proof" | "skip">("view");
  const [adj, setAdj] = useState<Record<number, Adj>>({});
  const [note, setNote] = useState("");
  const [skipReason, setSkipReason] = useState("");
  const [err, setErr] = useState<ApiError | null>(null);
  const [busy, setBusy] = useState(false);

  const ds = route?.stops.find((s) => s.stop.id === id);
  if (loading && !route) return <p className="muted">Loading…</p>;
  if (!ds) return <><p><Link to="/driver/route">← Route</Link></p><p>Stop not found on today's route.</p></>;
  const { stop, order } = ds;

  function adjustments(): Schemas["LineAdjustment"][] | undefined {
    const out: Schemas["LineAdjustment"][] = [];
    for (const l of order.lines) {
      const a = adj[l.id];
      if (!a) continue;
      const item: Schemas["LineAdjustment"] = { lineId: l.id };
      if (a.deliveredQty !== "") { const n = toHundredths(a.deliveredQty); if (n == null) throw new Error(`${l.sku}: bad quantity`); item.deliveredQty = n; }
      if (a.deliveredWeight !== "") { const n = toHundredths(a.deliveredWeight); if (n == null) throw new Error(`${l.sku}: bad weight`); item.deliveredWeight = n; }
      if (a.shortageNote) item.shortageNote = a.shortageNote;
      if (item.deliveredQty != null || item.deliveredWeight != null || item.shortageNote) out.push(item);
    }
    return out.length ? out : undefined;
  }

  async function send(body: Omit<Schemas["DriverAction"], "clientId">) {
    setBusy(true);
    setErr(null);
    try {
      const e = await act(id, { clientId: newClientId(), ...body });
      setErr(e);
      if (!e) { setMode("view"); setAdj({}); if (body.type !== "adjust") nav("/driver/route"); }
    } catch (ex) {
      setErr({ code: "invalid", message: (ex as Error).message });
    } finally {
      setBusy(false);
    }
  }

  const setA = (lineId: number, k: keyof Adj, v: string) => setAdj({ ...adj, [lineId]: { deliveredQty: "", deliveredWeight: "", shortageNote: "", ...adj[lineId], [k]: v } });

  return (
    <>
      <p><Link to="/driver/route">← Route</Link></p>
      <h1>{stop.sequence}. <strong>{stop.customerName}</strong> <StatusBadge status={stop.status} /> {pending[id] > 0 && <span className="badge pending">pending sync</span>}</h1>
      <div className="card">
        <div><a href={`https://maps.google.com/?q=${encodeURIComponent(stop.deliveryAddress)}`}>{stop.deliveryAddress}</a></div>
        {stop.phone && <div><a href={`tel:${stop.phone}`}>{stop.contactName ? `${stop.contactName} · ` : ""}{stop.phone}</a></div>}
        {stop.deliveryNotes && <p><strong>{stop.deliveryNotes}</strong></p>}
        {order.notes && <p className="muted">Order note: {order.notes}</p>}
        {stop.skipReason && <p className="error">Skipped: {stop.skipReason}</p>}
      </div>
      <h2>On the truck</h2>
      <table>
        <tbody>
          {order.lines.map((l) => (
            <tr key={l.id}>
              <td><strong>{l.productName}</strong><br /><span className="muted">{l.sku}</span></td>
              <td>{qty(l.orderedQty, l.sellUnit)}{l.catchWeight && <div>{lb(l.shippedWeight ?? l.estWeight)}</div>}
                {l.deliveredQty != null && l.deliveredQty !== l.orderedQty && <div className="error">delivered {qty(l.deliveredQty, l.sellUnit)}</div>}
                {l.catchWeight && l.deliveredWeight != null && l.deliveredWeight !== l.shippedWeight && <div className="error">delivered {lb(l.deliveredWeight)}</div>}
                {l.shortageNote && <div className="muted">{l.shortageNote}</div>}
              </td>
              {mode === "adjust" && (
                <td className="stack">
                  <input inputMode="decimal" placeholder={`qty (${hundredths(l.orderedQty)})`} value={adj[l.id]?.deliveredQty ?? ""} onChange={(e) => setA(l.id, "deliveredQty", e.target.value)} />
                  {l.catchWeight && <input inputMode="decimal" placeholder={`lb (${hundredths(l.shippedWeight ?? l.estWeight ?? 0)})`} value={adj[l.id]?.deliveredWeight ?? ""} onChange={(e) => setA(l.id, "deliveredWeight", e.target.value)} />}
                  <input placeholder="note (rejected, short…)" value={adj[l.id]?.shortageNote ?? ""} onChange={(e) => setA(l.id, "shortageNote", e.target.value)} />
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
      <ErrorBanner error={err} />
      {mode === "view" && stop.status === "pending" && (
        <div className="actions">
          <button onClick={() => setMode("proof")} disabled={busy}>Delivered</button>
          <button className="secondary" onClick={() => setMode("adjust")} disabled={busy}>Adjust quantities</button>
          <button className="secondary" onClick={() => setMode("skip")} disabled={busy}>Skip stop</button>
        </div>
      )}
      {mode === "view" && stop.status === "delivered" && (
        <div className="actions"><button className="secondary" onClick={() => setMode("adjust")} disabled={busy}>Adjust quantities</button></div>
      )}
      {mode === "adjust" && (
        <div className="actions">
          <input placeholder="note for the office" value={note} onChange={(e) => setNote(e.target.value)} />
          <button onClick={() => { try { send({ type: "adjust", lines: adjustments(), note }); } catch (ex) { setErr({ code: "invalid", message: (ex as Error).message }); } }} disabled={busy}>Save adjustments</button>
          <button className="link" onClick={() => setMode("view")}>Cancel</button>
        </div>
      )}
      {mode === "proof" && (
        <>
          <h2>Proof of delivery</h2>
          <input placeholder="note for the office (optional)" value={note} onChange={(e) => setNote(e.target.value)} />
          <ProofCapture onCancel={() => setMode("view")} onDone={(proof) => { try { send({ type: "deliver", proof, lines: adjustments(), note }); } catch (ex) { setErr({ code: "invalid", message: (ex as Error).message }); } }} />
        </>
      )}
      {mode === "skip" && (
        <div className="actions card">
          <label>Reason<input value={skipReason} onChange={(e) => setSkipReason(e.target.value)} autoFocus /></label>
          <button className="danger" disabled={!skipReason.trim() || busy} onClick={() => send({ type: "skip", skipReason: skipReason.trim(), note })}>Confirm skip</button>
          <button className="link" onClick={() => setMode("view")}>Cancel</button>
        </div>
      )}
    </>
  );
}
