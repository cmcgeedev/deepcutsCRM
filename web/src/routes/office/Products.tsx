import { useState, type FormEvent } from "react";
import { Link } from "react-router";
import { api, errorOf, type ApiError } from "../../api/client";
import { ErrorBanner, Field } from "../../components/Field";
import { useApi } from "../../components/useApi";
import { hundredths, money, toCents, toHundredths } from "../../lib/format";

export default function Products() {
  const [showInactive, setShowInactive] = useState(false);
  const list = useApi(() => api.GET("/api/office/products", { params: { query: { includeInactive: showInactive } } }), [showInactive]);
  const [f, setF] = useState({ sku: "", name: "", category: "", sellUnit: "case" as "lb" | "case" | "each", catchWeight: true, approxCaseWeight: "", basePrice: "" });
  const [err, setErr] = useState<ApiError | null>(null);
  const [busy, setBusy] = useState(false);

  async function create(e: FormEvent) {
    e.preventDefault();
    const basePriceCents = toCents(f.basePrice);
    const w = f.approxCaseWeight ? toHundredths(f.approxCaseWeight) : null;
    if (basePriceCents == null) return setErr({ code: "invalid", message: "", fields: { basePriceCents: "must be a dollar amount" } });
    setBusy(true);
    const res = await api.POST("/api/office/products", { body: { sku: f.sku, name: f.name, category: f.category, sellUnit: f.sellUnit, catchWeight: f.catchWeight, approxCaseWeight: w ?? undefined, basePriceCents, active: true } });
    setBusy(false);
    const e2 = errorOf(res);
    setErr(e2);
    if (!e2) { setF({ ...f, sku: "", name: "", approxCaseWeight: "", basePrice: "" }); list.reload(); }
  }

  return (
    <>
      <h1>Products</h1>
      <form onSubmit={create} className="row card">
        <Field label="SKU" error={err?.fields?.sku}><input className="short" value={f.sku} onChange={(e) => setF({ ...f, sku: e.target.value })} /></Field>
        <Field label="Name" error={err?.fields?.name}><input value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} /></Field>
        <Field label="Category"><input className="short" value={f.category} onChange={(e) => setF({ ...f, category: e.target.value })} /></Field>
        <Field label="Sell unit" error={err?.fields?.sellUnit}>
          <select value={f.sellUnit} onChange={(e) => setF({ ...f, sellUnit: e.target.value as typeof f.sellUnit })}><option value="case">case</option><option value="each">each</option><option value="lb">lb</option></select>
        </Field>
        <label className="row"><input type="checkbox" checked={f.catchWeight} onChange={(e) => setF({ ...f, catchWeight: e.target.checked })} /> catch-weight (priced per lb)</label>
        <Field label="Approx case lb" error={err?.fields?.approxCaseWeight}><input className="short" value={f.approxCaseWeight} onChange={(e) => setF({ ...f, approxCaseWeight: e.target.value })} disabled={!f.catchWeight} /></Field>
        <Field label={f.catchWeight ? "Base $/lb" : "Base price"} error={err?.fields?.basePriceCents}><input className="short" value={f.basePrice} onChange={(e) => setF({ ...f, basePrice: e.target.value })} /></Field>
        <button disabled={busy}>Add product</button>
        <ErrorBanner error={err && !err.fields ? err : null} />
      </form>
      <p><label className="row"><input type="checkbox" checked={showInactive} onChange={(e) => setShowInactive(e.target.checked)} /> show inactive</label></p>
      <ErrorBanner error={list.error} />
      {list.data && (
        <table>
          <thead><tr><th>SKU</th><th>Name</th><th>Category</th><th>Unit</th><th className="num">Base price</th><th></th></tr></thead>
          <tbody>
            {list.data.map((p) => (
              <tr key={p.id}>
                <td><Link to={`/office/products/${p.id}`}>{p.sku}</Link></td>
                <td>{p.name}</td>
                <td>{p.category}</td>
                <td>{p.catchWeight ? `case (~${hundredths(p.approxCaseWeight ?? 0)} lb), priced /lb` : p.sellUnit}</td>
                <td className="num">{money(p.basePriceCents)}</td>
                <td>{!p.active && <span className="badge">inactive</span>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </>
  );
}

