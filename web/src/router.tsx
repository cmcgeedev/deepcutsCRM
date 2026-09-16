import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import { RequireSession, SessionProvider } from "./auth/Session";
import OfficeLayout from "./routes/office/Layout";
import OfficeLogin from "./routes/office/Login";
import Day from "./routes/office/Day";
import Orders from "./routes/office/Orders";
import OrderDetail from "./routes/office/OrderDetail";
import Customers from "./routes/office/Customers";
import CustomerDetail from "./routes/office/CustomerDetail";
import Products from "./routes/office/Products";
import ProductDetail from "./routes/office/ProductDetail";
import DriverLayout from "./routes/driver/Layout";
import DriverLogin from "./routes/driver/Login";
import DriverRoute from "./routes/driver/Route";
import Stop from "./routes/driver/Stop";

function Office() {
  return (
    <SessionProvider realm="office">
      <Routes>
        <Route path="login" element={<OfficeLogin />} />
        <Route element={<RequireSession><OfficeLayout /></RequireSession>}>
          <Route index element={<Navigate to="day" replace />} />
          <Route path="day" element={<Day />} />
          <Route path="orders" element={<Orders />} />
          <Route path="orders/:id" element={<OrderDetail />} />
          <Route path="customers" element={<Customers />} />
          <Route path="customers/:id" element={<CustomerDetail />} />
          <Route path="products" element={<Products />} />
          <Route path="products/:id" element={<ProductDetail />} />
          <Route path="*" element={<Navigate to="day" replace />} />
        </Route>
      </Routes>
    </SessionProvider>
  );
}

function Driver() {
  return (
    <SessionProvider realm="driver">
      <Routes>
        <Route path="login" element={<DriverLogin />} />
        <Route element={<RequireSession><DriverLayout /></RequireSession>}>
          <Route index element={<Navigate to="route" replace />} />
          <Route path="route" element={<DriverRoute />} />
          <Route path="stops/:id" element={<Stop />} />
          <Route path="*" element={<Navigate to="route" replace />} />
        </Route>
      </Routes>
    </SessionProvider>
  );
}

export default function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Navigate to="/office" replace />} />
        <Route path="/office/*" element={<Office />} />
        <Route path="/driver/*" element={<Driver />} />
      </Routes>
    </BrowserRouter>
  );
}
