# Deep Cuts CRM: Phase 1 Design (Core + Delivery)

Status: draft, pending review
Date: 2026-09-10

## Purpose

A CRM for a small regional meat distributor, built for one anchor customer with the
intent to productize later. The distributor runs on spreadsheets plus QuickBooks Online
today. Drivers get their stops by text or printed sheet.

Phase 1 replaces the route sheet and the order spreadsheet: customers, products with
per-customer pricing, catch-weight orders, a daily delivery schedule, and a driver app that
works without signal.

## Roadmap

Each phase is useful on its own and gets its own spec and plan.

1. **Core + delivery** (this spec). Customers, products, per-customer prices, catch-weight
   orders, delivery routes, driver app, data import, local demo.
2. **Finalize to QuickBooks Online.** Finalizing a delivered order pushes an invoice to QBO.
   With catch-weight, the invoice cannot exist until after delivery; it is created from
   delivered quantities, never from ordered quantities.
3. **Inventory with lots.** Receiving, lot IDs with pack and use-by dates, picking assigns
   lots to order lines, first-expiry-first-out. Adds a child table under OrderLine; OrderLine
   itself does not change.
4. **Customer portal.** Customers log in, see their own prices, place orders. Third set of
   routes in the same web app, customer realm in the same API.
5. **Inbound email and SMS intake.** Free text becomes a draft order the office confirms in
   a review queue. Deliberately last: parsing free text against a catch-weight catalog without
   review produces wrong orders.

## Assumptions (confirm at review)

- Shipped weights are known at pick or pack, entered by the office before the route goes
  out. Drivers only adjust for shortages and rejections at the stop.
- Low volume: under 50 customers, a few hundred SKUs, a handful of drivers and trucks, tens
  of orders a day.
- Single distributor for the foreseeable future. No tenant column. If multi-tenancy is ever
  needed, the migration is adding a tenant column, not restructuring.
- QuickBooks Online remains the billing system through phase 2 at least. Invoicing may move
  into the CRM later, so orders carry prices and line amounts from day one.
- No route optimization. Stop order is set by the office.
- **Editing after confirmation (needs your answer):** food customers change orders up to
  the morning of delivery. The spec allows line edits through `scheduled` until the route is
  marked out. If the office instead needs a hard lock at confirmation, say so.
- The business runs in one time zone, configured as `DEEPCUTS_TIMEZONE`. "Today's route"
  and delivery dates are local dates in that zone; the server stores timestamps in UTC.

## Decisions

- **Stack:** Go backend, React frontend, SQLite database, single static binary that embeds
  the built frontend. Chosen because Go and JavaScript are the builder's professional
  languages and the app is form and state-machine heavy, not CPU heavy.
- **JSON API from day one** rather than server-rendered pages. The driver app needs offline
  behavior that is a solved problem in the PWA world, and phases 4 and 5 need the API anyway.
- **Idempotent driver actions** keyed by a client-generated UUID so the offline queue can
  replay safely.
- **Prices frozen on the order line** at creation. Later price changes never alter existing
  orders.
- **Three login realms** (office, driver, customer) enforced by middleware from the start.
  Customer realm has no login page until phase 4.
- **Imports are CLI subcommands,** not UI. They run a handful of times and must never delete.
- **Public repo.** Code is public. Customer data, price lists, uploads, and credentials never
  enter the repo: `data/` is gitignored and secrets come from environment or SSM.

## Data model

All amounts are stored as integer cents. Weights are stored as integer hundredths of a
pound. No floating point in money or weight columns.

**Customer**
- id, name, billing address, delivery address, contact name, phone, email
- delivery notes (dock hours, gate code), default delivery days (set of weekdays)
- qbo_customer_id (nullable), active, created_at, updated_at

**Product**
- id, sku (unique), name, category
- sell_unit: `lb`, `case`, `each`
- catch_weight (bool): ordered by the case, priced and invoiced by the pound, delivered in
  variable-weight cases. Catch-weight products always have sell_unit `case`.
- approx_case_weight (hundredths of lb, nullable; required when catch_weight)
- base_price_cents per sell unit, or per pound when catch_weight
- qbo_item_id (nullable), active, created_at, updated_at

