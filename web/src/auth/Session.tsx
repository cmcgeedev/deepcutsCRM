import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { Navigate, useLocation } from "react-router";
import { api, type Schemas } from "../api/client";

type Realm = "office" | "driver";
type User = Schemas["User"];
type Ctx = { realm: Realm; user: User | null; loading: boolean; setUser: (u: User | null) => void; logout: () => Promise<void> };

const SessionContext = createContext<Ctx | null>(null);

export function SessionProvider({ realm, children }: { realm: Realm; children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    const me = realm === "office" ? api.GET("/api/office/me") : api.GET("/api/driver/me");
    me.then((r) => setUser(r.data ?? null)).finally(() => setLoading(false));
  }, [realm]);
  const logout = useCallback(async () => {
    if (realm === "office") await api.POST("/api/office/logout");
    else await api.POST("/api/driver/logout");
    setUser(null);
  }, [realm]);
  return <SessionContext.Provider value={{ realm, user, loading, setUser, logout }}>{children}</SessionContext.Provider>;
}

export function useSession(): Ctx {
  const c = useContext(SessionContext);
  if (!c) throw new Error("useSession outside SessionProvider");
  return c;
}

export function RequireSession({ children }: { children: ReactNode }) {
  const { realm, user, loading } = useSession();
  const loc = useLocation();
  if (loading) return <p className="muted">Loading…</p>;
  if (!user) return <Navigate to={`/${realm}/login`} state={{ from: loc.pathname }} replace />;
  return <>{children}</>;
}
