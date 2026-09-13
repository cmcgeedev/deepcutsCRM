import { Outlet } from "react-router";
import { useSession } from "../../auth/Session";
import { useUnsyncedCount } from "../../offline/useRoute";

export default function DriverLayout() {
  const { user, logout } = useSession();
  const unsynced = useUnsyncedCount();
  return (
    <div className="driver">
      <header className="topbar">
        <strong>Deep Cuts</strong>
        <span className="spacer" />
        <span className="muted">{user?.displayName}</span>
        <button
          className="link"
          onClick={logout}
          disabled={unsynced > 0}
          title={unsynced > 0 ? `Waiting for ${unsynced} update(s) to sync` : undefined}
        >
          Sign out
        </button>
      </header>
      <main className="content"><Outlet /></main>
    </div>
  );
}
