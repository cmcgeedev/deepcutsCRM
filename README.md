# Deep Cuts CRM

CRM for a small regional meat distributor: customers, per-customer pricing, catch-weight
orders, daily delivery routes, and an offline-capable driver app. Single Go binary with an
embedded React app and a SQLite database.

## Requirements

Go 1.27+, Node 26+. Everything else is pinned in `go.mod` and `web/package.json`.

## Quick start

    make demo        # build, seed demo data, serve on http://localhost:8080

Office login and driver PINs are printed by the seed step.

## Development

    make generate    # regenerate sqlc, oapi-codegen and the TS client
    make test        # Go and web tests
    make e2e         # build, seed, walk an order through the API

Design: `docs/superpowers/specs/2026-09-10-phase1-core-delivery-design.md`.