**CustomerPrice**
- id, customer_id, product_id, price_cents, effective_from (date), created_at
- Latest effective_from on or before the order date wins. No end date; a new row supersedes.

**Order**
- id, customer_id, requested_delivery_date
- status: `draft` → `confirmed` → `scheduled` → `delivered` → `finalized`; `cancelled`
  reachable from draft, confirmed, scheduled
- notes, created_by (user id), needs_review (bool, set when delivered differs from shipped)
- created_at, updated_at, finalized_at (nullable)

**OrderLine**
- id, order_id, product_id
- ordered_qty (in sell unit, hundredths for lb; cases for catch-weight)
- unit_price_cents (frozen at creation), price_overridden (bool)
- est_weight (hundredths of lb, nullable; catch-weight only, from ordered cases × approx
  case weight)
- shipped_weight (hundredths of lb, nullable; entered at pick or pack)
- delivered_qty (nullable; defaults to ordered_qty on delivery for every line)
- delivered_weight (hundredths of lb, nullable; catch-weight only, defaults to shipped_weight
  on delivery)
- shortage_note (nullable)
- Extended amount is computed, never stored: catch-weight lines use delivered_weight (or
  shipped, or estimated, in that order of availability) × unit price per lb; other lines use
  delivered_qty (or ordered_qty) × unit price. Rounding: the product of hundredths and cents
  is divided by 100 and rounded half-up to whole cents.

**DeliveryRoute**
- id, date, driver_user_id, truck_label
- status: `planned` → `out` → `complete`
- created_at, out_at, completed_at

**DeliveryStop**
- id, route_id, order_id (unique per route; an order may have stops on several routes over
  time if it was skipped and rescheduled, but at most one stop with status `pending` or
  `delivered`), sequence
- status: `pending` → `delivered` | `skipped`
- delivered_at, proof_type (`signature` | `photo` | `name`), proof_ref (file path or name)
- skip_reason, driver_note

**DriverAction**
- client_id (UUID, primary key), stop_id, action_type (`deliver` | `adjust` | `skip`)
- payload (JSON), received_at
- A POST with a known client_id is acknowledged with 200 and not reapplied.

**User**
- id, realm (`office` | `driver` | `customer`), display_name
- email (office), password_hash (office), pin_hash (driver)
- customer_id (nullable, customer realm only, unused in phase 1)
- active, created_at

**Session**
- id (random token), user_id, realm, expires_at

Phase 3 hook: a `LotAllocation` table (order_line_id, lot_id, weight, qty) will hang under
OrderLine. Nothing on OrderLine changes when it arrives.

Migration constraint: no SQLite-specific column types or functions. Schema lives in goose
migrations and queries in sqlc files, so a Postgres move means editing dialect differences
(parameter style, upsert, RETURNING) in one place, regenerating, and running the tests.

## Pricing and order lifecycle

**Price resolution** runs once when a line is added: latest CustomerPrice for that customer
and product with effective_from on or before the order's requested delivery date, else the
product base price. The result is frozen on the line. Office can override the frozen price
by hand; the line records `price_overridden`.

**Lifecycle**
1. **Draft.** Office picks a customer, adds lines. Catch-weight lines show an estimated
   weight and amount.
2. **Confirmed.** Office marked it ready. Confirming requires at least one line. Lines
   remain editable (add, remove, change quantity, shipped weight, price override) through
   `scheduled` until the route is marked out, after which only shipped weights and price
   overrides change. See the assumption on editing after confirmation.
3. **Scheduled.** Order is on a DeliveryRoute for its date. It can move between routes or
   return to confirmed until the route goes out.
4. **Delivered.** Driver marked the stop. delivered_qty defaults to ordered_qty and
   delivered_weight defaults to shipped_weight unless the driver adjusted.
5. **Finalized.** Office reviewed and finalized. Read-only afterward. Phase 2 hooks the
   QBO push here; phase 1 just records finalized_at.

**Shipped weights** are entered per line from the office before the route goes out. If a
catch-weight line has no shipped weight when the route is marked out, the app warns but
does not block.

