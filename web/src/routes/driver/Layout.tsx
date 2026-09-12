import { Outlet } from "react-router";
import { useSession } from "../../auth/Session";

export default function DriverLayout() {
  const { user, logout } = useSession();
  return (
    <div className="driver">
      <header className="topbar">
        <strong>Deep Cuts</strong>
        <span className="spacer" />
        <span className="muted">{user?.displayName}</span>
        <button className="link" onClick={logout}>Sign out</button>
      </header>
      <main className="content"><Outlet /></main>
    </div>
  );
}
