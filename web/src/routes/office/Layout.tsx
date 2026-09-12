import { NavLink, Outlet } from "react-router";
import { useSession } from "../../auth/Session";

export default function OfficeLayout() {
  const { user, logout } = useSession();
  return (
    <div className="office">
      <header className="topbar">
        <strong>Deep Cuts</strong>
        <nav>
          <NavLink to="/office/day">Day</NavLink>
          <NavLink to="/office/orders">Orders</NavLink>
          <NavLink to="/office/customers">Customers</NavLink>
          <NavLink to="/office/products">Products</NavLink>
        </nav>
        <span className="spacer" />
        <span className="muted">{user?.displayName}</span>
        <button className="link" onClick={logout}>Sign out</button>
      </header>
      <main className="content"><Outlet /></main>
    </div>
  );
}