**Shortages and rejections.** Driver reduces delivered weight or quantity and leaves a note.
A fully rejected line has delivered quantity zero. Any difference between delivered and
shipped sets `needs_review` on the order.

**Skipped stops.** When a driver skips a stop, the stop keeps status `skipped` with its
reason, and the order returns to `confirmed` with `needs_review` set so the office
reschedules it onto another route or cancels it. The route can still complete with skipped
stops on it.

**Finalize** refuses if any line is missing a delivered quantity or weight it needs, and
names the lines. Finalize clears `needs_review`.

**Cancellation** is allowed through scheduled. A cancelled scheduled order is removed from
its route. After delivery, corrections happen through delivered quantities and, in phase 2,
credits in QuickBooks.

## Delivery schedule and driver app

**Office day view.** For a chosen date: confirmed orders for that date on the left, routes
on the right. Office creates a route (driver, truck label), adds orders to it, and orders
the stops by drag or up and down buttons. Stops show the delivery address and notes.
Marking the route **out** locks the stop list and the order lines; only shipped weights and
price overrides remain editable from the office.

**Driver app.** Driver picks their name and enters a PIN. Today's route appears as a stop
list: sequence, customer, address with a tap-to-navigate link, delivery notes, phone number.
Tapping a stop shows the lines with shipped quantity and weight, and three actions:

- **Deliver.** Requires proof: signature drawn on screen, a photo, or a received-by name.
- **Adjust.** Edit delivered weight or quantity per line, add a note. Allowed before or
  after delivering.
- **Skip.** Reason required. The office sees it in the day view.

**Offline.** The driver app is a PWA. Opening the route with signal caches the route and its
stops in IndexedDB. Every action is written locally with a client UUID, then a sync queue
POSTs it with retry and backoff. Stops show a pending badge until the server acknowledges.
Photos and signatures are stored as size-capped blobs (target under 300 KB each) so the
queue stays small.

Queue error handling: 200 or a 409 on a replayed UUID means done; 5xx or network failure
means retry; any other 4xx marks the item stuck and shows the message to the driver.

**Route completion.** When every stop is delivered or skipped, the driver taps done and the
route becomes complete. The office day view shows which orders need review before finalize.

## Application structure

```
deepcutsCRM/
  cmd/deepcuts/main.go        wires config, db, router, embedded UI; serve, import, seed, user subcommands
  internal/
    config/                   env-driven settings, DEEPCUTS_* prefix
    db/                       sqlite open (modernc.org/sqlite), WAL pragma, embedded goose migrations
    db/queries/               sqlc-generated code
    domain/                   plain types and rules: pricing, order and route state machines,
                              driver action application; no database access
    service/                  transactional use cases over domain + db
    http/                     chi router, middleware, JSON handlers by realm
    auth/                     sessions, password and PIN login, realm guard
    storage/                  proof uploads to local dir now, S3 later
    importer/                 CSV and QBO customer/product/price imports
  sql/
    schema/                   goose migrations
    queries/                  sqlc query files
  api/openapi.yaml            the API contract; Go and TS code are generated from it
  web/                        Vite + React (TypeScript)
    src/routes/office/        day view, orders, customers, products, prices
    src/routes/driver/        login, route list, stop detail, proof capture
    src/api/                  client generated from api/openapi.yaml
    src/offline/              IndexedDB cache and sync queue
    public/                   PWA manifest; service worker via vite-plugin-pwa
  data/                       gitignored: sqlite file, uploads
  infra/                      Terraform (phase 1 tail, after local demo works)
  docs/
  Makefile
```

**API.** All routes under `/api`. Prefixes `/api/office/*`, `/api/driver/*`, and
`/api/customer/*` (reserved). Spec-first: `api/openapi.yaml` is the contract. `oapi-codegen`
generates the Go server interface and request/response types; `openapi-typescript` plus
`openapi-fetch` generate the web client. Both generations run in `make generate` and CI
fails if generated code is stale. Any non-API path serves the SPA.

