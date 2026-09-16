import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { Navigate, useLocation } from "react-router";
import { api, type Schemas } from "../api/client";
import { clearRoute } from "../offline/db";

type Realm = "office" | "driver";
type User = Schemas["User"];
type Ctx = {
  realm: Realm;
  user: User | null;
  loading: boolean;
  /** True once the initial /me call has rejected (no network) and we're running on cached identity. */
  offline: boolean;
  setUser: (u: User | null) => void;
  logout: () => Promise<void>;
};

const SessionContext = createContext<Ctx | null>(null);

function storageKey(realm: Realm) {
  return `deepcuts.session.${realm}`;
}

function loadStoredUser(realm: Realm): User | null {
  try {
    const raw = localStorage.getItem(storageKey(realm));
    return raw ? (JSON.parse(raw) as User) : null;
  } catch {
    return null;
  }
}

function storeUser(realm: Realm, user: User | null) {
  try {
    if (user) localStorage.setItem(storageKey(realm), JSON.stringify(user));
    else localStorage.removeItem(storageKey(realm));
  } catch {
    // localStorage unavailable (private mode, quota) -- session just won't survive a cold start.
  }
}

export function SessionProvider({ realm, children }: { realm: Realm; children: ReactNode }) {
  const [user, setUserState] = useState<User | null>(() => loadStoredUser(realm));
  const [loading, setLoading] = useState(true);
  const [offline, setOffline] = useState(false);

  const setUser = useCallback((u: User | null) => {
    setUserState(u);
    storeUser(realm, u);
  }, [realm]);

  useEffect(() => {
    let alive = true;
    const me = realm === "office" ? api.GET("/api/office/me") : api.GET("/api/driver/me");
    me.then((r) => {
      if (!alive) return;
      setOffline(false);
      if (r.response.status === 401) setUser(null);
      else if (r.data) setUser(r.data);
    }).catch(() => {
      // Network failure: keep whatever identity we loaded from storage at mount.
      if (!alive) return;
      setOffline(true);
    }).finally(() => {
      if (alive) setLoading(false);
    });
    return () => { alive = false; };
  }, [realm, setUser]);

  const logout = useCallback(async () => {
    try {
      if (realm === "office") await api.POST("/api/office/logout");
      else await api.POST("/api/driver/logout");
    } finally {
      setUser(null);
      if (realm === "driver") await clearRoute().catch(() => {});
    }
  }, [realm, setUser]);

  return <SessionContext.Provider value={{ realm, user, loading, offline, setUser, logout }}>{children}</SessionContext.Provider>;
}

export function useSession(): Ctx {
  const c = useContext(SessionContext);
  if (!c) throw new Error("useSession outside SessionProvider");
  return c;
}

export function RequireSession({ children }: { children: ReactNode }) {
  const { realm, user, loading, offline } = useSession();
  const loc = useLocation();
  if (loading) return <p className="muted">Loading…</p>;
  if (!user) {
    if (offline) return <p className="muted">You're offline and not signed in on this phone.</p>;
    return <Navigate to={`/${realm}/login`} state={{ from: loc.pathname }} replace />;
  }
  return <>{children}</>;
}
