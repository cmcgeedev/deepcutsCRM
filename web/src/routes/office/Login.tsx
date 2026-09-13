import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router";
import { api, errorOf } from "../../api/client";
import { useSession } from "../../auth/Session";

export default function OfficeLogin() {
  const { setUser } = useSession();
  const nav = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await api.POST("/api/office/login", { body: { email, password } });
      const err = errorOf(res);
      if (err) return setError(err.message);
      setUser(res.data!);
      nav("/office/day", { replace: true });
    } catch {
      setError("Can't reach the server. Check that it is running, then try again.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="narrow">
      <h1>Deep Cuts — Office</h1>
      <form onSubmit={submit} className="stack">
        <label>Email<input type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="username" required /></label>
        <label>Password<input type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" required /></label>
        {error && <p className="error" role="alert">{error}</p>}
        <button disabled={busy}>Sign in</button>
      </form>
    </main>
  );
}
