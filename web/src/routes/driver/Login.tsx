import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router";
import { api, errorOf } from "../../api/client";
import { useSession } from "../../auth/Session";
import { useApi } from "../../components/useApi";

export default function DriverLogin() {
  const { setUser } = useSession();
  const nav = useNavigate();
  const drivers = useApi(() => api.GET("/api/driver/drivers"), []);
  const [userId, setUserId] = useState(0);
  const [pin, setPin] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await api.POST("/api/driver/login", { body: { userId, pin } });
      const err = errorOf(res);
      if (err) return setError(err.code === "rate_limited" ? err.message : "Wrong PIN");
      setUser(res.data!);
      nav("/driver/route", { replace: true });
    } catch {
      setError("Can't reach the server. Check that it is running, then try again.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="narrow driver">
      <h1>Deep Cuts — Driver</h1>
      <form onSubmit={submit} className="stack">
        <label>Who are you?
          <select value={userId} onChange={(e) => setUserId(Number(e.target.value))} required>
            <option value={0}>— choose —</option>
            {drivers.data?.map((d) => <option key={d.id} value={d.id}>{d.displayName}</option>)}
          </select>
        </label>
        <label>PIN<input type="password" inputMode="numeric" pattern="[0-9]{6}" maxLength={6} value={pin} onChange={(e) => setPin(e.target.value)} autoComplete="off" required /></label>
        {error && <p className="error" role="alert">{error}</p>}
        <button disabled={busy || !userId || pin.length !== 6}>Start</button>
      </form>
    </main>
  );
}