**Auth.** Cookie sessions stored in the Session table, each tagged with a realm. Middleware
rejects a request whose session realm does not match the route prefix. Office logs in with
email and password (bcrypt). Drivers pick a name from the active driver list and enter a
6 digit PIN (bcrypt). Because the driver list is visible before login, PIN attempts are
rate limited per user and per IP, and the driver login page is only reachable on the
`/driver` path.

**User management** is CLI in phase 1, matching imports: `deepcuts user add-office`,
`deepcuts user add-driver`, `deepcuts user set-pin`, `deepcuts user deactivate`.

**Errors.** Business rule violations return 409 with a stable `code` and `message`.
Validation failures return 422 with field names. The web app maps codes to inline messages.

**Single binary.** `make build` runs the Vite build, embeds `web/dist` via `embed.FS`, and
compiles for linux/arm64 and the host platform. Migrations run at startup.

**Background work.** None in phase 1.

## Data import

CLI subcommands of the binary. Each prints created, updated, and rejected counts, never
deletes, and is safe to re-run (matches on QBO ID or SKU and updates in place).

- `deepcuts import customers <file.csv>` — from a QBO customer export or the QBO API
  (read scope only). Stores qbo_customer_id when present.
- `deepcuts import products <file.csv>` — QBO items give SKU, name, base price, item ID.
  Spreadsheet columns supply sell unit, catch-weight flag, approximate case weight, category.
  Unmatched rows are reported.
- `deepcuts import prices <file.csv>` — customer, product, price. Both sides must exist.
  Writes CustomerPrice rows dated today.

Live QBO API pull is preferred for customers and products but CSV is the fallback if OAuth
setup lags. The QBO app is registered in phase 1 with read scope only.

Delivery notes, default delivery days, and dock details are filled in by hand in the office
UI.

## Local demo

`make demo` must work on a fresh clone with only Go and Node installed:

1. Build the frontend and binary.
2. Create `data/demo.sqlite`, run migrations.
3. `deepcuts seed demo` loads fixture data: one office user, two drivers, a dozen customers
   with delivery notes, forty products (half catch-weight), per-customer prices for a few
   customers, and a week of orders in mixed states including one route already out.
4. Start the server on localhost and print the office and driver logins.

The demo uses the same binary and code paths as production. No mocks.

**Phone on the LAN.** Service workers and installability require a secure origin, and only
`localhost` is exempt. To show the driver app on a real phone, `make demo-tls` generates a
local certificate with `mkcert` (documented one-time install and root trust on the phone)
and starts the server with `--tls-cert` and `--tls-key`. The printed URL uses the machine's
LAN address. A tunnel such as `tailscale serve` is an equivalent documented alternative.

## Testing

- **domain:** table-driven tests for price resolution, every order and route transition
  including disallowed ones, extended amount computation, idempotent action replay.
- **service:** tests against a temporary SQLite file: order creation with frozen prices,
  scheduling, route out lock, delivery adjustment, finalize refusal and success, all
  three imports with good and bad rows, re-run safety.
- **http:** `httptest` per realm: guard rejects cross-realm sessions; driver action endpoint
  acknowledges a repeated UUID without reapplying; error shapes match the contract.
- **web:** component tests for stop detail and proof capture; one test of the sync queue
  simulating a failed then succeeding POST.
- **end-to-end:** script builds the binary, migrates an empty database, seeds demo data, and
  walks an order from draft to finalized through the API.

## Infrastructure (after the local demo works)

Same lean AWS shape as Conjurate: one t4g instance, Caddy for TLS, Litestream replicating the
SQLite file to S3, systemd running the binary. Terraform in `infra/`. A cost rundown is
presented before any Terraform is written; expected range is $20 to 30 per month.

Secrets (session key, QBO client credentials) in SSM parameters, read at boot. CI runs Go
tests, web tests, and a build on every push. Deploy is a manual workflow dispatch. Uploads in
`data/uploads` sync nightly to S3 alongside Litestream.

## Out of scope for phase 1

QuickBooks invoice push, inventory and lots, customer portal, inbound email or SMS intake,
route optimization, credits and returns, standing orders, split deliveries, multi-tenancy.
