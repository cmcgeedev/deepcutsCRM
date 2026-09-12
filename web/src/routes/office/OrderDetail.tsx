import { useState, type FormEvent } from "react";
import { Link, useParams } from "react-router";
import { api, errorOf, type ApiError, type Schemas } from "../../api/client";
import { ErrorBanner, Field } from "../../components/Field";
import { StatusBadge } from "../../components/StatusBadge";
import { useApi } from "../../components/useApi";
import { hundredths, lb, money, qty, toCents, toHundredths } from "../../lib/format";

type Order = Schemas["Order"];
type Line = Schemas["OrderLine"];

function scope(o: Order) {
  const routeOut = o.routeStatus === "out";
  return {
    lines: ["draft", "confirmed", "scheduled"].includes(o.status) && !routeOut,
    shippedAndPrice: ["draft", "confirmed", "scheduled"].includes(o.status),
    delivered: o.status === "delivered",
  };
}

function LineRow({ o, l, onPatch, onDelete }: { o: Order; l: Line; onPatch: (lineId: number, body: Schemas["LinePatch"]) => Promise<ApiError | null>; onDelete: (lineId: number) => void }) {
  const s = scope(o);
  const [err, setErr] = useState<ApiError | null>(null);
  const [edit, setEdit] = useState<{ orderedQty: string; unitPrice: string; shippedWeight: string; deliveredQty: string; deliveredWeight: string; shortageNote: string }>({
    orderedQty: hundredths(l.orderedQty), unitPrice: (l.unitPriceCents / 100).toFixed(2), shippedWeight: l.shippedWeight != null ? hundredths(l.shippedWeight) : "",
    deliveredQty: l.deliveredQty != null ? hundredths(l.deliveredQty) : "", deliveredWeight: l.deliveredWeight != null ? hundredths(l.deliveredWeight) : "", shortageNote: l.shortageNote,
  });
  async function commit(field: keyof typeof edit) {
    const body: Schemas["LinePatch"] = {};
    const v = edit[field];
    if (field === "orderedQty") { const n = toHundredths(v); if (n == null) return setErr({ code: "invalid", message: "quantity must be a number" }); body.orderedQty = n; }
    if (field === "unitPrice") { const n = toCents(v); if (n == null) return setErr({ code: "invalid", message: "price must be a dollar amount" }); body.unitPriceCents = n; }
    if (field === "shippedWeight") { if (v === "") return; const n = toHundredths(v); if (n == null) return setErr({ code: "invalid", message: "weight must be a number" }); body.shippedWeight = n; }
    if (field === "deliveredQty") { if (v === "") return; const n = toHundredths(v); if (n == null) return setErr({ code: "invalid", message: "quantity must be a number" }); body.deliveredQty = n; }
    if (field === "deliveredWeight") { if (v === "") return; const n = toHundredths(v); if (n == null) return setErr({ code: "invalid", message: "weight must be a number" }); body.deliveredWeight = n; }
    if (field === "shortageNote") body.shortageNote = v;
    setErr(await onPatch(l.id, body));
  }
  const num = (field: keyof typeof edit, enabled: boolean) => (
    <input className="short" value={edit[field]} disabled={!enabled} onChange={(e) => setEdit({ ...edit, [field]: e.target.value })} onBlur={() => enabled && commit(field)} />
  );
  return (
    <tr>
      <td>{l.sku}<br /><span className="muted">{l.productName}</span></td>
      <td>{s.lines ? num("orderedQty", true) : qty(l.orderedQty, l.sellUnit)}{l.catchWeight && <div className="muted">est {lb(l.estWeight)}</div>}</td>
      <td>{s.shippedAndPrice ? num("unitPrice", true) : money(l.unitPriceCents)}{l.priceOverridden && <div className="muted">overridden</div>}<div className="muted">{l.catchWeight ? "/lb" : `/${l.sellUnit}`}</div></td>
      <td>{l.catchWeight ? (s.shippedAndPrice ? num("shippedWeight", true) : lb(l.shippedWeight)) : "—"}</td>
      <td>{s.delivered ? <>{num("deliveredQty", true)}{l.catchWeight && <> {num("deliveredWeight", true)}</>}</> : <>{l.deliveredQty != null ? qty(l.deliveredQty, l.sellUnit) : "—"}{l.catchWeight && <div className="muted">{lb(l.deliveredWeight)}</div>}</>}
        {(s.delivered || l.shortageNote) && <input value={edit.shortageNote} disabled={!s.delivered} placeholder="shortage note" onChange={(e) => setEdit({ ...edit, shortageNote: e.target.value })} onBlur={() => s.delivered && commit("shortageNote")} />}
      </td>
      <td className="num">{money(l.amountCents)}<div className="muted">{l.amountSource}</div></td>
      <td>{s.lines && <button className="link" onClick={() => onDelete(l.id)}>remove</button>}{err && <div className="error">{err.message}</div>}</td>
    </tr>
  );
}

