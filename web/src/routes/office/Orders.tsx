import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router";
import { api, errorOf, type ApiError } from "../../api/client";
import { ErrorBanner, Field } from "../../components/Field";
import { StatusBadge } from "../../components/StatusBadge";
import { useApi } from "../../components/useApi";

const STATUSES = ["", "draft", "confirmed", "scheduled", "delivered", "finalized", "cancelled"];

export default function Orders() {
  const nav = useNavigate();
  const [date, setDate] = useState("");
  const [status, setStatus] = useState("");
  const list = useApi(() => api.GET("/api/office/orders", { params: { query: { date: date || undefined, status: status || undefined } } }), [date, status]);
  const customers = useApi(() => api.GET("/api/office/customers"), []);
  const [customerId, setCustomerId] = useState(0);
  const [newDate, setNewDate] = useState(new Date(Date.now() + 86400000).toISOString().slice(0, 10));
  const [err, setErr] = useState<ApiError | null>(null);

  async function create(e: FormEvent) {
    e.preventDefault();
    const res = await api.POST("/api/office/orders", { body: { customerId, requestedDeliveryDate: newDate } });
    setErr(errorOf(res));
    if (res.data) nav(`/office/orders/${res.data.id}`);
  }

  return (
    <>
      <h1>Orders</h1>
      <form onSubmit={create} className="row card">
        <Field label="Customer" error={err?.fields?.customerId}>
          <select value={customerId} onChange={(e) => setCustomerId(Number(e.target.value))}>
            <option value={0}>— choose —</option>
            {customers.data?.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
        </Field>
        <Field label="Delivery date" error={err?.fields?.requestedDeliveryDate}><input type="date" value={newDate} onChange={(e) => setNewDate(e.target.value)} /></Field>
        <button disabled={!customerId}>New order</button>
        <ErrorBanner error={err && !err.fields ? err : null} />
      </form>
      <div className="row">
        <label>Date<input type="date" value={date} onChange={(e) => setDate(e.target.value)} /></label>
        <label>Status<select value={status} onChange={(e) => setStatus(e.target.value)}>{STATUSES.map((s) => <option key={s} value={s}>{s || "any"}</option>)}</select></label>
      </div>
      <ErrorBanner error={list.error} />
      {list.data && (
        <table>
          <thead><tr><th>#</th><th>Customer</th><th>Delivery</th><th>Status</th><th className="num">Lines</th><th></th></tr></thead>
          <tbody>
            {list.data.map((o) => (
              <tr key={o.id}>
                <td><Link to={`/office/orders/${o.id}`}>{o.id}</Link></td>
                <td>{o.customerName}</td>
                <td>{o.requestedDeliveryDate}</td>
                <td><StatusBadge status={o.status} /></td>
                <td className="num">{o.lineCount}</td>
                <td>{o.needsReview && <span className="badge warn">needs review</span>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </>
  );
}
