import { useState, type FormEvent } from "react";
import { Link, useParams } from "react-router";
import { api, errorOf, type ApiError } from "../../api/client";
import { ErrorBanner, Field } from "../../components/Field";
import { useApi } from "../../components/useApi";
import { hundredths, toCents, toHundredths } from "../../lib/format";

export default function ProductDetail() {
  const id = Number(useParams().id);
  const prod = useApi(() => api.GET("/api/office/products/{productId}", { params: { path: { productId: id } } }), [id]);
  const [err, setErr] = useState<ApiError | null>(null);
  const [saved, setSaved] = useState(false);
  const [f, setF] = useState<{ sku: string; name: string; category: string; sellUnit: "lb" | "case" | "each"; catchWeight: boolean; approxCaseWeight: string; basePrice: string; qboItemId: string; active: boolean } | null>(null);

  const p = prod.data;
  if (prod.error) return <ErrorBanner error={prod.error} />;
  if (!p) return <p className="muted">Loading…</p>;
  const v = f ?? { sku: p.sku, name: p.name, category: p.category, sellUnit: p.sellUnit, catchWeight: p.catchWeight, approxCaseWeight: p.approxCaseWeight ? hundredths(p.approxCaseWeight) : "", basePrice: (p.basePriceCents / 100).toFixed(2), qboItemId: p.qboItemId ?? "", active: p.active };

  async function save(e: FormEvent) {
    e.preventDefault();
    const basePriceCents = toCents(v.basePrice);
    const w = v.approxCaseWeight ? toHundredths(v.approxCaseWeight) : null;
    if (basePriceCents == null) return setErr({ code: "invalid", message: "", fields: { basePriceCents: "must be a dollar amount" } });
    const res = await api.PUT("/api/office/products/{productId}", { params: { path: { productId: id } }, body: { sku: v.sku, name: v.name, category: v.category, sellUnit: v.sellUnit, catchWeight: v.catchWeight, approxCaseWeight: w ?? undefined, basePriceCents, qboItemId: v.qboItemId, active: v.active } });
    setErr(errorOf(res));
    if (res.data) { prod.setData(res.data); setF(null); setSaved(true); }
  }

  return (
    <>
      <p><Link to="/office/products">← Products</Link></p>
      <h1>{p.sku} · {p.name}</h1>
      {saved && <p className="muted">Saved.</p>}
      <ErrorBanner error={err && !err.fields ? err : null} />
      <form onSubmit={save} className="stack card">
        <div className="cols">
          <Field label="SKU" error={err?.fields?.sku}><input value={v.sku} onChange={(e) => setF({ ...v, sku: e.target.value })} /></Field>
          <Field label="Name" error={err?.fields?.name}><input value={v.name} onChange={(e) => setF({ ...v, name: e.target.value })} /></Field>
          <Field label="Category"><input value={v.category} onChange={(e) => setF({ ...v, category: e.target.value })} /></Field>
          <Field label="Sell unit" error={err?.fields?.sellUnit}>
            <select value={v.sellUnit} onChange={(e) => setF({ ...v, sellUnit: e.target.value as typeof v.sellUnit })}><option value="case">case</option><option value="each">each</option><option value="lb">lb</option></select>
          </Field>
          <Field label="Approx case lb" error={err?.fields?.approxCaseWeight}><input value={v.approxCaseWeight} onChange={(e) => setF({ ...v, approxCaseWeight: e.target.value })} disabled={!v.catchWeight} /></Field>
          <Field label={v.catchWeight ? "Base $/lb" : "Base price"} error={err?.fields?.basePriceCents}><input value={v.basePrice} onChange={(e) => setF({ ...v, basePrice: e.target.value })} /></Field>
          <Field label="QuickBooks item id"><input value={v.qboItemId} onChange={(e) => setF({ ...v, qboItemId: e.target.value })} /></Field>
        </div>
        <label className="row"><input type="checkbox" checked={v.catchWeight} onChange={(e) => setF({ ...v, catchWeight: e.target.checked })} /> catch-weight (ordered by the case, priced per lb)</label>
        <label className="row"><input type="checkbox" checked={v.active} onChange={(e) => setF({ ...v, active: e.target.checked })} /> active</label>
        <div><button>Save</button></div>
      </form>
    </>
  );
}

