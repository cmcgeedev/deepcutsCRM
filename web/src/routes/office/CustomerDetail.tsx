import { useMemo, useState, type FormEvent } from "react";
import { Link, useParams } from "react-router";
import { api, errorOf, type ApiError, type Schemas } from "../../api/client";
import { ErrorBanner, Field } from "../../components/Field";
import { useApi } from "../../components/useApi";
import { money, toCents } from "../../lib/format";

const DAYS = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
type Input = Schemas["CustomerInput"];

export function CustomerForm({ initial, onSubmit, fields, busy }: { initial: Input; onSubmit: (v: Input) => void; fields?: Record<string, string>; busy: boolean }) {
  // `initial` only ever changes when the CustomerDetail parent gives this component a
  // new `key` (see its render below), so a plain useState seed -- no reset effect needed.
  const [v, setV] = useState<Input>(initial);
  const set = (k: keyof Input) => (e: { target: { value: string } }) => setV({ ...v, [k]: e.target.value });
  const days = v.deliveryDays ?? [];
  return (
    <form onSubmit={(e: FormEvent) => { e.preventDefault(); onSubmit(v); }} className="stack card">
      <Field label="Name" error={fields?.name}><input value={v.name} onChange={set("name")} /></Field>
      <div className="cols">
        <Field label="Delivery address"><textarea value={v.deliveryAddress ?? ""} onChange={set("deliveryAddress")} /></Field>
        <Field label="Billing address"><textarea value={v.billingAddress ?? ""} onChange={set("billingAddress")} /></Field>
        <Field label="Contact name"><input value={v.contactName ?? ""} onChange={set("contactName")} /></Field>
        <Field label="Phone"><input value={v.phone ?? ""} onChange={set("phone")} /></Field>
        <Field label="Email"><input value={v.email ?? ""} onChange={set("email")} /></Field>
        <Field label="QuickBooks customer id"><input value={v.qboCustomerId ?? ""} onChange={set("qboCustomerId")} /></Field>
      </div>
      <Field label="Delivery notes (dock hours, gate codes)"><textarea value={v.deliveryNotes ?? ""} onChange={set("deliveryNotes")} /></Field>
      <div className="row">
        <span>Delivery days:</span>
        {DAYS.map((d) => (
          <label key={d} className="row"><input type="checkbox" checked={days.includes(d)} onChange={(e) => setV({ ...v, deliveryDays: e.target.checked ? [...days, d] : days.filter((x) => x !== d) })} />{d}</label>
        ))}
        {fields?.deliveryDays && <span className="error">{fields.deliveryDays}</span>}
      </div>
      <label className="row"><input type="checkbox" checked={v.active ?? true} onChange={(e) => setV({ ...v, active: e.target.checked })} /> active</label>
      <div><button disabled={busy}>Save</button></div>
    </form>
  );
}

export default function CustomerDetail() {
  const id = Number(useParams().id);
  const cust = useApi(() => api.GET("/api/office/customers/{customerId}", { params: { path: { customerId: id } } }), [id]);
  const prices = useApi(() => api.GET("/api/office/customers/{customerId}/prices", { params: { path: { customerId: id } } }), [id]);
  const products = useApi(() => api.GET("/api/office/products"), []);
  const [err, setErr] = useState<ApiError | null>(null);
  const [busy, setBusy] = useState(false);
  const [saved, setSaved] = useState(false);
  const [productId, setProductId] = useState<number>(0);
  const [price, setPrice] = useState("");
  const [effectiveFrom, setEffectiveFrom] = useState(new Date().toISOString().slice(0, 10));
  const [perr, setPerr] = useState<ApiError | null>(null);

  // Memoized so CustomerForm (seeded from `initial` via useState, reset by the
  // `key={id}` below on navigation to a different customer) doesn't get a
  // new object identity -- and doesn't lose in-progress edits -- on every
  // unrelated CustomerDetail re-render (e.g. the negotiated-price form fields).
  const c = cust.data;
  const initial = useMemo<Input | null>(
    () =>
      c
        ? { name: c.name, billingAddress: c.billingAddress, deliveryAddress: c.deliveryAddress, contactName: c.contactName, phone: c.phone, email: c.email, deliveryNotes: c.deliveryNotes, deliveryDays: c.deliveryDays, qboCustomerId: c.qboCustomerId ?? "", active: c.active }
        : null,
    [c],
  );

  async function save(v: Input) {
    setBusy(true); setSaved(false);
    const res = await api.PUT("/api/office/customers/{customerId}", { params: { path: { customerId: id } }, body: v });
    setBusy(false);
    setErr(errorOf(res));
    if (res.data) { cust.setData(res.data); setSaved(true); }
  }

  async function addPrice(e: FormEvent) {
    e.preventDefault();
    const cents = toCents(price);
    if (cents == null) return setPerr({ code: "invalid", message: "price must be a dollar amount like 5.99", fields: { priceCents: "must be a dollar amount" } });
    const res = await api.POST("/api/office/customers/{customerId}/prices", { params: { path: { customerId: id } }, body: { productId, priceCents: cents, effectiveFrom } });
    setPerr(errorOf(res));
    if (res.data) { setPrice(""); prices.reload(); }
  }

  if (cust.error) return <ErrorBanner error={cust.error} />;
  if (!c || !initial) return <p className="muted">Loading…</p>;
  return (
    <>
      <p><Link to="/office/customers">← Customers</Link></p>
      <h1>{c.name}</h1>
      <ErrorBanner error={err && !err.fields ? err : null} />
      {saved && <p className="muted">Saved.</p>}
      <CustomerForm key={id} initial={initial} onSubmit={save} fields={err?.fields} busy={busy} />

      <h2>Negotiated prices</h2>
      <form onSubmit={addPrice} className="row card">
        <Field label="Product" error={perr?.fields?.productId}>
          <select value={productId} onChange={(e) => setProductId(Number(e.target.value))}>
            <option value={0}>— choose —</option>
            {products.data?.map((p) => <option key={p.id} value={p.id}>{p.sku} · {p.name} ({money(p.basePriceCents)} base)</option>)}
          </select>
        </Field>
        <Field label="Price" error={perr?.fields?.priceCents}><input className="short" value={price} onChange={(e) => setPrice(e.target.value)} placeholder="5.49" /></Field>
        <Field label="Effective from" error={perr?.fields?.effectiveFrom}><input type="date" value={effectiveFrom} onChange={(e) => setEffectiveFrom(e.target.value)} /></Field>
        <button disabled={!productId}>Set price</button>
        <ErrorBanner error={perr && !perr.fields ? perr : null} />
      </form>
      {prices.data && prices.data.length === 0 && <p className="muted">No negotiated prices; this customer pays base prices.</p>}
      {prices.data && prices.data.length > 0 && (
        <table>
          <thead><tr><th>SKU</th><th>Product</th><th className="num">Price</th><th>Effective</th></tr></thead>
          <tbody>{prices.data.map((p) => <tr key={p.id}><td>{p.sku}</td><td>{p.productName}</td><td className="num">{money(p.priceCents)}</td><td>{p.effectiveFrom}</td></tr>)}</tbody>
        </table>
      )}
    </>
  );
}
