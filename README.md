# Deep Cuts CRM

CRM for a small regional meat distributor: customers with negotiated prices, catch-weight
orders, daily delivery routes, and an offline-capable driver app. One Go binary with an
embedded React app and a SQLite database.

Design: `docs/superpowers/specs/2026-09-10-phase1-core-delivery-design.md`
Plan: `docs/superpowers/plans/2026-09-10-phase1-core-delivery.md`

## Requirements

Go 1.27+, Node 26+. Everything else is pinned in `go.mod` and `web/package.json`.
For the phone demo over HTTPS: `brew install mkcert && mkcert -install`.

## Run the demo

    make demo        # build, seed demo data into data/demo.sqlite, serve http://localhost:8080
    make demo-tls    # same over https://<lan-ip>:8443 so a phone can install the driver app

Logins are printed by the seed step (office@demo.local / demo1234; drivers Sam 111111, Riley 222222).
Office: `/office`. Driver: `/driver`.

## Real data

    bin/deepcuts user add-office you@example.com "Your Name"
    bin/deepcuts user add-driver "Sam"
    bin/deepcuts import customers customers.csv
    bin/deepcuts import products products.csv
    bin/deepcuts import prices prices.csv --effective 2026-10-01
    bin/deepcuts serve

CSV formats: `docs/import-formats.md`. Config is environment only:
`DEEPCUTS_DB_PATH` (data/deepcuts.sqlite), `DEEPCUTS_DATA_DIR` (data), `DEEPCUTS_ADDR` (:8080),
`DEEPCUTS_TIMEZONE` (America/New_York), `DEEPCUTS_TLS_CERT` / `DEEPCUTS_TLS_KEY`,
`DEEPCUTS_SECURE_COOKIES` (bool, default false -- set true when running behind a
TLS-terminating proxy so session cookies are still marked Secure).

## Development

    make generate    # sqlc, oapi-codegen, TypeScript client from api/openapi.yaml
    make test        # Go and web tests
    make e2e         # build, seed, walk an order draft → finalized through the API
    cd web && npm run dev   # Vite dev server proxying /api to :8080

Layout: `cmd/deepcuts` (CLI), `internal/domain` (rules, no I/O), `internal/service` (use cases),
`internal/httpapi` (chi + generated strict server), `internal/db/migrations` (goose),
`sql/queries` (sqlc), `api/openapi.yaml` (contract), `web/` (Vite + React, driver PWA).

Units: money in cents, weights in hundredths of a pound, quantities in hundredths of the sell
unit. Catch-weight products are ordered by the case and priced per pound; the invoice amount
is only final after delivery.
