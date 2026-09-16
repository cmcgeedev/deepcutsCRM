-- +goose Up
CREATE TABLE customers (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  billing_address TEXT NOT NULL DEFAULT '',
  delivery_address TEXT NOT NULL DEFAULT '',
  contact_name TEXT NOT NULL DEFAULT '',
  phone TEXT NOT NULL DEFAULT '',
  email TEXT NOT NULL DEFAULT '',
  delivery_notes TEXT NOT NULL DEFAULT '',
  delivery_days TEXT NOT NULL DEFAULT '',
  qbo_customer_id TEXT,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
CREATE UNIQUE INDEX customers_qbo ON customers(qbo_customer_id) WHERE qbo_customer_id IS NOT NULL;

CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  realm TEXT NOT NULL CHECK (realm IN ('office','driver','customer')),
  display_name TEXT NOT NULL,
  email TEXT,
  password_hash TEXT,
  pin_hash TEXT,
  customer_id INTEGER REFERENCES customers(id),
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME NOT NULL
);
CREATE UNIQUE INDEX users_email ON users(email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX users_driver_name ON users(display_name) WHERE realm = 'driver';

CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES users(id),
  realm TEXT NOT NULL,
  expires_at DATETIME NOT NULL
);

CREATE TABLE products (
  id INTEGER PRIMARY KEY,
  sku TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  category TEXT NOT NULL DEFAULT '',
  sell_unit TEXT NOT NULL CHECK (sell_unit IN ('lb','case','each')),
  catch_weight BOOLEAN NOT NULL DEFAULT FALSE,
  approx_case_weight INTEGER, -- hundredths of lb
  base_price_cents INTEGER NOT NULL,
  qbo_item_id TEXT,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE customer_prices (
  id INTEGER PRIMARY KEY,
  customer_id INTEGER NOT NULL REFERENCES customers(id),
  product_id INTEGER NOT NULL REFERENCES products(id),
  price_cents INTEGER NOT NULL,
  effective_from TEXT NOT NULL,
  created_at DATETIME NOT NULL
);
CREATE INDEX customer_prices_lookup ON customer_prices(customer_id, product_id, effective_from);

CREATE TABLE orders (
  id INTEGER PRIMARY KEY,
  customer_id INTEGER NOT NULL REFERENCES customers(id),
  requested_delivery_date TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('draft','confirmed','scheduled','delivered','finalized','cancelled')),
  notes TEXT NOT NULL DEFAULT '',
  created_by INTEGER REFERENCES users(id),
  needs_review BOOLEAN NOT NULL DEFAULT FALSE,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  finalized_at DATETIME
);
CREATE INDEX orders_date ON orders(requested_delivery_date, status);

CREATE TABLE order_lines (
  id INTEGER PRIMARY KEY,
  order_id INTEGER NOT NULL REFERENCES orders(id),
  product_id INTEGER NOT NULL REFERENCES products(id),
  ordered_qty INTEGER NOT NULL, -- hundredths of the sell unit
  unit_price_cents INTEGER NOT NULL,
  price_overridden BOOLEAN NOT NULL DEFAULT FALSE,
  est_weight INTEGER, -- hundredths of lb
  shipped_weight INTEGER, -- hundredths of lb
  delivered_qty INTEGER, -- hundredths of the sell unit
  delivered_weight INTEGER, -- hundredths of lb
  shortage_note TEXT NOT NULL DEFAULT ''
);
CREATE INDEX order_lines_order ON order_lines(order_id);

CREATE TABLE delivery_routes (
  id INTEGER PRIMARY KEY,
  route_date TEXT NOT NULL,
  driver_user_id INTEGER NOT NULL REFERENCES users(id),
  truck_label TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('planned','out','complete')),
  created_at DATETIME NOT NULL,
  out_at DATETIME,
  completed_at DATETIME
);
CREATE INDEX delivery_routes_date ON delivery_routes(route_date);

CREATE TABLE delivery_stops (
  id INTEGER PRIMARY KEY,
  route_id INTEGER NOT NULL REFERENCES delivery_routes(id),
  order_id INTEGER NOT NULL REFERENCES orders(id),
  sequence INTEGER NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending','delivered','skipped')),
  delivered_at DATETIME,
  proof_type TEXT,
  proof_ref TEXT,
  skip_reason TEXT NOT NULL DEFAULT '',
  driver_note TEXT NOT NULL DEFAULT '',
  UNIQUE (route_id, order_id)
);
CREATE UNIQUE INDEX delivery_stops_active_order ON delivery_stops(order_id) WHERE status <> 'skipped';

CREATE TABLE driver_actions (
  client_id TEXT PRIMARY KEY,
  stop_id INTEGER NOT NULL REFERENCES delivery_stops(id),
  action_type TEXT NOT NULL,
  payload TEXT NOT NULL,
  received_at DATETIME NOT NULL
);

-- +goose Down
DROP TABLE driver_actions;
DROP TABLE delivery_stops;
DROP TABLE delivery_routes;
DROP TABLE order_lines;
DROP TABLE orders;
DROP TABLE customer_prices;
DROP TABLE products;
DROP TABLE sessions;
DROP TABLE users;
DROP TABLE customers;
