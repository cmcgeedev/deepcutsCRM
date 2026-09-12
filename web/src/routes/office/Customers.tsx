import { useState, type FormEvent } from "react";
import { Link } from "react-router";
import { api, errorOf, type ApiError } from "../../api/client";
import { ErrorBanner, Field } from "../../components/Field";
import { useApi } from "../../components/useApi";

export default function Customers() {
  const [showInactive, setShowInactive] = useState(false);
  const list = useApi(() => api.GET("/api/office/customers", { params: { query: { includeInactive: showInactive } } }), [showInactive]);
  const [name, setName] = useState("");
  const [deliveryAddress, setDeliveryAddress] = useState("");
  const [err, setErr] = useState<ApiError | null>(null);
  const [busy, setBusy] = useState(false);

  async function create(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    const res = await api.POST("/api/office/customers", { body: { name, deliveryAddress, active: true } });
    setBusy(false);
    const e2 = errorOf(res);
    setErr(e2);
    if (!e2) { setName(""); setDeliveryAddress(""); list.reload(); }
  }

  return (
    <>
      <h1>Customers</h1>
      <form onSubmit={create} className="row card">
        <Field label="Name" error={err?.fields?.name}><input value={name} onChange={(e) => setName(e.target.value)} /></Field>
        <Field label="Delivery address" error={err?.fields?.deliveryAddress}><input value={deliveryAddress} onChange={(e) => setDeliveryAddress(e.target.value)} /></Field>
        <button disabled={busy}>Add customer</button>
        <ErrorBanner error={err && !err.fields ? err : null} />
      </form>
      <p><label className="row"><input type="checkbox" checked={showInactive} onChange={(e) => setShowInactive(e.target.checked)} /> show inactive</label></p>
      <ErrorBanner error={list.error} />
      {list.data && list.data.length === 0 && <p className="muted">No customers yet.</p>}
      {list.data && list.data.length > 0 && (
        <table>
          <thead><tr><th>Name</th><th>Delivery address</th><th>Contact</th><th>Days</th><th></th></tr></thead>
          <tbody>
            {list.data.map((c) => (
              <tr key={c.id}>
                <td><Link to={`/office/customers/${c.id}`}>{c.name}</Link></td>
                <td>{c.deliveryAddress}</td>
                <td>{c.contactName} {c.phone}</td>
                <td>{c.deliveryDays.join(", ")}</td>
                <td>{!c.active && <span className="badge">inactive</span>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </>
  );
}