export default function OrderDetail() {
  const id = Number(useParams().id);
  const ord = useApi(() => api.GET("/api/office/orders/{orderId}", { params: { path: { orderId: id } } }), [id]);
  const products = useApi(() => api.GET("/api/office/products"), []);
  const [err, setErr] = useState<ApiError | null>(null);
  const [productId, setProductId] = useState(0);
  const [qtyStr, setQtyStr] = useState("1");
  const [notes, setNotes] = useState<string | null>(null);

  const o = ord.data;
  if (ord.error) return <ErrorBanner error={ord.error} />;
  if (!o) return <p className="muted">Loading…</p>;
  const s = scope(o);
  const path = { params: { path: { orderId: id } } };

  const apply = async (p: Promise<{ data?: Order; error?: unknown; response: Response }>) => {
    const res = await p;
    const e = errorOf(res);
    setErr(e);
    if (res.data) ord.setData(res.data);
    return e;
  };

  async function addLine(e: FormEvent) {
    e.preventDefault();
    const n = toHundredths(qtyStr);
    if (n == null || n === 0) return setErr({ code: "invalid", message: "quantity must be a positive number" });
    if (!(await apply(api.POST("/api/office/orders/{orderId}/lines", { ...path, body: { productId, orderedQty: n } })))) setQtyStr("1");
  }

  const action = (name: "confirm" | "unconfirm" | "cancel" | "finalize") => () => {
    switch (name) {
      case "confirm": return apply(api.POST("/api/office/orders/{orderId}/confirm", path));
      case "unconfirm": return apply(api.POST("/api/office/orders/{orderId}/unconfirm", path));
      case "cancel": return apply(api.POST("/api/office/orders/{orderId}/cancel", path));
      case "finalize": return apply(api.POST("/api/office/orders/{orderId}/finalize", path));
    }
  };

  return (
    <>
      <p><Link to="/office/orders">← Orders</Link></p>
      <h1>Order #{o.id} <StatusBadge status={o.status} /> {o.needsReview && <span className="badge warn">needs review</span>}</h1>
      <div className="cols">
        <div className="card">
          <strong>{o.customer.name}</strong><br />
          {o.customer.deliveryAddress}<br />
          <span className="muted">{o.customer.contactName} {o.customer.phone}</span>
          {o.customer.deliveryNotes && <p className="muted">{o.customer.deliveryNotes}</p>}
        </div>
        <div className="card stack">
          <div>Delivery date: <strong>{o.requestedDeliveryDate}</strong>{o.routeId && <> · on route <Link to={`/office/day?date=${o.requestedDeliveryDate}`}>#{o.routeId}</Link> ({o.routeStatus})</>}</div>
          <Field label="Notes"><textarea value={notes ?? o.notes} disabled={!s.lines} onChange={(e) => setNotes(e.target.value)} onBlur={() => notes != null && notes !== o.notes && apply(api.PATCH("/api/office/orders/{orderId}", { ...path, body: { notes } }))} /></Field>
          <div className="row">
            {o.status === "draft" && <button onClick={action("confirm")}>Confirm</button>}
            {o.status === "confirmed" && <button className="secondary" onClick={action("unconfirm")}>Back to draft</button>}
            {o.status === "delivered" && <button onClick={action("finalize")}>Finalize</button>}
            {["draft", "confirmed", "scheduled"].includes(o.status) && <button className="danger" onClick={() => confirm("Cancel this order?") && action("cancel")()}>Cancel order</button>}
          </div>
        </div>
      </div>
      <ErrorBanner error={err} />
      {err?.fields && <ul className="error">{Object.entries(err.fields).map(([k, v]) => <li key={k}>{k}: {v}</li>)}</ul>}

      <h2>Lines</h2>
      <table>
        <thead><tr><th>Product</th><th>Ordered</th><th>Unit price</th><th>Shipped wt</th><th>Delivered</th><th className="num">Amount</th><th></th></tr></thead>
        <tbody>
          {o.lines.map((l) => (
            <LineRow key={l.id} o={o} l={l}
              onPatch={(lineId, body) => apply(api.PATCH("/api/office/orders/{orderId}/lines/{lineId}", { params: { path: { orderId: id, lineId } }, body }))}
              onDelete={(lineId) => apply(api.DELETE("/api/office/orders/{orderId}/lines/{lineId}", { params: { path: { orderId: id, lineId } } }))} />
          ))}
          <tr><td colSpan={5} className="num"><strong>Total</strong></td><td className="num"><strong>{money(o.totalCents)}</strong></td><td /></tr>
        </tbody>
      </table>
      {s.lines && (
        <form onSubmit={addLine} className="row card">
          <Field label="Product">
            <select value={productId} onChange={(e) => setProductId(Number(e.target.value))}>
              <option value={0}>— choose —</option>
              {products.data?.map((p) => <option key={p.id} value={p.id}>{p.sku} · {p.name}</option>)}
            </select>
          </Field>
          <Field label="Quantity"><input className="short" value={qtyStr} onChange={(e) => setQtyStr(e.target.value)} /></Field>
          <button disabled={!productId}>Add line</button>
        </form>
      )}
    </>
  );
}
