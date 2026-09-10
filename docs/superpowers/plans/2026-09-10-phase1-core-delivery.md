# Phase 1: Core + Delivery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A single Go binary with an embedded React app that lets an office manage customers, products, per-customer prices, catch-weight orders and daily delivery routes, and lets drivers work a route from a phone with offline support, demoable locally with seeded data.

**Architecture:** Go backend (chi + oapi-codegen strict server, sqlc over SQLite via modernc, goose migrations) with business rules in a database-free `domain` package and transactional use cases in `service`. Spec-first OpenAPI contract at `api/openapi.yaml` generates both the Go server interface and the TypeScript client. React (Vite, TypeScript) app with office and driver route groups; the driver group is a PWA with an IndexedDB cache and an idempotent sync queue. The built web app is embedded into the binary.

**Tech Stack:** Go 1.27, chi v5, oapi-codegen v2.8 (strict server), sqlc v1.31 (sqlite engine), goose v3.28, modernc.org/sqlite v1.58, golang.org/x/crypto (bcrypt); Node 26, Vite, React 19, react-router v7, openapi-typescript + openapi-fetch, vite-plugin-pwa, idb, signature_pad, vitest + @testing-library/react.

**Spec:** `docs/superpowers/specs/2026-09-10-phase1-core-delivery-design.md`

## Global Constraints

- Module path `github.com/cmcgeedev/deepcutsCRM`. Go 1.27. Tools pinned in `go.mod` `tool` directives and run with `go tool <name>`; never rely on a globally installed sqlc, goose or oapi-codegen.
- Money is integer cents. Weights are integer hundredths of a pound. **Quantities are integer hundredths of the sell unit for every unit** (3 cases is stored as 300). No floating point in money, weight or quantity columns or Go types. Rounding of extended amounts: `(hundredths × cents + 50) / 100` integer division (half-up).
- No SQLite-specific column types or functions in migrations or queries. Allowed types: INTEGER, TEXT, BOOLEAN, DATETIME. Dates are TEXT `YYYY-MM-DD`. Timestamps are DATETIME stored in UTC.
- Business time zone from `DEEPCUTS_TIMEZONE` (default `America/New_York`). "Today" is the local date in that zone.
- Three realms: `office`, `driver`, `customer`. API prefixes `/api/office/*`, `/api/driver/*`, `/api/customer/*` (reserved, unused). A session carries one realm; middleware rejects mismatches with 401.
- Driver PINs are exactly 6 digits. PIN attempts rate limited per user and per IP.
- Driver actions are idempotent on `clientId` (UUID). A replay returns 200 with `applied: false`.
- Errors: business rule violations return 409 `{code, message}`; validation returns 422 `{code: "invalid", message, fields}`; not found 404; auth 401. All errors use the `Error` schema.
- `data/` is gitignored and holds the sqlite file and uploads. No secrets or customer data in the repo. The repo is public.
- Every generated file (`internal/db/queries/*.go`, `internal/api/api.gen.go`, `web/src/api/schema.d.ts`) is committed. `make generate` regenerates; CI fails if `git diff --exit-code` shows drift after generate.
- Commit after every task; push after every task (`git push -u origin main`).
- Commands: prefix with `export PATH=/opt/homebrew/bin:$PATH` in every shell. Use explicit timeouts. Reads ≤200 lines, edits ≤80 lines per tool call.
- Terraform / AWS deployment is **not** part of this plan. It is a follow-on plan gated on a cost rundown.
- QuickBooks Online live API pull is **not** part of this plan (needs an Intuit developer app and Chris's credentials). Imports are CSV; the `qbo_customer_id` and `qbo_item_id` columns are populated from CSV columns when present.

## File Map

Backend (Go):
- `go.mod`, `Makefile`, `sqlc.yaml`, `oapi-codegen.yaml`, `.github/workflows/ci.yml`
- `cmd/deepcuts/main.go` — subcommand dispatch: `serve`, `migrate`, `import`, `user`, `seed`
- `cmd/deepcuts/cmd_serve.go`, `cmd_import.go`, `cmd_user.go`, `cmd_seed.go` — one file per subcommand group
- `internal/config/config.go` — env parsing into `Config`
- `internal/db/db.go` — `Open(path)`, `Migrate(db)`, embedded migrations
- `internal/db/queries/` — sqlc output (generated)
- `internal/db/migrations/00001_init.sql` — the whole phase 1 schema (goose, embedded)
- `sql/queries/*.sql` — sqlc queries, one file per table group
- `internal/domain/money.go` — `Cents`, `Hundredths`, `Extended`
- `internal/domain/pricing.go` — `ResolvePrice`, `EstimateWeight`, `LineAmount`
- `internal/domain/states.go` — order/route/stop status types and transition tables
- `internal/domain/actions.go` — driver action payload types and validation
- `internal/service/errors.go` — `Error` type, constructors
- `internal/service/store.go` — `Store` (db + queries + tx helper + clock + location)
- `internal/service/catalog.go` — customers, products, customer prices
- `internal/service/orders.go` — orders and lines
- `internal/service/routes.go` — routes, stops, scheduling, day view
- `internal/service/driver.go` — driver route view, idempotent actions, finalize helpers
- `internal/storage/local.go` — proof file storage
- `internal/auth/auth.go` — users, password/PIN, sessions, rate limiter, middleware
- `internal/importer/importer.go` — CSV imports
- `internal/seed/demo.go` — demo data
- `internal/api/api.gen.go` — oapi-codegen output (generated)
- `internal/httpapi/server.go` — router assembly, SPA fallback, error handler
- `internal/httpapi/handlers_auth.go`, `handlers_catalog.go`, `handlers_orders.go`, `handlers_routes.go`, `handlers_driver.go` — `StrictServerInterface` implementation split by area
- `internal/httpapi/convert.go` — queries/domain → API types
- `web/embed.go` — `//go:embed all:dist`
- `api/openapi.yaml` — the contract

Frontend (`web/`):
- `package.json`, `vite.config.ts`, `tsconfig.json`, `index.html`
- `src/main.tsx`, `src/router.tsx`
- `src/api/schema.d.ts` (generated), `src/api/client.ts`
- `src/lib/format.ts` — cents/hundredths formatting
- `src/auth/OfficeAuth.tsx`, `src/auth/DriverAuth.tsx`
- `src/routes/office/Layout.tsx`, `Login.tsx`, `Customers.tsx`, `CustomerDetail.tsx`, `Products.tsx`, `ProductDetail.tsx`, `Orders.tsx`, `OrderDetail.tsx`, `Day.tsx`
- `src/routes/driver/Layout.tsx`, `Login.tsx`, `Route.tsx`, `Stop.tsx`, `Proof.tsx`
- `src/offline/db.ts` (IndexedDB), `src/offline/queue.ts` (sync queue), `src/offline/useRoute.ts`
- `public/manifest.webmanifest` (generated by vite-plugin-pwa config)

Tests:
- Go: `*_test.go` next to each package; `internal/service/testdb_test.go` shared helper
- Web: `src/**/*.test.tsx`, `src/offline/queue.test.ts`
- `scripts/e2e.sh` — build, migrate, seed, walk an order through the API with curl

---

### Task 1: Repository scaffold, config, CLI skeleton

**Files:**
- Create: `go.mod`, `Makefile`, `cmd/deepcuts/main.go`, `internal/config/config.go`, `internal/config/config_test.go`, `README.md`
- Modify: `.gitignore`

**Interfaces:**
- Produces: `config.Config{DBPath, DataDir, Addr, Timezone string, Location *time.Location, TLSCert, TLSKey string}` and `config.FromEnv(getenv func(string) string) (Config, error)`.
- Produces: `cmd/deepcuts` `main()` that dispatches on `os.Args[1]` to `runServe`, `runMigrate`, `runImport`, `runUser`, `runSeed` (later tasks fill these; this task stubs them to print "not implemented" and exit 2).

- [ ] **Step 1: Initialize the module and pin tools**

```bash
export PATH=/opt/homebrew/bin:$PATH
cd ~/deepcutsCRM
go mod init github.com/cmcgeedev/deepcutsCRM
go get -tool github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
go get -tool github.com/pressly/goose/v3/cmd/goose@v3.28.0
go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.8.0
go get modernc.org/sqlite@v1.58.0 github.com/go-chi/chi/v5@v5.3.2 golang.org/x/crypto@v0.57.0 github.com/google/uuid@latest
```

- [ ] **Step 2: Write the failing config test**

`internal/config/config_test.go`:
```go
package config

import "testing"

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestDefaults(t *testing.T) {
	c, err := FromEnv(env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if c.DBPath != "data/deepcuts.sqlite" || c.DataDir != "data" || c.Addr != ":8080" {
		t.Fatalf("bad defaults: %+v", c)
	}
	if c.Timezone != "America/New_York" || c.Location == nil {
		t.Fatalf("bad tz: %+v", c)
	}
}

func TestOverridesAndBadTimezone(t *testing.T) {
	c, err := FromEnv(env(map[string]string{
		"DEEPCUTS_DB_PATH": "/tmp/x.sqlite", "DEEPCUTS_ADDR": ":9999",
		"DEEPCUTS_TIMEZONE": "America/Chicago", "DEEPCUTS_TLS_CERT": "c.pem", "DEEPCUTS_TLS_KEY": "k.pem",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if c.DBPath != "/tmp/x.sqlite" || c.Addr != ":9999" || c.Location.String() != "America/Chicago" || c.TLSCert != "c.pem" {
		t.Fatalf("overrides not applied: %+v", c)
	}
	if _, err := FromEnv(env(map[string]string{"DEEPCUTS_TIMEZONE": "Mars/Olympus"})); err == nil {
		t.Fatal("expected error for bad timezone")
	}
	if _, err := FromEnv(env(map[string]string{"DEEPCUTS_TLS_CERT": "c.pem"})); err == nil {
		t.Fatal("expected error when only one of cert/key is set")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/config/ 2>&1 | head -5`
Expected: build failure, `FromEnv` undefined.

- [ ] **Step 4: Implement config**

`internal/config/config.go`:
```go
// Package config reads DEEPCUTS_* environment variables into a Config.
package config

import (
	"fmt"
	"time"
)

type Config struct {
	DBPath   string
	DataDir  string
	Addr     string
	Timezone string
	Location *time.Location
	TLSCert  string
	TLSKey   string
}

func FromEnv(getenv func(string) string) (Config, error) {
	get := func(k, def string) string {
		if v := getenv(k); v != "" {
			return v
		}
		return def
	}
	c := Config{
		DBPath:   get("DEEPCUTS_DB_PATH", "data/deepcuts.sqlite"),
		DataDir:  get("DEEPCUTS_DATA_DIR", "data"),
		Addr:     get("DEEPCUTS_ADDR", ":8080"),
		Timezone: get("DEEPCUTS_TIMEZONE", "America/New_York"),
		TLSCert:  getenv("DEEPCUTS_TLS_CERT"),
		TLSKey:   getenv("DEEPCUTS_TLS_KEY"),
	}
	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		return Config{}, fmt.Errorf("DEEPCUTS_TIMEZONE %q: %w", c.Timezone, err)
	}
	c.Location = loc
	if (c.TLSCert == "") != (c.TLSKey == "") {
		return Config{}, fmt.Errorf("DEEPCUTS_TLS_CERT and DEEPCUTS_TLS_KEY must be set together")
	}
	return c, nil
}

// TLS reports whether the server should listen with TLS.
func (c Config) TLS() bool { return c.TLSCert != "" }
```

- [ ] **Step 5: Run test to verify it passes**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/config/`
Expected: `ok`

- [ ] **Step 6: Write the CLI skeleton**

`cmd/deepcuts/main.go`:
```go
// Command deepcuts is the single binary: API server, migrations, imports, users, demo seed.
package main

import (
	"fmt"
	"os"

	"github.com/cmcgeedev/deepcutsCRM/internal/config"
)

const usage = `usage: deepcuts <command> [args]

commands:
  serve                       run the API server and web app
  migrate                     apply database migrations and exit
  import customers <file.csv> import customers from CSV
  import products <file.csv>  import products from CSV
  import prices <file.csv>    import per-customer prices from CSV
  user add-office <email> <display name>   create an office user (prompts for password)
  user add-driver <display name>           create a driver (prompts for 6 digit PIN)
  user set-pin <display name>              reset a driver PIN
  user deactivate <email or display name>  deactivate a user
  seed demo                   load demo data into an empty database
`

type command func(cfg config.Config, args []string) error

var commands = map[string]command{
	"serve":   runServe,
	"migrate": runMigrate,
	"import":  runImport,
	"user":    runUser,
	"seed":    runSeed,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd, ok := commands[os.Args[1]]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown command %q\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	cfg, err := config.FromEnv(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	if err := cmd(cfg, os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func notImplemented(name string) command {
	return func(config.Config, []string) error { return fmt.Errorf("%s: not implemented yet", name) }
}

var (
	runServe   = notImplemented("serve")
	runMigrate = notImplemented("migrate")
	runImport  = notImplemented("import")
	runUser    = notImplemented("user")
	runSeed    = notImplemented("seed")
)
```

Later tasks replace each `notImplemented` var with a real function defined in its own file (`cmd_serve.go` etc.) by deleting the corresponding line from this `var` block and defining `func runServe(cfg config.Config, args []string) error` there.

- [ ] **Step 7: Makefile, gitignore, README**

`Makefile`:
```make
export PATH := /opt/homebrew/bin:$(PATH)
BIN := bin/deepcuts

.PHONY: all generate build build-web test test-go test-web demo demo-tls clean e2e check-generated

all: build

generate:
	go tool sqlc generate
	go tool oapi-codegen -config oapi-codegen.yaml api/openapi.yaml
	cd web && npm run generate

check-generated: generate
	git diff --exit-code -- internal/db/queries internal/api web/src/api/schema.d.ts

build-web:
	cd web && npm ci --no-audit --no-fund && npm run build

build: build-web
	go build -o $(BIN) ./cmd/deepcuts

build-linux: build-web
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bin/deepcuts-linux-arm64 ./cmd/deepcuts

test-go:
	go vet ./... && go test ./...

test-web:
	cd web && npm test -- --run

test: test-go test-web

demo: build
	rm -f data/demo.sqlite data/demo.sqlite-wal data/demo.sqlite-shm
	DEEPCUTS_DB_PATH=data/demo.sqlite $(BIN) seed demo
	DEEPCUTS_DB_PATH=data/demo.sqlite $(BIN) serve

demo-tls: build
	./scripts/demo-tls.sh

e2e: build
	./scripts/e2e.sh

clean:
	rm -rf bin web/dist
```

Append to `.gitignore`:
```
web/dist/*
!web/dist/.gitkeep
certs/
```
and remove the earlier `web/dist/` line so the `.gitkeep` exception works.

`README.md`:
```markdown
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
```

- [ ] **Step 8: Verify build and commit**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go build ./... && go run ./cmd/deepcuts 2>&1 | head -3; git add -A && git commit -q -m "Scaffold module, config, CLI skeleton, Makefile" && git push -u origin main`
Expected: build ok; running without args prints usage; commit and push succeed.

---

### Task 2: Schema, migrations, sqlc queries for users, customers, products, prices

**Files:**
- Create: `internal/db/migrations/00001_init.sql`, `sqlc.yaml`, `internal/db/db.go`, `internal/db/db_test.go`, `sql/queries/users.sql`, `sql/queries/customers.sql`, `sql/queries/products.sql`, `sql/queries/prices.sql`, `cmd/deepcuts/cmd_migrate.go`
- Generated: `internal/db/queries/*.go`
- Modify: `cmd/deepcuts/main.go` (remove `runMigrate` stub line)

**Interfaces:**
- Produces: `db.Open(path string) (*sql.DB, error)` opening with WAL, busy_timeout 5000, foreign keys on; `db.Migrate(ctx, *sql.DB) error`; `db.OpenAndMigrate(ctx, path) (*sql.DB, error)`.
- Produces: `queries.New(dbtx) *Queries` (sqlc) with the query names listed in Step 4. Field types follow the sqlc sqlite mapping: BOOLEAN→bool, DATETIME→time.Time, nullable→sql.Null*.

- [ ] **Step 1: Write the schema**

`internal/db/migrations/00001_init.sql` (goose reads it from the embedded FS; sqlc reads it as the schema):
```sql
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
  approx_case_weight INTEGER,
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
  ordered_qty INTEGER NOT NULL,
  unit_price_cents INTEGER NOT NULL,
  price_overridden BOOLEAN NOT NULL DEFAULT FALSE,
  est_weight INTEGER,
  shipped_weight INTEGER,
  delivered_qty INTEGER,
  delivered_weight INTEGER,
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
```

`sqlc.yaml`:
```yaml
version: "2"
sql:
  - engine: "sqlite"
    schema: "internal/db/migrations"
    queries: "sql/queries"
    gen:
      go:
        package: "queries"
        out: "internal/db/queries"
```

- [ ] **Step 2: Write the failing db test**

`internal/db/db_test.go`:
```go
package db

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenAndMigrate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.sqlite")
	d, err := OpenAndMigrate(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	var mode string
	if err := d.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		t.Fatalf("journal_mode=%q err=%v", mode, err)
	}
	var fk int
	if err := d.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil || fk != 1 {
		t.Fatalf("foreign_keys=%d err=%v", fk, err)
	}
	var n int
	if err := d.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name IN ('customers','users','sessions','products','customer_prices','orders','order_lines','delivery_routes','delivery_stops','driver_actions')").Scan(&n); err != nil || n != 10 {
		t.Fatalf("tables=%d err=%v", n, err)
	}
	// migrate again is a no-op
	if err := Migrate(context.Background(), d); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/db/ 2>&1 | head -3`
Expected: build failure, `OpenAndMigrate` undefined.

- [ ] **Step 4: Implement db.go**

`internal/db/db.go`:
```go
// Package db opens the SQLite database and applies embedded goose migrations.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open opens (creating if needed) the SQLite file with WAL, a 5 s busy timeout and foreign keys on.
func Open(path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)", path)
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	d.SetMaxOpenConns(1) // SQLite serializes writes; one connection avoids SQLITE_BUSY under load
	if err := d.Ping(); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

// Migrate applies all pending migrations.
func Migrate(ctx context.Context, d *sql.DB) error {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	goose.SetBaseFS(sub)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.UpContext(ctx, d, ".")
}

func OpenAndMigrate(ctx context.Context, path string) (*sql.DB, error) {
	d, err := Open(path)
	if err != nil {
		return nil, err
	}
	if err := Migrate(ctx, d); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}
```

`go:embed` cannot reach outside the package directory, which is why migrations live under `internal/db/migrations/` rather than the spec's `sql/schema/`. Query files stay in `sql/queries/`.

Note on `SetMaxOpenConns(1)`: with a single connection, a transaction that opens a second query on the same `*sql.DB` deadlocks. Every service method must run its reads and writes on the `*sql.Tx` it opened (via `queries.WithTx`), never on the bare `*sql.DB` inside a transaction. Task 5's `Store.Tx` helper enforces this by only handing the transaction-bound `*Queries` to callbacks.

- [ ] **Step 5: Run test to verify it passes**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/db/`
Expected: `ok`

- [ ] **Step 6: Write the first sqlc query files**

`sql/queries/users.sql`:
```sql
-- name: CreateUser :one
INSERT INTO users (realm, display_name, email, password_hash, pin_hash, customer_id, active, created_at)
VALUES (?, ?, ?, ?, ?, ?, TRUE, ?) RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ? AND realm = 'office';

-- name: GetDriverByName :one
SELECT * FROM users WHERE display_name = ? AND realm = 'driver';

-- name: ListUsersByRealm :many
SELECT * FROM users WHERE realm = ? AND active = TRUE ORDER BY display_name;

-- name: SetUserPinHash :exec
UPDATE users SET pin_hash = ? WHERE id = ?;

-- name: SetUserActive :exec
UPDATE users SET active = ? WHERE id = ?;

-- name: CreateSession :exec
INSERT INTO sessions (id, user_id, realm, expires_at) VALUES (?, ?, ?, ?);

-- name: GetSession :one
SELECT s.id, s.user_id, s.realm, s.expires_at, u.display_name, u.active
FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.id = ?;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < ?;
```

`sql/queries/customers.sql`:
```sql
-- name: CreateCustomer :one
INSERT INTO customers (name, billing_address, delivery_address, contact_name, phone, email,
  delivery_notes, delivery_days, qbo_customer_id, active, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, TRUE, ?, ?) RETURNING *;

-- name: UpdateCustomer :one
UPDATE customers SET name = ?, billing_address = ?, delivery_address = ?, contact_name = ?,
  phone = ?, email = ?, delivery_notes = ?, delivery_days = ?, qbo_customer_id = ?, active = ?, updated_at = ?
WHERE id = ? RETURNING *;

-- name: GetCustomer :one
SELECT * FROM customers WHERE id = ?;

-- name: GetCustomerByQBO :one
SELECT * FROM customers WHERE qbo_customer_id = ?;

-- name: GetCustomerByName :one
SELECT * FROM customers WHERE name = ?;

-- name: ListCustomers :many
SELECT * FROM customers WHERE (sqlc.arg(include_inactive) = TRUE OR active = TRUE) ORDER BY name;
```

`sql/queries/products.sql`:
```sql
-- name: CreateProduct :one
INSERT INTO products (sku, name, category, sell_unit, catch_weight, approx_case_weight,
  base_price_cents, qbo_item_id, active, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, TRUE, ?, ?) RETURNING *;

-- name: UpdateProduct :one
UPDATE products SET sku = ?, name = ?, category = ?, sell_unit = ?, catch_weight = ?,
  approx_case_weight = ?, base_price_cents = ?, qbo_item_id = ?, active = ?, updated_at = ?
WHERE id = ? RETURNING *;

-- name: GetProduct :one
SELECT * FROM products WHERE id = ?;

-- name: GetProductBySKU :one
SELECT * FROM products WHERE sku = ?;

-- name: ListProducts :many
SELECT * FROM products WHERE (sqlc.arg(include_inactive) = TRUE OR active = TRUE) ORDER BY category, name;
```

`sql/queries/prices.sql`:
```sql
-- name: CreateCustomerPrice :one
INSERT INTO customer_prices (customer_id, product_id, price_cents, effective_from, created_at)
VALUES (?, ?, ?, ?, ?) RETURNING *;

-- name: ListCustomerPriceRows :many
SELECT * FROM customer_prices WHERE customer_id = ? AND product_id = ? ORDER BY effective_from DESC, id DESC;

-- name: ListCurrentCustomerPrices :many
SELECT cp.*, p.sku, p.name AS product_name
FROM customer_prices cp
JOIN products p ON p.id = cp.product_id
WHERE cp.customer_id = ?
  AND cp.id = (
    SELECT c2.id FROM customer_prices c2
    WHERE c2.customer_id = cp.customer_id AND c2.product_id = cp.product_id AND c2.effective_from <= sqlc.arg(as_of)
    ORDER BY c2.effective_from DESC, c2.id DESC LIMIT 1)
ORDER BY p.name;
```

- [ ] **Step 7: Generate and verify it compiles**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go tool sqlc generate && go build ./... && ls internal/db/queries/`
Expected: `customers.sql.go db.go models.go prices.sql.go products.sql.go users.sql.go`; build ok. sqlc may type `include_inactive` as `interface{}` because it is only compared to `TRUE`; passing a Go `bool` still works. Named args that are also compared to a column (`date`, `status`, `customer_id`, `as_of`) get that column's type.

- [ ] **Step 8: Wire the migrate subcommand**

`cmd/deepcuts/cmd_migrate.go`:
```go
package main

import (
	"context"
	"fmt"

	"github.com/cmcgeedev/deepcutsCRM/internal/config"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
)

func runMigrate(cfg config.Config, args []string) error {
	d, err := db.OpenAndMigrate(context.Background(), cfg.DBPath)
	if err != nil {
		return err
	}
	defer d.Close()
	fmt.Println("migrated", cfg.DBPath)
	return nil
}
```
Delete the `runMigrate = notImplemented("migrate")` line in `cmd/deepcuts/main.go`.

- [ ] **Step 9: Verify and commit**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go build ./... && DEEPCUTS_DB_PATH=$(mktemp -d)/t.sqlite go run ./cmd/deepcuts migrate && go test ./... && git add -A && git commit -q -m "Add schema, migrations, sqlc queries for catalog and users" && git push`
Expected: `migrated ...`, tests ok, pushed.

---

### Task 3: Domain: money, weights, pricing, line amounts

**Files:**
- Create: `internal/domain/money.go`, `internal/domain/pricing.go`, `internal/domain/money_test.go`, `internal/domain/pricing_test.go`

**Interfaces:**
- Produces:
  - `type Cents int64`, `type Hundredths int64`
  - `func Extended(qty Hundredths, unitPrice Cents) Cents` — half-up rounding
  - `type Unit string` with `UnitLb`, `UnitCase`, `UnitEach`; `func ParseUnit(string) (Unit, bool)`
  - `type Product struct { ID int64; SKU, Name string; SellUnit Unit; CatchWeight bool; ApproxCaseWeight Hundredths; BasePrice Cents }`
  - `func (p Product) Validate() map[string]string` — field errors (empty when valid)
  - `type PriceRow struct { Price Cents; EffectiveFrom string }`
  - `func ResolvePrice(base Cents, rows []PriceRow, orderDate string) (price Cents, fromCustomer bool)`
  - `func EstimateWeight(p Product, orderedQty Hundredths) (Hundredths, bool)` — false when not catch-weight
  - `type Line struct { Product Product; OrderedQty Hundredths; UnitPrice Cents; EstWeight, ShippedWeight, DeliveredQty, DeliveredWeight *Hundredths }`
  - `func LineAmount(l Line) Cents`
  - `func (l Line) BillableQty() (Hundredths, string)` — the quantity used and which source (`delivered`, `shipped`, `estimated`, `ordered`)

- [ ] **Step 1: Write the failing tests**

`internal/domain/money_test.go`:
```go
package domain

import "testing"

func TestExtendedRoundsHalfUp(t *testing.T) {
	cases := []struct {
		qty   Hundredths
		price Cents
		want  Cents
	}{
		{100, 1000, 1000},  // 1.00 × $10.00
		{250, 1000, 2500},  // 2.50 × $10.00
		{333, 1000, 3330},  // 3.33 × $10.00
		{1, 1, 0},          // 0.01 × $0.01 = 0.0001 → 0
		{50, 1, 1},         // 0.50 × $0.01 = 0.005 → rounds up to 1
		{49, 1, 0},         // 0.49 × $0.01 = 0.0049 → 0
		{1234, 799, 9860},  // 12.34 × $7.99 = 98.5966 → 98.60
		{0, 999, 0},
	}
	for _, c := range cases {
		if got := Extended(c.qty, c.price); got != c.want {
			t.Errorf("Extended(%d,%d)=%d want %d", c.qty, c.price, got, c.want)
		}
	}
}

func TestParseUnit(t *testing.T) {
	for _, s := range []string{"lb", "case", "each"} {
		if _, ok := ParseUnit(s); !ok {
			t.Errorf("%q should parse", s)
		}
	}
	if _, ok := ParseUnit("kg"); ok {
		t.Error("kg should not parse")
	}
}

func TestProductValidate(t *testing.T) {
	good := Product{SKU: "BRIS-01", Name: "Brisket", SellUnit: UnitCase, CatchWeight: true, ApproxCaseWeight: 6000, BasePrice: 599}
	if errs := good.Validate(); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	bad := Product{SKU: "", Name: "", SellUnit: "kg", CatchWeight: true, ApproxCaseWeight: 0, BasePrice: -1}
	errs := bad.Validate()
	for _, f := range []string{"sku", "name", "sellUnit", "approxCaseWeight", "basePriceCents"} {
		if errs[f] == "" {
			t.Errorf("expected error for %s: %v", f, errs)
		}
	}
	cwLb := Product{SKU: "X", Name: "X", SellUnit: UnitLb, CatchWeight: true, ApproxCaseWeight: 100, BasePrice: 1}
	if cwLb.Validate()["sellUnit"] == "" {
		t.Error("catch-weight products must be sold by the case")
	}
}
```

`internal/domain/pricing_test.go`:
```go
package domain

import "testing"

func TestResolvePrice(t *testing.T) {
	rows := []PriceRow{
		{Price: 500, EffectiveFrom: "2026-01-01"},
		{Price: 550, EffectiveFrom: "2026-06-01"},
		{Price: 600, EffectiveFrom: "2026-12-01"},
	}
	if p, ok := ResolvePrice(700, nil, "2026-09-10"); p != 700 || ok {
		t.Errorf("no rows: got %d %v", p, ok)
	}
	if p, ok := ResolvePrice(700, rows, "2025-12-31"); p != 700 || ok {
		t.Errorf("before all rows: got %d %v", p, ok)
	}
	if p, ok := ResolvePrice(700, rows, "2026-06-01"); p != 550 || !ok {
		t.Errorf("on effective date: got %d %v", p, ok)
	}
	if p, ok := ResolvePrice(700, rows, "2026-09-10"); p != 550 || !ok {
		t.Errorf("between: got %d %v", p, ok)
	}
	if p, _ := ResolvePrice(700, rows, "2027-01-01"); p != 600 {
		t.Errorf("latest: got %d", p)
	}
}

func TestEstimateWeight(t *testing.T) {
	cw := Product{SellUnit: UnitCase, CatchWeight: true, ApproxCaseWeight: 6000} // 60 lb cases
	if w, ok := EstimateWeight(cw, 300); !ok || w != 18000 {
		t.Errorf("3 cases × 60 lb: got %d %v", w, ok)
	}
	if w, ok := EstimateWeight(cw, 150); !ok || w != 9000 {
		t.Errorf("1.5 cases: got %d %v", w, ok)
	}
	if _, ok := EstimateWeight(Product{SellUnit: UnitEach}, 100); ok {
		t.Error("non catch-weight must return false")
	}
}

func h(v int64) *Hundredths { x := Hundredths(v); return &x }

func TestLineAmountCatchWeight(t *testing.T) {
	p := Product{SellUnit: UnitCase, CatchWeight: true, ApproxCaseWeight: 6000, BasePrice: 599}
	l := Line{Product: p, OrderedQty: 200, UnitPrice: 599, EstWeight: h(12000)}
	if got := LineAmount(l); got != 71880 { // 120.00 lb × $5.99
		t.Errorf("estimated: %d", got)
	}
	if _, src := l.BillableQty(); src != "estimated" {
		t.Errorf("src=%s", src)
	}
	l.ShippedWeight = h(11850)
	if got := LineAmount(l); got != 70982 { // 118.50 × 5.99 = 709.815 → 709.82
		t.Errorf("shipped: %d", got)
	}
	l.DeliveredWeight = h(6000)
	if got := LineAmount(l); got != 35940 {
		t.Errorf("delivered: %d", got)
	}
	if _, src := l.BillableQty(); src != "delivered" {
		t.Errorf("src=%s", src)
	}
}

func TestLineAmountUnitPriced(t *testing.T) {
	p := Product{SellUnit: UnitEach, BasePrice: 1250}
	l := Line{Product: p, OrderedQty: 400, UnitPrice: 1250}
	if got := LineAmount(l); got != 5000 {
		t.Errorf("ordered: %d", got)
	}
	l.DeliveredQty = h(300)
	if got := LineAmount(l); got != 3750 {
		t.Errorf("delivered: %d", got)
	}
	l.DeliveredQty = h(0)
	if got := LineAmount(l); got != 0 {
		t.Errorf("rejected line must be zero, got %d", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/domain/ 2>&1 | head -3`
Expected: build failure (undefined types).

- [ ] **Step 3: Implement money.go and pricing.go**

`internal/domain/money.go`:
```go
// Package domain holds business rules with no database or HTTP dependencies.
package domain

// Cents is a money amount in integer cents.
type Cents int64

// Hundredths is a quantity or weight in hundredths of a unit (1.00 lb == 100).
type Hundredths int64

// Extended multiplies a quantity in hundredths by a unit price in cents and rounds half-up to cents.
func Extended(qty Hundredths, unitPrice Cents) Cents {
	n := int64(qty) * int64(unitPrice)
	if n < 0 {
		return Cents(-((-n + 50) / 100))
	}
	return Cents((n + 50) / 100)
}

type Unit string

const (
	UnitLb   Unit = "lb"
	UnitCase Unit = "case"
	UnitEach Unit = "each"
)

func ParseUnit(s string) (Unit, bool) {
	switch Unit(s) {
	case UnitLb, UnitCase, UnitEach:
		return Unit(s), true
	}
	return "", false
}

type Product struct {
	ID               int64
	SKU              string
	Name             string
	SellUnit         Unit
	CatchWeight      bool
	ApproxCaseWeight Hundredths
	BasePrice        Cents
}

// Validate returns field-name → message for every invalid field; empty when valid.
func (p Product) Validate() map[string]string {
	errs := map[string]string{}
	if p.SKU == "" {
		errs["sku"] = "required"
	}
	if p.Name == "" {
		errs["name"] = "required"
	}
	if _, ok := ParseUnit(string(p.SellUnit)); !ok {
		errs["sellUnit"] = "must be lb, case or each"
	} else if p.CatchWeight && p.SellUnit != UnitCase {
		errs["sellUnit"] = "catch-weight products are sold by the case"
	}
	if p.CatchWeight && p.ApproxCaseWeight <= 0 {
		errs["approxCaseWeight"] = "required for catch-weight products"
	}
	if p.BasePrice < 0 {
		errs["basePriceCents"] = "must not be negative"
	}
	return errs
}
```

`internal/domain/pricing.go`:
```go
package domain

// PriceRow is one negotiated customer price with the date it takes effect (YYYY-MM-DD).
type PriceRow struct {
	Price         Cents
	EffectiveFrom string
}

// ResolvePrice picks the customer price with the latest EffectiveFrom on or before orderDate,
// falling back to base. Dates compare lexically because they are YYYY-MM-DD.
func ResolvePrice(base Cents, rows []PriceRow, orderDate string) (Cents, bool) {
	best := -1
	for i, r := range rows {
		if r.EffectiveFrom > orderDate {
			continue
		}
		if best == -1 || r.EffectiveFrom > rows[best].EffectiveFrom {
			best = i
		}
	}
	if best == -1 {
		return base, false
	}
	return rows[best].Price, true
}

// EstimateWeight returns ordered cases × approximate case weight for catch-weight products.
func EstimateWeight(p Product, orderedQty Hundredths) (Hundredths, bool) {
	if !p.CatchWeight {
		return 0, false
	}
	return Hundredths(int64(orderedQty) * int64(p.ApproxCaseWeight) / 100), true
}

type Line struct {
	Product         Product
	OrderedQty      Hundredths
	UnitPrice       Cents
	EstWeight       *Hundredths
	ShippedWeight   *Hundredths
	DeliveredQty    *Hundredths
	DeliveredWeight *Hundredths
}

// BillableQty returns the quantity the amount is computed from and where it came from.
// Catch-weight: delivered weight, else shipped, else estimated. Otherwise: delivered qty, else ordered.
func (l Line) BillableQty() (Hundredths, string) {
	if l.Product.CatchWeight {
		switch {
		case l.DeliveredWeight != nil:
			return *l.DeliveredWeight, "delivered"
		case l.ShippedWeight != nil:
			return *l.ShippedWeight, "shipped"
		case l.EstWeight != nil:
			return *l.EstWeight, "estimated"
		default:
			return 0, "estimated"
		}
	}
	if l.DeliveredQty != nil {
		return *l.DeliveredQty, "delivered"
	}
	return l.OrderedQty, "ordered"
}

func LineAmount(l Line) Cents {
	q, _ := l.BillableQty()
	return Extended(q, l.UnitPrice)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/domain/ -v 2>&1 | tail -3`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
cd ~/deepcutsCRM && git add internal/domain && git commit -q -m "Add domain money, pricing and line amount rules" && git push
```

---

### Task 4: Domain: state machines and driver action payloads

**Files:**
- Create: `internal/domain/states.go`, `internal/domain/states_test.go`, `internal/domain/actions.go`, `internal/domain/actions_test.go`

**Interfaces:**
- Produces:
  - `type OrderStatus string` consts `OrderDraft, OrderConfirmed, OrderScheduled, OrderDelivered, OrderFinalized, OrderCancelled`; `func (s OrderStatus) CanTransition(to OrderStatus) bool`
  - `type RouteStatus string` consts `RoutePlanned, RouteOut, RouteComplete`; `func (s RouteStatus) CanTransition(to RouteStatus) bool`
  - `type StopStatus string` consts `StopPending, StopDelivered, StopSkipped`
  - `type LineEditScope int` consts `EditNone, EditShippedAndPrice, EditAll, EditDelivered`; `func LineEditScopeFor(order OrderStatus, route RouteStatus) LineEditScope`
  - `type ActionType string` consts `ActionDeliver, ActionAdjust, ActionSkip`
  - `type ProofType string` consts `ProofSignature, ProofPhoto, ProofName`
  - `type LineAdjustment struct { LineID int64; DeliveredQty *Hundredths; DeliveredWeight *Hundredths; ShortageNote *string }`
  - `type DriverAction struct { ClientID string; Type ActionType; Proof *Proof; Lines []LineAdjustment; Note string; SkipReason string }`
  - `type Proof struct { Type ProofType; Data []byte; Name string }`
  - `func (a DriverAction) Validate() map[string]string`
  - `const MaxProofBytes = 300 * 1024`

- [ ] **Step 1: Write the failing tests**

`internal/domain/states_test.go`:
```go
package domain

import "testing"

func TestOrderTransitions(t *testing.T) {
	allowed := map[OrderStatus][]OrderStatus{
		OrderDraft:     {OrderConfirmed, OrderCancelled},
		OrderConfirmed: {OrderScheduled, OrderCancelled, OrderDraft},
		OrderScheduled: {OrderConfirmed, OrderDelivered, OrderCancelled},
		OrderDelivered: {OrderFinalized},
		OrderFinalized: {},
		OrderCancelled: {},
	}
	all := []OrderStatus{OrderDraft, OrderConfirmed, OrderScheduled, OrderDelivered, OrderFinalized, OrderCancelled}
	for from, tos := range allowed {
		ok := map[OrderStatus]bool{}
		for _, to := range tos {
			ok[to] = true
		}
		for _, to := range all {
			if got := from.CanTransition(to); got != ok[to] {
				t.Errorf("%s→%s: got %v want %v", from, to, got, ok[to])
			}
		}
	}
}

func TestRouteTransitions(t *testing.T) {
	if !RoutePlanned.CanTransition(RouteOut) || !RouteOut.CanTransition(RouteComplete) {
		t.Error("forward transitions must be allowed")
	}
	if RoutePlanned.CanTransition(RouteComplete) || RouteOut.CanTransition(RoutePlanned) || RouteComplete.CanTransition(RouteOut) {
		t.Error("skips and reversals must be rejected")
	}
}

func TestLineEditScope(t *testing.T) {
	cases := []struct {
		o    OrderStatus
		r    RouteStatus
		want LineEditScope
	}{
		{OrderDraft, "", EditAll},
		{OrderConfirmed, "", EditAll},
		{OrderScheduled, RoutePlanned, EditAll},
		{OrderScheduled, RouteOut, EditShippedAndPrice},
		{OrderDelivered, RouteOut, EditDelivered},
		{OrderDelivered, RouteComplete, EditDelivered},
		{OrderFinalized, RouteComplete, EditNone},
		{OrderCancelled, "", EditNone},
	}
	for _, c := range cases {
		if got := LineEditScopeFor(c.o, c.r); got != c.want {
			t.Errorf("%s/%s: got %v want %v", c.o, c.r, got, c.want)
		}
	}
}
```

`internal/domain/actions_test.go`:
```go
package domain

import (
	"strings"
	"testing"
)

const uuid1 = "6f1c2f1e-5b7a-4c1e-9c3a-1d2e3f4a5b6c"

func TestDriverActionValidate(t *testing.T) {
	ok := DriverAction{ClientID: uuid1, Type: ActionDeliver, Proof: &Proof{Type: ProofName, Name: "Pat"}}
	if errs := ok.Validate(); len(errs) != 0 {
		t.Fatalf("unexpected: %v", errs)
	}
	if errs := (DriverAction{ClientID: "nope", Type: ActionDeliver, Proof: &Proof{Type: ProofName, Name: "P"}}).Validate(); errs["clientId"] == "" {
		t.Error("bad uuid must fail")
	}
	if errs := (DriverAction{ClientID: uuid1, Type: "fly"}).Validate(); errs["type"] == "" {
		t.Error("bad type must fail")
	}
	if errs := (DriverAction{ClientID: uuid1, Type: ActionDeliver}).Validate(); errs["proof"] == "" {
		t.Error("deliver without proof must fail")
	}
	if errs := (DriverAction{ClientID: uuid1, Type: ActionDeliver, Proof: &Proof{Type: ProofSignature}}).Validate(); errs["proof"] == "" {
		t.Error("signature without data must fail")
	}
	big := &Proof{Type: ProofPhoto, Data: []byte(strings.Repeat("x", MaxProofBytes+1))}
	if errs := (DriverAction{ClientID: uuid1, Type: ActionDeliver, Proof: big}).Validate(); errs["proof"] == "" {
		t.Error("oversized proof must fail")
	}
	if errs := (DriverAction{ClientID: uuid1, Type: ActionSkip}).Validate(); errs["skipReason"] == "" {
		t.Error("skip without reason must fail")
	}
	if errs := (DriverAction{ClientID: uuid1, Type: ActionAdjust}).Validate(); errs["lines"] == "" {
		t.Error("adjust without lines must fail")
	}
	neg := Hundredths(-1)
	if errs := (DriverAction{ClientID: uuid1, Type: ActionAdjust, Lines: []LineAdjustment{{LineID: 1, DeliveredQty: &neg}}}).Validate(); errs["lines"] == "" {
		t.Error("negative delivered qty must fail")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/domain/ 2>&1 | head -3`
Expected: build failure.

- [ ] **Step 3: Implement states.go and actions.go**

`internal/domain/states.go`:
```go
package domain

type OrderStatus string

const (
	OrderDraft     OrderStatus = "draft"
	OrderConfirmed OrderStatus = "confirmed"
	OrderScheduled OrderStatus = "scheduled"
	OrderDelivered OrderStatus = "delivered"
	OrderFinalized OrderStatus = "finalized"
	OrderCancelled OrderStatus = "cancelled"
)

var orderTransitions = map[OrderStatus]map[OrderStatus]bool{
	OrderDraft:     {OrderConfirmed: true, OrderCancelled: true},
	OrderConfirmed: {OrderScheduled: true, OrderCancelled: true, OrderDraft: true},
	OrderScheduled: {OrderConfirmed: true, OrderDelivered: true, OrderCancelled: true},
	OrderDelivered: {OrderFinalized: true},
}

func (s OrderStatus) CanTransition(to OrderStatus) bool { return orderTransitions[s][to] }

type RouteStatus string

const (
	RoutePlanned  RouteStatus = "planned"
	RouteOut      RouteStatus = "out"
	RouteComplete RouteStatus = "complete"
)

func (s RouteStatus) CanTransition(to RouteStatus) bool {
	return (s == RoutePlanned && to == RouteOut) || (s == RouteOut && to == RouteComplete)
}

type StopStatus string

const (
	StopPending   StopStatus = "pending"
	StopDelivered StopStatus = "delivered"
	StopSkipped   StopStatus = "skipped"
)

// LineEditScope says which order-line fields the office may change.
type LineEditScope int

const (
	EditNone            LineEditScope = iota // finalized or cancelled
	EditShippedAndPrice                      // route is out: shipped weight and price override only
	EditAll                                  // draft, confirmed, or scheduled on a planned route
	EditDelivered                            // delivered, awaiting finalize: delivered qty/weight and notes
)

// LineEditScopeFor takes the order status and the status of its active route ("" when none).
func LineEditScopeFor(order OrderStatus, route RouteStatus) LineEditScope {
	switch order {
	case OrderDraft, OrderConfirmed:
		return EditAll
	case OrderScheduled:
		if route == RouteOut {
			return EditShippedAndPrice
		}
		return EditAll
	case OrderDelivered:
		return EditDelivered
	}
	return EditNone
}
```

`internal/domain/actions.go`:
```go
package domain

import "regexp"

type ActionType string

const (
	ActionDeliver ActionType = "deliver"
	ActionAdjust  ActionType = "adjust"
	ActionSkip    ActionType = "skip"
)

type ProofType string

const (
	ProofSignature ProofType = "signature"
	ProofPhoto     ProofType = "photo"
	ProofName      ProofType = "name"
)

// MaxProofBytes caps a decoded signature or photo so the offline queue stays small.
const MaxProofBytes = 300 * 1024

type Proof struct {
	Type ProofType
	Data []byte // PNG or JPEG bytes for signature/photo
	Name string // received-by name for ProofName
}

type LineAdjustment struct {
	LineID          int64
	DeliveredQty    *Hundredths
	DeliveredWeight *Hundredths
	ShortageNote    *string
}

type DriverAction struct {
	ClientID   string
	Type       ActionType
	Proof      *Proof
	Lines      []LineAdjustment
	Note       string
	SkipReason string
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (a DriverAction) Validate() map[string]string {
	errs := map[string]string{}
	if !uuidRe.MatchString(a.ClientID) {
		errs["clientId"] = "must be a UUID"
	}
	switch a.Type {
	case ActionDeliver:
		if a.Proof == nil {
			errs["proof"] = "required to deliver"
		} else if msg := a.Proof.validate(); msg != "" {
			errs["proof"] = msg
		}
	case ActionAdjust:
		if len(a.Lines) == 0 {
			errs["lines"] = "at least one line adjustment required"
		}
	case ActionSkip:
		if a.SkipReason == "" {
			errs["skipReason"] = "required to skip"
		}
	default:
		errs["type"] = "must be deliver, adjust or skip"
	}
	for _, l := range a.Lines {
		if (l.DeliveredQty != nil && *l.DeliveredQty < 0) || (l.DeliveredWeight != nil && *l.DeliveredWeight < 0) {
			errs["lines"] = "delivered quantities must not be negative"
		}
	}
	return errs
}

func (p Proof) validate() string {
	switch p.Type {
	case ProofName:
		if p.Name == "" {
			return "received-by name required"
		}
	case ProofSignature, ProofPhoto:
		if len(p.Data) == 0 {
			return "image data required"
		}
		if len(p.Data) > MaxProofBytes {
			return "image larger than 300 KB"
		}
	default:
		return "type must be signature, photo or name"
	}
	return ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/domain/`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
cd ~/deepcutsCRM && git add internal/domain && git commit -q -m "Add order/route state machines and driver action validation" && git push
```

---

### Task 5: Service foundation and catalog (customers, products, prices)

**Files:**
- Create: `internal/service/errors.go`, `internal/service/store.go`, `internal/service/catalog.go`, `internal/service/testdb_test.go`, `internal/service/catalog_test.go`

**Interfaces:**
- Produces:
  - `type Error struct { Status int; Code, Message string; Fields map[string]string }` implementing `error`; constructors `NotFound(what string)`, `Conflict(code, msg string)`, `Invalid(fields map[string]string)`, `Unauthorized(msg string)`; `func AsError(err error) (*Error, bool)`.
  - `type Service struct { DB *sql.DB; Q *queries.Queries; Now func() time.Time; Loc *time.Location }`; `func New(db *sql.DB, loc *time.Location) *Service`; `func (s *Service) Tx(ctx, fn func(q *queries.Queries) error) error`; `func (s *Service) Today() string`.
  - Helpers (unexported, used by every later service file): `nullStr(string) sql.NullString`, `strOf(sql.NullString) string`, `nullInt(*int64) sql.NullInt64`, `intPtr(sql.NullInt64) *int64`, `nullTime(*time.Time) sql.NullTime`, `hPtr(sql.NullInt64) *domain.Hundredths`, `productOf(queries.Product) domain.Product`.
  - `type CustomerInput struct { Name, BillingAddress, DeliveryAddress, ContactName, Phone, Email, DeliveryNotes string; DeliveryDays []string; QBOCustomerID string; Active bool }`
  - `CreateCustomer(ctx, CustomerInput) (queries.Customer, error)`, `UpdateCustomer(ctx, id int64, CustomerInput) (queries.Customer, error)`, `GetCustomer(ctx, id) (queries.Customer, error)`, `ListCustomers(ctx, includeInactive bool) ([]queries.Customer, error)`
  - `type ProductInput struct { SKU, Name, Category, SellUnit string; CatchWeight bool; ApproxCaseWeight *int64; BasePriceCents int64; QBOItemID string; Active bool }`
  - `CreateProduct`, `UpdateProduct`, `GetProduct`, `ListProducts(ctx, includeInactive)` mirroring customers.
  - `type PriceInput struct { ProductID int64; PriceCents int64; EffectiveFrom string }`; `SetCustomerPrice(ctx, customerID int64, PriceInput) (queries.CustomerPrice, error)`; `ListCustomerPrices(ctx, customerID int64, asOf string) ([]queries.ListCurrentCustomerPricesRow, error)`
  - `resolvePrice(ctx, q *queries.Queries, customerID, productID int64, base domain.Cents, date string) (domain.Cents, bool, error)`
  - `ParseDeliveryDays([]string) (string, error)` / `SplitDeliveryDays(string) []string`
  - `ValidDate(string) bool` — `YYYY-MM-DD` parse check.

- [ ] **Step 1: Write errors.go and store.go (no test of their own; exercised by every service test)**

`internal/service/errors.go`:
```go
// Package service implements transactional use cases over the domain rules and sqlc queries.
package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
)

type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

func NotFound(what string) *Error {
	return &Error{Status: http.StatusNotFound, Code: "not_found", Message: what + " not found"}
}
func Conflict(code, msg string) *Error {
	return &Error{Status: http.StatusConflict, Code: code, Message: msg}
}
func Invalid(fields map[string]string) *Error {
	return &Error{Status: http.StatusUnprocessableEntity, Code: "invalid", Message: "validation failed", Fields: fields}
}
func Unauthorized(msg string) *Error {
	return &Error{Status: http.StatusUnauthorized, Code: "unauthorized", Message: msg}
}

func AsError(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// notFoundIf converts sql.ErrNoRows into a 404 for the named thing.
func notFoundIf(err error, what string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return NotFound(what)
	}
	return err
}
```

`internal/service/store.go`:
```go
package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
)

type Service struct {
	DB  *sql.DB
	Q   *queries.Queries
	Now func() time.Time
	Loc *time.Location
}

func New(d *sql.DB, loc *time.Location) *Service {
	return &Service{DB: d, Q: queries.New(d), Now: time.Now, Loc: loc}
}

// Tx runs fn inside a transaction. All queries inside fn must go through q.
func (s *Service) Tx(ctx context.Context, fn func(q *queries.Queries) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(s.Q.WithTx(tx)); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (s *Service) now() time.Time { return s.Now().UTC() }

// Today is the business-local date.
func (s *Service) Today() string { return s.Now().In(s.Loc).Format("2006-01-02") }

func ValidDate(d string) bool {
	_, err := time.Parse("2006-01-02", d)
	return err == nil
}

var weekdays = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

func ParseDeliveryDays(days []string) (string, error) {
	var out []string
	for _, d := range days {
		d = strings.ToLower(strings.TrimSpace(d))
		ok := false
		for _, w := range weekdays {
			if w == d {
				ok = true
			}
		}
		if !ok {
			return "", fmt.Errorf("unknown weekday %q", d)
		}
		out = append(out, d)
	}
	return strings.Join(out, ","), nil
}

func SplitDeliveryDays(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, ",")
}

func nullStr(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }
func strOf(n sql.NullString) string   { return n.String }
func nullInt(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}
func intPtr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}
func hPtr(n sql.NullInt64) *domain.Hundredths {
	if !n.Valid {
		return nil
	}
	v := domain.Hundredths(n.Int64)
	return &v
}
func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func productOf(p queries.Product) domain.Product {
	return domain.Product{
		ID: p.ID, SKU: p.Sku, Name: p.Name, SellUnit: domain.Unit(p.SellUnit),
		CatchWeight: p.CatchWeight, ApproxCaseWeight: domain.Hundredths(p.ApproxCaseWeight.Int64),
		BasePrice: domain.Cents(p.BasePriceCents),
	}
}
```

- [ ] **Step 2: Write the shared test helper and the failing catalog test**

`internal/service/testdb_test.go`:
```go
package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/db"
)

// newTestService opens a fresh migrated SQLite file in a temp dir with a fixed clock.
func newTestService(t *testing.T) *Service {
	t.Helper()
	d, err := db.OpenAndMigrate(context.Background(), filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	loc, _ := time.LoadLocation("America/New_York")
	s := New(d, loc)
	s.Now = func() time.Time { return time.Date(2026, 9, 10, 15, 0, 0, 0, time.UTC) } // 11:00 New York
	return s
}

func mustNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	e, ok := AsError(err)
	if !ok || e.Code != code {
		t.Fatalf("want error code %q, got %v", code, err)
	}
}
```

`internal/service/catalog_test.go`:
```go
package service

import (
	"context"
	"testing"
)

var ctx = context.Background()

func TestCustomerCRUD(t *testing.T) {
	s := newTestService(t)
	c, err := s.CreateCustomer(ctx, CustomerInput{Name: "Blue Plate Diner", DeliveryDays: []string{"Mon", "thu"}, QBOCustomerID: "qbo-1", Active: true})
	mustNoErr(t, err)
	if c.DeliveryDays != "mon,thu" || c.QboCustomerID.String != "qbo-1" {
		t.Fatalf("bad row: %+v", c)
	}
	_, err = s.CreateCustomer(ctx, CustomerInput{Name: "", Active: true})
	wantCode(t, err, "invalid")
	_, err = s.CreateCustomer(ctx, CustomerInput{Name: "X", DeliveryDays: []string{"funday"}, Active: true})
	wantCode(t, err, "invalid")
	_, err = s.CreateCustomer(ctx, CustomerInput{Name: "Dup", QBOCustomerID: "qbo-1", Active: true})
	wantCode(t, err, "duplicate")

	u, err := s.UpdateCustomer(ctx, c.ID, CustomerInput{Name: "Blue Plate", Active: false})
	mustNoErr(t, err)
	if u.Name != "Blue Plate" || u.Active {
		t.Fatalf("update failed: %+v", u)
	}
	active, _ := s.ListCustomers(ctx, false)
	all, _ := s.ListCustomers(ctx, true)
	if len(active) != 0 || len(all) != 1 {
		t.Fatalf("list: active=%d all=%d", len(active), len(all))
	}
	_, err = s.GetCustomer(ctx, 999)
	wantCode(t, err, "not_found")
}

func TestProductCRUD(t *testing.T) {
	s := newTestService(t)
	w := int64(6000)
	p, err := s.CreateProduct(ctx, ProductInput{SKU: "BRIS", Name: "Brisket", SellUnit: "case", CatchWeight: true, ApproxCaseWeight: &w, BasePriceCents: 599, Active: true})
	mustNoErr(t, err)
	if !p.CatchWeight || p.ApproxCaseWeight.Int64 != 6000 {
		t.Fatalf("bad row: %+v", p)
	}
	_, err = s.CreateProduct(ctx, ProductInput{SKU: "BRIS", Name: "Dup", SellUnit: "each", BasePriceCents: 1, Active: true})
	wantCode(t, err, "duplicate")
	_, err = s.CreateProduct(ctx, ProductInput{SKU: "BAD", Name: "Bad", SellUnit: "lb", CatchWeight: true, ApproxCaseWeight: &w, BasePriceCents: 1, Active: true})
	wantCode(t, err, "invalid")
	u, err := s.UpdateProduct(ctx, p.ID, ProductInput{SKU: "BRIS", Name: "Whole Brisket", SellUnit: "case", CatchWeight: true, ApproxCaseWeight: &w, BasePriceCents: 649, Active: true})
	mustNoErr(t, err)
	if u.Name != "Whole Brisket" || u.BasePriceCents != 649 {
		t.Fatalf("update failed: %+v", u)
	}
}

func TestCustomerPrices(t *testing.T) {
	s := newTestService(t)
	c, _ := s.CreateCustomer(ctx, CustomerInput{Name: "A", Active: true})
	p, _ := s.CreateProduct(ctx, ProductInput{SKU: "S1", Name: "Sausage", SellUnit: "each", BasePriceCents: 400, Active: true})
	_, err := s.SetCustomerPrice(ctx, c.ID, PriceInput{ProductID: p.ID, PriceCents: 350, EffectiveFrom: "2026-01-01"})
	mustNoErr(t, err)
	_, err = s.SetCustomerPrice(ctx, c.ID, PriceInput{ProductID: p.ID, PriceCents: 375, EffectiveFrom: "2026-10-01"})
	mustNoErr(t, err)
	_, err = s.SetCustomerPrice(ctx, c.ID, PriceInput{ProductID: p.ID, PriceCents: 1, EffectiveFrom: "not-a-date"})
	wantCode(t, err, "invalid")
	_, err = s.SetCustomerPrice(ctx, c.ID, PriceInput{ProductID: 999, PriceCents: 1, EffectiveFrom: "2026-01-01"})
	wantCode(t, err, "not_found")

	rows, err := s.ListCustomerPrices(ctx, c.ID, "2026-09-10")
	mustNoErr(t, err)
	if len(rows) != 1 || rows[0].PriceCents != 350 {
		t.Fatalf("as of sept: %+v", rows)
	}
	rows, _ = s.ListCustomerPrices(ctx, c.ID, "2026-10-01")
	if len(rows) != 1 || rows[0].PriceCents != 375 {
		t.Fatalf("as of oct: %+v", rows)
	}
	price, custom, err := s.resolvePrice(ctx, s.Q, c.ID, p.ID, 400, "2025-06-01")
	mustNoErr(t, err)
	if price != 400 || custom {
		t.Fatalf("fallback to base: %d %v", price, custom)
	}
	price, custom, _ = s.resolvePrice(ctx, s.Q, c.ID, p.ID, 400, "2026-12-01")
	if price != 375 || !custom {
		t.Fatalf("latest custom: %d %v", price, custom)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/service/ 2>&1 | head -3`
Expected: build failure.

- [ ] **Step 4: Implement catalog.go**

`internal/service/catalog.go`:
```go
package service

import (
	"context"
	"strings"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
)

type CustomerInput struct {
	Name, BillingAddress, DeliveryAddress, ContactName, Phone, Email, DeliveryNotes string
	DeliveryDays                                                                    []string
	QBOCustomerID                                                                   string
	Active                                                                          bool
}

func (in CustomerInput) validate() (days string, err error) {
	fields := map[string]string{}
	if strings.TrimSpace(in.Name) == "" {
		fields["name"] = "required"
	}
	days, derr := ParseDeliveryDays(in.DeliveryDays)
	if derr != nil {
		fields["deliveryDays"] = derr.Error()
	}
	if len(fields) > 0 {
		return "", Invalid(fields)
	}
	return days, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (s *Service) CreateCustomer(ctx context.Context, in CustomerInput) (queries.Customer, error) {
	days, err := in.validate()
	if err != nil {
		return queries.Customer{}, err
	}
	now := s.now()
	c, err := s.Q.CreateCustomer(ctx, queries.CreateCustomerParams{
		Name: strings.TrimSpace(in.Name), BillingAddress: in.BillingAddress, DeliveryAddress: in.DeliveryAddress,
		ContactName: in.ContactName, Phone: in.Phone, Email: in.Email, DeliveryNotes: in.DeliveryNotes,
		DeliveryDays: days, QboCustomerID: nullStr(in.QBOCustomerID), CreatedAt: now, UpdatedAt: now,
	})
	if isUniqueViolation(err) {
		return queries.Customer{}, Conflict("duplicate", "a customer with that QuickBooks id already exists")
	}
	return c, err
}

func (s *Service) UpdateCustomer(ctx context.Context, id int64, in CustomerInput) (queries.Customer, error) {
	days, err := in.validate()
	if err != nil {
		return queries.Customer{}, err
	}
	if _, err := s.Q.GetCustomer(ctx, id); err != nil {
		return queries.Customer{}, notFoundIf(err, "customer")
	}
	c, err := s.Q.UpdateCustomer(ctx, queries.UpdateCustomerParams{
		Name: strings.TrimSpace(in.Name), BillingAddress: in.BillingAddress, DeliveryAddress: in.DeliveryAddress,
		ContactName: in.ContactName, Phone: in.Phone, Email: in.Email, DeliveryNotes: in.DeliveryNotes,
		DeliveryDays: days, QboCustomerID: nullStr(in.QBOCustomerID), Active: in.Active, UpdatedAt: s.now(), ID: id,
	})
	if isUniqueViolation(err) {
		return queries.Customer{}, Conflict("duplicate", "a customer with that QuickBooks id already exists")
	}
	return c, err
}

func (s *Service) GetCustomer(ctx context.Context, id int64) (queries.Customer, error) {
	c, err := s.Q.GetCustomer(ctx, id)
	return c, notFoundIf(err, "customer")
}

func (s *Service) ListCustomers(ctx context.Context, includeInactive bool) ([]queries.Customer, error) {
	return s.Q.ListCustomers(ctx, includeInactive)
}

type ProductInput struct {
	SKU, Name, Category, SellUnit string
	CatchWeight                   bool
	ApproxCaseWeight              *int64
	BasePriceCents                int64
	QBOItemID                     string
	Active                        bool
}

func (in ProductInput) domain() (domain.Product, error) {
	p := domain.Product{
		SKU: strings.TrimSpace(in.SKU), Name: strings.TrimSpace(in.Name), SellUnit: domain.Unit(in.SellUnit),
		CatchWeight: in.CatchWeight, BasePrice: domain.Cents(in.BasePriceCents),
	}
	if in.ApproxCaseWeight != nil {
		p.ApproxCaseWeight = domain.Hundredths(*in.ApproxCaseWeight)
	}
	if errs := p.Validate(); len(errs) > 0 {
		return p, Invalid(errs)
	}
	return p, nil
}

func (s *Service) CreateProduct(ctx context.Context, in ProductInput) (queries.Product, error) {
	p, err := in.domain()
	if err != nil {
		return queries.Product{}, err
	}
	now := s.now()
	row, err := s.Q.CreateProduct(ctx, queries.CreateProductParams{
		Sku: p.SKU, Name: p.Name, Category: in.Category, SellUnit: string(p.SellUnit), CatchWeight: p.CatchWeight,
		ApproxCaseWeight: nullInt(in.ApproxCaseWeight), BasePriceCents: in.BasePriceCents, QboItemID: nullStr(in.QBOItemID),
		CreatedAt: now, UpdatedAt: now,
	})
	if isUniqueViolation(err) {
		return queries.Product{}, Conflict("duplicate", "a product with that SKU already exists")
	}
	return row, err
}

func (s *Service) UpdateProduct(ctx context.Context, id int64, in ProductInput) (queries.Product, error) {
	p, err := in.domain()
	if err != nil {
		return queries.Product{}, err
	}
	if _, err := s.Q.GetProduct(ctx, id); err != nil {
		return queries.Product{}, notFoundIf(err, "product")
	}
	row, err := s.Q.UpdateProduct(ctx, queries.UpdateProductParams{
		Sku: p.SKU, Name: p.Name, Category: in.Category, SellUnit: string(p.SellUnit), CatchWeight: p.CatchWeight,
		ApproxCaseWeight: nullInt(in.ApproxCaseWeight), BasePriceCents: in.BasePriceCents, QboItemID: nullStr(in.QBOItemID),
		Active: in.Active, UpdatedAt: s.now(), ID: id,
	})
	if isUniqueViolation(err) {
		return queries.Product{}, Conflict("duplicate", "a product with that SKU already exists")
	}
	return row, err
}

func (s *Service) GetProduct(ctx context.Context, id int64) (queries.Product, error) {
	p, err := s.Q.GetProduct(ctx, id)
	return p, notFoundIf(err, "product")
}

func (s *Service) ListProducts(ctx context.Context, includeInactive bool) ([]queries.Product, error) {
	return s.Q.ListProducts(ctx, includeInactive)
}

type PriceInput struct {
	ProductID     int64
	PriceCents    int64
	EffectiveFrom string
}

func (s *Service) SetCustomerPrice(ctx context.Context, customerID int64, in PriceInput) (queries.CustomerPrice, error) {
	fields := map[string]string{}
	if !ValidDate(in.EffectiveFrom) {
		fields["effectiveFrom"] = "must be YYYY-MM-DD"
	}
	if in.PriceCents < 0 {
		fields["priceCents"] = "must not be negative"
	}
	if len(fields) > 0 {
		return queries.CustomerPrice{}, Invalid(fields)
	}
	if _, err := s.Q.GetCustomer(ctx, customerID); err != nil {
		return queries.CustomerPrice{}, notFoundIf(err, "customer")
	}
	if _, err := s.Q.GetProduct(ctx, in.ProductID); err != nil {
		return queries.CustomerPrice{}, notFoundIf(err, "product")
	}
	return s.Q.CreateCustomerPrice(ctx, queries.CreateCustomerPriceParams{
		CustomerID: customerID, ProductID: in.ProductID, PriceCents: in.PriceCents, EffectiveFrom: in.EffectiveFrom, CreatedAt: s.now(),
	})
}

func (s *Service) ListCustomerPrices(ctx context.Context, customerID int64, asOf string) ([]queries.ListCurrentCustomerPricesRow, error) {
	if _, err := s.Q.GetCustomer(ctx, customerID); err != nil {
		return nil, notFoundIf(err, "customer")
	}
	return s.Q.ListCurrentCustomerPrices(ctx, queries.ListCurrentCustomerPricesParams{CustomerID: customerID, AsOf: asOf})
}

// resolvePrice applies domain.ResolvePrice using the customer's price rows. q may be transaction-bound.
func (s *Service) resolvePrice(ctx context.Context, q *queries.Queries, customerID, productID int64, base domain.Cents, date string) (domain.Cents, bool, error) {
	rows, err := q.ListCustomerPriceRows(ctx, queries.ListCustomerPriceRowsParams{CustomerID: customerID, ProductID: productID})
	if err != nil {
		return 0, false, err
	}
	prs := make([]domain.PriceRow, len(rows))
	for i, r := range rows {
		prs[i] = domain.PriceRow{Price: domain.Cents(r.PriceCents), EffectiveFrom: r.EffectiveFrom}
	}
	p, custom := domain.ResolvePrice(base, prs, date)
	return p, custom, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/service/ 2>&1 | tail -3`
Expected: `ok`. If a sqlc param name differs (e.g. `IncludeInactive` vs positional), read `internal/db/queries/customers.sql.go` and match the generated names.

- [ ] **Step 6: Commit**

```bash
cd ~/deepcutsCRM && git add internal/service && git commit -q -m "Add service foundation and catalog use cases" && git push
```

---

### Task 6: Orders service: create, lines with frozen prices, transitions

**Files:**
- Create: `sql/queries/orders.sql`, `internal/service/orders.go`, `internal/service/orders_test.go`
- Generated: `internal/db/queries/orders.sql.go`

**Interfaces:**
- Produces:
  - `type OrderInput struct { CustomerID int64; RequestedDeliveryDate, Notes string; CreatedBy int64 }`
  - `type OrderFilter struct { Date, Status string; CustomerID int64 }`
  - `type LineDetail struct { Line queries.OrderLine; Product queries.Product; AmountCents int64; AmountSource string }`
  - `type OrderDetail struct { Order queries.Order; Customer queries.Customer; Lines []LineDetail; TotalCents int64; RouteID, StopID *int64; RouteStatus string }`
  - `type LinePatch struct { OrderedQty, UnitPriceCents, ShippedWeight, DeliveredQty, DeliveredWeight *int64; ShortageNote *string }`
  - `CreateOrder(ctx, OrderInput) (OrderDetail, error)`, `GetOrder(ctx, id) (OrderDetail, error)`, `ListOrders(ctx, OrderFilter) ([]queries.ListOrdersRow, error)`, `UpdateOrder(ctx, id, notes, date *string) (OrderDetail, error)`
  - `AddLine(ctx, orderID, productID int64, orderedQty int64) (OrderDetail, error)`, `UpdateLine(ctx, orderID, lineID int64, LinePatch) (OrderDetail, error)`, `DeleteLine(ctx, orderID, lineID) (OrderDetail, error)`
  - `ConfirmOrder`, `UnconfirmOrder`, `CancelOrder`, `FinalizeOrder` — all `(ctx, id int64) (OrderDetail, error)`
  - `transitionOrder(ctx, q, order queries.Order, to domain.OrderStatus) error` — shared by routes/driver services
  - `loadOrder(ctx, q, id) (OrderDetail, error)` — shared
  - Query names: `CreateOrder, GetOrder, ListOrders, UpdateOrderMeta, UpdateOrderStatus, SetOrderNeedsReview, FinalizeOrder, CreateOrderLine, GetOrderLine, ListOrderLines, UpdateOrderLine, DeleteOrderLine, CountOrderLines, GetActiveStopForOrder`

- [ ] **Step 1: Write the queries**

`sql/queries/orders.sql`:
```sql
-- name: CreateOrder :one
INSERT INTO orders (customer_id, requested_delivery_date, status, notes, created_by, needs_review, created_at, updated_at)
VALUES (?, ?, 'draft', ?, ?, FALSE, ?, ?) RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders WHERE id = ?;

-- name: ListOrders :many
SELECT o.*, c.name AS customer_name,
  (SELECT count(*) FROM order_lines ol WHERE ol.order_id = o.id) AS line_count
FROM orders o JOIN customers c ON c.id = o.customer_id
WHERE (sqlc.arg(date) = '' OR o.requested_delivery_date = sqlc.arg(date))
  AND (sqlc.arg(status) = '' OR o.status = sqlc.arg(status))
  AND (sqlc.arg(customer_id) = 0 OR o.customer_id = sqlc.arg(customer_id))
ORDER BY o.requested_delivery_date DESC, o.id DESC
LIMIT 500;

-- name: ListUnscheduledOrdersForDate :many
SELECT o.*, c.name AS customer_name,
  (SELECT count(*) FROM order_lines ol WHERE ol.order_id = o.id) AS line_count
FROM orders o JOIN customers c ON c.id = o.customer_id
WHERE o.requested_delivery_date = ? AND o.status = 'confirmed'
ORDER BY c.name;

-- name: UpdateOrderMeta :exec
UPDATE orders SET notes = ?, requested_delivery_date = ?, updated_at = ? WHERE id = ?;

-- name: UpdateOrderStatus :exec
UPDATE orders SET status = ?, updated_at = ? WHERE id = ?;

-- name: SetOrderNeedsReview :exec
UPDATE orders SET needs_review = ?, updated_at = ? WHERE id = ?;

-- name: FinalizeOrder :exec
UPDATE orders SET status = 'finalized', needs_review = FALSE, finalized_at = ?, updated_at = ? WHERE id = ?;

-- name: CreateOrderLine :one
INSERT INTO order_lines (order_id, product_id, ordered_qty, unit_price_cents, price_overridden, est_weight)
VALUES (?, ?, ?, ?, FALSE, ?) RETURNING *;

-- name: GetOrderLine :one
SELECT * FROM order_lines WHERE id = ? AND order_id = ?;

-- name: ListOrderLines :many
SELECT ol.*, p.sku, p.name AS product_name, p.category, p.sell_unit, p.catch_weight, p.approx_case_weight, p.base_price_cents
FROM order_lines ol JOIN products p ON p.id = ol.product_id
WHERE ol.order_id = ? ORDER BY ol.id;

-- name: UpdateOrderLine :exec
UPDATE order_lines SET ordered_qty = ?, unit_price_cents = ?, price_overridden = ?, est_weight = ?,
  shipped_weight = ?, delivered_qty = ?, delivered_weight = ?, shortage_note = ?
WHERE id = ?;

-- name: DeleteOrderLine :exec
DELETE FROM order_lines WHERE id = ? AND order_id = ?;

-- name: CountOrderLines :one
SELECT count(*) FROM order_lines WHERE order_id = ?;

-- name: GetActiveStopForOrder :one
SELECT s.id AS stop_id, s.route_id, s.status AS stop_status, r.status AS route_status, r.route_date, r.driver_user_id
FROM delivery_stops s JOIN delivery_routes r ON r.id = s.route_id
WHERE s.order_id = ? AND s.status <> 'skipped';
```

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go tool sqlc generate && go build ./...`
Expected: ok. Read `internal/db/queries/orders.sql.go` (≤200 lines at a time) to confirm the generated param struct field names before writing orders.go.

- [ ] **Step 2: Write the failing tests**

`internal/service/orders_test.go`:
```go
package service

import "testing"

type fixture struct {
	s        *Service
	customer int64
	brisket  int64 // catch-weight, 60 lb cases, base $5.99/lb, customer price $5.49
	sausage  int64 // each, $4.00
}

func newFixture(t *testing.T) fixture {
	s := newTestService(t)
	c, err := s.CreateCustomer(ctx, CustomerInput{Name: "Blue Plate", Active: true})
	mustNoErr(t, err)
	w := int64(6000)
	b, err := s.CreateProduct(ctx, ProductInput{SKU: "BRIS", Name: "Brisket", SellUnit: "case", CatchWeight: true, ApproxCaseWeight: &w, BasePriceCents: 599, Active: true})
	mustNoErr(t, err)
	sg, err := s.CreateProduct(ctx, ProductInput{SKU: "SAUS", Name: "Sausage", SellUnit: "each", BasePriceCents: 400, Active: true})
	mustNoErr(t, err)
	_, err = s.SetCustomerPrice(ctx, c.ID, PriceInput{ProductID: b.ID, PriceCents: 549, EffectiveFrom: "2026-01-01"})
	mustNoErr(t, err)
	return fixture{s: s, customer: c.ID, brisket: b.ID, sausage: sg.ID}
}

func TestCreateOrderAndLinesFreezePrice(t *testing.T) {
	f := newFixture(t)
	o, err := f.s.CreateOrder(ctx, OrderInput{CustomerID: f.customer, RequestedDeliveryDate: "2026-09-12"})
	mustNoErr(t, err)
	if o.Order.Status != "draft" || len(o.Lines) != 0 {
		t.Fatalf("new order: %+v", o.Order)
	}
	_, err = f.s.CreateOrder(ctx, OrderInput{CustomerID: f.customer, RequestedDeliveryDate: "12/09/2026"})
	wantCode(t, err, "invalid")
	_, err = f.s.CreateOrder(ctx, OrderInput{CustomerID: 999, RequestedDeliveryDate: "2026-09-12"})
	wantCode(t, err, "not_found")

	o, err = f.s.AddLine(ctx, o.Order.ID, f.brisket, 200) // 2 cases
	mustNoErr(t, err)
	l := o.Lines[0]
	if l.Line.UnitPriceCents != 549 || l.Line.EstWeight.Int64 != 12000 || l.AmountSource != "estimated" || l.AmountCents != 65880 {
		t.Fatalf("brisket line: %+v amount=%d src=%s", l.Line, l.AmountCents, l.AmountSource)
	}
	o, _ = f.s.AddLine(ctx, o.Order.ID, f.sausage, 1000) // 10 each
	if o.Lines[1].Line.UnitPriceCents != 400 || o.Lines[1].AmountCents != 4000 || o.TotalCents != 69880 {
		t.Fatalf("sausage line / total: %+v total=%d", o.Lines[1], o.TotalCents)
	}
	// price change after the fact does not touch the line
	_, _ = f.s.SetCustomerPrice(ctx, f.customer, PriceInput{ProductID: f.brisket, PriceCents: 100, EffectiveFrom: "2026-01-02"})
	o, _ = f.s.GetOrder(ctx, o.Order.ID)
	if o.Lines[0].Line.UnitPriceCents != 549 {
		t.Fatal("price was not frozen")
	}
	_, err = f.s.AddLine(ctx, o.Order.ID, f.brisket, 0)
	wantCode(t, err, "invalid")
}

func TestLineEditsAndOverride(t *testing.T) {
	f := newFixture(t)
	o, _ := f.s.CreateOrder(ctx, OrderInput{CustomerID: f.customer, RequestedDeliveryDate: "2026-09-12"})
	o, _ = f.s.AddLine(ctx, o.Order.ID, f.brisket, 200)
	lineID := o.Lines[0].Line.ID
	price := int64(500)
	qty := int64(300)
	o, err := f.s.UpdateLine(ctx, o.Order.ID, lineID, LinePatch{OrderedQty: &qty, UnitPriceCents: &price})
	mustNoErr(t, err)
	l := o.Lines[0].Line
	if l.OrderedQty != 300 || l.UnitPriceCents != 500 || !l.PriceOverridden || l.EstWeight.Int64 != 18000 {
		t.Fatalf("patch: %+v", l)
	}
	shipped := int64(17550)
	o, _ = f.s.UpdateLine(ctx, o.Order.ID, lineID, LinePatch{ShippedWeight: &shipped})
	if o.Lines[0].AmountSource != "shipped" || o.Lines[0].AmountCents != 87750 {
		t.Fatalf("shipped amount: %+v", o.Lines[0])
	}
	o, err = f.s.DeleteLine(ctx, o.Order.ID, lineID)
	mustNoErr(t, err)
	if len(o.Lines) != 0 {
		t.Fatal("line not deleted")
	}
	_, err = f.s.DeleteLine(ctx, o.Order.ID, 999)
	wantCode(t, err, "not_found")
}

func TestOrderTransitionsAndLocks(t *testing.T) {
	f := newFixture(t)
	o, _ := f.s.CreateOrder(ctx, OrderInput{CustomerID: f.customer, RequestedDeliveryDate: "2026-09-12"})
	_, err := f.s.ConfirmOrder(ctx, o.Order.ID)
	wantCode(t, err, "no_lines")
	o, _ = f.s.AddLine(ctx, o.Order.ID, f.sausage, 100)
	o, err = f.s.ConfirmOrder(ctx, o.Order.ID)
	mustNoErr(t, err)
	if o.Order.Status != "confirmed" {
		t.Fatal(o.Order.Status)
	}
	// still editable while confirmed
	o, err = f.s.AddLine(ctx, o.Order.ID, f.brisket, 100)
	mustNoErr(t, err)
	_, err = f.s.FinalizeOrder(ctx, o.Order.ID)
	wantCode(t, err, "invalid_transition")
	o, err = f.s.UnconfirmOrder(ctx, o.Order.ID)
	mustNoErr(t, err)
	if o.Order.Status != "draft" {
		t.Fatal(o.Order.Status)
	}
	o, _ = f.s.CancelOrder(ctx, o.Order.ID)
	if o.Order.Status != "cancelled" {
		t.Fatal(o.Order.Status)
	}
	_, err = f.s.AddLine(ctx, o.Order.ID, f.sausage, 100)
	wantCode(t, err, "locked")
	_, err = f.s.ConfirmOrder(ctx, o.Order.ID)
	wantCode(t, err, "invalid_transition")
}

func TestListOrdersFilters(t *testing.T) {
	f := newFixture(t)
	a, _ := f.s.CreateOrder(ctx, OrderInput{CustomerID: f.customer, RequestedDeliveryDate: "2026-09-12"})
	_, _ = f.s.CreateOrder(ctx, OrderInput{CustomerID: f.customer, RequestedDeliveryDate: "2026-09-13"})
	_, _ = f.s.AddLine(ctx, a.Order.ID, f.sausage, 100)
	_, _ = f.s.ConfirmOrder(ctx, a.Order.ID)
	all, _ := f.s.ListOrders(ctx, OrderFilter{})
	byDate, _ := f.s.ListOrders(ctx, OrderFilter{Date: "2026-09-12"})
	byStatus, _ := f.s.ListOrders(ctx, OrderFilter{Status: "confirmed"})
	if len(all) != 2 || len(byDate) != 1 || len(byStatus) != 1 || byStatus[0].LineCount != 1 || byStatus[0].CustomerName != "Blue Plate" {
		t.Fatalf("all=%d date=%d status=%d", len(all), len(byDate), len(byStatus))
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/service/ 2>&1 | head -3`
Expected: build failure.

- [ ] **Step 4: Implement orders.go**

`internal/service/orders.go`:
```go
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
)

type OrderInput struct {
	CustomerID            int64
	RequestedDeliveryDate string
	Notes                 string
	CreatedBy             int64
}

type OrderFilter struct {
	Date       string
	Status     string
	CustomerID int64
}

type LineDetail struct {
	Line         queries.OrderLine
	Product      queries.Product
	AmountCents  int64
	AmountSource string
}

type OrderDetail struct {
	Order       queries.Order
	Customer    queries.Customer
	Lines       []LineDetail
	TotalCents  int64
	RouteID     *int64
	StopID      *int64
	RouteStatus string // "" when not scheduled
}

type LinePatch struct {
	OrderedQty, UnitPriceCents, ShippedWeight, DeliveredQty, DeliveredWeight *int64
	ShortageNote                                                             *string
}

func (s *Service) CreateOrder(ctx context.Context, in OrderInput) (OrderDetail, error) {
	if !ValidDate(in.RequestedDeliveryDate) {
		return OrderDetail{}, Invalid(map[string]string{"requestedDeliveryDate": "must be YYYY-MM-DD"})
	}
	var out OrderDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		if _, err := q.GetCustomer(ctx, in.CustomerID); err != nil {
			return notFoundIf(err, "customer")
		}
		now := s.now()
		o, err := q.CreateOrder(ctx, queries.CreateOrderParams{
			CustomerID: in.CustomerID, RequestedDeliveryDate: in.RequestedDeliveryDate, Notes: in.Notes,
			CreatedBy: nullInt(nonZero(in.CreatedBy)), CreatedAt: now, UpdatedAt: now,
		})
		if err != nil {
			return err
		}
		out, err = s.loadOrder(ctx, q, o.ID)
		return err
	})
	return out, err
}

func nonZero(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}

func (s *Service) GetOrder(ctx context.Context, id int64) (OrderDetail, error) {
	return s.loadOrder(ctx, s.Q, id)
}

func (s *Service) ListOrders(ctx context.Context, f OrderFilter) ([]queries.ListOrdersRow, error) {
	return s.Q.ListOrders(ctx, queries.ListOrdersParams{Date: f.Date, Status: f.Status, CustomerID: f.CustomerID})
}

func (s *Service) UpdateOrder(ctx context.Context, id int64, notes, date *string) (OrderDetail, error) {
	var out OrderDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		o, err := q.GetOrder(ctx, id)
		if err != nil {
			return notFoundIf(err, "order")
		}
		if scope, _ := s.editScope(ctx, q, o); scope != domain.EditAll {
			return Conflict("locked", "order can no longer be edited")
		}
		if notes != nil {
			o.Notes = *notes
		}
		if date != nil {
			if !ValidDate(*date) {
				return Invalid(map[string]string{"requestedDeliveryDate": "must be YYYY-MM-DD"})
			}
			if o.Status == "scheduled" {
				return Conflict("locked", "unschedule the order before changing its date")
			}
			o.RequestedDeliveryDate = *date
		}
		if err := q.UpdateOrderMeta(ctx, queries.UpdateOrderMetaParams{Notes: o.Notes, RequestedDeliveryDate: o.RequestedDeliveryDate, UpdatedAt: s.now(), ID: id}); err != nil {
			return err
		}
		out, err = s.loadOrder(ctx, q, id)
		return err
	})
	return out, err
}

// editScope returns what may be edited on the order's lines, and the active route status if any.
func (s *Service) editScope(ctx context.Context, q *queries.Queries, o queries.Order) (domain.LineEditScope, string) {
	routeStatus := ""
	if stop, err := q.GetActiveStopForOrder(ctx, o.ID); err == nil {
		routeStatus = stop.RouteStatus
	}
	return domain.LineEditScopeFor(domain.OrderStatus(o.Status), domain.RouteStatus(routeStatus)), routeStatus
}

func (s *Service) AddLine(ctx context.Context, orderID, productID, orderedQty int64) (OrderDetail, error) {
	if orderedQty <= 0 {
		return OrderDetail{}, Invalid(map[string]string{"orderedQty": "must be positive"})
	}
	var out OrderDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		o, err := q.GetOrder(ctx, orderID)
		if err != nil {
			return notFoundIf(err, "order")
		}
		if scope, _ := s.editScope(ctx, q, o); scope != domain.EditAll {
			return Conflict("locked", "lines cannot be added to this order")
		}
		p, err := q.GetProduct(ctx, productID)
		if err != nil {
			return notFoundIf(err, "product")
		}
		dp := productOf(p)
		price, _, err := s.resolvePrice(ctx, q, o.CustomerID, productID, dp.BasePrice, o.RequestedDeliveryDate)
		if err != nil {
			return err
		}
		var est sql.NullInt64
		if w, ok := domain.EstimateWeight(dp, domain.Hundredths(orderedQty)); ok {
			est = sql.NullInt64{Int64: int64(w), Valid: true}
		}
		if _, err := q.CreateOrderLine(ctx, queries.CreateOrderLineParams{
			OrderID: orderID, ProductID: productID, OrderedQty: orderedQty, UnitPriceCents: int64(price), EstWeight: est,
		}); err != nil {
			return err
		}
		if err := q.UpdateOrderMeta(ctx, queries.UpdateOrderMetaParams{Notes: o.Notes, RequestedDeliveryDate: o.RequestedDeliveryDate, UpdatedAt: s.now(), ID: orderID}); err != nil {
			return err
		}
		out, err = s.loadOrder(ctx, q, orderID)
		return err
	})
	return out, err
}

func (s *Service) UpdateLine(ctx context.Context, orderID, lineID int64, patch LinePatch) (OrderDetail, error) {
	var out OrderDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		o, err := q.GetOrder(ctx, orderID)
		if err != nil {
			return notFoundIf(err, "order")
		}
		l, err := q.GetOrderLine(ctx, queries.GetOrderLineParams{ID: lineID, OrderID: orderID})
		if err != nil {
			return notFoundIf(err, "line")
		}
		p, err := q.GetProduct(ctx, l.ProductID)
		if err != nil {
			return err
		}
		scope, _ := s.editScope(ctx, q, o)
		if err := applyLinePatch(&l, productOf(p), patch, scope); err != nil {
			return err
		}
		if err := q.UpdateOrderLine(ctx, queries.UpdateOrderLineParams{
			OrderedQty: l.OrderedQty, UnitPriceCents: l.UnitPriceCents, PriceOverridden: l.PriceOverridden, EstWeight: l.EstWeight,
			ShippedWeight: l.ShippedWeight, DeliveredQty: l.DeliveredQty, DeliveredWeight: l.DeliveredWeight, ShortageNote: l.ShortageNote, ID: l.ID,
		}); err != nil {
			return err
		}
		out, err = s.loadOrder(ctx, q, orderID)
		return err
	})
	return out, err
}

// applyLinePatch mutates l according to the patch, enforcing the edit scope.
func applyLinePatch(l *queries.OrderLine, p domain.Product, patch LinePatch, scope domain.LineEditScope) error {
	fields := map[string]string{}
	locked := func(f string) { fields[f] = "cannot be changed in the order's current state" }
	if patch.OrderedQty != nil {
		if scope != domain.EditAll {
			locked("orderedQty")
		} else if *patch.OrderedQty <= 0 {
			fields["orderedQty"] = "must be positive"
		} else {
			l.OrderedQty = *patch.OrderedQty
			l.EstWeight = sql.NullInt64{}
			if w, ok := domain.EstimateWeight(p, domain.Hundredths(l.OrderedQty)); ok {
				l.EstWeight = sql.NullInt64{Int64: int64(w), Valid: true}
			}
		}
	}
	if patch.UnitPriceCents != nil {
		if scope != domain.EditAll && scope != domain.EditShippedAndPrice {
			locked("unitPriceCents")
		} else if *patch.UnitPriceCents < 0 {
			fields["unitPriceCents"] = "must not be negative"
		} else {
			l.UnitPriceCents = *patch.UnitPriceCents
			l.PriceOverridden = true
		}
	}
	if patch.ShippedWeight != nil {
		if scope != domain.EditAll && scope != domain.EditShippedAndPrice {
			locked("shippedWeight")
		} else if !p.CatchWeight {
			fields["shippedWeight"] = "only catch-weight lines have a shipped weight"
		} else if *patch.ShippedWeight < 0 {
			fields["shippedWeight"] = "must not be negative"
		} else {
			l.ShippedWeight = sql.NullInt64{Int64: *patch.ShippedWeight, Valid: true}
		}
	}
	for name, v := range map[string]*int64{"deliveredQty": patch.DeliveredQty, "deliveredWeight": patch.DeliveredWeight} {
		if v == nil {
			continue
		}
		if scope != domain.EditDelivered {
			locked(name)
		} else if *v < 0 {
			fields[name] = "must not be negative"
		} else if name == "deliveredQty" {
			l.DeliveredQty = sql.NullInt64{Int64: *v, Valid: true}
		} else if !p.CatchWeight {
			fields[name] = "only catch-weight lines have a delivered weight"
		} else {
			l.DeliveredWeight = sql.NullInt64{Int64: *v, Valid: true}
		}
	}
	if patch.ShortageNote != nil {
		if scope == domain.EditNone {
			locked("shortageNote")
		} else {
			l.ShortageNote = *patch.ShortageNote
		}
	}
	if len(fields) > 0 {
		return Invalid(fields)
	}
	return nil
}

func (s *Service) DeleteLine(ctx context.Context, orderID, lineID int64) (OrderDetail, error) {
	var out OrderDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		o, err := q.GetOrder(ctx, orderID)
		if err != nil {
			return notFoundIf(err, "order")
		}
		if scope, _ := s.editScope(ctx, q, o); scope != domain.EditAll {
			return Conflict("locked", "lines cannot be removed from this order")
		}
		if _, err := q.GetOrderLine(ctx, queries.GetOrderLineParams{ID: lineID, OrderID: orderID}); err != nil {
			return notFoundIf(err, "line")
		}
		if err := q.DeleteOrderLine(ctx, queries.DeleteOrderLineParams{ID: lineID, OrderID: orderID}); err != nil {
			return err
		}
		out, err = s.loadOrder(ctx, q, orderID)
		return err
	})
	return out, err
}

// transitionOrder checks the state machine and writes the new status.
func (s *Service) transitionOrder(ctx context.Context, q *queries.Queries, o queries.Order, to domain.OrderStatus) error {
	from := domain.OrderStatus(o.Status)
	if !from.CanTransition(to) {
		return Conflict("invalid_transition", fmt.Sprintf("order cannot go from %s to %s", from, to))
	}
	return q.UpdateOrderStatus(ctx, queries.UpdateOrderStatusParams{Status: string(to), UpdatedAt: s.now(), ID: o.ID})
}

func (s *Service) simpleTransition(ctx context.Context, id int64, to domain.OrderStatus, pre func(q *queries.Queries, o queries.Order) error) (OrderDetail, error) {
	var out OrderDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		o, err := q.GetOrder(ctx, id)
		if err != nil {
			return notFoundIf(err, "order")
		}
		if pre != nil {
			if err := pre(q, o); err != nil {
				return err
			}
		}
		if err := s.transitionOrder(ctx, q, o, to); err != nil {
			return err
		}
		out, err = s.loadOrder(ctx, q, id)
		return err
	})
	return out, err
}

func (s *Service) ConfirmOrder(ctx context.Context, id int64) (OrderDetail, error) {
	return s.simpleTransition(ctx, id, domain.OrderConfirmed, func(q *queries.Queries, o queries.Order) error {
		if !domain.OrderStatus(o.Status).CanTransition(domain.OrderConfirmed) {
			return Conflict("invalid_transition", fmt.Sprintf("order cannot go from %s to confirmed", o.Status))
		}
		n, err := q.CountOrderLines(ctx, id)
		if err != nil {
			return err
		}
		if n == 0 {
			return Conflict("no_lines", "add at least one line before confirming")
		}
		return nil
	})
}

func (s *Service) UnconfirmOrder(ctx context.Context, id int64) (OrderDetail, error) {
	return s.simpleTransition(ctx, id, domain.OrderDraft, nil)
}

// CancelOrder cancels a draft, confirmed or scheduled order. A scheduled order's stop is removed
// only while the route is planned; once the route is out the office must skip it via the driver flow.
func (s *Service) CancelOrder(ctx context.Context, id int64) (OrderDetail, error) {
	return s.simpleTransition(ctx, id, domain.OrderCancelled, func(q *queries.Queries, o queries.Order) error {
		stop, err := q.GetActiveStopForOrder(ctx, o.ID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if stop.RouteStatus != string(domain.RoutePlanned) {
			return Conflict("locked", "route is already out; skip the stop from the driver app instead")
		}
		return q.DeleteStop(ctx, stop.StopID)
	})
}

// FinalizeOrder locks a delivered order after checking every line has what it needs to be billed.
func (s *Service) FinalizeOrder(ctx context.Context, id int64) (OrderDetail, error) {
	var out OrderDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		o, err := q.GetOrder(ctx, id)
		if err != nil {
			return notFoundIf(err, "order")
		}
		if !domain.OrderStatus(o.Status).CanTransition(domain.OrderFinalized) {
			return Conflict("invalid_transition", fmt.Sprintf("order cannot go from %s to finalized", o.Status))
		}
		lines, err := q.ListOrderLines(ctx, id)
		if err != nil {
			return err
		}
		fields := map[string]string{}
		for _, l := range lines {
			key := fmt.Sprintf("line:%d", l.ID)
			if !l.DeliveredQty.Valid {
				fields[key] = "missing delivered quantity"
			} else if l.CatchWeight && !l.DeliveredWeight.Valid {
				fields[key] = "missing delivered weight"
			}
		}
		if len(fields) > 0 {
			return &Error{Status: 409, Code: "incomplete", Message: "some lines are missing delivered quantities", Fields: fields}
		}
		now := s.now()
		if err := q.FinalizeOrder(ctx, queries.FinalizeOrderParams{FinalizedAt: sql.NullTime{Time: now, Valid: true}, UpdatedAt: now, ID: id}); err != nil {
			return err
		}
		out, err = s.loadOrder(ctx, q, id)
		return err
	})
	return out, err
}

// loadOrder assembles an OrderDetail with computed amounts.
func (s *Service) loadOrder(ctx context.Context, q *queries.Queries, id int64) (OrderDetail, error) {
	o, err := q.GetOrder(ctx, id)
	if err != nil {
		return OrderDetail{}, notFoundIf(err, "order")
	}
	c, err := q.GetCustomer(ctx, o.CustomerID)
	if err != nil {
		return OrderDetail{}, err
	}
	rows, err := q.ListOrderLines(ctx, id)
	if err != nil {
		return OrderDetail{}, err
	}
	d := OrderDetail{Order: o, Customer: c, Lines: make([]LineDetail, 0, len(rows))}
	for _, r := range rows {
		p := queries.Product{ID: r.ProductID, Sku: r.Sku, Name: r.ProductName, Category: r.Category, SellUnit: r.SellUnit,
			CatchWeight: r.CatchWeight, ApproxCaseWeight: r.ApproxCaseWeight, BasePriceCents: r.BasePriceCents}
		line := queries.OrderLine{ID: r.ID, OrderID: r.OrderID, ProductID: r.ProductID, OrderedQty: r.OrderedQty,
			UnitPriceCents: r.UnitPriceCents, PriceOverridden: r.PriceOverridden, EstWeight: r.EstWeight, ShippedWeight: r.ShippedWeight,
			DeliveredQty: r.DeliveredQty, DeliveredWeight: r.DeliveredWeight, ShortageNote: r.ShortageNote}
		dl := domain.Line{Product: productOf(p), OrderedQty: domain.Hundredths(line.OrderedQty), UnitPrice: domain.Cents(line.UnitPriceCents),
			EstWeight: hPtr(line.EstWeight), ShippedWeight: hPtr(line.ShippedWeight), DeliveredQty: hPtr(line.DeliveredQty), DeliveredWeight: hPtr(line.DeliveredWeight)}
		_, src := dl.BillableQty()
		amt := int64(domain.LineAmount(dl))
		d.Lines = append(d.Lines, LineDetail{Line: line, Product: p, AmountCents: amt, AmountSource: src})
		d.TotalCents += amt
	}
	if stop, err := q.GetActiveStopForOrder(ctx, id); err == nil {
		d.RouteID, d.StopID, d.RouteStatus = &stop.RouteID, &stop.StopID, stop.RouteStatus
	}
	return d, nil
}
```

`DeleteStop` is defined in Task 7's `routes.sql`. To keep this task compiling on its own, add to `sql/queries/orders.sql` now:
```sql
-- name: DeleteStop :exec
DELETE FROM delivery_stops WHERE id = ?;
```
and regenerate. Task 7 must **not** define it again.

- [ ] **Step 5: Run tests to verify they pass**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go tool sqlc generate && go test ./internal/service/ 2>&1 | tail -5`
Expected: `ok`. Generated row field names for `ListOrderLines` are `Sku`, `ProductName`, `Category`, `SellUnit`, `CatchWeight`, `ApproxCaseWeight`, `BasePriceCents`; for `ListOrders` they are `CustomerName`, `LineCount`. Match whatever sqlc emitted.

- [ ] **Step 6: Commit**

```bash
cd ~/deepcutsCRM && git add -A && git commit -q -m "Add orders service: lines with frozen prices, transitions, finalize" && git push
```

---

### Task 7: Routes service: routes, stops, scheduling, day view

**Files:**
- Create: `sql/queries/routes.sql`, `internal/service/routes.go`, `internal/service/routes_test.go`
- Generated: `internal/db/queries/routes.sql.go`

**Interfaces:**
- Produces:
  - `type RouteInput struct { RouteDate string; DriverUserID int64; TruckLabel string }`
  - `type StopDetail struct { Stop queries.DeliveryStop; Order queries.Order; Customer queries.Customer; LineCount int64 }`
  - `type RouteDetail struct { Route queries.DeliveryRoute; DriverName string; Stops []StopDetail }`
  - `type DayView struct { Date string; Unscheduled []queries.ListUnscheduledOrdersForDateRow; Routes []RouteDetail }`
  - `CreateRoute(ctx, RouteInput) (RouteDetail, error)`, `GetRoute(ctx, id) (RouteDetail, error)`, `DayView(ctx, date string) (DayView, error)`
  - `AddStop(ctx, routeID, orderID int64) (RouteDetail, error)` — schedules the order (confirmed→scheduled), appends at the end
  - `RemoveStop(ctx, routeID, stopID int64) (RouteDetail, error)` — only while planned; order→confirmed
  - `ReorderStops(ctx, routeID int64, stopIDs []int64) (RouteDetail, error)` — must be a permutation of the route's stops
  - `RouteOut(ctx, routeID) (RouteDetail, error)`, `RouteComplete(ctx, routeID) (RouteDetail, error)`
  - `loadRoute(ctx, q, id) (RouteDetail, error)` — shared with driver service
  - `ListDrivers(ctx) ([]queries.User, error)`
  - Query names: `CreateRoute, GetRoute, ListRoutesByDate, UpdateRouteStatus, CreateStop, GetStop, ListStopsForRoute, UpdateStopSequence, MaxStopSequence, CountPendingStops, UpdateStopDelivered, UpdateStopSkipped, GetRouteForDriverDate, CreateDriverAction, GetDriverAction`

- [ ] **Step 1: Write the queries**

`sql/queries/routes.sql`:
```sql
-- name: CreateRoute :one
INSERT INTO delivery_routes (route_date, driver_user_id, truck_label, status, created_at)
VALUES (?, ?, ?, 'planned', ?) RETURNING *;

-- name: GetRoute :one
SELECT * FROM delivery_routes WHERE id = ?;

-- name: ListRoutesByDate :many
SELECT * FROM delivery_routes WHERE route_date = ? ORDER BY id;

-- name: UpdateRouteStatus :exec
UPDATE delivery_routes SET status = ?, out_at = ?, completed_at = ? WHERE id = ?;

-- name: CreateStop :one
INSERT INTO delivery_stops (route_id, order_id, sequence, status) VALUES (?, ?, ?, 'pending') RETURNING *;

-- name: GetStop :one
SELECT * FROM delivery_stops WHERE id = ?;

-- name: ListStopsForRoute :many
SELECT s.*, o.customer_id, o.status AS order_status, o.needs_review, o.requested_delivery_date, o.notes AS order_notes,
  c.name AS customer_name, c.delivery_address, c.phone, c.contact_name, c.delivery_notes,
  (SELECT count(*) FROM order_lines ol WHERE ol.order_id = o.id) AS line_count
FROM delivery_stops s
JOIN orders o ON o.id = s.order_id
JOIN customers c ON c.id = o.customer_id
WHERE s.route_id = ? ORDER BY s.sequence, s.id;

-- name: UpdateStopSequence :exec
UPDATE delivery_stops SET sequence = ? WHERE id = ? AND route_id = ?;

-- name: MaxStopSequence :one
SELECT coalesce(max(sequence), 0) FROM delivery_stops WHERE route_id = ?;

-- name: CountPendingStops :one
SELECT count(*) FROM delivery_stops WHERE route_id = ? AND status = 'pending';

-- name: UpdateStopDelivered :exec
UPDATE delivery_stops SET status = 'delivered', delivered_at = ?, proof_type = ?, proof_ref = ?, driver_note = ? WHERE id = ?;

-- name: UpdateStopSkipped :exec
UPDATE delivery_stops SET status = 'skipped', skip_reason = ?, driver_note = ? WHERE id = ?;

-- name: UpdateStopNote :exec
UPDATE delivery_stops SET driver_note = ? WHERE id = ?;

-- name: GetRouteForDriverDate :one
SELECT * FROM delivery_routes WHERE driver_user_id = ? AND route_date = ? AND status <> 'complete' ORDER BY id LIMIT 1;

-- name: CreateDriverAction :exec
INSERT INTO driver_actions (client_id, stop_id, action_type, payload, received_at) VALUES (?, ?, ?, ?, ?);

-- name: GetDriverAction :one
SELECT * FROM driver_actions WHERE client_id = ?;
```

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go tool sqlc generate && go build ./...`

- [ ] **Step 2: Write the failing tests**

`internal/service/routes_test.go`:
```go
package service

import (
	"testing"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
)

func (f fixture) confirmedOrder(t *testing.T, date string) int64 {
	t.Helper()
	o, err := f.s.CreateOrder(ctx, OrderInput{CustomerID: f.customer, RequestedDeliveryDate: date})
	mustNoErr(t, err)
	_, err = f.s.AddLine(ctx, o.Order.ID, f.brisket, 100)
	mustNoErr(t, err)
	_, err = f.s.ConfirmOrder(ctx, o.Order.ID)
	mustNoErr(t, err)
	return o.Order.ID
}

func (f fixture) driver(t *testing.T, name string) int64 {
	t.Helper()
	u, err := f.s.Q.CreateUser(ctx, queries.CreateUserParams{Realm: "driver", DisplayName: name, CreatedAt: f.s.now()})
	mustNoErr(t, err)
	return u.ID
}

func TestRouteSchedulingAndDayView(t *testing.T) {
	f := newFixture(t)
	drv := f.driver(t, "Sam")
	o1 := f.confirmedOrder(t, "2026-09-12")
	o2 := f.confirmedOrder(t, "2026-09-12")
	o3 := f.confirmedOrder(t, "2026-09-13")

	_, err := f.s.CreateRoute(ctx, RouteInput{RouteDate: "bad", DriverUserID: drv})
	wantCode(t, err, "invalid")
	_, err = f.s.CreateRoute(ctx, RouteInput{RouteDate: "2026-09-12", DriverUserID: 999})
	wantCode(t, err, "not_found")
	r, err := f.s.CreateRoute(ctx, RouteInput{RouteDate: "2026-09-12", DriverUserID: drv, TruckLabel: "Reefer 1"})
	mustNoErr(t, err)
	if r.Route.Status != "planned" || r.DriverName != "Sam" {
		t.Fatalf("route: %+v", r)
	}

	r, err = f.s.AddStop(ctx, r.Route.ID, o1)
	mustNoErr(t, err)
	r, err = f.s.AddStop(ctx, r.Route.ID, o2)
	mustNoErr(t, err)
	if len(r.Stops) != 2 || r.Stops[0].Stop.Sequence != 1 || r.Stops[1].Stop.Sequence != 2 || r.Stops[0].Order.Status != "scheduled" {
		t.Fatalf("stops: %+v", r.Stops)
	}
	_, err = f.s.AddStop(ctx, r.Route.ID, o1)
	wantCode(t, err, "invalid_transition") // already scheduled
	_, err = f.s.AddStop(ctx, r.Route.ID, o3)
	wantCode(t, err, "date_mismatch")

	dv, err := f.s.DayView(ctx, "2026-09-12")
	mustNoErr(t, err)
	if len(dv.Unscheduled) != 0 || len(dv.Routes) != 1 || len(dv.Routes[0].Stops) != 2 {
		t.Fatalf("day view: %+v", dv)
	}

	r, err = f.s.ReorderStops(ctx, r.Route.ID, []int64{r.Stops[1].Stop.ID, r.Stops[0].Stop.ID})
	mustNoErr(t, err)
	if r.Stops[0].Order.ID != o2 || r.Stops[0].Stop.Sequence != 1 {
		t.Fatalf("reorder: %+v", r.Stops)
	}
	_, err = f.s.ReorderStops(ctx, r.Route.ID, []int64{r.Stops[0].Stop.ID})
	wantCode(t, err, "invalid")

	r, err = f.s.RemoveStop(ctx, r.Route.ID, r.Stops[1].Stop.ID)
	mustNoErr(t, err)
	if len(r.Stops) != 1 {
		t.Fatal("stop not removed")
	}
	od, _ := f.s.GetOrder(ctx, o1)
	if od.Order.Status != "confirmed" || od.RouteID != nil {
		t.Fatalf("unscheduled order: %+v", od.Order)
	}
	dv, _ = f.s.DayView(ctx, "2026-09-12")
	if len(dv.Unscheduled) != 1 {
		t.Fatalf("unscheduled should list o1: %+v", dv.Unscheduled)
	}
}

func TestRouteOutLocksAndComplete(t *testing.T) {
	f := newFixture(t)
	drv := f.driver(t, "Sam")
	o1 := f.confirmedOrder(t, "2026-09-12")
	r, _ := f.s.CreateRoute(ctx, RouteInput{RouteDate: "2026-09-12", DriverUserID: drv})
	_, err := f.s.RouteOut(ctx, r.Route.ID)
	wantCode(t, err, "empty_route")
	r, _ = f.s.AddStop(ctx, r.Route.ID, o1)
	r, err = f.s.RouteOut(ctx, r.Route.ID)
	mustNoErr(t, err)
	if r.Route.Status != "out" || !r.Route.OutAt.Valid {
		t.Fatalf("out: %+v", r.Route)
	}
	_, err = f.s.RemoveStop(ctx, r.Route.ID, r.Stops[0].Stop.ID)
	wantCode(t, err, "locked")
	_, err = f.s.AddStop(ctx, r.Route.ID, f.confirmedOrder(t, "2026-09-12"))
	wantCode(t, err, "locked")
	qty := int64(200)
	_, err = f.s.UpdateLine(ctx, o1, mustLine(t, f, o1), LinePatch{OrderedQty: &qty})
	wantCode(t, err, "invalid") // ordered qty locked once out
	shipped := int64(5900)
	_, err = f.s.UpdateLine(ctx, o1, mustLine(t, f, o1), LinePatch{ShippedWeight: &shipped})
	mustNoErr(t, err)
	_, err = f.s.CancelOrder(ctx, o1)
	wantCode(t, err, "locked")
	_, err = f.s.RouteComplete(ctx, r.Route.ID)
	wantCode(t, err, "stops_pending")
}

func mustLine(t *testing.T, f fixture, orderID int64) int64 {
	t.Helper()
	o, err := f.s.GetOrder(ctx, orderID)
	mustNoErr(t, err)
	return o.Lines[0].Line.ID
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/service/ 2>&1 | head -3`
Expected: build failure.

- [ ] **Step 4: Implement routes.go**

`internal/service/routes.go`:
```go
package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
)

type RouteInput struct {
	RouteDate    string
	DriverUserID int64
	TruckLabel   string
}

type StopDetail struct {
	Stop      queries.DeliveryStop
	Order     queries.Order
	Customer  queries.Customer
	LineCount int64
}

type RouteDetail struct {
	Route      queries.DeliveryRoute
	DriverName string
	Stops      []StopDetail
}

type DayView struct {
	Date        string
	Unscheduled []queries.ListUnscheduledOrdersForDateRow
	Routes      []RouteDetail
}

func (s *Service) ListDrivers(ctx context.Context) ([]queries.User, error) {
	return s.Q.ListUsersByRealm(ctx, "driver")
}

func (s *Service) CreateRoute(ctx context.Context, in RouteInput) (RouteDetail, error) {
	if !ValidDate(in.RouteDate) {
		return RouteDetail{}, Invalid(map[string]string{"routeDate": "must be YYYY-MM-DD"})
	}
	var out RouteDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		u, err := q.GetUser(ctx, in.DriverUserID)
		if err != nil || u.Realm != "driver" || !u.Active {
			return NotFound("driver")
		}
		r, err := q.CreateRoute(ctx, queries.CreateRouteParams{RouteDate: in.RouteDate, DriverUserID: in.DriverUserID, TruckLabel: in.TruckLabel, CreatedAt: s.now()})
		if err != nil {
			return err
		}
		out, err = s.loadRoute(ctx, q, r.ID)
		return err
	})
	return out, err
}

func (s *Service) GetRoute(ctx context.Context, id int64) (RouteDetail, error) {
	return s.loadRoute(ctx, s.Q, id)
}

func (s *Service) DayView(ctx context.Context, date string) (DayView, error) {
	if !ValidDate(date) {
		return DayView{}, Invalid(map[string]string{"date": "must be YYYY-MM-DD"})
	}
	un, err := s.Q.ListUnscheduledOrdersForDate(ctx, date)
	if err != nil {
		return DayView{}, err
	}
	routes, err := s.Q.ListRoutesByDate(ctx, date)
	if err != nil {
		return DayView{}, err
	}
	dv := DayView{Date: date, Unscheduled: un, Routes: []RouteDetail{}}
	for _, r := range routes {
		rd, err := s.loadRoute(ctx, s.Q, r.ID)
		if err != nil {
			return DayView{}, err
		}
		dv.Routes = append(dv.Routes, rd)
	}
	return dv, nil
}

func (s *Service) AddStop(ctx context.Context, routeID, orderID int64) (RouteDetail, error) {
	var out RouteDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		r, err := q.GetRoute(ctx, routeID)
		if err != nil {
			return notFoundIf(err, "route")
		}
		if r.Status != string(domain.RoutePlanned) {
			return Conflict("locked", "stops cannot be added once the route is out")
		}
		o, err := q.GetOrder(ctx, orderID)
		if err != nil {
			return notFoundIf(err, "order")
		}
		if o.RequestedDeliveryDate != r.RouteDate {
			return Conflict("date_mismatch", "order is for a different delivery date than the route")
		}
		if err := s.transitionOrder(ctx, q, o, domain.OrderScheduled); err != nil {
			return err
		}
		maxSeq, err := q.MaxStopSequence(ctx, routeID)
		if err != nil {
			return err
		}
		if _, err := q.CreateStop(ctx, queries.CreateStopParams{RouteID: routeID, OrderID: orderID, Sequence: maxSeq + 1}); err != nil {
			return err
		}
		out, err = s.loadRoute(ctx, q, routeID)
		return err
	})
	return out, err
}

func (s *Service) RemoveStop(ctx context.Context, routeID, stopID int64) (RouteDetail, error) {
	var out RouteDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		r, err := q.GetRoute(ctx, routeID)
		if err != nil {
			return notFoundIf(err, "route")
		}
		if r.Status != string(domain.RoutePlanned) {
			return Conflict("locked", "stops cannot be removed once the route is out")
		}
		st, err := q.GetStop(ctx, stopID)
		if err != nil || st.RouteID != routeID {
			return NotFound("stop")
		}
		o, err := q.GetOrder(ctx, st.OrderID)
		if err != nil {
			return err
		}
		if err := s.transitionOrder(ctx, q, o, domain.OrderConfirmed); err != nil {
			return err
		}
		if err := q.DeleteStop(ctx, stopID); err != nil {
			return err
		}
		out, err = s.loadRoute(ctx, q, routeID)
		return err
	})
	return out, err
}

func (s *Service) ReorderStops(ctx context.Context, routeID int64, stopIDs []int64) (RouteDetail, error) {
	var out RouteDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		r, err := q.GetRoute(ctx, routeID)
		if err != nil {
			return notFoundIf(err, "route")
		}
		if r.Status != string(domain.RoutePlanned) {
			return Conflict("locked", "stop order is fixed once the route is out")
		}
		existing, err := q.ListStopsForRoute(ctx, routeID)
		if err != nil {
			return err
		}
		have := map[int64]bool{}
		for _, st := range existing {
			have[st.ID] = true
		}
		seen := map[int64]bool{}
		for _, id := range stopIDs {
			if !have[id] || seen[id] {
				return Invalid(map[string]string{"stopIds": "must list each stop on the route exactly once"})
			}
			seen[id] = true
		}
		if len(seen) != len(have) {
			return Invalid(map[string]string{"stopIds": "must list each stop on the route exactly once"})
		}
		for i, id := range stopIDs {
			if err := q.UpdateStopSequence(ctx, queries.UpdateStopSequenceParams{Sequence: int64(i + 1), ID: id, RouteID: routeID}); err != nil {
				return err
			}
		}
		out, err = s.loadRoute(ctx, q, routeID)
		return err
	})
	return out, err
}

func (s *Service) RouteOut(ctx context.Context, routeID int64) (RouteDetail, error) {
	var out RouteDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		r, err := q.GetRoute(ctx, routeID)
		if err != nil {
			return notFoundIf(err, "route")
		}
		if !domain.RouteStatus(r.Status).CanTransition(domain.RouteOut) {
			return Conflict("invalid_transition", fmt.Sprintf("route cannot go from %s to out", r.Status))
		}
		n, err := q.MaxStopSequence(ctx, routeID)
		if err != nil {
			return err
		}
		if n == 0 {
			return Conflict("empty_route", "add at least one stop before marking the route out")
		}
		now := s.now()
		if err := q.UpdateRouteStatus(ctx, queries.UpdateRouteStatusParams{Status: string(domain.RouteOut), OutAt: sql.NullTime{Time: now, Valid: true}, ID: routeID}); err != nil {
			return err
		}
		out, err = s.loadRoute(ctx, q, routeID)
		return err
	})
	return out, err
}

func (s *Service) RouteComplete(ctx context.Context, routeID int64) (RouteDetail, error) {
	var out RouteDetail
	err := s.Tx(ctx, func(q *queries.Queries) error {
		r, err := q.GetRoute(ctx, routeID)
		if err != nil {
			return notFoundIf(err, "route")
		}
		if !domain.RouteStatus(r.Status).CanTransition(domain.RouteComplete) {
			return Conflict("invalid_transition", fmt.Sprintf("route cannot go from %s to complete", r.Status))
		}
		pending, err := q.CountPendingStops(ctx, routeID)
		if err != nil {
			return err
		}
		if pending > 0 {
			return Conflict("stops_pending", fmt.Sprintf("%d stops are still pending", pending))
		}
		now := s.now()
		if err := q.UpdateRouteStatus(ctx, queries.UpdateRouteStatusParams{Status: string(domain.RouteComplete), OutAt: r.OutAt, CompletedAt: sql.NullTime{Time: now, Valid: true}, ID: routeID}); err != nil {
			return err
		}
		out, err = s.loadRoute(ctx, q, routeID)
		return err
	})
	return out, err
}

func (s *Service) loadRoute(ctx context.Context, q *queries.Queries, id int64) (RouteDetail, error) {
	r, err := q.GetRoute(ctx, id)
	if err != nil {
		return RouteDetail{}, notFoundIf(err, "route")
	}
	u, err := q.GetUser(ctx, r.DriverUserID)
	if err != nil {
		return RouteDetail{}, err
	}
	rows, err := q.ListStopsForRoute(ctx, id)
	if err != nil {
		return RouteDetail{}, err
	}
	rd := RouteDetail{Route: r, DriverName: u.DisplayName, Stops: make([]StopDetail, 0, len(rows))}
	for _, x := range rows {
		rd.Stops = append(rd.Stops, StopDetail{
			Stop: queries.DeliveryStop{ID: x.ID, RouteID: x.RouteID, OrderID: x.OrderID, Sequence: x.Sequence, Status: x.Status,
				DeliveredAt: x.DeliveredAt, ProofType: x.ProofType, ProofRef: x.ProofRef, SkipReason: x.SkipReason, DriverNote: x.DriverNote},
			Order: queries.Order{ID: x.OrderID, CustomerID: x.CustomerID, Status: x.OrderStatus, NeedsReview: x.NeedsReview,
				RequestedDeliveryDate: x.RequestedDeliveryDate, Notes: x.OrderNotes},
			Customer: queries.Customer{ID: x.CustomerID, Name: x.CustomerName, DeliveryAddress: x.DeliveryAddress, Phone: x.Phone,
				ContactName: x.ContactName, DeliveryNotes: x.DeliveryNotes},
			LineCount: x.LineCount,
		})
	}
	return rd, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/service/ 2>&1 | tail -5`
Expected: `ok`

- [ ] **Step 6: Commit**

```bash
cd ~/deepcutsCRM && git add -A && git commit -q -m "Add routes service: scheduling, stops, day view, out/complete" && git push
```

---

### Task 8: Proof storage and driver service (idempotent actions)

**Files:**
- Create: `internal/storage/local.go`, `internal/storage/local_test.go`, `internal/service/driver.go`, `internal/service/driver_test.go`

**Interfaces:**
- Produces:
  - `storage.Local{Dir string}`; `func NewLocal(dir string) (*Local, error)`; `func (l *Local) Save(kind string, data []byte) (ref string, err error)` — sniffs PNG/JPEG, writes `<dir>/<kind>/<uuid>.<ext>`, returns `kind/<uuid>.<ext>`; `func (l *Local) Open(ref string) (io.ReadCloser, string /*content type*/, error)` rejecting refs with `..` or absolute paths.
  - `type ProofStore interface { Save(kind string, data []byte) (string, error) }` in package service; `Service.Proofs ProofStore` field set by the caller (nil allowed in tests that never deliver with an image).
  - `type DriverStop struct { Stop queries.DeliveryStop; Customer queries.Customer; Order OrderDetail }`
  - `type DriverRoute struct { Route queries.DeliveryRoute; Stops []DriverStop }`
  - `DriverRouteForDate(ctx, driverUserID int64, date string) (DriverRoute, error)` — 404 when none
  - `ApplyDriverAction(ctx, driverUserID, stopID int64, a domain.DriverAction) (applied bool, stop DriverStop, err error)`
  - `DriverCompleteRoute(ctx, driverUserID, routeID int64) (DriverRoute, error)`
  - `loadDriverStop(ctx, q, stopID) (DriverStop, error)`

- [ ] **Step 1: Write the failing storage test**

`internal/storage/local_test.go`:
```go
package storage

import (
	"io"
	"testing"
)

var png = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0}
var jpg = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0}

func TestSaveAndOpen(t *testing.T) {
	l, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ref, err := l.Save("proofs", png)
	if err != nil {
		t.Fatal(err)
	}
	rc, ct, err := l.Open(ref)
	if err != nil || ct != "image/png" {
		t.Fatalf("open: %v %s", err, ct)
	}
	b, _ := io.ReadAll(rc)
	rc.Close()
	if string(b) != string(png) {
		t.Fatal("content mismatch")
	}
	ref2, _ := l.Save("proofs", jpg)
	if _, ct, _ := l.Open(ref2); ct != "image/jpeg" {
		t.Fatalf("jpeg content type: %s", ct)
	}
	if _, err := l.Save("proofs", []byte("not an image")); err == nil {
		t.Fatal("non-image must be rejected")
	}
	for _, bad := range []string{"../etc/passwd", "/etc/passwd", "proofs/../../x.png"} {
		if _, _, err := l.Open(bad); err == nil {
			t.Fatalf("path %q must be rejected", bad)
		}
	}
}
```

- [ ] **Step 2: Implement storage/local.go**

`internal/storage/local.go`:
```go
// Package storage saves proof images under a local directory. S3 can replace it later.
package storage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type Local struct{ Dir string }

func NewLocal(dir string) (*Local, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Local{Dir: dir}, nil
}

func sniff(data []byte) (ext, contentType string, ok bool) {
	switch {
	case bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}):
		return "png", "image/png", true
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return "jpg", "image/jpeg", true
	}
	return "", "", false
}

func (l *Local) Save(kind string, data []byte) (string, error) {
	ext, _, ok := sniff(data)
	if !ok {
		return "", errors.New("proof must be a PNG or JPEG image")
	}
	if strings.ContainsAny(kind, "/\\.") {
		return "", errors.New("invalid kind")
	}
	if err := os.MkdirAll(filepath.Join(l.Dir, kind), 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s.%s", uuid.NewString(), ext)
	if err := os.WriteFile(filepath.Join(l.Dir, kind, name), data, 0o644); err != nil {
		return "", err
	}
	return kind + "/" + name, nil
}

func (l *Local) Open(ref string) (io.ReadCloser, string, error) {
	if filepath.IsAbs(ref) || strings.Contains(ref, "..") {
		return nil, "", errors.New("invalid ref")
	}
	ct := "application/octet-stream"
	switch filepath.Ext(ref) {
	case ".png":
		ct = "image/png"
	case ".jpg":
		ct = "image/jpeg"
	}
	f, err := os.Open(filepath.Join(l.Dir, filepath.FromSlash(ref)))
	if err != nil {
		return nil, "", err
	}
	return f, ct, nil
}
```

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/storage/`
Expected: `ok`

- [ ] **Step 3: Write the failing driver service tests**

`internal/service/driver_test.go`:
```go
package service

import (
	"testing"

	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
)

type memProofs struct{ saved int }

func (m *memProofs) Save(kind string, data []byte) (string, error) { m.saved++; return "proofs/x.png", nil }

const cid1 = "11111111-1111-4111-8111-111111111111"
const cid2 = "22222222-2222-4222-8222-222222222222"

func outRoute(t *testing.T) (fixture, int64, int64, int64) {
	t.Helper()
	f := newFixture(t)
	drv := f.driver(t, "Sam")
	o := f.confirmedOrder(t, "2026-09-10") // today in the fixture clock
	r, _ := f.s.CreateRoute(ctx, RouteInput{RouteDate: "2026-09-10", DriverUserID: drv})
	r, _ = f.s.AddStop(ctx, r.Route.ID, o)
	shipped := int64(5900)
	_, err := f.s.UpdateLine(ctx, o, mustLine(t, f, o), LinePatch{ShippedWeight: &shipped})
	mustNoErr(t, err)
	r, err = f.s.RouteOut(ctx, r.Route.ID)
	mustNoErr(t, err)
	return f, drv, r.Route.ID, r.Stops[0].Stop.ID
}

func TestDriverRouteForDate(t *testing.T) {
	f, drv, routeID, _ := outRoute(t)
	dr, err := f.s.DriverRouteForDate(ctx, drv, f.s.Today())
	mustNoErr(t, err)
	if dr.Route.ID != routeID || len(dr.Stops) != 1 || dr.Stops[0].Customer.Name != "Blue Plate" || len(dr.Stops[0].Order.Lines) != 1 {
		t.Fatalf("driver route: %+v", dr)
	}
	_, err = f.s.DriverRouteForDate(ctx, drv, "2026-01-01")
	wantCode(t, err, "not_found")
	other := f.driver(t, "Riley")
	_, err = f.s.DriverRouteForDate(ctx, other, f.s.Today())
	wantCode(t, err, "not_found")
}

func TestDeliverIsIdempotentAndDefaultsDelivered(t *testing.T) {
	f, drv, _, stopID := outRoute(t)
	f.s.Proofs = &memProofs{}
	a := domain.DriverAction{ClientID: cid1, Type: domain.ActionDeliver, Proof: &domain.Proof{Type: domain.ProofName, Name: "Pat"}, Note: "left at dock"}
	applied, st, err := f.s.ApplyDriverAction(ctx, drv, stopID, a)
	mustNoErr(t, err)
	if !applied || st.Stop.Status != "delivered" || st.Stop.ProofType.String != "name" || st.Stop.ProofRef.String != "Pat" || st.Stop.DriverNote != "left at dock" {
		t.Fatalf("deliver: applied=%v %+v", applied, st.Stop)
	}
	l := st.Order.Lines[0].Line
	if !l.DeliveredQty.Valid || l.DeliveredQty.Int64 != 100 || l.DeliveredWeight.Int64 != 5900 || st.Order.Order.Status != "delivered" || st.Order.Order.NeedsReview {
		t.Fatalf("defaults: %+v order=%+v", l, st.Order.Order)
	}
	applied, _, err = f.s.ApplyDriverAction(ctx, drv, stopID, a)
	mustNoErr(t, err)
	if applied {
		t.Fatal("replay must not apply")
	}
	_, _, err = f.s.ApplyDriverAction(ctx, drv, stopID, domain.DriverAction{ClientID: cid2, Type: domain.ActionDeliver, Proof: a.Proof})
	wantCode(t, err, "invalid_transition") // already delivered
}

func TestDeliverWithImageProofSaves(t *testing.T) {
	f, drv, _, stopID := outRoute(t)
	m := &memProofs{}
	f.s.Proofs = m
	a := domain.DriverAction{ClientID: cid1, Type: domain.ActionDeliver, Proof: &domain.Proof{Type: domain.ProofSignature, Data: []byte{0x89, 'P', 'N', 'G'}}}
	_, st, err := f.s.ApplyDriverAction(ctx, drv, stopID, a)
	mustNoErr(t, err)
	if m.saved != 1 || st.Stop.ProofRef.String != "proofs/x.png" {
		t.Fatalf("proof not saved: %+v", st.Stop)
	}
}

func TestAdjustSetsNeedsReview(t *testing.T) {
	f, drv, _, stopID := outRoute(t)
	f.s.Proofs = &memProofs{}
	lineID := mustLine(t, f, f.orderOf(t, stopID))
	short := domain.Hundredths(5000)
	note := "one case rejected, temp"
	adj := domain.DriverAction{ClientID: cid1, Type: domain.ActionAdjust, Lines: []domain.LineAdjustment{{LineID: lineID, DeliveredWeight: &short, ShortageNote: &note}}}
	applied, st, err := f.s.ApplyDriverAction(ctx, drv, stopID, adj)
	mustNoErr(t, err)
	if !applied || st.Order.Lines[0].Line.DeliveredWeight.Int64 != 5000 || st.Order.Lines[0].Line.ShortageNote != note {
		t.Fatalf("adjust before deliver: %+v", st.Order.Lines[0].Line)
	}
	_, st, err = f.s.ApplyDriverAction(ctx, drv, stopID, domain.DriverAction{ClientID: cid2, Type: domain.ActionDeliver, Proof: &domain.Proof{Type: domain.ProofName, Name: "Pat"}})
	mustNoErr(t, err)
	if st.Order.Lines[0].Line.DeliveredWeight.Int64 != 5000 || !st.Order.Order.NeedsReview {
		t.Fatalf("deliver must keep adjustment and flag review: %+v %+v", st.Order.Lines[0].Line, st.Order.Order)
	}
}

func (f fixture) orderOf(t *testing.T, stopID int64) int64 {
	t.Helper()
	st, err := f.s.Q.GetStop(ctx, stopID)
	mustNoErr(t, err)
	return st.OrderID
}

func TestSkipReturnsOrderToConfirmed(t *testing.T) {
	f, drv, routeID, stopID := outRoute(t)
	orderID := f.orderOf(t, stopID)
	_, st, err := f.s.ApplyDriverAction(ctx, drv, stopID, domain.DriverAction{ClientID: cid1, Type: domain.ActionSkip, SkipReason: "closed"})
	mustNoErr(t, err)
	if st.Stop.Status != "skipped" || st.Stop.SkipReason != "closed" || st.Order.Order.Status != "confirmed" || !st.Order.Order.NeedsReview {
		t.Fatalf("skip: %+v %+v", st.Stop, st.Order.Order)
	}
	dr, err := f.s.DriverCompleteRoute(ctx, drv, routeID)
	mustNoErr(t, err)
	if dr.Route.Status != "complete" {
		t.Fatal(dr.Route.Status)
	}
	// reschedule onto a new route works: the skipped stop stays, a new pending stop is created
	r2, _ := f.s.CreateRoute(ctx, RouteInput{RouteDate: "2026-09-10", DriverUserID: drv})
	r2, err = f.s.AddStop(ctx, r2.Route.ID, orderID)
	mustNoErr(t, err)
	if len(r2.Stops) != 1 || r2.Stops[0].Order.Status != "scheduled" {
		t.Fatalf("reschedule: %+v", r2.Stops)
	}
}

func TestDriverCannotTouchOtherRoute(t *testing.T) {
	f, _, routeID, stopID := outRoute(t)
	other := f.driver(t, "Riley")
	_, _, err := f.s.ApplyDriverAction(ctx, other, stopID, domain.DriverAction{ClientID: cid1, Type: domain.ActionSkip, SkipReason: "x"})
	wantCode(t, err, "not_found")
	_, err = f.s.DriverCompleteRoute(ctx, other, routeID)
	wantCode(t, err, "not_found")
}
```

Also add:

```go
func TestAdjustRejectsForeignLine(t *testing.T) {
	f, drv, _, stopID := outRoute(t)
	w := domain.Hundredths(1)
	_, _, err := f.s.ApplyDriverAction(ctx, drv, stopID, domain.DriverAction{ClientID: cid1, Type: domain.ActionAdjust, Lines: []domain.LineAdjustment{{LineID: 999, DeliveredWeight: &w}}})
	wantCode(t, err, "invalid")
}
```

- [ ] **Step 4: Run tests to verify they fail**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/service/ 2>&1 | head -3`
Expected: build failure.

- [ ] **Step 5: Implement driver.go**

`internal/service/driver.go`:
```go
package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
)

// ProofStore persists signature/photo images and returns a reference.
type ProofStore interface {
	Save(kind string, data []byte) (string, error)
}

type DriverStop struct {
	Stop     queries.DeliveryStop
	Customer queries.Customer
	Order    OrderDetail
}

type DriverRoute struct {
	Route queries.DeliveryRoute
	Stops []DriverStop
}

func (s *Service) DriverRouteForDate(ctx context.Context, driverUserID int64, date string) (DriverRoute, error) {
	r, err := s.Q.GetRouteForDriverDate(ctx, queries.GetRouteForDriverDateParams{DriverUserID: driverUserID, RouteDate: date})
	if err != nil {
		return DriverRoute{}, notFoundIf(err, "route")
	}
	return s.loadDriverRoute(ctx, s.Q, r)
}

func (s *Service) loadDriverRoute(ctx context.Context, q *queries.Queries, r queries.DeliveryRoute) (DriverRoute, error) {
	stops, err := q.ListStopsForRoute(ctx, r.ID)
	if err != nil {
		return DriverRoute{}, err
	}
	out := DriverRoute{Route: r, Stops: make([]DriverStop, 0, len(stops))}
	for _, st := range stops {
		ds, err := s.loadDriverStop(ctx, q, st.ID)
		if err != nil {
			return DriverRoute{}, err
		}
		out.Stops = append(out.Stops, ds)
	}
	return out, nil
}

func (s *Service) loadDriverStop(ctx context.Context, q *queries.Queries, stopID int64) (DriverStop, error) {
	st, err := q.GetStop(ctx, stopID)
	if err != nil {
		return DriverStop{}, notFoundIf(err, "stop")
	}
	od, err := s.loadOrder(ctx, q, st.OrderID)
	if err != nil {
		return DriverStop{}, err
	}
	return DriverStop{Stop: st, Customer: od.Customer, Order: od}, nil
}

// ownedStop loads a stop and verifies it belongs to a route driven by driverUserID.
func (s *Service) ownedStop(ctx context.Context, q *queries.Queries, driverUserID, stopID int64) (queries.DeliveryStop, queries.DeliveryRoute, error) {
	st, err := q.GetStop(ctx, stopID)
	if err != nil {
		return st, queries.DeliveryRoute{}, notFoundIf(err, "stop")
	}
	r, err := q.GetRoute(ctx, st.RouteID)
	if err != nil {
		return st, r, err
	}
	if r.DriverUserID != driverUserID {
		return st, r, NotFound("stop")
	}
	return st, r, nil
}

// ApplyDriverAction applies a deliver/adjust/skip action exactly once per ClientID.
func (s *Service) ApplyDriverAction(ctx context.Context, driverUserID, stopID int64, a domain.DriverAction) (bool, DriverStop, error) {
	if errs := a.Validate(); len(errs) > 0 {
		return false, DriverStop{}, Invalid(errs)
	}
	applied := false
	var out DriverStop
	err := s.Tx(ctx, func(q *queries.Queries) error {
		st, r, err := s.ownedStop(ctx, q, driverUserID, stopID)
		if err != nil {
			return err
		}
		if prev, err := q.GetDriverAction(ctx, a.ClientID); err == nil {
			if prev.StopID != stopID {
				return Conflict("client_id_reused", "this action id was already used for another stop")
			}
			out, err = s.loadDriverStop(ctx, q, stopID)
			return err
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if r.Status != string(domain.RouteOut) {
			return Conflict("route_not_out", "the route is not out")
		}
		o, err := q.GetOrder(ctx, st.OrderID)
		if err != nil {
			return err
		}
		switch a.Type {
		case domain.ActionAdjust:
			if st.Status == string(domain.StopSkipped) {
				return Conflict("invalid_transition", "stop was skipped")
			}
			if err := s.applyAdjustments(ctx, q, o, a.Lines); err != nil {
				return err
			}
			if a.Note != "" {
				if err := q.UpdateStopNote(ctx, queries.UpdateStopNoteParams{DriverNote: a.Note, ID: stopID}); err != nil {
					return err
				}
			}
		case domain.ActionDeliver:
			if st.Status != string(domain.StopPending) {
				return Conflict("invalid_transition", "stop is already "+st.Status)
			}
			ref := a.Proof.Name
			if a.Proof.Type != domain.ProofName {
				if s.Proofs == nil {
					return errors.New("proof storage not configured")
				}
				ref, err = s.Proofs.Save("proofs", a.Proof.Data)
				if err != nil {
					return Invalid(map[string]string{"proof": err.Error()})
				}
			}
			if err := s.applyAdjustments(ctx, q, o, a.Lines); err != nil {
				return err
			}
			if err := s.defaultDelivered(ctx, q, o.ID); err != nil {
				return err
			}
			if err := q.UpdateStopDelivered(ctx, queries.UpdateStopDeliveredParams{
				DeliveredAt: sql.NullTime{Time: s.now(), Valid: true}, ProofType: nullStr(string(a.Proof.Type)), ProofRef: nullStr(ref), DriverNote: a.Note, ID: stopID,
			}); err != nil {
				return err
			}
			if err := s.transitionOrder(ctx, q, o, domain.OrderDelivered); err != nil {
				return err
			}
		case domain.ActionSkip:
			if st.Status != string(domain.StopPending) {
				return Conflict("invalid_transition", "stop is already "+st.Status)
			}
			if err := q.UpdateStopSkipped(ctx, queries.UpdateStopSkippedParams{SkipReason: a.SkipReason, DriverNote: a.Note, ID: stopID}); err != nil {
				return err
			}
			if err := s.transitionOrder(ctx, q, o, domain.OrderConfirmed); err != nil {
				return err
			}
			if err := q.SetOrderNeedsReview(ctx, queries.SetOrderNeedsReviewParams{NeedsReview: true, UpdatedAt: s.now(), ID: o.ID}); err != nil {
				return err
			}
		}
		payload, _ := json.Marshal(map[string]any{"type": a.Type, "note": a.Note, "skipReason": a.SkipReason, "lines": len(a.Lines), "proof": proofType(a.Proof)})
		if err := q.CreateDriverAction(ctx, queries.CreateDriverActionParams{ClientID: a.ClientID, StopID: stopID, ActionType: string(a.Type), Payload: string(payload), ReceivedAt: s.now()}); err != nil {
			return err
		}
		applied = true
		out, err = s.loadDriverStop(ctx, q, stopID)
		return err
	})
	return applied, out, err
}

func proofType(p *domain.Proof) string {
	if p == nil {
		return ""
	}
	return string(p.Type)
}

// applyAdjustments writes delivered quantities/weights and notes for the given lines.
func (s *Service) applyAdjustments(ctx context.Context, q *queries.Queries, o queries.Order, adj []domain.LineAdjustment) error {
	for _, a := range adj {
		l, err := q.GetOrderLine(ctx, queries.GetOrderLineParams{ID: a.LineID, OrderID: o.ID})
		if err != nil {
			return Invalid(map[string]string{"lines": fmt.Sprintf("line %d is not on this order", a.LineID)})
		}
		p, err := q.GetProduct(ctx, l.ProductID)
		if err != nil {
			return err
		}
		if a.DeliveredQty != nil {
			l.DeliveredQty = sql.NullInt64{Int64: int64(*a.DeliveredQty), Valid: true}
		}
		if a.DeliveredWeight != nil {
			if !p.CatchWeight {
				return Invalid(map[string]string{"lines": fmt.Sprintf("line %d is not catch-weight", a.LineID)})
			}
			l.DeliveredWeight = sql.NullInt64{Int64: int64(*a.DeliveredWeight), Valid: true}
		}
		if a.ShortageNote != nil {
			l.ShortageNote = *a.ShortageNote
		}
		if err := q.UpdateOrderLine(ctx, queries.UpdateOrderLineParams{
			OrderedQty: l.OrderedQty, UnitPriceCents: l.UnitPriceCents, PriceOverridden: l.PriceOverridden, EstWeight: l.EstWeight,
			ShippedWeight: l.ShippedWeight, DeliveredQty: l.DeliveredQty, DeliveredWeight: l.DeliveredWeight, ShortageNote: l.ShortageNote, ID: l.ID,
		}); err != nil {
			return err
		}
	}
	return nil
}

// defaultDelivered fills delivered_qty from ordered_qty and delivered_weight from shipped_weight
// where the driver did not adjust, and flags the order for review if anything differs.
func (s *Service) defaultDelivered(ctx context.Context, q *queries.Queries, orderID int64) error {
	lines, err := q.ListOrderLines(ctx, orderID)
	if err != nil {
		return err
	}
	review := false
	for _, r := range lines {
		dq, dw := r.DeliveredQty, r.DeliveredWeight
		if !dq.Valid {
			dq = sql.NullInt64{Int64: r.OrderedQty, Valid: true}
		}
		if r.CatchWeight && !dw.Valid && r.ShippedWeight.Valid {
			dw = r.ShippedWeight
		}
		if dq.Int64 != r.OrderedQty || (r.CatchWeight && r.ShippedWeight.Valid && dw.Int64 != r.ShippedWeight.Int64) || r.ShortageNote != "" {
			review = true
		}
		if err := q.UpdateOrderLine(ctx, queries.UpdateOrderLineParams{
			OrderedQty: r.OrderedQty, UnitPriceCents: r.UnitPriceCents, PriceOverridden: r.PriceOverridden, EstWeight: r.EstWeight,
			ShippedWeight: r.ShippedWeight, DeliveredQty: dq, DeliveredWeight: dw, ShortageNote: r.ShortageNote, ID: r.ID,
		}); err != nil {
			return err
		}
	}
	if review {
		return q.SetOrderNeedsReview(ctx, queries.SetOrderNeedsReviewParams{NeedsReview: true, UpdatedAt: s.now(), ID: orderID})
	}
	return nil
}

func (s *Service) DriverCompleteRoute(ctx context.Context, driverUserID, routeID int64) (DriverRoute, error) {
	r, err := s.Q.GetRoute(ctx, routeID)
	if err != nil || r.DriverUserID != driverUserID {
		return DriverRoute{}, NotFound("route")
	}
	if _, err := s.RouteComplete(ctx, routeID); err != nil {
		return DriverRoute{}, err
	}
	r, _ = s.Q.GetRoute(ctx, routeID)
	return s.loadDriverRoute(ctx, s.Q, r)
}
```

Add the field `Proofs ProofStore` to `Service` in `store.go`.

- [ ] **Step 6: Run tests to verify they pass**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/service/ ./internal/storage/ 2>&1 | tail -5`
Expected: `ok` for both.

- [ ] **Step 7: Commit**

```bash
cd ~/deepcutsCRM && git add -A && git commit -q -m "Add proof storage and idempotent driver actions" && git push
```

---

### Task 9: Auth: users, sessions, PIN/password login, rate limiting, realm middleware, user CLI

**Files:**
- Create: `internal/auth/auth.go`, `internal/auth/limiter.go`, `internal/auth/middleware.go`, `internal/auth/auth_test.go`, `cmd/deepcuts/cmd_user.go`
- Modify: `cmd/deepcuts/main.go` (remove `runUser` stub line)

**Interfaces:**
- Produces:
  - `type Auth struct { Q *queries.Queries; Now func() time.Time; Limiter *Limiter; SessionTTL time.Duration }`; `func New(q *queries.Queries) *Auth` (TTL 30 days, limiter 10 attempts per 15 min)
  - `CreateOfficeUser(ctx, email, displayName, password string) (queries.User, error)`; `CreateDriver(ctx, displayName, pin string) (queries.User, error)`; `SetDriverPIN(ctx, displayName, pin string) error`; `DeactivateUser(ctx, emailOrName string) error`
  - `LoginOffice(ctx, email, password, ip string) (Session, error)`; `LoginDriver(ctx, userID int64, pin, ip string) (Session, error)`; `Logout(ctx, sessionID string) error`
  - `type Session struct { ID string; UserID int64; Realm string; DisplayName string; ExpiresAt time.Time }`
  - `Lookup(ctx, sessionID string) (Session, bool)`
  - `Middleware(realm string) func(http.Handler) http.Handler` — reads cookie `deepcuts_session`, verifies realm, stores `Session` in context; 401 JSON otherwise
  - `func SessionFrom(ctx) (Session, bool)`; `func SetCookie(w, session, secure bool)`; `func ClearCookie(w, secure bool)`; `const CookieName = "deepcuts_session"`
  - `type Limiter` with `Allow(key string) bool` and `Reset(key string)`; per-user key `user:<id>` and per-IP key `ip:<ip>`.
  - `ValidPIN(string) bool` — exactly 6 ASCII digits.

- [ ] **Step 1: Write the failing tests**

`internal/auth/auth_test.go`:
```go
package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
)

var ctx = context.Background()

func newAuth(t *testing.T) *Auth {
	t.Helper()
	d, err := db.OpenAndMigrate(ctx, filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	a := New(queries.New(d))
	now := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)
	a.Now = func() time.Time { return now }
	return a
}

func TestOfficeLogin(t *testing.T) {
	a := newAuth(t)
	_, err := a.CreateOfficeUser(ctx, "o@x.com", "Olive", "short")
	if err == nil {
		t.Fatal("password under 8 chars must be rejected")
	}
	u, err := a.CreateOfficeUser(ctx, "o@x.com", "Olive", "correct horse")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.CreateOfficeUser(ctx, "o@x.com", "Dup", "correct horse"); err == nil {
		t.Fatal("duplicate email must fail")
	}
	if _, err := a.LoginOffice(ctx, "o@x.com", "wrong", "1.1.1.1"); err == nil {
		t.Fatal("wrong password must fail")
	}
	s, err := a.LoginOffice(ctx, "O@X.com", "correct horse", "1.1.1.1")
	if err != nil || s.UserID != u.ID || s.Realm != "office" || s.DisplayName != "Olive" {
		t.Fatalf("login: %v %+v", err, s)
	}
	got, ok := a.Lookup(ctx, s.ID)
	if !ok || got.UserID != u.ID {
		t.Fatal("lookup failed")
	}
	if err := a.Logout(ctx, s.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := a.Lookup(ctx, s.ID); ok {
		t.Fatal("session should be gone")
	}
}

func TestDriverPinAndRateLimit(t *testing.T) {
	a := newAuth(t)
	if _, err := a.CreateDriver(ctx, "Sam", "12345"); err == nil {
		t.Fatal("5 digit pin must be rejected")
	}
	u, err := a.CreateDriver(ctx, "Sam", "123456")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err := a.LoginDriver(ctx, u.ID, "000000", "9.9.9.9"); err == nil {
			t.Fatal("wrong pin must fail")
		}
	}
	if _, err := a.LoginDriver(ctx, u.ID, "123456", "9.9.9.9"); err == nil {
		t.Fatal("11th attempt must be rate limited even with the right pin")
	}
	a.Limiter.Reset("user:" + itoa(u.ID))
	a.Limiter.Reset("ip:9.9.9.9")
	s, err := a.LoginDriver(ctx, u.ID, "123456", "9.9.9.9")
	if err != nil || s.Realm != "driver" {
		t.Fatalf("login after reset: %v", err)
	}
	if err := a.SetDriverPIN(ctx, "Sam", "654321"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.LoginDriver(ctx, u.ID, "123456", "8.8.8.8"); err == nil {
		t.Fatal("old pin must fail after reset")
	}
	if err := a.DeactivateUser(ctx, "Sam"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.LoginDriver(ctx, u.ID, "654321", "7.7.7.7"); err == nil {
		t.Fatal("inactive user must not log in")
	}
}

func TestSessionExpiry(t *testing.T) {
	a := newAuth(t)
	u, _ := a.CreateDriver(ctx, "Sam", "123456")
	s, _ := a.LoginDriver(ctx, u.ID, "123456", "1.1.1.1")
	a.Now = func() time.Time { return time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC) }
	if _, ok := a.Lookup(ctx, s.ID); ok {
		t.Fatal("expired session must not resolve")
	}
}

func TestMiddlewareRealms(t *testing.T) {
	a := newAuth(t)
	u, _ := a.CreateDriver(ctx, "Sam", "123456")
	s, _ := a.LoginDriver(ctx, u.ID, "123456", "1.1.1.1")
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, _ := SessionFrom(r.Context())
		w.Write([]byte(sess.DisplayName))
	})
	call := func(realm, cookie string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "/", nil)
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: CookieName, Value: cookie})
		}
		rec := httptest.NewRecorder()
		a.Middleware(realm)(ok).ServeHTTP(rec, req)
		return rec
	}
	if rec := call("driver", s.ID); rec.Code != 200 || rec.Body.String() != "Sam" {
		t.Fatalf("driver realm: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call("office", s.ID); rec.Code != 401 {
		t.Fatalf("cross realm must be 401, got %d", rec.Code)
	}
	if rec := call("driver", ""); rec.Code != 401 {
		t.Fatalf("no cookie must be 401, got %d", rec.Code)
	}
	if rec := call("driver", "garbage"); rec.Code != 401 {
		t.Fatalf("bad cookie must be 401, got %d", rec.Code)
	}
}

func itoa(i int64) string { return fmtInt(i) }
```

Add to the test file: `import "strconv"` and `func fmtInt(i int64) string { return strconv.FormatInt(i, 10) }`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/auth/ 2>&1 | head -3`
Expected: build failure.

- [ ] **Step 3: Implement limiter.go, auth.go, middleware.go**

`internal/auth/limiter.go`:
```go
package auth

import (
	"sync"
	"time"
)

// Limiter is a fixed-window counter: at most max attempts per key per window.
type Limiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	now    func() time.Time
	hits   map[string]entry
}

type entry struct {
	count int
	start time.Time
}

func NewLimiter(max int, window time.Duration, now func() time.Time) *Limiter {
	return &Limiter{max: max, window: window, now: now, hits: map[string]entry{}}
}

// Allow records an attempt and reports whether it is within the limit.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	t := l.now()
	e := l.hits[key]
	if e.start.IsZero() || t.Sub(e.start) > l.window {
		e = entry{start: t}
	}
	e.count++
	l.hits[key] = e
	return e.count <= l.max
}

func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}
```

`internal/auth/auth.go`:
```go
// Package auth owns users, credentials, sessions and the realm middleware.
package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

const (
	RealmOffice   = "office"
	RealmDriver   = "driver"
	RealmCustomer = "customer"
)

type Auth struct {
	Q          *queries.Queries
	Now        func() time.Time
	Limiter    *Limiter
	SessionTTL time.Duration
}

func New(q *queries.Queries) *Auth {
	a := &Auth{Q: q, Now: time.Now, SessionTTL: 30 * 24 * time.Hour}
	a.Limiter = NewLimiter(10, 15*time.Minute, func() time.Time { return a.Now() })
	return a
}

type Session struct {
	ID          string
	UserID      int64
	Realm       string
	DisplayName string
	ExpiresAt   time.Time
}

var pinRe = regexp.MustCompile(`^[0-9]{6}$`)

func ValidPIN(p string) bool { return pinRe.MatchString(p) }

var ErrBadCredentials = service.Unauthorized("invalid credentials")
var ErrRateLimited = &service.Error{Status: 429, Code: "rate_limited", Message: "too many attempts, try again later"}

func (a *Auth) CreateOfficeUser(ctx context.Context, email, displayName, password string) (queries.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !strings.Contains(email, "@") {
		return queries.User{}, service.Invalid(map[string]string{"email": "must be an email address"})
	}
	if len(password) < 8 {
		return queries.User{}, service.Invalid(map[string]string{"password": "must be at least 8 characters"})
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return queries.User{}, err
	}
	u, err := a.Q.CreateUser(ctx, queries.CreateUserParams{
		Realm: RealmOffice, DisplayName: displayName, Email: sql.NullString{String: email, Valid: true},
		PasswordHash: sql.NullString{String: string(hash), Valid: true}, CreatedAt: a.Now().UTC(),
	})
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return queries.User{}, service.Conflict("duplicate", "an office user with that email already exists")
	}
	return u, err
}

func (a *Auth) CreateDriver(ctx context.Context, displayName, pin string) (queries.User, error) {
	if !ValidPIN(pin) {
		return queries.User{}, service.Invalid(map[string]string{"pin": "must be exactly 6 digits"})
	}
	if strings.TrimSpace(displayName) == "" {
		return queries.User{}, service.Invalid(map[string]string{"displayName": "required"})
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return queries.User{}, err
	}
	u, err := a.Q.CreateUser(ctx, queries.CreateUserParams{
		Realm: RealmDriver, DisplayName: strings.TrimSpace(displayName), PinHash: sql.NullString{String: string(hash), Valid: true}, CreatedAt: a.Now().UTC(),
	})
	if err != nil && strings.Contains(err.Error(), "UNIQUE") {
		return queries.User{}, service.Conflict("duplicate", "a driver with that name already exists")
	}
	return u, err
}

func (a *Auth) SetDriverPIN(ctx context.Context, displayName, pin string) error {
	if !ValidPIN(pin) {
		return service.Invalid(map[string]string{"pin": "must be exactly 6 digits"})
	}
	u, err := a.Q.GetDriverByName(ctx, displayName)
	if err != nil {
		return service.NotFound("driver")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return a.Q.SetUserPinHash(ctx, queries.SetUserPinHashParams{PinHash: sql.NullString{String: string(hash), Valid: true}, ID: u.ID})
}

func (a *Auth) DeactivateUser(ctx context.Context, emailOrName string) error {
	u, err := a.Q.GetUserByEmail(ctx, sql.NullString{String: strings.ToLower(emailOrName), Valid: true})
	if err != nil {
		u, err = a.Q.GetDriverByName(ctx, emailOrName)
		if err != nil {
			return service.NotFound("user")
		}
	}
	return a.Q.SetUserActive(ctx, queries.SetUserActiveParams{Active: false, ID: u.ID})
}

func (a *Auth) LoginOffice(ctx context.Context, email, password, ip string) (Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !a.Limiter.Allow("ip:"+ip) || !a.Limiter.Allow("email:"+email) {
		return Session{}, ErrRateLimited
	}
	u, err := a.Q.GetUserByEmail(ctx, sql.NullString{String: email, Valid: true})
	if err != nil || !u.Active || !u.PasswordHash.Valid {
		return Session{}, ErrBadCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash.String), []byte(password)) != nil {
		return Session{}, ErrBadCredentials
	}
	a.Limiter.Reset("email:" + email)
	return a.createSession(ctx, u)
}

func (a *Auth) LoginDriver(ctx context.Context, userID int64, pin, ip string) (Session, error) {
	userKey := fmt.Sprintf("user:%d", userID)
	if !a.Limiter.Allow("ip:"+ip) || !a.Limiter.Allow(userKey) {
		return Session{}, ErrRateLimited
	}
	u, err := a.Q.GetUser(ctx, userID)
	if err != nil || u.Realm != RealmDriver || !u.Active || !u.PinHash.Valid {
		return Session{}, ErrBadCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PinHash.String), []byte(pin)) != nil {
		return Session{}, ErrBadCredentials
	}
	a.Limiter.Reset(userKey)
	return a.createSession(ctx, u)
}

func (a *Auth) createSession(ctx context.Context, u queries.User) (Session, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return Session{}, err
	}
	s := Session{ID: base64.RawURLEncoding.EncodeToString(buf), UserID: u.ID, Realm: u.Realm, DisplayName: u.DisplayName, ExpiresAt: a.Now().UTC().Add(a.SessionTTL)}
	if err := a.Q.CreateSession(ctx, queries.CreateSessionParams{ID: s.ID, UserID: s.UserID, Realm: s.Realm, ExpiresAt: s.ExpiresAt}); err != nil {
		return Session{}, err
	}
	return s, nil
}

func (a *Auth) Logout(ctx context.Context, sessionID string) error {
	return a.Q.DeleteSession(ctx, sessionID)
}

func (a *Auth) Lookup(ctx context.Context, sessionID string) (Session, bool) {
	if sessionID == "" {
		return Session{}, false
	}
	row, err := a.Q.GetSession(ctx, sessionID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return Session{}, false
		}
		return Session{}, false
	}
	if !row.Active || row.ExpiresAt.Before(a.Now().UTC()) {
		return Session{}, false
	}
	return Session{ID: row.ID, UserID: row.UserID, Realm: row.Realm, DisplayName: row.DisplayName, ExpiresAt: row.ExpiresAt}, true
}
```

`internal/auth/middleware.go`:
```go
package auth

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"
)

const CookieName = "deepcuts_session"

type ctxKey struct{}

func SessionFrom(ctx context.Context) (Session, bool) {
	s, ok := ctx.Value(ctxKey{}).(Session)
	return s, ok
}

// Middleware requires a valid session of the given realm and stores it in the request context.
func (a *Auth) Middleware(realm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(CookieName)
			if err != nil {
				unauthorized(w)
				return
			}
			s, ok := a.Lookup(r.Context(), c.Value)
			if !ok || s.Realm != realm {
				unauthorized(w)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, s)))
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"code": "unauthorized", "message": "login required"})
}

func SetCookie(w http.ResponseWriter, s Session, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: s.ID, Path: "/", HttpOnly: true, Secure: secure,
		SameSite: http.SameSiteLaxMode, Expires: s.ExpiresAt,
	})
}

func ClearCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "", Path: "/", HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, Expires: time.Unix(0, 0), MaxAge: -1})
}

// ClientIP returns the remote IP without port. Behind Caddy, X-Forwarded-For's first entry wins.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := indexByte(xff, ','); i > 0 {
			return xff[:i]
		}
		return xff
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/auth/ 2>&1 | tail -3`
Expected: `ok`. (`GetUserByEmail` takes `sql.NullString` because the column is nullable; if sqlc generated `string`, adapt the calls.)

- [ ] **Step 5: User CLI**

`cmd/deepcuts/cmd_user.go`:
```go
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/config"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
)

// readSecret prompts on stderr and reads without echo when stdin is a terminal; otherwise reads a line.
func readSecret(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		return string(b), err
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}

func runUser(cfg config.Config, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: deepcuts user add-office <email> <display name> | add-driver <display name> | set-pin <display name> | deactivate <email or name>")
	}
	ctx := context.Background()
	d, err := db.OpenAndMigrate(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer d.Close()
	a := auth.New(queries.New(d))
	switch args[0] {
	case "add-office":
		if len(args) < 3 {
			return fmt.Errorf("usage: deepcuts user add-office <email> <display name>")
		}
		pw, err := readSecret("password: ")
		if err != nil {
			return err
		}
		u, err := a.CreateOfficeUser(ctx, args[1], strings.Join(args[2:], " "), pw)
		if err != nil {
			return err
		}
		fmt.Println("created office user", u.ID, u.Email.String)
	case "add-driver":
		pin, err := readSecret("6 digit PIN: ")
		if err != nil {
			return err
		}
		u, err := a.CreateDriver(ctx, strings.Join(args[1:], " "), pin)
		if err != nil {
			return err
		}
		fmt.Println("created driver", u.ID, u.DisplayName)
	case "set-pin":
		pin, err := readSecret("new 6 digit PIN: ")
		if err != nil {
			return err
		}
		if err := a.SetDriverPIN(ctx, strings.Join(args[1:], " "), pin); err != nil {
			return err
		}
		fmt.Println("pin updated")
	case "deactivate":
		if err := a.DeactivateUser(ctx, strings.Join(args[1:], " ")); err != nil {
			return err
		}
		fmt.Println("deactivated")
	default:
		return fmt.Errorf("unknown user subcommand %q", args[0])
	}
	return nil
}
```

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go get golang.org/x/term@latest && go build ./...` and delete the `runUser = notImplemented("user")` line in `main.go`.

- [ ] **Step 6: Verify the CLI end to end and commit**

Run:
```bash
export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && T=$(mktemp -d) && \
echo "correct horse" | DEEPCUTS_DB_PATH=$T/t.sqlite go run ./cmd/deepcuts user add-office o@x.com Olive Office && \
echo "123456" | DEEPCUTS_DB_PATH=$T/t.sqlite go run ./cmd/deepcuts user add-driver Sam && \
go test ./... && git add -A && git commit -q -m "Add auth: users, sessions, PIN/password login, realm middleware, user CLI" && git push
```
Expected: `created office user 1 o@x.com`, `created driver 2 Sam`, tests ok, pushed.

---

### Task 10: OpenAPI contract, generated server, HTTP skeleton, login endpoints

**Files:**
- Create: `api/openapi.yaml`, `oapi-codegen.yaml`, `internal/httpapi/server.go`, `internal/httpapi/handlers_auth.go`, `internal/httpapi/convert.go`, `internal/httpapi/server_test.go`, `internal/httpapi/testserver_test.go`
- Generated: `internal/api/api.gen.go`
- Modify: `Makefile` (no change needed; `generate` already runs oapi-codegen)

**Interfaces:**
- Produces:
  - `httpapi.Deps{Svc *service.Service; Auth *auth.Auth; Proofs *storage.Local; Secure bool; Web fs.FS}`; `func NewRouter(d Deps) http.Handler`
  - `type Server struct{ d Deps }` implementing `api.StrictServerInterface` (methods added across Tasks 10 to 12); each handler returns `nil, err` for service errors and `writeError` maps them.
  - Realm guard: `/api/office/login` and `/api/driver/login`, `/api/driver/drivers` are public; every other `/api/office/*` requires an office session; every other `/api/driver/*` requires a driver session; `/api/customer/*` returns 404 in phase 1.
  - convert.go: `toCustomer(queries.Customer) api.Customer`, `toProduct(queries.Product) api.Product`, `toOrder(service.OrderDetail) api.Order`, `toOrderSummary(...)`, `toRoute(service.RouteDetail) api.Route`, `toStop(service.StopDetail) api.Stop`, `toDriverStop(service.DriverStop) api.DriverStop`, `toDriverRoute(service.DriverRoute) api.DriverRoute`, `toUser(auth.Session) api.User`.
  - Test helper: `newTestServer(t) (*httptest.Server, *service.Service, *auth.Auth)` with `officeClient`/`driverClient` cookie jars and `do(t, client, method, path, body any, out any) int`.

- [ ] **Step 1: Write the OpenAPI contract**

`api/openapi.yaml`:
```yaml
openapi: 3.0.3
info:
  title: Deep Cuts CRM API
  version: "1.0"
servers:
  - url: /
paths:
  /api/office/login:
    post:
      operationId: officeLogin
      tags: [auth]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/OfficeLogin" } } } }
      responses:
        "200": { description: logged in, content: { application/json: { schema: { $ref: "#/components/schemas/User" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/logout:
    post:
      operationId: officeLogout
      tags: [auth]
      responses:
        "204": { description: logged out }
        default: { $ref: "#/components/responses/Error" }
  /api/office/me:
    get:
      operationId: officeMe
      tags: [auth]
      responses:
        "200": { description: current user, content: { application/json: { schema: { $ref: "#/components/schemas/User" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/driver/drivers:
    get:
      operationId: listDriverLoginNames
      tags: [auth]
      responses:
        "200": { description: active drivers, content: { application/json: { schema: { type: array, items: { $ref: "#/components/schemas/Driver" } } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/driver/login:
    post:
      operationId: driverLogin
      tags: [auth]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/DriverLogin" } } } }
      responses:
        "200": { description: logged in, content: { application/json: { schema: { $ref: "#/components/schemas/User" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/driver/logout:
    post:
      operationId: driverLogout
      tags: [auth]
      responses:
        "204": { description: logged out }
        default: { $ref: "#/components/responses/Error" }
  /api/driver/me:
    get:
      operationId: driverMe
      tags: [auth]
      responses:
        "200": { description: current user, content: { application/json: { schema: { $ref: "#/components/schemas/User" } } } }
        default: { $ref: "#/components/responses/Error" }

  /api/office/customers:
    get:
      operationId: listCustomers
      tags: [catalog]
      parameters:
        - { name: includeInactive, in: query, schema: { type: boolean } }
      responses:
        "200": { description: customers, content: { application/json: { schema: { type: array, items: { $ref: "#/components/schemas/Customer" } } } } }
        default: { $ref: "#/components/responses/Error" }
    post:
      operationId: createCustomer
      tags: [catalog]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/CustomerInput" } } } }
      responses:
        "201": { description: created, content: { application/json: { schema: { $ref: "#/components/schemas/Customer" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/customers/{customerId}:
    parameters:
      - { name: customerId, in: path, required: true, schema: { type: integer, format: int64 } }
    get:
      operationId: getCustomer
      tags: [catalog]
      responses:
        "200": { description: customer, content: { application/json: { schema: { $ref: "#/components/schemas/Customer" } } } }
        default: { $ref: "#/components/responses/Error" }
    put:
      operationId: updateCustomer
      tags: [catalog]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/CustomerInput" } } } }
      responses:
        "200": { description: updated, content: { application/json: { schema: { $ref: "#/components/schemas/Customer" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/customers/{customerId}/prices:
    parameters:
      - { name: customerId, in: path, required: true, schema: { type: integer, format: int64 } }
    get:
      operationId: listCustomerPrices
      tags: [catalog]
      parameters:
        - { name: asOf, in: query, schema: { type: string }, description: "YYYY-MM-DD, defaults to today" }
      responses:
        "200": { description: current prices, content: { application/json: { schema: { type: array, items: { $ref: "#/components/schemas/CustomerPrice" } } } } }
        default: { $ref: "#/components/responses/Error" }
    post:
      operationId: setCustomerPrice
      tags: [catalog]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/PriceInput" } } } }
      responses:
        "201": { description: created, content: { application/json: { schema: { $ref: "#/components/schemas/CustomerPrice" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/products:
    get:
      operationId: listProducts
      tags: [catalog]
      parameters:
        - { name: includeInactive, in: query, schema: { type: boolean } }
      responses:
        "200": { description: products, content: { application/json: { schema: { type: array, items: { $ref: "#/components/schemas/Product" } } } } }
        default: { $ref: "#/components/responses/Error" }
    post:
      operationId: createProduct
      tags: [catalog]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/ProductInput" } } } }
      responses:
        "201": { description: created, content: { application/json: { schema: { $ref: "#/components/schemas/Product" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/products/{productId}:
    parameters:
      - { name: productId, in: path, required: true, schema: { type: integer, format: int64 } }
    get:
      operationId: getProduct
      tags: [catalog]
      responses:
        "200": { description: product, content: { application/json: { schema: { $ref: "#/components/schemas/Product" } } } }
        default: { $ref: "#/components/responses/Error" }
    put:
      operationId: updateProduct
      tags: [catalog]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/ProductInput" } } } }
      responses:
        "200": { description: updated, content: { application/json: { schema: { $ref: "#/components/schemas/Product" } } } }
        default: { $ref: "#/components/responses/Error" }

  /api/office/orders:
    get:
      operationId: listOrders
      tags: [orders]
      parameters:
        - { name: date, in: query, schema: { type: string } }
        - { name: status, in: query, schema: { type: string } }
        - { name: customerId, in: query, schema: { type: integer, format: int64 } }
      responses:
        "200": { description: orders, content: { application/json: { schema: { type: array, items: { $ref: "#/components/schemas/OrderSummary" } } } } }
        default: { $ref: "#/components/responses/Error" }
    post:
      operationId: createOrder
      tags: [orders]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/OrderInput" } } } }
      responses:
        "201": { description: created, content: { application/json: { schema: { $ref: "#/components/schemas/Order" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/orders/{orderId}:
    parameters:
      - { name: orderId, in: path, required: true, schema: { type: integer, format: int64 } }
    get:
      operationId: getOrder
      tags: [orders]
      responses:
        "200": { description: order, content: { application/json: { schema: { $ref: "#/components/schemas/Order" } } } }
        default: { $ref: "#/components/responses/Error" }
    patch:
      operationId: updateOrder
      tags: [orders]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/OrderPatch" } } } }
      responses:
        "200": { description: updated, content: { application/json: { schema: { $ref: "#/components/schemas/Order" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/orders/{orderId}/lines:
    parameters:
      - { name: orderId, in: path, required: true, schema: { type: integer, format: int64 } }
    post:
      operationId: addOrderLine
      tags: [orders]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/LineInput" } } } }
      responses:
        "200": { description: order with new line, content: { application/json: { schema: { $ref: "#/components/schemas/Order" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/orders/{orderId}/lines/{lineId}:
    parameters:
      - { name: orderId, in: path, required: true, schema: { type: integer, format: int64 } }
      - { name: lineId, in: path, required: true, schema: { type: integer, format: int64 } }
    patch:
      operationId: updateOrderLine
      tags: [orders]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/LinePatch" } } } }
      responses:
        "200": { description: order, content: { application/json: { schema: { $ref: "#/components/schemas/Order" } } } }
        default: { $ref: "#/components/responses/Error" }
    delete:
      operationId: deleteOrderLine
      tags: [orders]
      responses:
        "200": { description: order, content: { application/json: { schema: { $ref: "#/components/schemas/Order" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/orders/{orderId}/confirm:
    post:
      operationId: confirmOrder
      tags: [orders]
      parameters: [ { name: orderId, in: path, required: true, schema: { type: integer, format: int64 } } ]
      responses:
        "200": { description: order, content: { application/json: { schema: { $ref: "#/components/schemas/Order" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/orders/{orderId}/unconfirm:
    post:
      operationId: unconfirmOrder
      tags: [orders]
      parameters: [ { name: orderId, in: path, required: true, schema: { type: integer, format: int64 } } ]
      responses:
        "200": { description: order, content: { application/json: { schema: { $ref: "#/components/schemas/Order" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/orders/{orderId}/cancel:
    post:
      operationId: cancelOrder
      tags: [orders]
      parameters: [ { name: orderId, in: path, required: true, schema: { type: integer, format: int64 } } ]
      responses:
        "200": { description: order, content: { application/json: { schema: { $ref: "#/components/schemas/Order" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/orders/{orderId}/finalize:
    post:
      operationId: finalizeOrder
      tags: [orders]
      parameters: [ { name: orderId, in: path, required: true, schema: { type: integer, format: int64 } } ]
      responses:
        "200": { description: order, content: { application/json: { schema: { $ref: "#/components/schemas/Order" } } } }
        default: { $ref: "#/components/responses/Error" }

  /api/office/day:
    get:
      operationId: getDay
      tags: [routes]
      parameters:
        - { name: date, in: query, required: true, schema: { type: string } }
      responses:
        "200": { description: day view, content: { application/json: { schema: { $ref: "#/components/schemas/DayView" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/drivers:
    get:
      operationId: listDrivers
      tags: [routes]
      responses:
        "200": { description: drivers, content: { application/json: { schema: { type: array, items: { $ref: "#/components/schemas/Driver" } } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/routes:
    post:
      operationId: createRoute
      tags: [routes]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/RouteInput" } } } }
      responses:
        "201": { description: created, content: { application/json: { schema: { $ref: "#/components/schemas/Route" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/routes/{routeId}:
    parameters:
      - { name: routeId, in: path, required: true, schema: { type: integer, format: int64 } }
    get:
      operationId: getRoute
      tags: [routes]
      responses:
        "200": { description: route, content: { application/json: { schema: { $ref: "#/components/schemas/Route" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/routes/{routeId}/stops:
    parameters:
      - { name: routeId, in: path, required: true, schema: { type: integer, format: int64 } }
    post:
      operationId: addStop
      tags: [routes]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/AddStopInput" } } } }
      responses:
        "200": { description: route, content: { application/json: { schema: { $ref: "#/components/schemas/Route" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/routes/{routeId}/stops/{stopId}:
    parameters:
      - { name: routeId, in: path, required: true, schema: { type: integer, format: int64 } }
      - { name: stopId, in: path, required: true, schema: { type: integer, format: int64 } }
    delete:
      operationId: removeStop
      tags: [routes]
      responses:
        "200": { description: route, content: { application/json: { schema: { $ref: "#/components/schemas/Route" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/routes/{routeId}/sequence:
    put:
      operationId: reorderStops
      tags: [routes]
      parameters: [ { name: routeId, in: path, required: true, schema: { type: integer, format: int64 } } ]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/SequenceInput" } } } }
      responses:
        "200": { description: route, content: { application/json: { schema: { $ref: "#/components/schemas/Route" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/routes/{routeId}/out:
    post:
      operationId: routeOut
      tags: [routes]
      parameters: [ { name: routeId, in: path, required: true, schema: { type: integer, format: int64 } } ]
      responses:
        "200": { description: route, content: { application/json: { schema: { $ref: "#/components/schemas/Route" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/routes/{routeId}/complete:
    post:
      operationId: routeComplete
      tags: [routes]
      parameters: [ { name: routeId, in: path, required: true, schema: { type: integer, format: int64 } } ]
      responses:
        "200": { description: route, content: { application/json: { schema: { $ref: "#/components/schemas/Route" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/office/stops/{stopId}/proof:
    get:
      operationId: getStopProof
      tags: [routes]
      parameters: [ { name: stopId, in: path, required: true, schema: { type: integer, format: int64 } } ]
      responses:
        "200": { description: proof image, content: { image/png: { schema: { type: string, format: binary } }, image/jpeg: { schema: { type: string, format: binary } } } }
        default: { $ref: "#/components/responses/Error" }

  /api/driver/route:
    get:
      operationId: getDriverRoute
      tags: [driver]
      parameters:
        - { name: date, in: query, schema: { type: string }, description: "YYYY-MM-DD, defaults to today" }
      responses:
        "200": { description: today's route, content: { application/json: { schema: { $ref: "#/components/schemas/DriverRoute" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/driver/stops/{stopId}/actions:
    post:
      operationId: postDriverAction
      tags: [driver]
      parameters: [ { name: stopId, in: path, required: true, schema: { type: integer, format: int64 } } ]
      requestBody: { required: true, content: { application/json: { schema: { $ref: "#/components/schemas/DriverAction" } } } }
      responses:
        "200": { description: result, content: { application/json: { schema: { $ref: "#/components/schemas/DriverActionResult" } } } }
        default: { $ref: "#/components/responses/Error" }
  /api/driver/routes/{routeId}/complete:
    post:
      operationId: driverCompleteRoute
      tags: [driver]
      parameters: [ { name: routeId, in: path, required: true, schema: { type: integer, format: int64 } } ]
      responses:
        "200": { description: route, content: { application/json: { schema: { $ref: "#/components/schemas/DriverRoute" } } } }
        default: { $ref: "#/components/responses/Error" }

components:
  responses:
    Error:
      description: error
      content:
        application/json:
          schema: { $ref: "#/components/schemas/Error" }
  schemas:
    Error:
      type: object
      required: [code, message]
      properties:
        code: { type: string }
        message: { type: string }
        fields: { type: object, additionalProperties: { type: string } }
    User:
      type: object
      required: [id, realm, displayName]
      properties:
        id: { type: integer, format: int64 }
        realm: { type: string, enum: [office, driver, customer] }
        displayName: { type: string }
    Driver:
      type: object
      required: [id, displayName]
      properties:
        id: { type: integer, format: int64 }
        displayName: { type: string }
    OfficeLogin:
      type: object
      required: [email, password]
      properties:
        email: { type: string }
        password: { type: string }
    DriverLogin:
      type: object
      required: [userId, pin]
      properties:
        userId: { type: integer, format: int64 }
        pin: { type: string }
    Customer:
      type: object
      required: [id, name, billingAddress, deliveryAddress, contactName, phone, email, deliveryNotes, deliveryDays, active]
      properties:
        id: { type: integer, format: int64 }
        name: { type: string }
        billingAddress: { type: string }
        deliveryAddress: { type: string }
        contactName: { type: string }
        phone: { type: string }
        email: { type: string }
        deliveryNotes: { type: string }
        deliveryDays: { type: array, items: { type: string } }
        qboCustomerId: { type: string }
        active: { type: boolean }
    CustomerInput:
      type: object
      required: [name]
      properties:
        name: { type: string }
        billingAddress: { type: string }
        deliveryAddress: { type: string }
        contactName: { type: string }
        phone: { type: string }
        email: { type: string }
        deliveryNotes: { type: string }
        deliveryDays: { type: array, items: { type: string } }
        qboCustomerId: { type: string }
        active: { type: boolean, default: true }
    Product:
      type: object
      required: [id, sku, name, category, sellUnit, catchWeight, basePriceCents, active]
      properties:
        id: { type: integer, format: int64 }
        sku: { type: string }
        name: { type: string }
        category: { type: string }
        sellUnit: { type: string, enum: [lb, case, each] }
        catchWeight: { type: boolean }
        approxCaseWeight: { type: integer, format: int64, description: hundredths of a pound }
        basePriceCents: { type: integer, format: int64 }
        qboItemId: { type: string }
        active: { type: boolean }
    ProductInput:
      type: object
      required: [sku, name, sellUnit, basePriceCents]
      properties:
        sku: { type: string }
        name: { type: string }
        category: { type: string }
        sellUnit: { type: string, enum: [lb, case, each] }
        catchWeight: { type: boolean, default: false }
        approxCaseWeight: { type: integer, format: int64 }
        basePriceCents: { type: integer, format: int64 }
        qboItemId: { type: string }
        active: { type: boolean, default: true }
    CustomerPrice:
      type: object
      required: [id, productId, sku, productName, priceCents, effectiveFrom]
      properties:
        id: { type: integer, format: int64 }
        productId: { type: integer, format: int64 }
        sku: { type: string }
        productName: { type: string }
        priceCents: { type: integer, format: int64 }
        effectiveFrom: { type: string }
    PriceInput:
      type: object
      required: [productId, priceCents, effectiveFrom]
      properties:
        productId: { type: integer, format: int64 }
        priceCents: { type: integer, format: int64 }
        effectiveFrom: { type: string }
    OrderSummary:
      type: object
      required: [id, customerId, customerName, requestedDeliveryDate, status, needsReview, lineCount, notes]
      properties:
        id: { type: integer, format: int64 }
        customerId: { type: integer, format: int64 }
        customerName: { type: string }
        requestedDeliveryDate: { type: string }
        status: { $ref: "#/components/schemas/OrderStatus" }
        needsReview: { type: boolean }
        lineCount: { type: integer, format: int64 }
        notes: { type: string }
    OrderStatus:
      type: string
      enum: [draft, confirmed, scheduled, delivered, finalized, cancelled]
    OrderInput:
      type: object
      required: [customerId, requestedDeliveryDate]
      properties:
        customerId: { type: integer, format: int64 }
        requestedDeliveryDate: { type: string }
        notes: { type: string }
    OrderPatch:
      type: object
      properties:
        notes: { type: string }
        requestedDeliveryDate: { type: string }
    OrderLine:
      type: object
      required: [id, productId, sku, productName, sellUnit, catchWeight, orderedQty, unitPriceCents, priceOverridden, shortageNote, amountCents, amountSource]
      properties:
        id: { type: integer, format: int64 }
        productId: { type: integer, format: int64 }
        sku: { type: string }
        productName: { type: string }
        sellUnit: { type: string }
        catchWeight: { type: boolean }
        orderedQty: { type: integer, format: int64, description: hundredths of the sell unit }
        unitPriceCents: { type: integer, format: int64 }
        priceOverridden: { type: boolean }
        estWeight: { type: integer, format: int64 }
        shippedWeight: { type: integer, format: int64 }
        deliveredQty: { type: integer, format: int64 }
        deliveredWeight: { type: integer, format: int64 }
        shortageNote: { type: string }
        amountCents: { type: integer, format: int64 }
        amountSource: { type: string, enum: [delivered, shipped, estimated, ordered] }
    LineInput:
      type: object
      required: [productId, orderedQty]
      properties:
        productId: { type: integer, format: int64 }
        orderedQty: { type: integer, format: int64 }
    LinePatch:
      type: object
      properties:
        orderedQty: { type: integer, format: int64 }
        unitPriceCents: { type: integer, format: int64 }
        shippedWeight: { type: integer, format: int64 }
        deliveredQty: { type: integer, format: int64 }
        deliveredWeight: { type: integer, format: int64 }
        shortageNote: { type: string }
    Order:
      type: object
      required: [id, customer, requestedDeliveryDate, status, notes, needsReview, lines, totalCents, createdAt]
      properties:
        id: { type: integer, format: int64 }
        customer: { $ref: "#/components/schemas/Customer" }
        requestedDeliveryDate: { type: string }
        status: { $ref: "#/components/schemas/OrderStatus" }
        notes: { type: string }
        needsReview: { type: boolean }
        lines: { type: array, items: { $ref: "#/components/schemas/OrderLine" } }
        totalCents: { type: integer, format: int64 }
        routeId: { type: integer, format: int64 }
        stopId: { type: integer, format: int64 }
        routeStatus: { type: string }
        createdAt: { type: string, format: date-time }
        finalizedAt: { type: string, format: date-time }
    RouteInput:
      type: object
      required: [routeDate, driverUserId]
      properties:
        routeDate: { type: string }
        driverUserId: { type: integer, format: int64 }
        truckLabel: { type: string }
    Stop:
      type: object
      required: [id, routeId, orderId, sequence, status, hasProofImage, skipReason, driverNote, customerName, deliveryAddress, phone, contactName, deliveryNotes, orderStatus, needsReview, lineCount]
      properties:
        id: { type: integer, format: int64 }
        routeId: { type: integer, format: int64 }
        orderId: { type: integer, format: int64 }
        sequence: { type: integer, format: int64 }
        status: { type: string, enum: [pending, delivered, skipped] }
        deliveredAt: { type: string, format: date-time }
        proofType: { type: string, enum: [signature, photo, name] }
        proofName: { type: string, description: received-by name when proofType is name }
        hasProofImage: { type: boolean }
        skipReason: { type: string }
        driverNote: { type: string }
        customerName: { type: string }
        deliveryAddress: { type: string }
        phone: { type: string }
        contactName: { type: string }
        deliveryNotes: { type: string }
        orderStatus: { $ref: "#/components/schemas/OrderStatus" }
        needsReview: { type: boolean }
        lineCount: { type: integer, format: int64 }
    Route:
      type: object
      required: [id, routeDate, driverUserId, driverName, truckLabel, status, stops]
      properties:
        id: { type: integer, format: int64 }
        routeDate: { type: string }
        driverUserId: { type: integer, format: int64 }
        driverName: { type: string }
        truckLabel: { type: string }
        status: { type: string, enum: [planned, out, complete] }
        outAt: { type: string, format: date-time }
        completedAt: { type: string, format: date-time }
        stops: { type: array, items: { $ref: "#/components/schemas/Stop" } }
    AddStopInput:
      type: object
      required: [orderId]
      properties:
        orderId: { type: integer, format: int64 }
    SequenceInput:
      type: object
      required: [stopIds]
      properties:
        stopIds: { type: array, items: { type: integer, format: int64 } }
    DayView:
      type: object
      required: [date, unscheduled, routes]
      properties:
        date: { type: string }
        unscheduled: { type: array, items: { $ref: "#/components/schemas/OrderSummary" } }
        routes: { type: array, items: { $ref: "#/components/schemas/Route" } }
    DriverStop:
      type: object
      required: [stop, order]
      properties:
        stop: { $ref: "#/components/schemas/Stop" }
        order: { $ref: "#/components/schemas/Order" }
    DriverRoute:
      type: object
      required: [route, stops]
      properties:
        route: { $ref: "#/components/schemas/Route" }
        stops: { type: array, items: { $ref: "#/components/schemas/DriverStop" } }
    Proof:
      type: object
      required: [type]
      properties:
        type: { type: string, enum: [signature, photo, name] }
        dataBase64: { type: string, description: PNG or JPEG bytes, base64, at most 300 KB decoded }
        name: { type: string }
    LineAdjustment:
      type: object
      required: [lineId]
      properties:
        lineId: { type: integer, format: int64 }
        deliveredQty: { type: integer, format: int64 }
        deliveredWeight: { type: integer, format: int64 }
        shortageNote: { type: string }
    DriverAction:
      type: object
      required: [clientId, type]
      properties:
        clientId: { type: string, description: client-generated UUID; replays are acknowledged and not reapplied }
        type: { type: string, enum: [deliver, adjust, skip] }
        proof: { $ref: "#/components/schemas/Proof" }
        lines: { type: array, items: { $ref: "#/components/schemas/LineAdjustment" } }
        note: { type: string }
        skipReason: { type: string }
    DriverActionResult:
      type: object
      required: [applied, stop]
      properties:
        applied: { type: boolean }
        stop: { $ref: "#/components/schemas/DriverStop" }
```

`oapi-codegen.yaml`:
```yaml
package: api
output: internal/api/api.gen.go
generate:
  chi-server: true
  strict-server: true
  models: true
  embedded-spec: false
output-options:
  exclude-operation-ids:
    - officeLogin
    - officeLogout
    - driverLogin
    - driverLogout
    - listDriverLoginNames
    - getStopProof
```

The six excluded operations are implemented as plain chi handlers because they set cookies or stream a file; they stay in the contract so the TypeScript client has them.

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && mkdir -p internal/api && go tool oapi-codegen -config oapi-codegen.yaml api/openapi.yaml && go build ./... && grep -c "ctx context.Context, request" internal/api/api.gen.go`
Expected: build ok; the count of strict interface methods is 31.

- [ ] **Step 2: Write the test server helper and the failing auth-flow tests**

`internal/httpapi/testserver_test.go`:
```go
package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
	"github.com/cmcgeedev/deepcutsCRM/internal/storage"
)

type testEnv struct {
	srv    *httptest.Server
	svc    *service.Service
	auth   *auth.Auth
	office *http.Client
	driver *http.Client
	anon   *http.Client
	driverID int64
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	dir := t.TempDir()
	d, err := db.OpenAndMigrate(context.Background(), filepath.Join(dir, "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	loc, _ := time.LoadLocation("America/New_York")
	svc := service.New(d, loc)
	svc.Now = func() time.Time { return time.Date(2026, 9, 10, 15, 0, 0, 0, time.UTC) }
	proofs, _ := storage.NewLocal(filepath.Join(dir, "uploads"))
	svc.Proofs = proofs
	a := auth.New(queries.New(d))
	a.Now = svc.Now
	web := fstest.MapFS{"index.html": {Data: []byte("<html>spa</html>")}}
	srv := httptest.NewServer(NewRouter(Deps{Svc: svc, Auth: a, Proofs: proofs, Secure: false, Web: web}))
	t.Cleanup(srv.Close)

	env := &testEnv{srv: srv, svc: svc, auth: a, office: jarClient(), driver: jarClient(), anon: jarClient()}
	ctx := context.Background()
	if _, err := a.CreateOfficeUser(ctx, "o@x.com", "Olive", "correct horse"); err != nil {
		t.Fatal(err)
	}
	drv, err := a.CreateDriver(ctx, "Sam", "123456")
	if err != nil {
		t.Fatal(err)
	}
	env.driverID = drv.ID
	if code := env.do(t, env.office, "POST", "/api/office/login", map[string]any{"email": "o@x.com", "password": "correct horse"}, nil); code != 200 {
		t.Fatalf("office login: %d", code)
	}
	if code := env.do(t, env.driver, "POST", "/api/driver/login", map[string]any{"userId": drv.ID, "pin": "123456"}, nil); code != 200 {
		t.Fatalf("driver login: %d", code)
	}
	return env
}

func jarClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

// do sends JSON and decodes the JSON reply into out (may be nil). Returns the status code.
func (e *testEnv) do(t *testing.T, c *http.Client, method, path string, body any, out any) int {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("decode %s %s: %v: %s", method, path, err, raw)
		}
	}
	return res.StatusCode
}
```

`internal/httpapi/server_test.go`:
```go
package httpapi

import "testing"

func TestRealmGuards(t *testing.T) {
	env := newTestEnv(t)
	var me map[string]any
	if code := env.do(t, env.office, "GET", "/api/office/me", nil, &me); code != 200 || me["displayName"] != "Olive" || me["realm"] != "office" {
		t.Fatalf("office me: %d %v", code, me)
	}
	if code := env.do(t, env.driver, "GET", "/api/driver/me", nil, &me); code != 200 || me["realm"] != "driver" {
		t.Fatalf("driver me: %d %v", code, me)
	}
	if code := env.do(t, env.driver, "GET", "/api/office/me", nil, nil); code != 401 {
		t.Fatalf("driver on office path: %d", code)
	}
	if code := env.do(t, env.office, "GET", "/api/driver/route", nil, nil); code != 401 {
		t.Fatalf("office on driver path: %d", code)
	}
	if code := env.do(t, env.anon, "GET", "/api/office/customers", nil, nil); code != 401 {
		t.Fatalf("anon: %d", code)
	}
	if code := env.do(t, env.anon, "GET", "/api/customer/anything", nil, nil); code != 404 {
		t.Fatalf("customer realm reserved: %d", code)
	}
}

func TestLoginErrorsAndLogout(t *testing.T) {
	env := newTestEnv(t)
	var e map[string]any
	if code := env.do(t, env.anon, "POST", "/api/office/login", map[string]any{"email": "o@x.com", "password": "nope"}, &e); code != 401 || e["code"] != "unauthorized" {
		t.Fatalf("bad password: %d %v", code, e)
	}
	if code := env.do(t, env.anon, "POST", "/api/office/login", map[string]any{"email": "o@x.com"}, &e); code != 422 || e["code"] != "invalid" {
		t.Fatalf("missing field: %d %v", code, e)
	}
	var drivers []map[string]any
	if code := env.do(t, env.anon, "GET", "/api/driver/drivers", nil, &drivers); code != 200 || len(drivers) != 1 || drivers[0]["displayName"] != "Sam" {
		t.Fatalf("driver names: %d %v", code, drivers)
	}
	if code := env.do(t, env.office, "POST", "/api/office/logout", nil, nil); code != 204 {
		t.Fatalf("logout: %d", code)
	}
	if code := env.do(t, env.office, "GET", "/api/office/me", nil, nil); code != 401 {
		t.Fatalf("after logout: %d", code)
	}
}

func TestSPAFallbackAndErrorShape(t *testing.T) {
	env := newTestEnv(t)
	res, err := env.anon.Get(env.srv.URL + "/office/orders/12")
	if err != nil || res.StatusCode != 200 || res.Header.Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("spa fallback: %v %d %s", err, res.StatusCode, res.Header.Get("Content-Type"))
	}
	var e map[string]any
	if code := env.do(t, env.office, "GET", "/api/office/customers/999", nil, &e); code != 404 || e["code"] != "not_found" {
		t.Fatalf("404 shape: %d %v", code, e)
	}
	if code := env.do(t, env.office, "POST", "/api/office/customers", map[string]any{"name": ""}, &e); code != 422 || e["fields"].(map[string]any)["name"] == nil {
		t.Fatalf("422 shape: %d %v", code, e)
	}
}
```

The customer endpoints in the last test come from Task 11; when running this task alone, that test compiles but the `customers` cases fail. That is expected until Task 11 lands. Run only `TestRealmGuards` and `TestLoginErrorsAndLogout` here.

- [ ] **Step 3: Run tests to verify they fail**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/httpapi/ 2>&1 | head -3`
Expected: build failure (`NewRouter`, `Deps` undefined).

- [ ] **Step 4: Implement server.go, handlers_auth.go, convert.go**

`internal/httpapi/server.go`:
```go
// Package httpapi assembles the chi router: auth endpoints, the generated strict API, and the SPA.
package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
	"github.com/cmcgeedev/deepcutsCRM/internal/storage"
)

type Deps struct {
	Svc    *service.Service
	Auth   *auth.Auth
	Proofs *storage.Local
	Secure bool  // set Secure on cookies (TLS)
	Web    fs.FS // built SPA; index.html at the root
}

type Server struct{ d Deps }

var _ api.StrictServerInterface = (*Server)(nil)

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RealIP, middleware.Recoverer, middleware.NoCache)
	s := &Server{d: d}

	r.Post("/api/office/login", s.officeLogin)
	r.Post("/api/driver/login", s.driverLogin)
	r.Get("/api/driver/drivers", s.listDriverLoginNames)

	r.Group(func(g chi.Router) {
		g.Use(realmGuard(d.Auth))
		g.Post("/api/office/logout", s.logout)
		g.Post("/api/driver/logout", s.logout)
		g.Get("/api/office/stops/{stopId}/proof", s.getStopProof)
		strict := api.NewStrictHandlerWithOptions(s, nil, api.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  func(w http.ResponseWriter, r *http.Request, err error) { writeError(w, service.Invalid(map[string]string{"body": err.Error()})) },
			ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) { writeError(w, err) },
		})
		api.HandlerWithOptions(strict, api.ChiServerOptions{BaseRouter: g, ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeError(w, service.Invalid(map[string]string{"request": err.Error()}))
		}})
	})

	r.HandleFunc("/api/*", func(w http.ResponseWriter, r *http.Request) { writeError(w, service.NotFound("endpoint")) })
	r.NotFound(spaHandler(d.Web))
	return r
}

// realmGuard picks the realm from the path prefix and requires a matching session.
func realmGuard(a *auth.Auth) func(http.Handler) http.Handler {
	office := a.Middleware(auth.RealmOffice)
	driver := a.Middleware(auth.RealmDriver)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.HasPrefix(r.URL.Path, "/api/office/"):
				office(next).ServeHTTP(w, r)
			case strings.HasPrefix(r.URL.Path, "/api/driver/"):
				driver(next).ServeHTTP(w, r)
			default:
				writeError(w, service.NotFound("endpoint"))
			}
		})
	}
}

func writeError(w http.ResponseWriter, err error) {
	e, ok := service.AsError(err)
	if !ok {
		e = &service.Error{Status: http.StatusInternalServerError, Code: "internal", Message: "internal error"}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.Status)
	body := api.Error{Code: e.Code, Message: e.Message}
	if len(e.Fields) > 0 {
		f := e.Fields
		body.Fields = &f
	}
	json.NewEncoder(w).Encode(body)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := dec.Decode(v); err != nil {
		return service.Invalid(map[string]string{"body": "malformed JSON"})
	}
	return nil
}

// spaHandler serves files from the embedded build and falls back to index.html for client routes.
func spaHandler(web fs.FS) http.HandlerFunc {
	files := http.FS(web)
	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p != "" {
			if f, err := web.Open(p); err == nil {
				f.Close()
				http.FileServer(files).ServeHTTP(w, r)
				return
			}
		}
		idx, err := fs.ReadFile(web, "index.html")
		if err != nil {
			http.Error(w, "web app not built", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(idx)
	}
}

var errNoSession = errors.New("no session")

func sessionOf(r *http.Request) (auth.Session, error) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		return s, errNoSession
	}
	return s, nil
}
```

`internal/httpapi/handlers_auth.go`:
```go
package httpapi

import (
	"context"
	"net/http"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

func (s *Server) officeLogin(w http.ResponseWriter, r *http.Request) {
	var in api.OfficeLogin
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if in.Email == "" || in.Password == "" {
		writeError(w, service.Invalid(map[string]string{"email": "required", "password": "required"}))
		return
	}
	sess, err := s.d.Auth.LoginOffice(r.Context(), in.Email, in.Password, auth.ClientIP(r))
	if err != nil {
		writeError(w, err)
		return
	}
	auth.SetCookie(w, sess, s.d.Secure)
	writeJSON(w, http.StatusOK, toUser(sess))
}

func (s *Server) driverLogin(w http.ResponseWriter, r *http.Request) {
	var in api.DriverLogin
	if err := decode(r, &in); err != nil {
		writeError(w, err)
		return
	}
	if in.UserId == 0 || in.Pin == "" {
		writeError(w, service.Invalid(map[string]string{"userId": "required", "pin": "required"}))
		return
	}
	sess, err := s.d.Auth.LoginDriver(r.Context(), in.UserId, in.Pin, auth.ClientIP(r))
	if err != nil {
		writeError(w, err)
		return
	}
	auth.SetCookie(w, sess, s.d.Secure)
	writeJSON(w, http.StatusOK, toUser(sess))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if sess, err := sessionOf(r); err == nil {
		_ = s.d.Auth.Logout(r.Context(), sess.ID)
	}
	auth.ClearCookie(w, s.d.Secure)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listDriverLoginNames(w http.ResponseWriter, r *http.Request) {
	users, err := s.d.Svc.ListDrivers(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]api.Driver, 0, len(users))
	for _, u := range users {
		out = append(out, api.Driver{Id: u.ID, DisplayName: u.DisplayName})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) OfficeMe(ctx context.Context, _ api.OfficeMeRequestObject) (api.OfficeMeResponseObject, error) {
	sess, ok := auth.SessionFrom(ctx)
	if !ok {
		return nil, service.Unauthorized("login required")
	}
	return api.OfficeMe200JSONResponse(toUser(sess)), nil
}

func (s *Server) DriverMe(ctx context.Context, _ api.DriverMeRequestObject) (api.DriverMeResponseObject, error) {
	sess, ok := auth.SessionFrom(ctx)
	if !ok {
		return nil, service.Unauthorized("login required")
	}
	return api.DriverMe200JSONResponse(toUser(sess)), nil
}
```

The strict handler receives the request context, and `auth.Middleware` stored the session there, so `auth.SessionFrom(ctx)` works inside strict handlers.

`internal/httpapi/convert.go` (auth part now; catalog/order/route converters are added in Tasks 11 and 12 in this same file):
```go
package httpapi

import (
	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
)

func toUser(s auth.Session) api.User {
	return api.User{Id: s.UserID, Realm: api.UserRealm(s.Realm), DisplayName: s.DisplayName}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func i64Ptr(v *int64) *int64 { return v }
```

Until Tasks 11 and 12 add the remaining methods, the `var _ api.StrictServerInterface = (*Server)(nil)` assertion fails to compile. For this task only, generate stubs so the package compiles: create `internal/httpapi/stubs_tmp.go` with every other strict method returning `nil, service.Conflict("not_implemented", "not implemented")`. Get the method list with `grep -oE "^\t[A-Z][A-Za-z]+\(ctx context.Context, request [A-Za-z]+RequestObject\) \([A-Za-z]+ResponseObject, error\)" internal/api/api.gen.go`. Tasks 11 and 12 delete the stubs they replace; Task 12 deletes the file.

- [ ] **Step 5: Run tests to verify they pass**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go vet ./... && go test ./internal/httpapi/ -run 'TestRealmGuards|TestLoginErrorsAndLogout' 2>&1 | tail -3`
Expected: `ok`

- [ ] **Step 6: Commit**

```bash
cd ~/deepcutsCRM && git add -A && git commit -q -m "Add OpenAPI contract, generated strict server, router, login endpoints" && git push
```

---

### Task 11: Office API handlers: customers, products, prices, orders

**Files:**
- Create: `internal/httpapi/handlers_catalog.go`, `internal/httpapi/handlers_orders.go`, `internal/httpapi/catalog_test.go`, `internal/httpapi/orders_test.go`
- Modify: `internal/httpapi/convert.go` (add converters), `internal/httpapi/stubs_tmp.go` (delete the stubs replaced here)

**Interfaces:**
- Consumes: service methods from Tasks 5 and 6; generated request/response types in `internal/api/api.gen.go` (read the file in ≤200-line chunks for exact names: request objects are `<OperationId>RequestObject` with `Params`/`Body`/path fields, responses are `<OperationId><Status>JSONResponse`).
- Produces in convert.go: `toCustomer`, `customerInput`, `toProduct`, `productInput`, `toPriceRow`, `toOrderSummary`, `toOrder`, `toLine`, `deref(*string) string`, `derefBool(*bool, def bool) bool`.

- [ ] **Step 1: Write the failing tests**

`internal/httpapi/catalog_test.go`:
```go
package httpapi

import "testing"

func TestCustomerEndpoints(t *testing.T) {
	env := newTestEnv(t)
	var c map[string]any
	code := env.do(t, env.office, "POST", "/api/office/customers", map[string]any{
		"name": "Blue Plate", "deliveryAddress": "1 Main St", "deliveryDays": []string{"mon", "thu"}, "qboCustomerId": "q1",
	}, &c)
	if code != 201 || c["name"] != "Blue Plate" || c["active"] != true || len(c["deliveryDays"].([]any)) != 2 {
		t.Fatalf("create: %d %v", code, c)
	}
	id := int64(c["id"].(float64))
	var list []map[string]any
	if code := env.do(t, env.office, "GET", "/api/office/customers", nil, &list); code != 200 || len(list) != 1 {
		t.Fatalf("list: %d %v", code, list)
	}
	code = env.do(t, env.office, "PUT", "/api/office/customers/"+itoa(id), map[string]any{"name": "Blue Plate Diner", "active": false}, &c)
	if code != 200 || c["name"] != "Blue Plate Diner" || c["active"] != false {
		t.Fatalf("update: %d %v", code, c)
	}
	if code := env.do(t, env.office, "GET", "/api/office/customers", nil, &list); code != 200 || len(list) != 0 {
		t.Fatalf("inactive hidden: %d %v", code, list)
	}
	if code := env.do(t, env.office, "GET", "/api/office/customers?includeInactive=true", nil, &list); code != 200 || len(list) != 1 {
		t.Fatalf("includeInactive: %d %v", code, list)
	}
	var e map[string]any
	if code := env.do(t, env.office, "POST", "/api/office/customers", map[string]any{"name": "Dup", "qboCustomerId": "q1"}, &e); code != 409 || e["code"] != "duplicate" {
		t.Fatalf("duplicate: %d %v", code, e)
	}
}

func TestProductAndPriceEndpoints(t *testing.T) {
	env := newTestEnv(t)
	var p map[string]any
	code := env.do(t, env.office, "POST", "/api/office/products", map[string]any{
		"sku": "BRIS", "name": "Brisket", "sellUnit": "case", "catchWeight": true, "approxCaseWeight": 6000, "basePriceCents": 599,
	}, &p)
	if code != 201 || p["catchWeight"] != true || p["approxCaseWeight"].(float64) != 6000 {
		t.Fatalf("create product: %d %v", code, p)
	}
	pid := int64(p["id"].(float64))
	var e map[string]any
	if code := env.do(t, env.office, "POST", "/api/office/products", map[string]any{"sku": "X", "name": "X", "sellUnit": "lb", "catchWeight": true, "basePriceCents": 1}, &e); code != 422 {
		t.Fatalf("invalid product: %d %v", code, e)
	}
	var c map[string]any
	env.do(t, env.office, "POST", "/api/office/customers", map[string]any{"name": "A"}, &c)
	cid := int64(c["id"].(float64))
	var pr map[string]any
	code = env.do(t, env.office, "POST", "/api/office/customers/"+itoa(cid)+"/prices", map[string]any{"productId": pid, "priceCents": 549, "effectiveFrom": "2026-01-01"}, &pr)
	if code != 201 || pr["sku"] != "BRIS" || pr["priceCents"].(float64) != 549 {
		t.Fatalf("set price: %d %v", code, pr)
	}
	var prices []map[string]any
	if code := env.do(t, env.office, "GET", "/api/office/customers/"+itoa(cid)+"/prices", nil, &prices); code != 200 || len(prices) != 1 {
		t.Fatalf("list prices: %d %v", code, prices)
	}
	if code := env.do(t, env.office, "GET", "/api/office/customers/"+itoa(cid)+"/prices?asOf=2025-01-01", nil, &prices); code != 200 || len(prices) != 0 {
		t.Fatalf("list prices before effective: %d %v", code, prices)
	}
}
```

Add to `testserver_test.go`: `import "strconv"` and `func itoa(i int64) string { return strconv.FormatInt(i, 10) }`.

`internal/httpapi/orders_test.go`:
```go
package httpapi

import "testing"

// seedCatalog creates a customer with a custom brisket price and returns (customerId, brisketId, sausageId).
func seedCatalog(t *testing.T, env *testEnv) (int64, int64, int64) {
	t.Helper()
	var c, b, s map[string]any
	env.do(t, env.office, "POST", "/api/office/customers", map[string]any{"name": "Blue Plate", "deliveryAddress": "1 Main St"}, &c)
	env.do(t, env.office, "POST", "/api/office/products", map[string]any{"sku": "BRIS", "name": "Brisket", "sellUnit": "case", "catchWeight": true, "approxCaseWeight": 6000, "basePriceCents": 599}, &b)
	env.do(t, env.office, "POST", "/api/office/products", map[string]any{"sku": "SAUS", "name": "Sausage", "sellUnit": "each", "basePriceCents": 400}, &s)
	cid, bid, sid := int64(c["id"].(float64)), int64(b["id"].(float64)), int64(s["id"].(float64))
	env.do(t, env.office, "POST", "/api/office/customers/"+itoa(cid)+"/prices", map[string]any{"productId": bid, "priceCents": 549, "effectiveFrom": "2026-01-01"}, nil)
	return cid, bid, sid
}

func TestOrderEndpoints(t *testing.T) {
	env := newTestEnv(t)
	cid, bid, sid := seedCatalog(t, env)
	var o map[string]any
	code := env.do(t, env.office, "POST", "/api/office/orders", map[string]any{"customerId": cid, "requestedDeliveryDate": "2026-09-12", "notes": "back door"}, &o)
	if code != 201 || o["status"] != "draft" || o["customer"].(map[string]any)["name"] != "Blue Plate" {
		t.Fatalf("create: %d %v", code, o)
	}
	oid := itoa(int64(o["id"].(float64)))
	code = env.do(t, env.office, "POST", "/api/office/orders/"+oid+"/lines", map[string]any{"productId": bid, "orderedQty": 200}, &o)
	lines := o["lines"].([]any)
	l0 := lines[0].(map[string]any)
	if code != 200 || len(lines) != 1 || l0["unitPriceCents"].(float64) != 549 || l0["estWeight"].(float64) != 12000 || l0["amountSource"] != "estimated" || o["totalCents"].(float64) != 65880 {
		t.Fatalf("add line: %d %v", code, o)
	}
	env.do(t, env.office, "POST", "/api/office/orders/"+oid+"/lines", map[string]any{"productId": sid, "orderedQty": 1000}, &o)
	lid := itoa(int64(l0["id"].(float64)))
	code = env.do(t, env.office, "PATCH", "/api/office/orders/"+oid+"/lines/"+lid, map[string]any{"shippedWeight": 11850, "unitPriceCents": 500}, &o)
	l0 = o["lines"].([]any)[0].(map[string]any)
	if code != 200 || l0["priceOverridden"] != true || l0["amountSource"] != "shipped" || l0["amountCents"].(float64) != 59250 {
		t.Fatalf("patch line: %d %v", code, l0)
	}
	var e map[string]any
	if code := env.do(t, env.office, "POST", "/api/office/orders/"+oid+"/finalize", nil, &e); code != 409 || e["code"] != "invalid_transition" {
		t.Fatalf("finalize draft: %d %v", code, e)
	}
	if code := env.do(t, env.office, "POST", "/api/office/orders/"+oid+"/confirm", nil, &o); code != 200 || o["status"] != "confirmed" {
		t.Fatalf("confirm: %d %v", code, o)
	}
	var list []map[string]any
	if code := env.do(t, env.office, "GET", "/api/office/orders?status=confirmed", nil, &list); code != 200 || len(list) != 1 || list[0]["lineCount"].(float64) != 2 {
		t.Fatalf("list: %d %v", code, list)
	}
	if code := env.do(t, env.office, "PATCH", "/api/office/orders/"+oid, map[string]any{"notes": "side door"}, &o); code != 200 || o["notes"] != "side door" {
		t.Fatalf("patch order: %d %v", code, o)
	}
	if code := env.do(t, env.office, "DELETE", "/api/office/orders/"+oid+"/lines/"+lid, nil, &o); code != 200 || len(o["lines"].([]any)) != 1 {
		t.Fatalf("delete line: %d %v", code, o)
	}
	if code := env.do(t, env.office, "POST", "/api/office/orders/"+oid+"/unconfirm", nil, &o); code != 200 || o["status"] != "draft" {
		t.Fatalf("unconfirm: %d %v", code, o)
	}
	if code := env.do(t, env.office, "POST", "/api/office/orders/"+oid+"/cancel", nil, &o); code != 200 || o["status"] != "cancelled" {
		t.Fatalf("cancel: %d %v", code, o)
	}
	if code := env.do(t, env.office, "POST", "/api/office/orders/"+oid+"/lines", map[string]any{"productId": sid, "orderedQty": 100}, &e); code != 409 || e["code"] != "locked" {
		t.Fatalf("locked: %d %v", code, e)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/httpapi/ -run 'TestCustomer|TestProduct|TestOrder' 2>&1 | tail -5`
Expected: failures with `not_implemented` (stubs) or 409 codes.

- [ ] **Step 3: Add converters**

Append to `internal/httpapi/convert.go`:
```go
import (
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefBool(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func derefI64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func nullI64(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

func nullT(n sql.NullTime) *time.Time {
	if !n.Valid {
		return nil
	}
	v := n.Time
	return &v
}

func toCustomer(c queries.Customer) api.Customer {
	return api.Customer{
		Id: c.ID, Name: c.Name, BillingAddress: c.BillingAddress, DeliveryAddress: c.DeliveryAddress, ContactName: c.ContactName,
		Phone: c.Phone, Email: c.Email, DeliveryNotes: c.DeliveryNotes, DeliveryDays: service.SplitDeliveryDays(c.DeliveryDays),
		QboCustomerId: strPtr(c.QboCustomerID.String), Active: c.Active,
	}
}

func customerInput(in api.CustomerInput) service.CustomerInput {
	days := []string{}
	if in.DeliveryDays != nil {
		days = *in.DeliveryDays
	}
	return service.CustomerInput{
		Name: in.Name, BillingAddress: deref(in.BillingAddress), DeliveryAddress: deref(in.DeliveryAddress), ContactName: deref(in.ContactName),
		Phone: deref(in.Phone), Email: deref(in.Email), DeliveryNotes: deref(in.DeliveryNotes), DeliveryDays: days,
		QBOCustomerID: deref(in.QboCustomerId), Active: derefBool(in.Active, true),
	}
}

func toProduct(p queries.Product) api.Product {
	return api.Product{
		Id: p.ID, Sku: p.Sku, Name: p.Name, Category: p.Category, SellUnit: api.ProductSellUnit(p.SellUnit), CatchWeight: p.CatchWeight,
		ApproxCaseWeight: nullI64(p.ApproxCaseWeight), BasePriceCents: p.BasePriceCents, QboItemId: strPtr(p.QboItemID.String), Active: p.Active,
	}
}

func productInput(in api.ProductInput) service.ProductInput {
	return service.ProductInput{
		SKU: in.Sku, Name: in.Name, Category: deref(in.Category), SellUnit: string(in.SellUnit), CatchWeight: derefBool(in.CatchWeight, false),
		ApproxCaseWeight: in.ApproxCaseWeight, BasePriceCents: in.BasePriceCents, QBOItemID: deref(in.QboItemId), Active: derefBool(in.Active, true),
	}
}

func toPriceRow(id, productID int64, sku, name string, cents int64, from string) api.CustomerPrice {
	return api.CustomerPrice{Id: id, ProductId: productID, Sku: sku, ProductName: name, PriceCents: cents, EffectiveFrom: from}
}

func toOrderSummary(id, customerID int64, customerName, date, status string, needsReview bool, lineCount int64, notes string) api.OrderSummary {
	return api.OrderSummary{Id: id, CustomerId: customerID, CustomerName: customerName, RequestedDeliveryDate: date,
		Status: api.OrderStatus(status), NeedsReview: needsReview, LineCount: lineCount, Notes: notes}
}

func toLine(l service.LineDetail) api.OrderLine {
	return api.OrderLine{
		Id: l.Line.ID, ProductId: l.Product.ID, Sku: l.Product.Sku, ProductName: l.Product.Name, SellUnit: l.Product.SellUnit,
		CatchWeight: l.Product.CatchWeight, OrderedQty: l.Line.OrderedQty, UnitPriceCents: l.Line.UnitPriceCents, PriceOverridden: l.Line.PriceOverridden,
		EstWeight: nullI64(l.Line.EstWeight), ShippedWeight: nullI64(l.Line.ShippedWeight), DeliveredQty: nullI64(l.Line.DeliveredQty),
		DeliveredWeight: nullI64(l.Line.DeliveredWeight), ShortageNote: l.Line.ShortageNote, AmountCents: l.AmountCents,
		AmountSource: api.OrderLineAmountSource(l.AmountSource),
	}
}

func toOrder(d service.OrderDetail) api.Order {
	lines := make([]api.OrderLine, 0, len(d.Lines))
	for _, l := range d.Lines {
		lines = append(lines, toLine(l))
	}
	return api.Order{
		Id: d.Order.ID, Customer: toCustomer(d.Customer), RequestedDeliveryDate: d.Order.RequestedDeliveryDate, Status: api.OrderStatus(d.Order.Status),
		Notes: d.Order.Notes, NeedsReview: d.Order.NeedsReview, Lines: lines, TotalCents: d.TotalCents, RouteId: d.RouteID, StopId: d.StopID,
		RouteStatus: strPtr(d.RouteStatus), CreatedAt: d.Order.CreatedAt, FinalizedAt: nullT(d.Order.FinalizedAt),
	}
}
```
(Merge the import block with the existing one; add `database/sql` and `time`.)

- [ ] **Step 4: Implement handlers_catalog.go**

```go
package httpapi

import (
	"context"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

func (s *Server) ListCustomers(ctx context.Context, r api.ListCustomersRequestObject) (api.ListCustomersResponseObject, error) {
	rows, err := s.d.Svc.ListCustomers(ctx, derefBool(r.Params.IncludeInactive, false))
	if err != nil {
		return nil, err
	}
	out := make(api.ListCustomers200JSONResponse, 0, len(rows))
	for _, c := range rows {
		out = append(out, toCustomer(c))
	}
	return out, nil
}

func (s *Server) CreateCustomer(ctx context.Context, r api.CreateCustomerRequestObject) (api.CreateCustomerResponseObject, error) {
	c, err := s.d.Svc.CreateCustomer(ctx, customerInput(*r.Body))
	if err != nil {
		return nil, err
	}
	return api.CreateCustomer201JSONResponse(toCustomer(c)), nil
}

func (s *Server) GetCustomer(ctx context.Context, r api.GetCustomerRequestObject) (api.GetCustomerResponseObject, error) {
	c, err := s.d.Svc.GetCustomer(ctx, r.CustomerId)
	if err != nil {
		return nil, err
	}
	return api.GetCustomer200JSONResponse(toCustomer(c)), nil
}

func (s *Server) UpdateCustomer(ctx context.Context, r api.UpdateCustomerRequestObject) (api.UpdateCustomerResponseObject, error) {
	c, err := s.d.Svc.UpdateCustomer(ctx, r.CustomerId, customerInput(*r.Body))
	if err != nil {
		return nil, err
	}
	return api.UpdateCustomer200JSONResponse(toCustomer(c)), nil
}

func (s *Server) ListCustomerPrices(ctx context.Context, r api.ListCustomerPricesRequestObject) (api.ListCustomerPricesResponseObject, error) {
	asOf := deref(r.Params.AsOf)
	if asOf == "" {
		asOf = s.d.Svc.Today()
	}
	if !service.ValidDate(asOf) {
		return nil, service.Invalid(map[string]string{"asOf": "must be YYYY-MM-DD"})
	}
	rows, err := s.d.Svc.ListCustomerPrices(ctx, r.CustomerId, asOf)
	if err != nil {
		return nil, err
	}
	out := make(api.ListCustomerPrices200JSONResponse, 0, len(rows))
	for _, p := range rows {
		out = append(out, toPriceRow(p.ID, p.ProductID, p.Sku, p.ProductName, p.PriceCents, p.EffectiveFrom))
	}
	return out, nil
}

func (s *Server) SetCustomerPrice(ctx context.Context, r api.SetCustomerPriceRequestObject) (api.SetCustomerPriceResponseObject, error) {
	p, err := s.d.Svc.SetCustomerPrice(ctx, r.CustomerId, service.PriceInput{ProductID: r.Body.ProductId, PriceCents: r.Body.PriceCents, EffectiveFrom: r.Body.EffectiveFrom})
	if err != nil {
		return nil, err
	}
	prod, err := s.d.Svc.GetProduct(ctx, p.ProductID)
	if err != nil {
		return nil, err
	}
	return api.SetCustomerPrice201JSONResponse(toPriceRow(p.ID, p.ProductID, prod.Sku, prod.Name, p.PriceCents, p.EffectiveFrom)), nil
}

func (s *Server) ListProducts(ctx context.Context, r api.ListProductsRequestObject) (api.ListProductsResponseObject, error) {
	rows, err := s.d.Svc.ListProducts(ctx, derefBool(r.Params.IncludeInactive, false))
	if err != nil {
		return nil, err
	}
	out := make(api.ListProducts200JSONResponse, 0, len(rows))
	for _, p := range rows {
		out = append(out, toProduct(p))
	}
	return out, nil
}

func (s *Server) CreateProduct(ctx context.Context, r api.CreateProductRequestObject) (api.CreateProductResponseObject, error) {
	p, err := s.d.Svc.CreateProduct(ctx, productInput(*r.Body))
	if err != nil {
		return nil, err
	}
	return api.CreateProduct201JSONResponse(toProduct(p)), nil
}

func (s *Server) GetProduct(ctx context.Context, r api.GetProductRequestObject) (api.GetProductResponseObject, error) {
	p, err := s.d.Svc.GetProduct(ctx, r.ProductId)
	if err != nil {
		return nil, err
	}
	return api.GetProduct200JSONResponse(toProduct(p)), nil
}

func (s *Server) UpdateProduct(ctx context.Context, r api.UpdateProductRequestObject) (api.UpdateProductResponseObject, error) {
	p, err := s.d.Svc.UpdateProduct(ctx, r.ProductId, productInput(*r.Body))
	if err != nil {
		return nil, err
	}
	return api.UpdateProduct200JSONResponse(toProduct(p)), nil
}
```

- [ ] **Step 5: Implement handlers_orders.go**

```go
package httpapi

import (
	"context"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

func (s *Server) ListOrders(ctx context.Context, r api.ListOrdersRequestObject) (api.ListOrdersResponseObject, error) {
	rows, err := s.d.Svc.ListOrders(ctx, service.OrderFilter{Date: deref(r.Params.Date), Status: deref(r.Params.Status), CustomerID: derefI64(r.Params.CustomerId)})
	if err != nil {
		return nil, err
	}
	out := make(api.ListOrders200JSONResponse, 0, len(rows))
	for _, o := range rows {
		out = append(out, toOrderSummary(o.ID, o.CustomerID, o.CustomerName, o.RequestedDeliveryDate, o.Status, o.NeedsReview, o.LineCount, o.Notes))
	}
	return out, nil
}

func (s *Server) CreateOrder(ctx context.Context, r api.CreateOrderRequestObject) (api.CreateOrderResponseObject, error) {
	sess, _ := auth.SessionFrom(ctx)
	o, err := s.d.Svc.CreateOrder(ctx, service.OrderInput{CustomerID: r.Body.CustomerId, RequestedDeliveryDate: r.Body.RequestedDeliveryDate, Notes: deref(r.Body.Notes), CreatedBy: sess.UserID})
	if err != nil {
		return nil, err
	}
	return api.CreateOrder201JSONResponse(toOrder(o)), nil
}

func (s *Server) GetOrder(ctx context.Context, r api.GetOrderRequestObject) (api.GetOrderResponseObject, error) {
	o, err := s.d.Svc.GetOrder(ctx, r.OrderId)
	if err != nil {
		return nil, err
	}
	return api.GetOrder200JSONResponse(toOrder(o)), nil
}

func (s *Server) UpdateOrder(ctx context.Context, r api.UpdateOrderRequestObject) (api.UpdateOrderResponseObject, error) {
	o, err := s.d.Svc.UpdateOrder(ctx, r.OrderId, r.Body.Notes, r.Body.RequestedDeliveryDate)
	if err != nil {
		return nil, err
	}
	return api.UpdateOrder200JSONResponse(toOrder(o)), nil
}

func (s *Server) AddOrderLine(ctx context.Context, r api.AddOrderLineRequestObject) (api.AddOrderLineResponseObject, error) {
	o, err := s.d.Svc.AddLine(ctx, r.OrderId, r.Body.ProductId, r.Body.OrderedQty)
	if err != nil {
		return nil, err
	}
	return api.AddOrderLine200JSONResponse(toOrder(o)), nil
}

func (s *Server) UpdateOrderLine(ctx context.Context, r api.UpdateOrderLineRequestObject) (api.UpdateOrderLineResponseObject, error) {
	b := r.Body
	o, err := s.d.Svc.UpdateLine(ctx, r.OrderId, r.LineId, service.LinePatch{OrderedQty: b.OrderedQty, UnitPriceCents: b.UnitPriceCents, ShippedWeight: b.ShippedWeight,
		DeliveredQty: b.DeliveredQty, DeliveredWeight: b.DeliveredWeight, ShortageNote: b.ShortageNote})
	if err != nil {
		return nil, err
	}
	return api.UpdateOrderLine200JSONResponse(toOrder(o)), nil
}

func (s *Server) DeleteOrderLine(ctx context.Context, r api.DeleteOrderLineRequestObject) (api.DeleteOrderLineResponseObject, error) {
	o, err := s.d.Svc.DeleteLine(ctx, r.OrderId, r.LineId)
	if err != nil {
		return nil, err
	}
	return api.DeleteOrderLine200JSONResponse(toOrder(o)), nil
}

func (s *Server) ConfirmOrder(ctx context.Context, r api.ConfirmOrderRequestObject) (api.ConfirmOrderResponseObject, error) {
	o, err := s.d.Svc.ConfirmOrder(ctx, r.OrderId)
	if err != nil {
		return nil, err
	}
	return api.ConfirmOrder200JSONResponse(toOrder(o)), nil
}

func (s *Server) UnconfirmOrder(ctx context.Context, r api.UnconfirmOrderRequestObject) (api.UnconfirmOrderResponseObject, error) {
	o, err := s.d.Svc.UnconfirmOrder(ctx, r.OrderId)
	if err != nil {
		return nil, err
	}
	return api.UnconfirmOrder200JSONResponse(toOrder(o)), nil
}

func (s *Server) CancelOrder(ctx context.Context, r api.CancelOrderRequestObject) (api.CancelOrderResponseObject, error) {
	o, err := s.d.Svc.CancelOrder(ctx, r.OrderId)
	if err != nil {
		return nil, err
	}
	return api.CancelOrder200JSONResponse(toOrder(o)), nil
}

func (s *Server) FinalizeOrder(ctx context.Context, r api.FinalizeOrderRequestObject) (api.FinalizeOrderResponseObject, error) {
	o, err := s.d.Svc.FinalizeOrder(ctx, r.OrderId)
	if err != nil {
		return nil, err
	}
	return api.FinalizeOrder200JSONResponse(toOrder(o)), nil
}
```

Delete the matching stubs from `stubs_tmp.go`.

- [ ] **Step 6: Run tests to verify they pass**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go vet ./... && go test ./internal/httpapi/ 2>&1 | tail -5`
Expected: `ok` (including `TestSPAFallbackAndErrorShape` from Task 10 now that customer endpoints exist).

- [ ] **Step 7: Commit**

```bash
cd ~/deepcutsCRM && git add -A && git commit -q -m "Add office API handlers for catalog and orders" && git push
```

---

### Task 12: Route, day view, and driver API handlers

**Files:**
- Create: `internal/httpapi/handlers_routes.go`, `internal/httpapi/handlers_driver.go`, `internal/httpapi/routes_test.go`, `internal/httpapi/driver_test.go`
- Modify: `internal/httpapi/convert.go`; delete `internal/httpapi/stubs_tmp.go`

**Interfaces:**
- Consumes: Tasks 7 and 8 service methods.
- Produces in convert.go: `toStop(service.StopDetail) api.Stop`, `toRoute(service.RouteDetail) api.Route`, `toDriverStop(service.DriverStop) api.DriverStop`, `toDriverRoute(service.DriverRoute, driverName string) api.DriverRoute`, `driverActionInput(api.DriverAction) (domain.DriverAction, error)`.

- [ ] **Step 1: Write the failing tests**

`internal/httpapi/routes_test.go`:
```go
package httpapi

import "testing"

// confirmedOrder creates a confirmed order for the date with one brisket line (2 cases) and returns its id.
func confirmedOrder(t *testing.T, env *testEnv, cid, bid int64, date string) int64 {
	t.Helper()
	var o map[string]any
	env.do(t, env.office, "POST", "/api/office/orders", map[string]any{"customerId": cid, "requestedDeliveryDate": date}, &o)
	oid := int64(o["id"].(float64))
	env.do(t, env.office, "POST", "/api/office/orders/"+itoa(oid)+"/lines", map[string]any{"productId": bid, "orderedQty": 200}, &o)
	if code := env.do(t, env.office, "POST", "/api/office/orders/"+itoa(oid)+"/confirm", nil, &o); code != 200 {
		t.Fatalf("confirm: %d %v", code, o)
	}
	return oid
}

func TestDayViewAndRoutes(t *testing.T) {
	env := newTestEnv(t)
	cid, bid, _ := seedCatalog(t, env)
	o1 := confirmedOrder(t, env, cid, bid, "2026-09-10")
	o2 := confirmedOrder(t, env, cid, bid, "2026-09-10")

	var day map[string]any
	if code := env.do(t, env.office, "GET", "/api/office/day?date=2026-09-10", nil, &day); code != 200 || len(day["unscheduled"].([]any)) != 2 || len(day["routes"].([]any)) != 0 {
		t.Fatalf("day: %d %v", code, day)
	}
	var drivers []map[string]any
	if code := env.do(t, env.office, "GET", "/api/office/drivers", nil, &drivers); code != 200 || len(drivers) != 1 {
		t.Fatalf("drivers: %d %v", code, drivers)
	}
	var r map[string]any
	if code := env.do(t, env.office, "POST", "/api/office/routes", map[string]any{"routeDate": "2026-09-10", "driverUserId": env.driverID, "truckLabel": "Reefer 1"}, &r); code != 201 || r["driverName"] != "Sam" {
		t.Fatalf("create route: %d %v", code, r)
	}
	rid := itoa(int64(r["id"].(float64)))
	env.do(t, env.office, "POST", "/api/office/routes/"+rid+"/stops", map[string]any{"orderId": o1}, &r)
	if code := env.do(t, env.office, "POST", "/api/office/routes/"+rid+"/stops", map[string]any{"orderId": o2}, &r); code != 200 || len(r["stops"].([]any)) != 2 {
		t.Fatalf("add stops: %d %v", code, r)
	}
	stops := r["stops"].([]any)
	s1 := itoa(int64(stops[0].(map[string]any)["id"].(float64)))
	s2 := itoa(int64(stops[1].(map[string]any)["id"].(float64)))
	if code := env.do(t, env.office, "PUT", "/api/office/routes/"+rid+"/sequence", map[string]any{"stopIds": []string{s2, s1}}, &r); code != 422 {
		t.Fatalf("sequence with strings must be 422: %d", code)
	}
	if code := env.do(t, env.office, "PUT", "/api/office/routes/"+rid+"/sequence", map[string]any{"stopIds": []int64{int64(stops[1].(map[string]any)["id"].(float64)), int64(stops[0].(map[string]any)["id"].(float64))}}, &r); code != 200 || r["stops"].([]any)[0].(map[string]any)["orderId"].(float64) != float64(o2) {
		t.Fatalf("reorder: %d %v", code, r)
	}
	if code := env.do(t, env.office, "DELETE", "/api/office/routes/"+rid+"/stops/"+s1, nil, &r); code != 200 || len(r["stops"].([]any)) != 1 {
		t.Fatalf("remove stop: %d %v", code, r)
	}
	env.do(t, env.office, "GET", "/api/office/day?date=2026-09-10", nil, &day)
	if len(day["unscheduled"].([]any)) != 1 || len(day["routes"].([]any)) != 1 {
		t.Fatalf("day after: %v", day)
	}
	if code := env.do(t, env.office, "POST", "/api/office/routes/"+rid+"/out", nil, &r); code != 200 || r["status"] != "out" {
		t.Fatalf("out: %d %v", code, r)
	}
	var e map[string]any
	if code := env.do(t, env.office, "POST", "/api/office/routes/"+rid+"/complete", nil, &e); code != 409 || e["code"] != "stops_pending" {
		t.Fatalf("complete with pending: %d %v", code, e)
	}
	if code := env.do(t, env.office, "GET", "/api/office/routes/"+rid, nil, &r); code != 200 || r["stops"].([]any)[0].(map[string]any)["customerName"] != "Blue Plate" {
		t.Fatalf("get route: %d %v", code, r)
	}
}
```

`internal/httpapi/driver_test.go`:
```go
package httpapi

import (
	"encoding/base64"
	"io"
	"testing"
)

const cidA = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
const cidB = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"

var pngBytes = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0}

// outRoute schedules one confirmed order on a route for today and marks it out. Returns (routeId, stopId, orderId).
func outRoute(t *testing.T, env *testEnv) (string, string, int64) {
	t.Helper()
	cid, bid, _ := seedCatalog(t, env)
	oid := confirmedOrder(t, env, cid, bid, "2026-09-10")
	var r map[string]any
	env.do(t, env.office, "POST", "/api/office/routes", map[string]any{"routeDate": "2026-09-10", "driverUserId": env.driverID}, &r)
	rid := itoa(int64(r["id"].(float64)))
	env.do(t, env.office, "POST", "/api/office/routes/"+rid+"/stops", map[string]any{"orderId": oid}, &r)
	var o map[string]any
	env.do(t, env.office, "GET", "/api/office/orders/"+itoa(oid), nil, &o)
	lid := itoa(int64(o["lines"].([]any)[0].(map[string]any)["id"].(float64)))
	env.do(t, env.office, "PATCH", "/api/office/orders/"+itoa(oid)+"/lines/"+lid, map[string]any{"shippedWeight": 11850}, nil)
	if code := env.do(t, env.office, "POST", "/api/office/routes/"+rid+"/out", nil, &r); code != 200 {
		t.Fatalf("out: %d %v", code, r)
	}
	sid := itoa(int64(r["stops"].([]any)[0].(map[string]any)["id"].(float64)))
	return rid, sid, oid
}

func TestDriverRouteAndActions(t *testing.T) {
	env := newTestEnv(t)
	rid, sid, oid := outRoute(t, env)
	var dr map[string]any
	if code := env.do(t, env.driver, "GET", "/api/driver/route", nil, &dr); code != 200 || len(dr["stops"].([]any)) != 1 {
		t.Fatalf("driver route: %d %v", code, dr)
	}
	stop := dr["stops"].([]any)[0].(map[string]any)
	if stop["stop"].(map[string]any)["customerName"] != "Blue Plate" || len(stop["order"].(map[string]any)["lines"].([]any)) != 1 {
		t.Fatalf("driver stop shape: %v", stop)
	}
	if code := env.do(t, env.driver, "GET", "/api/driver/route?date=2026-01-01", nil, nil); code != 404 {
		t.Fatalf("no route that day: %d", code)
	}
	lineID := int64(stop["order"].(map[string]any)["lines"].([]any)[0].(map[string]any)["id"].(float64))

	var res map[string]any
	code := env.do(t, env.driver, "POST", "/api/driver/stops/"+sid+"/actions", map[string]any{
		"clientId": cidA, "type": "adjust", "lines": []map[string]any{{"lineId": lineID, "deliveredWeight": 6000, "shortageNote": "one case rejected"}},
	}, &res)
	if code != 200 || res["applied"] != true {
		t.Fatalf("adjust: %d %v", code, res)
	}
	code = env.do(t, env.driver, "POST", "/api/driver/stops/"+sid+"/actions", map[string]any{
		"clientId": cidB, "type": "deliver", "proof": map[string]any{"type": "signature", "dataBase64": base64.StdEncoding.EncodeToString(pngBytes)}, "note": "left at dock",
	}, &res)
	st := res["stop"].(map[string]any)["stop"].(map[string]any)
	if code != 200 || res["applied"] != true || st["status"] != "delivered" || st["hasProofImage"] != true || st["proofType"] != "signature" {
		t.Fatalf("deliver: %d %v", code, res)
	}
	code = env.do(t, env.driver, "POST", "/api/driver/stops/"+sid+"/actions", map[string]any{
		"clientId": cidB, "type": "deliver", "proof": map[string]any{"type": "name", "name": "Pat"},
	}, &res)
	if code != 200 || res["applied"] != false {
		t.Fatalf("replay must be 200 applied=false: %d %v", code, res)
	}
	var e map[string]any
	if code := env.do(t, env.driver, "POST", "/api/driver/stops/"+sid+"/actions", map[string]any{"clientId": "nope", "type": "skip", "skipReason": "x"}, &e); code != 422 {
		t.Fatalf("bad uuid: %d %v", code, e)
	}
	// office sees the proof image and the review flag
	res2, err := env.office.Get(env.srv.URL + "/api/office/stops/" + sid + "/proof")
	if err != nil || res2.StatusCode != 200 || res2.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("proof: %v %d", err, res2.StatusCode)
	}
	b, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	if string(b) != string(pngBytes) {
		t.Fatal("proof bytes differ")
	}
	var o map[string]any
	env.do(t, env.office, "GET", "/api/office/orders/"+itoa(oid), nil, &o)
	if o["status"] != "delivered" || o["needsReview"] != true || o["lines"].([]any)[0].(map[string]any)["deliveredWeight"].(float64) != 6000 {
		t.Fatalf("order after delivery: %v", o)
	}
	if code := env.do(t, env.driver, "POST", "/api/driver/routes/"+rid+"/complete", nil, &dr); code != 200 || dr["route"].(map[string]any)["status"] != "complete" {
		t.Fatalf("complete: %d %v", code, dr)
	}
	if code := env.do(t, env.office, "POST", "/api/office/orders/"+itoa(oid)+"/finalize", nil, &o); code != 200 || o["status"] != "finalized" || o["needsReview"] != false {
		t.Fatalf("finalize: %d %v", code, o)
	}
}

func TestDriverProofTooLarge(t *testing.T) {
	env := newTestEnv(t)
	_, sid, _ := outRoute(t, env)
	big := make([]byte, 301*1024)
	copy(big, pngBytes)
	var e map[string]any
	code := env.do(t, env.driver, "POST", "/api/driver/stops/"+sid+"/actions", map[string]any{
		"clientId": cidA, "type": "deliver", "proof": map[string]any{"type": "photo", "dataBase64": base64.StdEncoding.EncodeToString(big)},
	}, &e)
	if code != 422 || e["fields"].(map[string]any)["proof"] == nil {
		t.Fatalf("oversized proof: %d %v", code, e)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/httpapi/ -run 'TestDayView|TestDriver' 2>&1 | tail -5`
Expected: failures.

- [ ] **Step 3: Add converters**

Append to `convert.go`:
```go
func toStop(sd service.StopDetail) api.Stop {
	st := sd.Stop
	out := api.Stop{
		Id: st.ID, RouteId: st.RouteID, OrderId: st.OrderID, Sequence: st.Sequence, Status: api.StopStatus(st.Status), DeliveredAt: nullT(st.DeliveredAt),
		SkipReason: st.SkipReason, DriverNote: st.DriverNote, CustomerName: sd.Customer.Name, DeliveryAddress: sd.Customer.DeliveryAddress,
		Phone: sd.Customer.Phone, ContactName: sd.Customer.ContactName, DeliveryNotes: sd.Customer.DeliveryNotes,
		OrderStatus: api.OrderStatus(sd.Order.Status), NeedsReview: sd.Order.NeedsReview, LineCount: sd.LineCount,
	}
	if st.ProofType.Valid {
		pt := api.StopProofType(st.ProofType.String)
		out.ProofType = &pt
		if st.ProofType.String == "name" {
			out.ProofName = strPtr(st.ProofRef.String)
		} else {
			out.HasProofImage = st.ProofRef.Valid
		}
	}
	return out
}

func toRoute(rd service.RouteDetail) api.Route {
	stops := make([]api.Stop, 0, len(rd.Stops))
	for _, s := range rd.Stops {
		stops = append(stops, toStop(s))
	}
	r := rd.Route
	return api.Route{Id: r.ID, RouteDate: r.RouteDate, DriverUserId: r.DriverUserID, DriverName: rd.DriverName, TruckLabel: r.TruckLabel,
		Status: api.RouteStatus(r.Status), OutAt: nullT(r.OutAt), CompletedAt: nullT(r.CompletedAt), Stops: stops}
}

func toDriverStop(ds service.DriverStop) api.DriverStop {
	sd := service.StopDetail{Stop: ds.Stop, Order: ds.Order.Order, Customer: ds.Customer, LineCount: int64(len(ds.Order.Lines))}
	return api.DriverStop{Stop: toStop(sd), Order: toOrder(ds.Order)}
}

func toDriverRoute(dr service.DriverRoute, driverName string) api.DriverRoute {
	stops := make([]api.DriverStop, 0, len(dr.Stops))
	for _, s := range dr.Stops {
		stops = append(stops, toDriverStop(s))
	}
	r := dr.Route
	return api.DriverRoute{
		Route: api.Route{Id: r.ID, RouteDate: r.RouteDate, DriverUserId: r.DriverUserID, DriverName: driverName, TruckLabel: r.TruckLabel,
			Status: api.RouteStatus(r.Status), OutAt: nullT(r.OutAt), CompletedAt: nullT(r.CompletedAt), Stops: []api.Stop{}},
		Stops: stops,
	}
}

func driverActionInput(in api.DriverAction) (domain.DriverAction, error) {
	a := domain.DriverAction{ClientID: in.ClientId, Type: domain.ActionType(in.Type), Note: deref(in.Note), SkipReason: deref(in.SkipReason)}
	if in.Proof != nil {
		p := &domain.Proof{Type: domain.ProofType(in.Proof.Type), Name: deref(in.Proof.Name)}
		if in.Proof.DataBase64 != nil {
			if len(*in.Proof.DataBase64) > (domain.MaxProofBytes*4/3)+4 {
				return a, service.Invalid(map[string]string{"proof": "image larger than 300 KB"})
			}
			data, err := base64.StdEncoding.DecodeString(*in.Proof.DataBase64)
			if err != nil {
				return a, service.Invalid(map[string]string{"proof": "dataBase64 is not valid base64"})
			}
			p.Data = data
		}
		a.Proof = p
	}
	if in.Lines != nil {
		for _, l := range *in.Lines {
			adj := domain.LineAdjustment{LineID: l.LineId, ShortageNote: l.ShortageNote}
			if l.DeliveredQty != nil {
				v := domain.Hundredths(*l.DeliveredQty)
				adj.DeliveredQty = &v
			}
			if l.DeliveredWeight != nil {
				v := domain.Hundredths(*l.DeliveredWeight)
				adj.DeliveredWeight = &v
			}
			a.Lines = append(a.Lines, adj)
		}
	}
	return a, nil
}
```
(add imports `encoding/base64` and `internal/domain`.)

- [ ] **Step 4: Implement handlers_routes.go**

```go
package httpapi

import (
	"context"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
	"strconv"
)

func (s *Server) GetDay(ctx context.Context, r api.GetDayRequestObject) (api.GetDayResponseObject, error) {
	dv, err := s.d.Svc.DayView(ctx, r.Params.Date)
	if err != nil {
		return nil, err
	}
	out := api.DayView{Date: dv.Date, Unscheduled: []api.OrderSummary{}, Routes: []api.Route{}}
	for _, o := range dv.Unscheduled {
		out.Unscheduled = append(out.Unscheduled, toOrderSummary(o.ID, o.CustomerID, o.CustomerName, o.RequestedDeliveryDate, o.Status, o.NeedsReview, o.LineCount, o.Notes))
	}
	for _, rt := range dv.Routes {
		out.Routes = append(out.Routes, toRoute(rt))
	}
	return api.GetDay200JSONResponse(out), nil
}

func (s *Server) ListDrivers(ctx context.Context, _ api.ListDriversRequestObject) (api.ListDriversResponseObject, error) {
	users, err := s.d.Svc.ListDrivers(ctx)
	if err != nil {
		return nil, err
	}
	out := make(api.ListDrivers200JSONResponse, 0, len(users))
	for _, u := range users {
		out = append(out, api.Driver{Id: u.ID, DisplayName: u.DisplayName})
	}
	return out, nil
}

func (s *Server) CreateRoute(ctx context.Context, r api.CreateRouteRequestObject) (api.CreateRouteResponseObject, error) {
	rd, err := s.d.Svc.CreateRoute(ctx, service.RouteInput{RouteDate: r.Body.RouteDate, DriverUserID: r.Body.DriverUserId, TruckLabel: deref(r.Body.TruckLabel)})
	if err != nil {
		return nil, err
	}
	return api.CreateRoute201JSONResponse(toRoute(rd)), nil
}

func (s *Server) GetRoute(ctx context.Context, r api.GetRouteRequestObject) (api.GetRouteResponseObject, error) {
	rd, err := s.d.Svc.GetRoute(ctx, r.RouteId)
	if err != nil {
		return nil, err
	}
	return api.GetRoute200JSONResponse(toRoute(rd)), nil
}

func (s *Server) AddStop(ctx context.Context, r api.AddStopRequestObject) (api.AddStopResponseObject, error) {
	rd, err := s.d.Svc.AddStop(ctx, r.RouteId, r.Body.OrderId)
	if err != nil {
		return nil, err
	}
	return api.AddStop200JSONResponse(toRoute(rd)), nil
}

func (s *Server) RemoveStop(ctx context.Context, r api.RemoveStopRequestObject) (api.RemoveStopResponseObject, error) {
	rd, err := s.d.Svc.RemoveStop(ctx, r.RouteId, r.StopId)
	if err != nil {
		return nil, err
	}
	return api.RemoveStop200JSONResponse(toRoute(rd)), nil
}

func (s *Server) ReorderStops(ctx context.Context, r api.ReorderStopsRequestObject) (api.ReorderStopsResponseObject, error) {
	rd, err := s.d.Svc.ReorderStops(ctx, r.RouteId, r.Body.StopIds)
	if err != nil {
		return nil, err
	}
	return api.ReorderStops200JSONResponse(toRoute(rd)), nil
}

func (s *Server) RouteOut(ctx context.Context, r api.RouteOutRequestObject) (api.RouteOutResponseObject, error) {
	rd, err := s.d.Svc.RouteOut(ctx, r.RouteId)
	if err != nil {
		return nil, err
	}
	return api.RouteOut200JSONResponse(toRoute(rd)), nil
}

func (s *Server) RouteComplete(ctx context.Context, r api.RouteCompleteRequestObject) (api.RouteCompleteResponseObject, error) {
	rd, err := s.d.Svc.RouteComplete(ctx, r.RouteId)
	if err != nil {
		return nil, err
	}
	return api.RouteComplete200JSONResponse(toRoute(rd)), nil
}

// getStopProof streams the signature/photo for a stop (plain handler: binary response).
func (s *Server) getStopProof(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "stopId"), 10, 64)
	if err != nil {
		writeError(w, service.Invalid(map[string]string{"stopId": "must be an integer"}))
		return
	}
	st, err := s.d.Svc.Q.GetStop(r.Context(), id)
	if err != nil || !st.ProofRef.Valid || st.ProofType.String == "name" {
		writeError(w, service.NotFound("proof"))
		return
	}
	rc, ct, err := s.d.Proofs.Open(st.ProofRef.String)
	if err != nil {
		writeError(w, service.NotFound("proof"))
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", ct)
	io.Copy(w, rc)
}
```

- [ ] **Step 5: Implement handlers_driver.go**

```go
package httpapi

import (
	"context"

	"github.com/cmcgeedev/deepcutsCRM/internal/api"
	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

func (s *Server) GetDriverRoute(ctx context.Context, r api.GetDriverRouteRequestObject) (api.GetDriverRouteResponseObject, error) {
	sess, _ := auth.SessionFrom(ctx)
	date := deref(r.Params.Date)
	if date == "" {
		date = s.d.Svc.Today()
	}
	if !service.ValidDate(date) {
		return nil, service.Invalid(map[string]string{"date": "must be YYYY-MM-DD"})
	}
	dr, err := s.d.Svc.DriverRouteForDate(ctx, sess.UserID, date)
	if err != nil {
		return nil, err
	}
	return api.GetDriverRoute200JSONResponse(toDriverRoute(dr, sess.DisplayName)), nil
}

func (s *Server) PostDriverAction(ctx context.Context, r api.PostDriverActionRequestObject) (api.PostDriverActionResponseObject, error) {
	sess, _ := auth.SessionFrom(ctx)
	a, err := driverActionInput(*r.Body)
	if err != nil {
		return nil, err
	}
	applied, st, err := s.d.Svc.ApplyDriverAction(ctx, sess.UserID, r.StopId, a)
	if err != nil {
		return nil, err
	}
	return api.PostDriverAction200JSONResponse(api.DriverActionResult{Applied: applied, Stop: toDriverStop(st)}), nil
}

func (s *Server) DriverCompleteRoute(ctx context.Context, r api.DriverCompleteRouteRequestObject) (api.DriverCompleteRouteResponseObject, error) {
	sess, _ := auth.SessionFrom(ctx)
	dr, err := s.d.Svc.DriverCompleteRoute(ctx, sess.UserID, r.RouteId)
	if err != nil {
		return nil, err
	}
	return api.DriverCompleteRoute200JSONResponse(toDriverRoute(dr, sess.DisplayName)), nil
}
```

Delete `internal/httpapi/stubs_tmp.go`.

- [ ] **Step 6: Run the whole Go suite and commit**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go vet ./... && go test ./... 2>&1 | tail -8`
Expected: all `ok`.

```bash
cd ~/deepcutsCRM && git add -A && git commit -q -m "Add route, day view and driver API handlers; remove stubs" && git push
```

---

### Task 13: CSV importers and the import CLI

**Files:**
- Create: `internal/importer/importer.go`, `internal/importer/importer_test.go`, `cmd/deepcuts/cmd_import.go`, `docs/import-formats.md`
- Modify: `cmd/deepcuts/main.go` (remove `runImport` stub line)

**Interfaces:**
- Produces:
  - `type Summary struct { Created, Updated, Rejected int; Errors []string }`; `func (s Summary) String() string`
  - `func Customers(ctx, svc *service.Service, r io.Reader) (Summary, error)`
  - `func Products(ctx, svc *service.Service, r io.Reader) (Summary, error)`
  - `func Prices(ctx, svc *service.Service, r io.Reader, effectiveFrom string) (Summary, error)`
  - CSV headers (case-insensitive, order-free):
    - customers: `name, billing_address, delivery_address, contact_name, phone, email, delivery_notes, delivery_days, qbo_customer_id`. Match on `qbo_customer_id` when present, else exact `name`. `delivery_days` is `mon;thu`.
    - products: `sku, name, category, sell_unit, catch_weight, approx_case_weight_lb, base_price, qbo_item_id`. Match on `sku`. `catch_weight` is `true/false/yes/no/1/0`. `approx_case_weight_lb` and `base_price` are decimals (`60`, `5.99`), converted to hundredths and cents.
    - prices: `customer, sku, price`. `customer` matches `qbo_customer_id` then `name`. `price` decimal dollars.
  - Never deletes. Re-run safe: existing rows are updated in place.

- [ ] **Step 1: Write the failing tests**

`internal/importer/importer_test.go`:
```go
package importer

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

var ctx = context.Background()

func newSvc(t *testing.T) *service.Service {
	t.Helper()
	d, err := db.OpenAndMigrate(ctx, filepath.Join(t.TempDir(), "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	loc, _ := time.LoadLocation("America/New_York")
	return service.New(d, loc)
}

func TestCustomersImportAndRerun(t *testing.T) {
	s := newSvc(t)
	csv := "Name,Delivery_Address,delivery_days,qbo_customer_id,phone\nBlue Plate,1 Main St,mon;thu,q1,555\nNo QBO,2 Side St,,,\n,3 Nowhere,,,\n"
	sum, err := Customers(ctx, s, strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	if sum.Created != 2 || sum.Rejected != 1 || len(sum.Errors) != 1 {
		t.Fatalf("first run: %+v", sum)
	}
	csv2 := "name,delivery_address,qbo_customer_id\nBlue Plate Diner,9 New St,q1\nNo QBO,2 Side St,\n"
	sum, err = Customers(ctx, s, strings.NewReader(csv2))
	if err != nil {
		t.Fatal(err)
	}
	if sum.Created != 0 || sum.Updated != 2 {
		t.Fatalf("rerun: %+v", sum)
	}
	all, _ := s.ListCustomers(ctx, true)
	if len(all) != 2 {
		t.Fatalf("expected 2 customers, got %d", len(all))
	}
	for _, c := range all {
		if c.QboCustomerID.String == "q1" && (c.Name != "Blue Plate Diner" || c.DeliveryAddress != "9 New St" || c.DeliveryDays != "mon,thu") {
			t.Fatalf("q1 not updated in place, or delivery days lost: %+v", c)
		}
	}
	if _, err := Customers(ctx, s, strings.NewReader("phone\n555\n")); err == nil {
		t.Fatal("missing name column must be an error")
	}
}

func TestProductsImport(t *testing.T) {
	s := newSvc(t)
	csv := "sku,name,category,sell_unit,catch_weight,approx_case_weight_lb,base_price,qbo_item_id\n" +
		"BRIS,Brisket,Beef,case,yes,60,5.99,i1\n" +
		"SAUS,Sausage,Pork,each,no,,4,i2\n" +
		"BAD,Bad,Beef,kg,no,,1,\n" +
		"CW,Catch no weight,Beef,case,true,,1,\n"
	sum, err := Products(ctx, s, strings.NewReader(csv))
	if err != nil {
		t.Fatal(err)
	}
	if sum.Created != 2 || sum.Rejected != 2 {
		t.Fatalf("products: %+v", sum)
	}
	p, err := s.Q.GetProductBySKU(ctx, "BRIS")
	if err != nil || !p.CatchWeight || p.ApproxCaseWeight.Int64 != 6000 || p.BasePriceCents != 599 || p.QboItemID.String != "i1" {
		t.Fatalf("brisket: %+v %v", p, err)
	}
	sum, _ = Products(ctx, s, strings.NewReader("sku,name,sell_unit,base_price\nBRIS,Whole Brisket,case,6.49\n"))
	if sum.Updated != 1 {
		t.Fatalf("rerun: %+v", sum)
	}
	p, _ = s.Q.GetProductBySKU(ctx, "BRIS")
	if p.Name != "Whole Brisket" || p.BasePriceCents != 649 || !p.CatchWeight || p.ApproxCaseWeight.Int64 != 6000 {
		t.Fatalf("partial update must keep unspecified fields: %+v", p)
	}
}

func TestPricesImport(t *testing.T) {
	s := newSvc(t)
	Customers(ctx, s, strings.NewReader("name,qbo_customer_id\nBlue Plate,q1\nRed Barn,\n"))
	Products(ctx, s, strings.NewReader("sku,name,sell_unit,base_price\nBRIS,Brisket,lb,5.99\n"))
	csv := "customer,sku,price\nq1,BRIS,5.49\nRed Barn,BRIS,5.79\nNobody,BRIS,1\nq1,NOPE,1\nq1,BRIS,abc\n"
	sum, err := Prices(ctx, s, strings.NewReader(csv), "2026-09-10")
	if err != nil {
		t.Fatal(err)
	}
	if sum.Created != 2 || sum.Rejected != 3 {
		t.Fatalf("prices: %+v", sum)
	}
	c, _ := s.Q.GetCustomerByQBO(ctx, sqlStr("q1"))
	rows, _ := s.ListCustomerPrices(ctx, c.ID, "2026-09-10")
	if len(rows) != 1 || rows[0].PriceCents != 549 {
		t.Fatalf("q1 price: %+v", rows)
	}
}
```

Add to the test file:
```go
import "database/sql"
func sqlStr(s string) sql.NullString { return sql.NullString{String: s, Valid: true} }
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/importer/ 2>&1 | head -3`
Expected: build failure.

- [ ] **Step 3: Implement importer.go**

```go
// Package importer loads customers, products and prices from CSV. It never deletes and is re-run safe.
package importer

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

type Summary struct {
	Created, Updated, Rejected int
	Errors                     []string
}

func (s Summary) String() string {
	out := fmt.Sprintf("created %d, updated %d, rejected %d", s.Created, s.Updated, s.Rejected)
	for _, e := range s.Errors {
		out += "\n  " + e
	}
	return out
}

func (s *Summary) reject(line int, msg string) {
	s.Rejected++
	s.Errors = append(s.Errors, fmt.Sprintf("line %d: %s", line, msg))
}

// table reads a CSV into rows keyed by lower-cased header.
type table struct {
	headers []string
	rows    []map[string]string
}

func readTable(r io.Reader, required ...string) (table, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	recs, err := cr.ReadAll()
	if err != nil {
		return table{}, err
	}
	if len(recs) == 0 {
		return table{}, errors.New("empty file")
	}
	t := table{}
	for _, h := range recs[0] {
		t.headers = append(t.headers, strings.ToLower(strings.TrimSpace(h)))
	}
	for _, req := range required {
		found := false
		for _, h := range t.headers {
			if h == req {
				found = true
			}
		}
		if !found {
			return table{}, fmt.Errorf("missing required column %q", req)
		}
	}
	for _, rec := range recs[1:] {
		row := map[string]string{}
		for i, h := range t.headers {
			if i < len(rec) {
				row[h] = strings.TrimSpace(rec[i])
			}
		}
		t.rows = append(t.rows, row)
	}
	return t, nil
}

func (t table) has(col string) bool {
	for _, h := range t.headers {
		if h == col {
			return true
		}
	}
	return false
}

// decimalToHundredths parses "5.99" → 599 (also used for weights: "60" → 6000).
func decimalToHundredths(s string) (int64, error) {
	f, err := strconv.ParseFloat(strings.TrimPrefix(s, "$"), 64)
	if err != nil || f < 0 || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("%q is not a non-negative decimal", s)
	}
	return int64(math.Round(f * 100)), nil
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "yes", "y", "1":
		return true, nil
	case "false", "no", "n", "0", "":
		return false, nil
	}
	return false, fmt.Errorf("%q is not a boolean", s)
}

func nullable(s string) sql.NullString { return sql.NullString{String: s, Valid: s != ""} }

func Customers(ctx context.Context, svc *service.Service, r io.Reader) (Summary, error) {
	t, err := readTable(r, "name")
	if err != nil {
		return Summary{}, err
	}
	var sum Summary
	for i, row := range t.rows {
		line := i + 2
		if row["name"] == "" {
			sum.reject(line, "name is required")
			continue
		}
		var existing *queries.Customer
		if q := row["qbo_customer_id"]; q != "" {
			if c, err := svc.Q.GetCustomerByQBO(ctx, nullable(q)); err == nil {
				existing = &c
			}
		}
		if existing == nil {
			if c, err := svc.Q.GetCustomerByName(ctx, row["name"]); err == nil {
				existing = &c
			}
		}
		in := service.CustomerInput{Active: true}
		if existing != nil {
			in = service.CustomerInput{
				Name: existing.Name, BillingAddress: existing.BillingAddress, DeliveryAddress: existing.DeliveryAddress, ContactName: existing.ContactName,
				Phone: existing.Phone, Email: existing.Email, DeliveryNotes: existing.DeliveryNotes, DeliveryDays: service.SplitDeliveryDays(existing.DeliveryDays),
				QBOCustomerID: existing.QboCustomerID.String, Active: existing.Active,
			}
		}
		in.Name = row["name"]
		set := func(col string, dst *string) {
			if t.has(col) {
				*dst = row[col]
			}
		}
		set("billing_address", &in.BillingAddress)
		set("delivery_address", &in.DeliveryAddress)
		set("contact_name", &in.ContactName)
		set("phone", &in.Phone)
		set("email", &in.Email)
		set("delivery_notes", &in.DeliveryNotes)
		set("qbo_customer_id", &in.QBOCustomerID)
		if t.has("delivery_days") && row["delivery_days"] != "" {
			in.DeliveryDays = strings.Split(strings.ReplaceAll(row["delivery_days"], ",", ";"), ";")
		}
		if existing == nil {
			_, err = svc.CreateCustomer(ctx, in)
		} else {
			_, err = svc.UpdateCustomer(ctx, existing.ID, in)
		}
		if err != nil {
			sum.reject(line, err.Error())
			continue
		}
		if existing == nil {
			sum.Created++
		} else {
			sum.Updated++
		}
	}
	return sum, nil
}

func Products(ctx context.Context, svc *service.Service, r io.Reader) (Summary, error) {
	t, err := readTable(r, "sku", "name")
	if err != nil {
		return Summary{}, err
	}
	var sum Summary
	for i, row := range t.rows {
		line := i + 2
		var existing *queries.Product
		if p, err := svc.Q.GetProductBySKU(ctx, row["sku"]); err == nil {
			existing = &p
		}
		in := service.ProductInput{SKU: row["sku"], Active: true}
		if existing != nil {
			in = service.ProductInput{SKU: existing.Sku, Name: existing.Name, Category: existing.Category, SellUnit: existing.SellUnit, CatchWeight: existing.CatchWeight,
				BasePriceCents: existing.BasePriceCents, QBOItemID: existing.QboItemID.String, Active: existing.Active}
			if existing.ApproxCaseWeight.Valid {
				w := existing.ApproxCaseWeight.Int64
				in.ApproxCaseWeight = &w
			}
		}
		in.Name = row["name"]
		if t.has("category") {
			in.Category = row["category"]
		}
		if t.has("sell_unit") {
			in.SellUnit = strings.ToLower(row["sell_unit"])
		}
		if t.has("qbo_item_id") {
			in.QBOItemID = row["qbo_item_id"]
		}
		var perr error
		if t.has("catch_weight") {
			in.CatchWeight, perr = parseBool(row["catch_weight"])
		}
		if perr == nil && t.has("approx_case_weight_lb") && row["approx_case_weight_lb"] != "" {
			var w int64
			w, perr = decimalToHundredths(row["approx_case_weight_lb"])
			in.ApproxCaseWeight = &w
		}
		if perr == nil && t.has("base_price") && row["base_price"] != "" {
			in.BasePriceCents, perr = decimalToHundredths(row["base_price"])
		}
		if perr != nil {
			sum.reject(line, perr.Error())
			continue
		}
		if existing == nil {
			_, err = svc.CreateProduct(ctx, in)
		} else {
			_, err = svc.UpdateProduct(ctx, existing.ID, in)
		}
		if err != nil {
			sum.reject(line, err.Error())
			continue
		}
		if existing == nil {
			sum.Created++
		} else {
			sum.Updated++
		}
	}
	return sum, nil
}

func Prices(ctx context.Context, svc *service.Service, r io.Reader, effectiveFrom string) (Summary, error) {
	if !service.ValidDate(effectiveFrom) {
		return Summary{}, fmt.Errorf("effective date %q must be YYYY-MM-DD", effectiveFrom)
	}
	t, err := readTable(r, "customer", "sku", "price")
	if err != nil {
		return Summary{}, err
	}
	var sum Summary
	for i, row := range t.rows {
		line := i + 2
		c, err := svc.Q.GetCustomerByQBO(ctx, nullable(row["customer"]))
		if err != nil {
			c, err = svc.Q.GetCustomerByName(ctx, row["customer"])
		}
		if err != nil {
			sum.reject(line, fmt.Sprintf("customer %q not found", row["customer"]))
			continue
		}
		p, err := svc.Q.GetProductBySKU(ctx, row["sku"])
		if err != nil {
			sum.reject(line, fmt.Sprintf("sku %q not found", row["sku"]))
			continue
		}
		cents, err := decimalToHundredths(row["price"])
		if err != nil {
			sum.reject(line, err.Error())
			continue
		}
		if _, err := svc.SetCustomerPrice(ctx, c.ID, service.PriceInput{ProductID: p.ID, PriceCents: cents, EffectiveFrom: effectiveFrom}); err != nil {
			sum.reject(line, err.Error())
			continue
		}
		sum.Created++
	}
	return sum, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/importer/ 2>&1 | tail -3`
Expected: `ok`

- [ ] **Step 5: Import CLI and format doc**

`cmd/deepcuts/cmd_import.go`:
```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cmcgeedev/deepcutsCRM/internal/config"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/importer"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

func runImport(cfg config.Config, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: deepcuts import customers|products|prices <file.csv> [--effective YYYY-MM-DD]")
	}
	ctx := context.Background()
	d, err := db.OpenAndMigrate(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer d.Close()
	svc := service.New(d, cfg.Location)
	f, err := os.Open(args[1])
	if err != nil {
		return err
	}
	defer f.Close()
	var sum importer.Summary
	switch args[0] {
	case "customers":
		sum, err = importer.Customers(ctx, svc, f)
	case "products":
		sum, err = importer.Products(ctx, svc, f)
	case "prices":
		eff := svc.Today()
		for i := 2; i+1 < len(args); i++ {
			if args[i] == "--effective" {
				eff = args[i+1]
			}
		}
		sum, err = importer.Prices(ctx, svc, f, eff)
	default:
		return fmt.Errorf("unknown import kind %q", args[0])
	}
	if err != nil {
		return err
	}
	fmt.Println(sum.String())
	return nil
}
```
Delete the `runImport = notImplemented("import")` line in `main.go`.

`docs/import-formats.md`:
```markdown
# CSV import formats

Headers are case-insensitive and may appear in any order. Extra columns are ignored.
Imports never delete rows and are safe to re-run: matched rows are updated in place and
columns absent from the file keep their current values.

## customers.csv

`name` (required), `billing_address`, `delivery_address`, `contact_name`, `phone`, `email`,
`delivery_notes`, `delivery_days` (`mon;thu`), `qbo_customer_id`.
Match order: `qbo_customer_id`, then exact `name`.

## products.csv

`sku` (required), `name` (required), `category`, `sell_unit` (`lb`, `case`, `each`),
`catch_weight` (`yes`/`no`), `approx_case_weight_lb` (decimal pounds), `base_price`
(decimal dollars), `qbo_item_id`. Match on `sku`. Catch-weight products must use `case`.

## prices.csv

`customer` (qbo id or exact name), `sku`, `price` (decimal dollars).
Every row creates a price effective from `--effective` (default today).

    deepcuts import customers customers.csv
    deepcuts import products products.csv
    deepcuts import prices prices.csv --effective 2026-10-01
```

- [ ] **Step 6: Verify and commit**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && printf 'name,qbo_customer_id\nBlue Plate,q1\n' > /tmp/c.csv && DEEPCUTS_DB_PATH=$(mktemp -d)/t.sqlite go run ./cmd/deepcuts import customers /tmp/c.csv && go test ./... 2>&1 | tail -3 && git add -A && git commit -q -m "Add CSV importers and import CLI" && git push`
Expected: `created 1, updated 0, rejected 0`, tests ok.

---

### Task 14: Demo seed, serve command, embedded web placeholder

**Files:**
- Create: `internal/seed/demo.go`, `internal/seed/demo_test.go`, `cmd/deepcuts/cmd_seed.go`, `cmd/deepcuts/cmd_serve.go`, `web/embed.go`, `web/dist/.gitkeep`
- Modify: `cmd/deepcuts/main.go` (remove `runSeed`, `runServe` stub lines)

**Interfaces:**
- Produces:
  - `seed.Demo(ctx, svc *service.Service, a *auth.Auth) (Info, error)`; `type Info struct { OfficeEmail, OfficePassword string; Drivers []DriverLogin; Customers, Products, Orders int }`; `type DriverLogin struct { Name, PIN string }`
  - Demo content: office `office@demo.local` / `demo1234`; drivers `Sam` PIN `111111`, `Riley` PIN `222222`; 12 customers with delivery notes and days; 40 products (20 catch-weight beef/pork primals with case weights, 20 unit-priced items); custom prices for 5 customers on 8 products; orders for the 7 days around today: yesterday has one route complete with delivered orders (one finalized, one needing review), today has one route out with 4 stops (one already delivered) and 2 unscheduled confirmed orders, tomorrow has 5 drafts/confirmed. Shipped weights filled on today's route.
  - `Demo` refuses if any customer already exists (`seed_not_empty`).
  - `web.Dist embed.FS` (package `web`, `//go:embed all:dist`), `func web.FS() (fs.FS, error)` returning `fs.Sub(Dist, "dist")`.
  - `deepcuts serve` listens on `cfg.Addr`, TLS when `cfg.TLS()`, logs the URL, graceful shutdown on SIGINT/SIGTERM.

- [ ] **Step 1: Write the failing seed test**

`internal/seed/demo_test.go`:
```go
package seed

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
	"github.com/cmcgeedev/deepcutsCRM/internal/storage"
)

func TestDemoSeed(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	d, err := db.OpenAndMigrate(ctx, filepath.Join(dir, "t.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	loc, _ := time.LoadLocation("America/New_York")
	svc := service.New(d, loc)
	svc.Proofs, _ = storage.NewLocal(filepath.Join(dir, "uploads"))
	a := auth.New(queries.New(d))
	info, err := Demo(ctx, svc, a)
	if err != nil {
		t.Fatal(err)
	}
	if info.Customers != 12 || info.Products != 40 || len(info.Drivers) != 2 || info.Orders < 12 {
		t.Fatalf("info: %+v", info)
	}
	if _, err := a.LoginOffice(ctx, info.OfficeEmail, info.OfficePassword, "t"); err != nil {
		t.Fatal("office login failed")
	}
	today := svc.Today()
	dv, err := svc.DayView(ctx, today)
	if err != nil || len(dv.Routes) != 1 || dv.Routes[0].Route.Status != "out" || len(dv.Routes[0].Stops) != 4 || len(dv.Unscheduled) != 2 {
		t.Fatalf("today: %v %+v", err, dv)
	}
	delivered := 0
	for _, st := range dv.Routes[0].Stops {
		if st.Stop.Status == "delivered" {
			delivered++
		}
	}
	if delivered != 1 {
		t.Fatalf("expected one delivered stop today, got %d", delivered)
	}
	drivers, _ := svc.ListDrivers(ctx)
	var sam int64
	for _, d := range drivers {
		if d.DisplayName == "Sam" {
			sam = d.ID
		}
	}
	dr, err := svc.DriverRouteForDate(ctx, sam, today)
	if err != nil || len(dr.Stops) != 4 {
		t.Fatalf("driver route: %v", err)
	}
	fin, _ := svc.ListOrders(ctx, service.OrderFilter{Status: "finalized"})
	rev, _ := svc.ListOrders(ctx, service.OrderFilter{Status: "delivered"})
	if len(fin) < 1 || len(rev) < 1 {
		t.Fatalf("yesterday: finalized=%d delivered=%d", len(fin), len(rev))
	}
	if _, err := Demo(ctx, svc, a); err == nil {
		t.Fatal("second seed must refuse")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/seed/ 2>&1 | head -3`
Expected: build failure.

- [ ] **Step 3: Implement seed/demo.go**

```go
// Package seed loads a realistic demo data set through the service layer so every state machine rule holds.
package seed

import (
	"context"
	"fmt"

	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/domain"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
)

type DriverLogin struct{ Name, PIN string }

type Info struct {
	OfficeEmail, OfficePassword string
	Drivers                     []DriverLogin
	Customers, Products, Orders int
}

type product struct {
	sku, name, category, unit string
	catch                     bool
	caseLb                    int64 // whole pounds
	price                     int64 // cents per unit (per lb when catch)
}

var products = []product{
	{"BF-BRIS", "Brisket, packer", "Beef", "case", true, 60, 599}, {"BF-RIB", "Ribeye, boneless", "Beef", "case", true, 45, 1499},
	{"BF-STRIP", "Strip loin", "Beef", "case", true, 50, 1199}, {"BF-TEND", "Tenderloin PSMO", "Beef", "case", true, 30, 1899},
	{"BF-CHUCK", "Chuck roll", "Beef", "case", true, 70, 449}, {"BF-RND", "Top round", "Beef", "case", true, 60, 479},
	{"BF-SHRT", "Short ribs", "Beef", "case", true, 40, 899}, {"BF-FLNK", "Flank steak", "Beef", "case", true, 25, 999},
	{"BF-SKRT", "Skirt steak, outside", "Beef", "case", true, 25, 1099}, {"BF-TRI", "Tri-tip", "Beef", "case", true, 30, 799},
	{"PK-BLLY", "Pork belly, skinless", "Pork", "case", true, 55, 389}, {"PK-SHLD", "Pork shoulder, bone-in", "Pork", "case", true, 65, 219},
	{"PK-LOIN", "Pork loin, boneless", "Pork", "case", true, 50, 279}, {"PK-RIBS", "St. Louis ribs", "Pork", "case", true, 30, 349},
	{"PK-BBR", "Baby back ribs", "Pork", "case", true, 30, 429}, {"PK-TEND", "Pork tenderloin", "Pork", "case", true, 20, 399},
	{"LM-LEG", "Lamb leg, bone-in", "Lamb", "case", true, 40, 749}, {"LM-RACK", "Lamb rack, frenched", "Lamb", "case", true, 20, 1999},
	{"CH-WHOLE", "Whole chicken, WOG", "Poultry", "case", true, 40, 189}, {"CH-THIGH", "Chicken thighs, boneless", "Poultry", "case", true, 40, 259},
	{"GB-80", "Ground beef 80/20, 10 lb tube", "Ground", "each", false, 0, 3490}, {"GB-90", "Ground beef 90/10, 10 lb tube", "Ground", "each", false, 0, 4290},
	{"GP-PORK", "Ground pork, 10 lb tube", "Ground", "each", false, 0, 2490}, {"SS-BRAT", "Bratwurst, 5 lb", "Sausage", "each", false, 0, 2250},
	{"SS-ITAL", "Italian sausage, 5 lb", "Sausage", "each", false, 0, 2250}, {"SS-CHOR", "Chorizo, 5 lb", "Sausage", "each", false, 0, 2450},
	{"SS-BKFT", "Breakfast links, 5 lb", "Sausage", "each", false, 0, 2150}, {"BC-SLAB", "Bacon, sliced, 15 lb", "Cured", "each", false, 0, 6750},
	{"BC-THK", "Bacon, thick cut, 15 lb", "Cured", "each", false, 0, 6950}, {"HM-BONE", "Ham, bone-in, each", "Cured", "each", false, 0, 3900},
	{"CH-WING", "Chicken wings, 40 lb", "Poultry", "each", false, 0, 9800}, {"CH-BRST", "Chicken breast, 40 lb", "Poultry", "each", false, 0, 11200},
	{"TK-BRST", "Turkey breast, each", "Poultry", "each", false, 0, 2800}, {"BF-PATTY", "Beef patties 6 oz, 40 ct", "Ground", "each", false, 0, 7200},
	{"BF-STEW", "Stew meat, 10 lb", "Beef", "each", false, 0, 5900}, {"BF-BONE", "Beef bones, 30 lb", "Beef", "each", false, 0, 3000},
	{"PK-FAT", "Pork fatback, 10 lb", "Pork", "each", false, 0, 1500}, {"DL-ROAST", "Deli roast beef, 8 lb", "Deli", "each", false, 0, 6400},
	{"DL-TURK", "Deli turkey, 8 lb", "Deli", "each", false, 0, 5600}, {"DL-HAM", "Deli ham, 8 lb", "Deli", "each", false, 0, 4800},
}

type customer struct{ name, addr, contact, phone, notes, days string }

var customers = []customer{
	{"Blue Plate Diner", "112 Main St, Ferndale", "Marcy", "555-0101", "Back door by the dumpsters, ring twice", "mon,thu"},
	{"The Butcher's Table", "9 Depot Rd, Ferndale", "Luis", "555-0102", "Walk-in cooler is left of the loading dock", "mon,wed,fri"},
	{"Red Barn Market", "4400 County Rd 12", "Dana", "555-0103", "Gate code 4471. Dock hours 6-10am", "tue,fri"},
	{"Hilltop Steakhouse", "1 Summit Ave", "Chef Ana", "555-0104", "Deliveries before 11am only", "mon,thu"},
	{"Corner Tavern", "88 Front St", "Mike", "555-0105", "Side alley, knock hard", "wed"},
	{"Greenfield Grocery", "210 Elm St, Greenfield", "Priya", "555-0106", "Receiving at rear, ask for Priya", "mon,wed,fri"},
	{"Smokehouse BBQ", "7 Pit Rd", "Earl", "555-0107", "Leave in the walk-in if nobody's there", "tue,thu"},
	{"Riverside Cafe", "300 River Rd", "Jo", "555-0108", "", "thu"},
	{"Lakeview Country Club", "1 Fairway Dr", "Kitchen", "555-0109", "Use the service road, security will wave you through", "wed,sat"},
	{"Mama Rosa's", "56 Union St", "Rosa", "555-0110", "Ring the bell at the kitchen door", "mon,thu"},
	{"Pinecrest School District", "1000 School Ln", "Cafeteria Mgr", "555-0111", "Dock hours 5:30-8am. No deliveries on holidays", "tue"},
	{"Two Rivers Brewing", "12 Brewery Way", "Sam K", "555-0112", "Kitchen is upstairs; use the freight elevator", "fri"},
}

func Demo(ctx context.Context, svc *service.Service, a *auth.Auth) (Info, error) {
	if existing, err := svc.ListCustomers(ctx, true); err != nil {
		return Info{}, err
	} else if len(existing) > 0 {
		return Info{}, service.Conflict("seed_not_empty", "database already has customers; seed only runs on an empty database")
	}
	info := Info{OfficeEmail: "office@demo.local", OfficePassword: "demo1234", Drivers: []DriverLogin{{"Sam", "111111"}, {"Riley", "222222"}}}
	if _, err := a.CreateOfficeUser(ctx, info.OfficeEmail, "Demo Office", info.OfficePassword); err != nil {
		return info, err
	}
	var driverIDs []int64
	for _, d := range info.Drivers {
		u, err := a.CreateDriver(ctx, d.Name, d.PIN)
		if err != nil {
			return info, err
		}
		driverIDs = append(driverIDs, u.ID)
	}
	var custIDs []int64
	for _, c := range customers {
		row, err := svc.CreateCustomer(ctx, service.CustomerInput{Name: c.name, BillingAddress: c.addr, DeliveryAddress: c.addr, ContactName: c.contact,
			Phone: c.phone, DeliveryNotes: c.notes, DeliveryDays: service.SplitDeliveryDays(c.days), Active: true})
		if err != nil {
			return info, err
		}
		custIDs = append(custIDs, row.ID)
	}
	info.Customers = len(custIDs)
	var prodIDs []int64
	for _, p := range products {
		in := service.ProductInput{SKU: p.sku, Name: p.name, Category: p.category, SellUnit: p.unit, CatchWeight: p.catch, BasePriceCents: p.price, Active: true}
		if p.catch {
			w := p.caseLb * 100
			in.ApproxCaseWeight = &w
		}
		row, err := svc.CreateProduct(ctx, in)
		if err != nil {
			return info, err
		}
		prodIDs = append(prodIDs, row.ID)
	}
	info.Products = len(prodIDs)
	// negotiated prices: first five customers get ~8% off eight products
	for ci := 0; ci < 5; ci++ {
		for pi := 0; pi < 8; pi++ {
			p := products[pi*3%len(products)]
			if _, err := svc.SetCustomerPrice(ctx, custIDs[ci], service.PriceInput{ProductID: prodIDs[pi*3%len(prodIDs)], PriceCents: p.price * 92 / 100, EffectiveFrom: "2026-01-01"}); err != nil {
				return info, err
			}
		}
	}
	today := svc.Now().In(svc.Loc)
	day := func(offset int) string { return today.AddDate(0, 0, offset).Format("2006-01-02") }

	// helper: confirmed order with n lines, deterministic product choice
	mk := func(ci int, date string, n int, confirm bool) (int64, error) {
		o, err := svc.CreateOrder(ctx, service.OrderInput{CustomerID: custIDs[ci], RequestedDeliveryDate: date})
		if err != nil {
			return 0, err
		}
		for k := 0; k < n; k++ {
			pi := (ci*7 + k*5) % len(prodIDs)
			qty := int64(100 * (1 + (ci+k)%3))
			if _, err := svc.AddLine(ctx, o.Order.ID, prodIDs[pi], qty); err != nil {
				return 0, err
			}
		}
		if confirm {
			if _, err := svc.ConfirmOrder(ctx, o.Order.ID); err != nil {
				return 0, err
			}
		}
		info.Orders++
		return o.Order.ID, nil
	}
	// fill shipped weights on catch-weight lines: approx weight ± a little
	ship := func(orderID int64) error {
		od, err := svc.GetOrder(ctx, orderID)
		if err != nil {
			return err
		}
		for i, l := range od.Lines {
			if !l.Product.CatchWeight {
				continue
			}
			w := l.Line.EstWeight.Int64 * int64(97+i*2) / 100
			if _, err := svc.UpdateLine(ctx, orderID, l.Line.ID, service.LinePatch{ShippedWeight: &w}); err != nil {
				return err
			}
		}
		return nil
	}
	deliver := func(driverID, stopID int64, clientID string, adjust bool) error {
		st, err := svc.Q.GetStop(ctx, stopID)
		if err != nil {
			return err
		}
		od, _ := svc.GetOrder(ctx, st.OrderID)
		act := domain.DriverAction{ClientID: clientID, Type: domain.ActionDeliver, Proof: &domain.Proof{Type: domain.ProofName, Name: "Received by " + od.Customer.ContactName}}
		if adjust && len(od.Lines) > 0 {
			short := domain.Hundredths(0)
			note := "customer refused, temperature"
			act.Lines = []domain.LineAdjustment{{LineID: od.Lines[0].Line.ID, DeliveredQty: &short, ShortageNote: &note}}
			if od.Lines[0].Product.CatchWeight {
				act.Lines[0].DeliveredWeight = &short
			}
		}
		_, _, err = svc.ApplyDriverAction(ctx, driverID, stopID, act)
		return err
	}

	// yesterday: complete route, one finalized order, one needing review
	y := day(-1)
	r, err := svc.CreateRoute(ctx, service.RouteInput{RouteDate: y, DriverUserID: driverIDs[0], TruckLabel: "Reefer 1"})
	if err != nil {
		return info, err
	}
	var yStops []int64
	for ci := 0; ci < 3; ci++ {
		oid, err := mk(ci, y, 3, true)
		if err != nil {
			return info, err
		}
		if err := ship(oid); err != nil {
			return info, err
		}
		if r, err = svc.AddStop(ctx, r.Route.ID, oid); err != nil {
			return info, err
		}
		yStops = append(yStops, r.Stops[len(r.Stops)-1].Stop.ID)
	}
	if _, err := svc.RouteOut(ctx, r.Route.ID); err != nil {
		return info, err
	}
	for i, sid := range yStops {
		if err := deliver(driverIDs[0], sid, fmt.Sprintf("00000000-0000-4000-8000-0000000000%02d", i), i == 1); err != nil {
			return info, err
		}
	}
	if _, err := svc.RouteComplete(ctx, r.Route.ID); err != nil {
		return info, err
	}
	first, _ := svc.Q.GetStop(ctx, yStops[0])
	if _, err := svc.FinalizeOrder(ctx, first.OrderID); err != nil {
		return info, err
	}

	// today: route out with 4 stops (one delivered), plus 2 unscheduled confirmed orders
	t := day(0)
	r, err = svc.CreateRoute(ctx, service.RouteInput{RouteDate: t, DriverUserID: driverIDs[0], TruckLabel: "Reefer 1"})
	if err != nil {
		return info, err
	}
	var tStops []int64
	for ci := 3; ci < 7; ci++ {
		oid, err := mk(ci, t, 4, true)
		if err != nil {
			return info, err
		}
		if err := ship(oid); err != nil {
			return info, err
		}
		if r, err = svc.AddStop(ctx, r.Route.ID, oid); err != nil {
			return info, err
		}
		tStops = append(tStops, r.Stops[len(r.Stops)-1].Stop.ID)
	}
	if _, err := svc.RouteOut(ctx, r.Route.ID); err != nil {
		return info, err
	}
	if err := deliver(driverIDs[0], tStops[0], "00000000-0000-4000-8000-0000000000aa", false); err != nil {
		return info, err
	}
	for ci := 7; ci < 9; ci++ {
		if _, err := mk(ci, t, 2, true); err != nil {
			return info, err
		}
	}
	// tomorrow and later: drafts and confirmed
	for ci := 0; ci < 12; ci++ {
		if _, err := mk(ci, day(1+ci%3), 2+ci%3, ci%2 == 0); err != nil {
			return info, err
		}
	}
	return info, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go test ./internal/seed/ 2>&1 | tail -3`
Expected: `ok`. If a step fails with `invalid_transition` or `locked`, the seed is violating a rule the services enforce; fix the seed order of operations, never the service.

- [ ] **Step 5: web embed, seed and serve commands**

`web/embed.go`:
```go
// Package web embeds the built React app. Run `make build-web` before building the binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var Dist embed.FS

func FS() (fs.FS, error) { return fs.Sub(Dist, "dist") }
```
Create the empty file `web/dist/.gitkeep` (the `.gitignore` from Task 1 keeps it).

`cmd/deepcuts/cmd_seed.go`:
```go
package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/config"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/seed"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
	"github.com/cmcgeedev/deepcutsCRM/internal/storage"
)

func runSeed(cfg config.Config, args []string) error {
	if len(args) != 1 || args[0] != "demo" {
		return fmt.Errorf("usage: deepcuts seed demo")
	}
	ctx := context.Background()
	d, err := db.OpenAndMigrate(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer d.Close()
	svc := service.New(d, cfg.Location)
	proofs, err := storage.NewLocal(filepath.Join(cfg.DataDir, "uploads"))
	if err != nil {
		return err
	}
	svc.Proofs = proofs
	info, err := seed.Demo(ctx, svc, auth.New(queries.New(d)))
	if err != nil {
		return err
	}
	fmt.Printf("seeded %d customers, %d products, %d orders into %s\n", info.Customers, info.Products, info.Orders, cfg.DBPath)
	fmt.Printf("office login:  %s / %s\n", info.OfficeEmail, info.OfficePassword)
	for _, dl := range info.Drivers {
		fmt.Printf("driver login:  %s PIN %s\n", dl.Name, dl.PIN)
	}
	return nil
}
```

`cmd/deepcuts/cmd_serve.go`:
```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cmcgeedev/deepcutsCRM/internal/auth"
	"github.com/cmcgeedev/deepcutsCRM/internal/config"
	"github.com/cmcgeedev/deepcutsCRM/internal/db"
	"github.com/cmcgeedev/deepcutsCRM/internal/db/queries"
	"github.com/cmcgeedev/deepcutsCRM/internal/httpapi"
	"github.com/cmcgeedev/deepcutsCRM/internal/service"
	"github.com/cmcgeedev/deepcutsCRM/internal/storage"
	"github.com/cmcgeedev/deepcutsCRM/web"
)

func runServe(cfg config.Config, args []string) error {
	ctx := context.Background()
	d, err := db.OpenAndMigrate(ctx, cfg.DBPath)
	if err != nil {
		return err
	}
	defer d.Close()
	svc := service.New(d, cfg.Location)
	proofs, err := storage.NewLocal(filepath.Join(cfg.DataDir, "uploads"))
	if err != nil {
		return err
	}
	svc.Proofs = proofs
	a := auth.New(queries.New(d))
	webFS, err := web.FS()
	if err != nil {
		return err
	}
	handler := httpapi.NewRouter(httpapi.Deps{Svc: svc, Auth: a, Proofs: proofs, Secure: cfg.TLS(), Web: webFS})
	srv := &http.Server{Addr: cfg.Addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		t := time.NewTicker(time.Hour)
		for range t.C {
			_ = queries.New(d).DeleteExpiredSessions(ctx, time.Now().UTC())
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		scheme := "http"
		if cfg.TLS() {
			scheme = "https"
		}
		log.Printf("deepcuts listening on %s://localhost%s (office: /office, driver: /driver)", scheme, cfg.Addr)
		if cfg.TLS() {
			errCh <- srv.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
		} else {
			errCh <- srv.ListenAndServe()
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-stop:
		fmt.Println("shutting down")
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
```

Delete the `runSeed` and `runServe` stub lines (and the now-empty `var (...)` block and `notImplemented` helper if nothing uses them) in `main.go`.

- [ ] **Step 6: Verify seed + serve end to end and commit**

Run:
```bash
export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && go vet ./... && go build -o bin/deepcuts ./cmd/deepcuts && \
T=$(mktemp -d) && DEEPCUTS_DB_PATH=$T/demo.sqlite DEEPCUTS_DATA_DIR=$T ./bin/deepcuts seed demo && \
(DEEPCUTS_DB_PATH=$T/demo.sqlite DEEPCUTS_DATA_DIR=$T DEEPCUTS_ADDR=:8099 ./bin/deepcuts serve & echo $! > $T/pid; sleep 1; \
 curl -s -c $T/jar -X POST localhost:8099/api/office/login -H 'Content-Type: application/json' -d '{"email":"office@demo.local","password":"demo1234"}'; echo; \
 curl -s -b $T/jar "localhost:8099/api/office/day?date=$(date +%F)" | head -c 300; echo; kill $(cat $T/pid))
```
Expected: seed prints logins; login returns the user JSON; day view returns JSON with one route.

```bash
cd ~/deepcutsCRM && go test ./... 2>&1 | tail -3 && git add -A && git commit -q -m "Add demo seed, serve command, embedded web placeholder" && git push
```

---

### Task 15: Web scaffold: Vite, React, generated client, auth, router, layouts

**Files:**
- Create: `web/` via Vite template, then `web/vite.config.ts`, `web/src/main.tsx`, `web/src/router.tsx`, `web/src/styles.css`, `web/src/api/client.ts`, `web/src/api/schema.d.ts` (generated), `web/src/lib/format.ts`, `web/src/lib/format.test.ts`, `web/src/auth/Session.tsx`, `web/src/routes/office/Layout.tsx`, `web/src/routes/office/Login.tsx`, `web/src/routes/driver/Layout.tsx`, `web/src/test/setup.ts`
- Modify: `Makefile` (none), `.gitignore` (none)

**Interfaces:**
- Produces:
  - `api` (openapi-fetch client), `type ApiError = { code: string; message: string; fields?: Record<string, string> }`, `errorOf(res: { error?: unknown; response: Response }): ApiError | null`, `type Schemas = components["schemas"]`.
  - `format.ts`: `money(cents: number): string` (`$5.99`), `hundredths(v: number): string` (`2.5`), `lb(v: number | undefined): string` (`118.50 lb` or `—`), `qty(v: number, unit: string): string` (`2 cases`, `10 each`, `2.50 lb`), `toHundredths(s: string): number | null` (`"2.5"` → 250), `toCents(s: string): number | null`.
  - `Session.tsx`: `<SessionProvider realm="office" | "driver">` fetching `/api/<realm>/me` on mount; `useSession()` → `{ user, loading, setUser, logout }`; `<RequireSession realm>` redirecting to `/<realm>/login`.
  - Router paths: `/` → `/office`; `/office/login`; `/office` layout with `customers`, `customers/:id`, `products`, `products/:id`, `orders`, `orders/:id`, `day`; `/driver/login`; `/driver` layout with `route`, `stops/:id`. Unknown office paths → `/office/day`; unknown driver paths → `/driver/route`.

- [ ] **Step 1: Scaffold with Vite and install dependencies**

```bash
export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM && \
npm create vite@latest web -- --template react-ts && cd web && \
npm install react-router openapi-fetch idb signature_pad && \
npm install -D openapi-typescript vite-plugin-pwa vitest jsdom @testing-library/react @testing-library/jest-dom @testing-library/user-event fake-indexeddb && \
rm -f src/App.css src/App.tsx src/assets/react.svg public/vite.svg src/index.css
```
If `npm create vite` prompts, answer: framework React, variant TypeScript. Do not choose the SWC or compiler variants.

Set scripts in `web/package.json`:
```json
"scripts": {
  "dev": "vite",
  "build": "tsc -b && vite build",
  "preview": "vite preview",
  "generate": "openapi-typescript ../api/openapi.yaml -o src/api/schema.d.ts",
  "test": "vitest",
  "lint": "eslint ."
}
```

- [ ] **Step 2: Vite config, test setup, generated client**

`web/vite.config.ts`:
```ts
/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { VitePWA } from "vite-plugin-pwa";

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: ["favicon.svg"],
      manifest: {
        name: "Deep Cuts Driver",
        short_name: "Deep Cuts",
        start_url: "/driver/route",
        scope: "/",
        display: "standalone",
        background_color: "#111111",
        theme_color: "#111111",
        icons: [{ src: "/icon.svg", sizes: "any", type: "image/svg+xml" }],
      },
      workbox: {
        navigateFallback: "/index.html",
        navigateFallbackDenylist: [/^\/api\//],
        globPatterns: ["**/*.{js,css,html,svg}"],
      },
    }),
  ],
  server: { proxy: { "/api": "http://localhost:8080" } },
  build: { outDir: "dist", emptyOutDir: true },
  test: { environment: "jsdom", setupFiles: ["src/test/setup.ts"], globals: true },
});
```

`web/public/icon.svg`: a 512×512 SVG with a dark square and a white "DC" (any simple SVG; it only needs to exist).
`web/public/favicon.svg`: same file.

`web/src/test/setup.ts`:
```ts
import "@testing-library/jest-dom/vitest";
import "fake-indexeddb/auto";
```

Generate the client types: `cd ~/deepcutsCRM/web && npm run generate` → `src/api/schema.d.ts` (commit it).

`web/src/api/client.ts`:
```ts
import createClient from "openapi-fetch";
import type { components, paths } from "./schema";

export const api = createClient<paths>({ baseUrl: "/", credentials: "include" });

export type Schemas = components["schemas"];
export type ApiError = { code: string; message: string; fields?: Record<string, string> };

/** Normalizes an openapi-fetch result into an ApiError, or null when the call succeeded. */
export function errorOf(res: { error?: unknown; response: Response }): ApiError | null {
  if (res.response.ok) return null;
  const e = res.error as Partial<ApiError> | undefined;
  return { code: e?.code ?? "http_" + res.response.status, message: e?.message ?? res.response.statusText, fields: e?.fields };
}
```

- [ ] **Step 3: Write the failing format test**

`web/src/lib/format.test.ts`:
```ts
import { describe, expect, it } from "vitest";
import { hundredths, lb, money, qty, toCents, toHundredths } from "./format";

describe("format", () => {
  it("money", () => {
    expect(money(599)).toBe("$5.99");
    expect(money(0)).toBe("$0.00");
    expect(money(123456)).toBe("$1,234.56");
  });
  it("hundredths and lb", () => {
    expect(hundredths(250)).toBe("2.5");
    expect(hundredths(300)).toBe("3");
    expect(lb(11850)).toBe("118.50 lb");
    expect(lb(undefined)).toBe("—");
  });
  it("qty by unit", () => {
    expect(qty(200, "case")).toBe("2 cases");
    expect(qty(100, "case")).toBe("1 case");
    expect(qty(1000, "each")).toBe("10 each");
    expect(qty(250, "lb")).toBe("2.50 lb");
  });
  it("parsing", () => {
    expect(toHundredths("2.5")).toBe(250);
    expect(toHundredths("2")).toBe(200);
    expect(toHundredths("abc")).toBeNull();
    expect(toHundredths("-1")).toBeNull();
    expect(toCents("5.99")).toBe(599);
    expect(toCents("$5.99")).toBe(599);
  });
});
```

Run: `cd ~/deepcutsCRM/web && npx vitest run src/lib 2>&1 | tail -5` → fails (module missing).

- [ ] **Step 4: Implement format.ts**

```ts
const usd = new Intl.NumberFormat("en-US", { style: "currency", currency: "USD" });

export function money(cents: number): string {
  return usd.format(cents / 100);
}

export function hundredths(v: number): string {
  return (v / 100).toString();
}

export function lb(v: number | undefined | null): string {
  return v == null ? "—" : (v / 100).toFixed(2) + " lb";
}

export function qty(v: number, unit: string): string {
  if (unit === "lb") return (v / 100).toFixed(2) + " lb";
  const n = v / 100;
  if (unit === "case") return `${n} ${n === 1 ? "case" : "cases"}`;
  return `${n} ${unit}`;
}

function parseDecimal(s: string): number | null {
  const cleaned = s.replace(/[$,\s]/g, "");
  if (!/^\d+(\.\d{0,2})?$/.test(cleaned)) return null;
  return Math.round(parseFloat(cleaned) * 100);
}

export const toHundredths = parseDecimal;
export const toCents = parseDecimal;
```

Run: `cd ~/deepcutsCRM/web && npx vitest run src/lib 2>&1 | tail -3` → passes.

- [ ] **Step 5: Session provider, layouts, login, router, styles, main**

`web/src/auth/Session.tsx`:
```tsx
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
```

`web/src/routes/office/Login.tsx`:
```tsx
import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router";
import { api, errorOf } from "../../api/client";
import { useSession } from "../../auth/Session";

export default function OfficeLogin() {
  const { setUser } = useSession();
  const nav = useNavigate();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    const res = await api.POST("/api/office/login", { body: { email, password } });
    setBusy(false);
    const err = errorOf(res);
    if (err) return setError(err.message);
    setUser(res.data!);
    nav("/office/day", { replace: true });
  }

  return (
    <main className="narrow">
      <h1>Deep Cuts — Office</h1>
      <form onSubmit={submit} className="stack">
        <label>Email<input type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="username" required /></label>
        <label>Password<input type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" required /></label>
        {error && <p className="error" role="alert">{error}</p>}
        <button disabled={busy}>Sign in</button>
      </form>
    </main>
  );
}
```

`web/src/routes/office/Layout.tsx`:
```tsx
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
```

`web/src/routes/driver/Layout.tsx`:
```tsx
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
```

`web/src/router.tsx` (placeholders for screens built in later tasks are plain components that render a heading; each later task replaces its import):
```tsx
import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import { RequireSession, SessionProvider } from "./auth/Session";
import OfficeLayout from "./routes/office/Layout";
import OfficeLogin from "./routes/office/Login";
import DriverLayout from "./routes/driver/Layout";

const Todo = ({ name }: { name: string }) => <h1>{name}</h1>;

function Office() {
  return (
    <SessionProvider realm="office">
      <Routes>
        <Route path="login" element={<OfficeLogin />} />
        <Route element={<RequireSession><OfficeLayout /></RequireSession>}>
          <Route index element={<Navigate to="day" replace />} />
          <Route path="day" element={<Todo name="Day" />} />
          <Route path="orders" element={<Todo name="Orders" />} />
          <Route path="orders/:id" element={<Todo name="Order" />} />
          <Route path="customers" element={<Todo name="Customers" />} />
          <Route path="customers/:id" element={<Todo name="Customer" />} />
          <Route path="products" element={<Todo name="Products" />} />
          <Route path="products/:id" element={<Todo name="Product" />} />
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
        <Route path="login" element={<Todo name="Driver login" />} />
        <Route element={<RequireSession><DriverLayout /></RequireSession>}>
          <Route index element={<Navigate to="route" replace />} />
          <Route path="route" element={<Todo name="Route" />} />
          <Route path="stops/:id" element={<Todo name="Stop" />} />
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
```

`web/src/main.tsx`:
```tsx
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import AppRouter from "./router";
import "./styles.css";

createRoot(document.getElementById("root")!).render(<StrictMode><AppRouter /></StrictMode>);
```

`web/src/styles.css` (complete, small, mobile-first; the driver realm gets large touch targets):
```css
:root { --bg: #fafafa; --fg: #161616; --muted: #6b6b6b; --line: #e2e2e2; --accent: #b3261e; --ok: #1b7f3b; --warn: #b26a00; font-family: system-ui, sans-serif; }
* { box-sizing: border-box; }
body { margin: 0; background: var(--bg); color: var(--fg); }
a { color: var(--accent); }
h1 { font-size: 1.4rem; margin: 0 0 1rem; }
h2 { font-size: 1.1rem; margin: 1.5rem 0 .5rem; }
.topbar { display: flex; gap: 1rem; align-items: center; padding: .6rem 1rem; background: #fff; border-bottom: 1px solid var(--line); position: sticky; top: 0; }
.topbar nav { display: flex; gap: .8rem; }
.topbar nav a { text-decoration: none; color: var(--fg); padding: .2rem .4rem; border-radius: 4px; }
.topbar nav a.active { background: var(--fg); color: #fff; }
.spacer { flex: 1; }
.content { padding: 1rem; max-width: 1100px; margin: 0 auto; }
.narrow { max-width: 420px; margin: 3rem auto; padding: 1rem; }
.stack { display: flex; flex-direction: column; gap: .8rem; }
.row { display: flex; gap: .6rem; align-items: center; flex-wrap: wrap; }
label { display: flex; flex-direction: column; gap: .25rem; font-size: .9rem; }
input, select, textarea { font: inherit; padding: .5rem; border: 1px solid var(--line); border-radius: 6px; background: #fff; }
input.short { width: 7rem; }
button { font: inherit; padding: .5rem .9rem; border: 1px solid var(--fg); background: var(--fg); color: #fff; border-radius: 6px; cursor: pointer; }
button.secondary { background: #fff; color: var(--fg); }
button.danger { background: var(--accent); border-color: var(--accent); }
button.link { background: none; border: none; color: var(--accent); padding: 0; }
button:disabled { opacity: .5; cursor: default; }
table { width: 100%; border-collapse: collapse; background: #fff; }
th, td { text-align: left; padding: .5rem; border-bottom: 1px solid var(--line); vertical-align: top; }
th { font-size: .8rem; color: var(--muted); text-transform: uppercase; }
td.num, th.num { text-align: right; font-variant-numeric: tabular-nums; }
.muted { color: var(--muted); }
.error { color: var(--accent); }
.badge { display: inline-block; padding: .1rem .5rem; border-radius: 999px; font-size: .75rem; background: #eee; }
.badge.ok { background: #dff3e4; color: var(--ok); }
.badge.warn { background: #fff1d6; color: var(--warn); }
.badge.bad { background: #fde2e0; color: var(--accent); }
.badge.pending { background: #e5e9ff; color: #2f3fa3; }
.card { background: #fff; border: 1px solid var(--line); border-radius: 8px; padding: .8rem; }
.cols { display: grid; grid-template-columns: 1fr 1fr; gap: 1rem; }
@media (max-width: 800px) { .cols { grid-template-columns: 1fr; } }
/* driver */
.driver .content { max-width: 600px; padding: .8rem; }
.driver button { min-height: 48px; font-size: 1.05rem; }
.driver .stop { display: block; text-decoration: none; color: inherit; margin-bottom: .6rem; }
.driver .stop h3 { margin: 0 0 .2rem; }
.driver .actions { display: grid; gap: .6rem; margin-top: 1rem; }
.driver canvas.sig { width: 100%; height: 200px; border: 1px dashed var(--muted); background: #fff; touch-action: none; }
```

Update `web/index.html` title to `Deep Cuts` and make sure it has `<meta name="viewport" content="width=device-width, initial-scale=1" />`, `<link rel="icon" href="/favicon.svg" />`, and `<div id="root"></div>` with `<script type="module" src="/src/main.tsx"></script>`.

- [ ] **Step 6: Verify build, tests, and a real login in the browser; commit**

Run: `export PATH=/opt/homebrew/bin:$PATH && cd ~/deepcutsCRM/web && npm run build 2>&1 | tail -5 && npx vitest run 2>&1 | tail -3`
Expected: `dist/` produced with `sw.js` and `manifest.webmanifest`; tests pass.

Then: `cd ~/deepcutsCRM && make demo` in the background, open `http://localhost:8080/office/login`, sign in with the printed demo login, confirm the layout renders with "Day" heading; stop the server.

```bash
cd ~/deepcutsCRM && git add -A && git commit -q -m "Scaffold web app: Vite, generated client, sessions, router, layouts" && git push
```

---

### Task 16: Office screens: customers, customer detail with prices, products

**Files:**
- Create: `web/src/routes/office/Customers.tsx`, `web/src/routes/office/CustomerDetail.tsx`, `web/src/routes/office/Products.tsx`, `web/src/routes/office/ProductDetail.tsx`, `web/src/components/Field.tsx`, `web/src/components/useApi.ts`, `web/src/routes/office/Customers.test.tsx`
- Modify: `web/src/router.tsx` (replace the four `Todo` placeholders)

**Interfaces:**
- Produces:
  - `useApi<T>(fn: () => Promise<{ data?: T; error?: unknown; response: Response }>, deps: unknown[])` → `{ data, error, loading, reload }`.
  - `<Field label error name>` wrapping an input with the field error from `ApiError.fields`.
  - `CustomerForm` and `ProductForm` components (inside their detail files) taking `initial`, `onSubmit(values)`, `fields` error map, `busy`.

- [ ] **Step 1: Write the failing list test**

`web/src/routes/office/Customers.test.tsx`:
```tsx
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import Customers from "./Customers";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("Customers", () => {
  afterEach(() => vi.restoreAllMocks());
  it("lists customers from the API", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(jsonResponse([
      { id: 1, name: "Blue Plate", billingAddress: "", deliveryAddress: "1 Main St", contactName: "", phone: "", email: "", deliveryNotes: "", deliveryDays: ["mon"], active: true },
    ]));
    render(<MemoryRouter><Customers /></MemoryRouter>);
    expect(await screen.findByText("Blue Plate")).toBeInTheDocument();
    expect(screen.getByText("1 Main St")).toBeInTheDocument();
  });
  it("shows a field error from a 422", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch");
    fetchMock.mockResolvedValueOnce(jsonResponse([]));
    fetchMock.mockResolvedValueOnce(jsonResponse({ code: "invalid", message: "validation failed", fields: { name: "required" } }, 422));
    render(<MemoryRouter><Customers /></MemoryRouter>);
    await screen.findByText(/no customers/i);
    (await screen.findByRole("button", { name: /add customer/i })).click();
    expect(await screen.findByText("required")).toBeInTheDocument();
  });
});
```

Run: `cd ~/deepcutsCRM/web && npx vitest run src/routes/office/Customers 2>&1 | tail -3` → fails (module missing).

- [ ] **Step 2: Shared hooks and components**

`web/src/components/useApi.ts`:
```ts
import { useCallback, useEffect, useState } from "react";
import { errorOf, type ApiError } from "../api/client";

type Result<T> = { data?: T; error?: unknown; response: Response };

export function useApi<T>(fn: () => Promise<Result<T>>, deps: unknown[]) {
  const [data, setData] = useState<T | undefined>();
  const [error, setError] = useState<ApiError | null>(null);
  const [loading, setLoading] = useState(true);
  const [tick, setTick] = useState(0);
  useEffect(() => {
    let alive = true;
    setLoading(true);
    fn().then((r) => {
      if (!alive) return;
      const e = errorOf(r);
      setError(e);
      setData(e ? undefined : r.data);
    }).finally(() => alive && setLoading(false));
    return () => { alive = false; };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, tick]);
  const reload = useCallback(() => setTick((t) => t + 1), []);
  return { data, error, loading, reload, setData };
}
```

`web/src/components/Field.tsx`:
```tsx
import type { ReactNode } from "react";

export function Field({ label, error, children }: { label: string; error?: string; children: ReactNode }) {
  return (
    <label>
      {label}
      {children}
      {error && <span className="error">{error}</span>}
    </label>
  );
}

export function ErrorBanner({ error }: { error: { message: string } | null | undefined }) {
  return error ? <p className="error" role="alert">{error.message}</p> : null;
}
```

- [ ] **Step 3: Customers list with inline create**

`web/src/routes/office/Customers.tsx`:
```tsx
import { useState, type FormEvent } from "react";
import { Link } from "react-router";
import { api, errorOf, type ApiError } from "../../api/client";
import { ErrorBanner, Field } from "../../components/Field";
import { useApi } from "../../components/useApi";

export default function Customers() {
  const [showInactive, setShowInactive] = useState(false);
  const list = useApi(() => api.GET("/api/office/customers", { params: { query: { includeInactive: showInactive } } }), [showInactive]);
  const [name, setName] = useState("");
  const [deliveryAddress, setDeliveryAddress] = useState("");
  const [err, setErr] = useState<ApiError | null>(null);
  const [busy, setBusy] = useState(false);

  async function create(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    const res = await api.POST("/api/office/customers", { body: { name, deliveryAddress } });
    setBusy(false);
    const e2 = errorOf(res);
    setErr(e2);
    if (!e2) { setName(""); setDeliveryAddress(""); list.reload(); }
  }

  return (
    <>
      <h1>Customers</h1>
      <form onSubmit={create} className="row card">
        <Field label="Name" error={err?.fields?.name}><input value={name} onChange={(e) => setName(e.target.value)} /></Field>
        <Field label="Delivery address" error={err?.fields?.deliveryAddress}><input value={deliveryAddress} onChange={(e) => setDeliveryAddress(e.target.value)} /></Field>
        <button disabled={busy}>Add customer</button>
        <ErrorBanner error={err && !err.fields ? err : null} />
      </form>
      <p><label className="row"><input type="checkbox" checked={showInactive} onChange={(e) => setShowInactive(e.target.checked)} /> show inactive</label></p>
      <ErrorBanner error={list.error} />
      {list.data && list.data.length === 0 && <p className="muted">No customers yet.</p>}
      {list.data && list.data.length > 0 && (
        <table>
          <thead><tr><th>Name</th><th>Delivery address</th><th>Contact</th><th>Days</th><th></th></tr></thead>
          <tbody>
            {list.data.map((c) => (
              <tr key={c.id}>
                <td><Link to={`/office/customers/${c.id}`}>{c.name}</Link></td>
                <td>{c.deliveryAddress}</td>
                <td>{c.contactName} {c.phone}</td>
                <td>{c.deliveryDays.join(", ")}</td>
                <td>{!c.active && <span className="badge">inactive</span>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </>
  );
}
```

- [ ] **Step 4: Customer detail with edit form and price list**

`web/src/routes/office/CustomerDetail.tsx`:
```tsx
import { useEffect, useState, type FormEvent } from "react";
import { Link, useParams } from "react-router";
import { api, errorOf, type ApiError, type Schemas } from "../../api/client";
import { ErrorBanner, Field } from "../../components/Field";
import { useApi } from "../../components/useApi";
import { money, toCents } from "../../lib/format";

const DAYS = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"];
type Input = Schemas["CustomerInput"];

export function CustomerForm({ initial, onSubmit, fields, busy }: { initial: Input; onSubmit: (v: Input) => void; fields?: Record<string, string>; busy: boolean }) {
  const [v, setV] = useState<Input>(initial);
  useEffect(() => setV(initial), [initial]);
  const set = (k: keyof Input) => (e: { target: { value: string } }) => setV({ ...v, [k]: e.target.value });
  const days = v.deliveryDays ?? [];
  return (
    <form onSubmit={(e: FormEvent) => { e.preventDefault(); onSubmit(v); }} className="stack card">
      <Field label="Name" error={fields?.name}><input value={v.name} onChange={set("name")} /></Field>
      <div className="cols">
        <Field label="Delivery address"><textarea value={v.deliveryAddress ?? ""} onChange={set("deliveryAddress")} /></Field>
        <Field label="Billing address"><textarea value={v.billingAddress ?? ""} onChange={set("billingAddress")} /></Field>
        <Field label="Contact name"><input value={v.contactName ?? ""} onChange={set("contactName")} /></Field>
        <Field label="Phone"><input value={v.phone ?? ""} onChange={set("phone")} /></Field>
        <Field label="Email"><input value={v.email ?? ""} onChange={set("email")} /></Field>
        <Field label="QuickBooks customer id"><input value={v.qboCustomerId ?? ""} onChange={set("qboCustomerId")} /></Field>
      </div>
      <Field label="Delivery notes (dock hours, gate codes)"><textarea value={v.deliveryNotes ?? ""} onChange={set("deliveryNotes")} /></Field>
      <div className="row">
        <span>Delivery days:</span>
        {DAYS.map((d) => (
          <label key={d} className="row"><input type="checkbox" checked={days.includes(d)} onChange={(e) => setV({ ...v, deliveryDays: e.target.checked ? [...days, d] : days.filter((x) => x !== d) })} />{d}</label>
        ))}
        {fields?.deliveryDays && <span className="error">{fields.deliveryDays}</span>}
      </div>
      <label className="row"><input type="checkbox" checked={v.active ?? true} onChange={(e) => setV({ ...v, active: e.target.checked })} /> active</label>
      <div><button disabled={busy}>Save</button></div>
    </form>
  );
}

export default function CustomerDetail() {
  const id = Number(useParams().id);
  const cust = useApi(() => api.GET("/api/office/customers/{customerId}", { params: { path: { customerId: id } } }), [id]);
  const prices = useApi(() => api.GET("/api/office/customers/{customerId}/prices", { params: { path: { customerId: id } } }), [id]);
  const products = useApi(() => api.GET("/api/office/products"), []);
  const [err, setErr] = useState<ApiError | null>(null);
  const [busy, setBusy] = useState(false);
  const [saved, setSaved] = useState(false);
  const [productId, setProductId] = useState<number>(0);
  const [price, setPrice] = useState("");
  const [effectiveFrom, setEffectiveFrom] = useState(new Date().toISOString().slice(0, 10));
  const [perr, setPerr] = useState<ApiError | null>(null);

  async function save(v: Input) {
    setBusy(true); setSaved(false);
    const res = await api.PUT("/api/office/customers/{customerId}", { params: { path: { customerId: id } }, body: v });
    setBusy(false);
    setErr(errorOf(res));
    if (res.data) { cust.setData(res.data); setSaved(true); }
  }

  async function addPrice(e: FormEvent) {
    e.preventDefault();
    const cents = toCents(price);
    if (cents == null) return setPerr({ code: "invalid", message: "price must be a dollar amount like 5.99", fields: { priceCents: "must be a dollar amount" } });
    const res = await api.POST("/api/office/customers/{customerId}/prices", { params: { path: { customerId: id } }, body: { productId, priceCents: cents, effectiveFrom } });
    setPerr(errorOf(res));
    if (res.data) { setPrice(""); prices.reload(); }
  }

  if (cust.error) return <ErrorBanner error={cust.error} />;
  if (!cust.data) return <p className="muted">Loading…</p>;
  const c = cust.data;
  const initial: Input = { name: c.name, billingAddress: c.billingAddress, deliveryAddress: c.deliveryAddress, contactName: c.contactName, phone: c.phone, email: c.email, deliveryNotes: c.deliveryNotes, deliveryDays: c.deliveryDays, qboCustomerId: c.qboCustomerId ?? "", active: c.active };
  return (
    <>
      <p><Link to="/office/customers">← Customers</Link></p>
      <h1>{c.name}</h1>
      <ErrorBanner error={err && !err.fields ? err : null} />
      {saved && <p className="muted">Saved.</p>}
      <CustomerForm initial={initial} onSubmit={save} fields={err?.fields} busy={busy} />

      <h2>Negotiated prices</h2>
      <form onSubmit={addPrice} className="row card">
        <Field label="Product" error={perr?.fields?.productId}>
          <select value={productId} onChange={(e) => setProductId(Number(e.target.value))}>
            <option value={0}>— choose —</option>
            {products.data?.map((p) => <option key={p.id} value={p.id}>{p.sku} · {p.name} ({money(p.basePriceCents)} base)</option>)}
          </select>
        </Field>
        <Field label="Price" error={perr?.fields?.priceCents}><input className="short" value={price} onChange={(e) => setPrice(e.target.value)} placeholder="5.49" /></Field>
        <Field label="Effective from" error={perr?.fields?.effectiveFrom}><input type="date" value={effectiveFrom} onChange={(e) => setEffectiveFrom(e.target.value)} /></Field>
        <button disabled={!productId}>Set price</button>
        <ErrorBanner error={perr && !perr.fields ? perr : null} />
      </form>
      {prices.data && prices.data.length === 0 && <p className="muted">No negotiated prices; this customer pays base prices.</p>}
      {prices.data && prices.data.length > 0 && (
        <table>
          <thead><tr><th>SKU</th><th>Product</th><th className="num">Price</th><th>Effective</th></tr></thead>
          <tbody>{prices.data.map((p) => <tr key={p.id}><td>{p.sku}</td><td>{p.productName}</td><td className="num">{money(p.priceCents)}</td><td>{p.effectiveFrom}</td></tr>)}</tbody>
        </table>
      )}
    </>
  );
}
```

- [ ] **Step 5: Products list and detail**

`web/src/routes/office/Products.tsx`:
```tsx
import { useState, type FormEvent } from "react";
import { Link } from "react-router";
import { api, errorOf, type ApiError } from "../../api/client";
import { ErrorBanner, Field } from "../../components/Field";
import { useApi } from "../../components/useApi";
import { hundredths, money, toCents, toHundredths } from "../../lib/format";

export default function Products() {
  const [showInactive, setShowInactive] = useState(false);
  const list = useApi(() => api.GET("/api/office/products", { params: { query: { includeInactive: showInactive } } }), [showInactive]);
  const [f, setF] = useState({ sku: "", name: "", category: "", sellUnit: "case" as "lb" | "case" | "each", catchWeight: true, approxCaseWeight: "", basePrice: "" });
  const [err, setErr] = useState<ApiError | null>(null);
  const [busy, setBusy] = useState(false);

  async function create(e: FormEvent) {
    e.preventDefault();
    const basePriceCents = toCents(f.basePrice);
    const w = f.approxCaseWeight ? toHundredths(f.approxCaseWeight) : null;
    if (basePriceCents == null) return setErr({ code: "invalid", message: "", fields: { basePriceCents: "must be a dollar amount" } });
    setBusy(true);
    const res = await api.POST("/api/office/products", { body: { sku: f.sku, name: f.name, category: f.category, sellUnit: f.sellUnit, catchWeight: f.catchWeight, approxCaseWeight: w ?? undefined, basePriceCents } });
    setBusy(false);
    const e2 = errorOf(res);
    setErr(e2);
    if (!e2) { setF({ ...f, sku: "", name: "", approxCaseWeight: "", basePrice: "" }); list.reload(); }
  }

  return (
    <>
      <h1>Products</h1>
      <form onSubmit={create} className="row card">
        <Field label="SKU" error={err?.fields?.sku}><input className="short" value={f.sku} onChange={(e) => setF({ ...f, sku: e.target.value })} /></Field>
        <Field label="Name" error={err?.fields?.name}><input value={f.name} onChange={(e) => setF({ ...f, name: e.target.value })} /></Field>
        <Field label="Category"><input className="short" value={f.category} onChange={(e) => setF({ ...f, category: e.target.value })} /></Field>
        <Field label="Sell unit" error={err?.fields?.sellUnit}>
          <select value={f.sellUnit} onChange={(e) => setF({ ...f, sellUnit: e.target.value as typeof f.sellUnit })}><option value="case">case</option><option value="each">each</option><option value="lb">lb</option></select>
        </Field>
        <label className="row"><input type="checkbox" checked={f.catchWeight} onChange={(e) => setF({ ...f, catchWeight: e.target.checked })} /> catch-weight (priced per lb)</label>
        <Field label="Approx case lb" error={err?.fields?.approxCaseWeight}><input className="short" value={f.approxCaseWeight} onChange={(e) => setF({ ...f, approxCaseWeight: e.target.value })} disabled={!f.catchWeight} /></Field>
        <Field label={f.catchWeight ? "Base $/lb" : "Base price"} error={err?.fields?.basePriceCents}><input className="short" value={f.basePrice} onChange={(e) => setF({ ...f, basePrice: e.target.value })} /></Field>
        <button disabled={busy}>Add product</button>
        <ErrorBanner error={err && !err.fields ? err : null} />
      </form>
      <p><label className="row"><input type="checkbox" checked={showInactive} onChange={(e) => setShowInactive(e.target.checked)} /> show inactive</label></p>
      <ErrorBanner error={list.error} />
      {list.data && (
        <table>
          <thead><tr><th>SKU</th><th>Name</th><th>Category</th><th>Unit</th><th className="num">Base price</th><th></th></tr></thead>
          <tbody>
            {list.data.map((p) => (
              <tr key={p.id}>
                <td><Link to={`/office/products/${p.id}`}>{p.sku}</Link></td>
                <td>{p.name}</td>
                <td>{p.category}</td>
                <td>{p.catchWeight ? `case (~${hundredths(p.approxCaseWeight ?? 0)} lb), priced /lb` : p.sellUnit}</td>
                <td className="num">{money(p.basePriceCents)}</td>
                <td>{!p.active && <span className="badge">inactive</span>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </>
  );
}
```

`web/src/routes/office/ProductDetail.tsx`:
```tsx
import { useState, type FormEvent } from "react";
import { Link, useParams } from "react-router";
import { api, errorOf, type ApiError } from "../../api/client";
import { ErrorBanner, Field } from "../../components/Field";
import { useApi } from "../../components/useApi";
import { hundredths, toCents, toHundredths } from "../../lib/format";

export default function ProductDetail() {
  const id = Number(useParams().id);
  const prod = useApi(() => api.GET("/api/office/products/{productId}", { params: { path: { productId: id } } }), [id]);
  const [err, setErr] = useState<ApiError | null>(null);
  const [saved, setSaved] = useState(false);
  const [f, setF] = useState<{ sku: string; name: string; category: string; sellUnit: "lb" | "case" | "each"; catchWeight: boolean; approxCaseWeight: string; basePrice: string; qboItemId: string; active: boolean } | null>(null);

  const p = prod.data;
  if (prod.error) return <ErrorBanner error={prod.error} />;
  if (!p) return <p className="muted">Loading…</p>;
  const v = f ?? { sku: p.sku, name: p.name, category: p.category, sellUnit: p.sellUnit, catchWeight: p.catchWeight, approxCaseWeight: p.approxCaseWeight ? hundredths(p.approxCaseWeight) : "", basePrice: (p.basePriceCents / 100).toFixed(2), qboItemId: p.qboItemId ?? "", active: p.active };

  async function save(e: FormEvent) {
    e.preventDefault();
    const basePriceCents = toCents(v.basePrice);
    const w = v.approxCaseWeight ? toHundredths(v.approxCaseWeight) : null;
    if (basePriceCents == null) return setErr({ code: "invalid", message: "", fields: { basePriceCents: "must be a dollar amount" } });
    const res = await api.PUT("/api/office/products/{productId}", { params: { path: { productId: id } }, body: { sku: v.sku, name: v.name, category: v.category, sellUnit: v.sellUnit, catchWeight: v.catchWeight, approxCaseWeight: w ?? undefined, basePriceCents, qboItemId: v.qboItemId, active: v.active } });
    setErr(errorOf(res));
    if (res.data) { prod.setData(res.data); setF(null); setSaved(true); }
  }

  return (
    <>
      <p><Link to="/office/products">← Products</Link></p>
      <h1>{p.sku} · {p.name}</h1>
      {saved && <p className="muted">Saved.</p>}
      <ErrorBanner error={err && !err.fields ? err : null} />
      <form onSubmit={save} className="stack card">
        <div className="cols">
          <Field label="SKU" error={err?.fields?.sku}><input value={v.sku} onChange={(e) => setF({ ...v, sku: e.target.value })} /></Field>
          <Field label="Name" error={err?.fields?.name}><input value={v.name} onChange={(e) => setF({ ...v, name: e.target.value })} /></Field>
          <Field label="Category"><input value={v.category} onChange={(e) => setF({ ...v, category: e.target.value })} /></Field>
          <Field label="Sell unit" error={err?.fields?.sellUnit}>
            <select value={v.sellUnit} onChange={(e) => setF({ ...v, sellUnit: e.target.value as typeof v.sellUnit })}><option value="case">case</option><option value="each">each</option><option value="lb">lb</option></select>
          </Field>
          <Field label="Approx case lb" error={err?.fields?.approxCaseWeight}><input value={v.approxCaseWeight} onChange={(e) => setF({ ...v, approxCaseWeight: e.target.value })} disabled={!v.catchWeight} /></Field>
          <Field label={v.catchWeight ? "Base $/lb" : "Base price"} error={err?.fields?.basePriceCents}><input value={v.basePrice} onChange={(e) => setF({ ...v, basePrice: e.target.value })} /></Field>
          <Field label="QuickBooks item id"><input value={v.qboItemId} onChange={(e) => setF({ ...v, qboItemId: e.target.value })} /></Field>
        </div>
        <label className="row"><input type="checkbox" checked={v.catchWeight} onChange={(e) => setF({ ...v, catchWeight: e.target.checked })} /> catch-weight (ordered by the case, priced per lb)</label>
        <label className="row"><input type="checkbox" checked={v.active} onChange={(e) => setF({ ...v, active: e.target.checked })} /> active</label>
        <div><button>Save</button></div>
      </form>
    </>
  );
}
```

Replace the four `Todo` routes in `router.tsx` with imports of `Customers`, `CustomerDetail`, `Products`, `ProductDetail`.

- [ ] **Step 6: Run tests, typecheck, verify in the browser, commit**

Run: `cd ~/deepcutsCRM/web && npx vitest run 2>&1 | tail -3 && npx tsc -b && npm run build 2>&1 | tail -2`
Expected: tests pass, no type errors, build succeeds.

Verify in the browser against `make demo`: customers list shows the 12 demo customers, open one, change the phone, save, add a negotiated price, see it in the table; products list shows 40 rows.

```bash
cd ~/deepcutsCRM && git add -A && git commit -q -m "Add office customer and product screens with price editor" && git push
```

---

### Task 17: Office screens: orders list and order detail

**Files:**
- Create: `web/src/routes/office/Orders.tsx`, `web/src/routes/office/OrderDetail.tsx`, `web/src/routes/office/OrderDetail.test.tsx`, `web/src/components/StatusBadge.tsx`
- Modify: `web/src/router.tsx`

**Interfaces:**
- Produces: `<StatusBadge status>` mapping order statuses to badge classes (`draft`→default, `confirmed`→pending, `scheduled`→pending, `delivered`→warn, `finalized`→ok, `cancelled`→bad); `<LineRow>` inside OrderDetail with inline editing according to the order status.
- Editing rules mirrored from the API (the API is the authority; the UI just hides what will be rejected): ordered qty and delete only while `draft`/`confirmed`/`scheduled` with route not out; price override and shipped weight while not delivered/finalized/cancelled; delivered qty/weight only while `delivered`; nothing when `finalized`/`cancelled`.

- [ ] **Step 1: Write the failing detail test**

`web/src/routes/office/OrderDetail.test.tsx`:
```tsx
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import OrderDetail from "./OrderDetail";

const order = {
  id: 7, customer: { id: 1, name: "Blue Plate", billingAddress: "", deliveryAddress: "1 Main St", contactName: "", phone: "", email: "", deliveryNotes: "", deliveryDays: [], active: true },
  requestedDeliveryDate: "2026-09-12", status: "draft", notes: "", needsReview: false, totalCents: 65880, createdAt: "2026-09-10T15:00:00Z",
  lines: [{ id: 3, productId: 2, sku: "BRIS", productName: "Brisket", sellUnit: "case", catchWeight: true, orderedQty: 200, unitPriceCents: 549, priceOverridden: false, estWeight: 12000, shortageNote: "", amountCents: 65880, amountSource: "estimated" }],
};

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("OrderDetail", () => {
  afterEach(() => vi.restoreAllMocks());
  it("renders lines with amounts and the estimated source", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url = String(input);
      if (url.includes("/products")) return jsonResponse([]);
      return jsonResponse(order);
    });
    render(<MemoryRouter initialEntries={["/office/orders/7"]}><Routes><Route path="/office/orders/:id" element={<OrderDetail />} /></Routes></MemoryRouter>);
    expect(await screen.findByText("Blue Plate")).toBeInTheDocument();
    expect(screen.getByText("Brisket")).toBeInTheDocument();
    expect(screen.getAllByText("$658.80").length).toBeGreaterThan(0); // line amount and total
    expect(screen.getByText(/estimated/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /confirm/i })).toBeInTheDocument();
  });
});
```

Run: `cd ~/deepcutsCRM/web && npx vitest run src/routes/office/OrderDetail 2>&1 | tail -3` → fails.

- [ ] **Step 2: StatusBadge and Orders list**

`web/src/components/StatusBadge.tsx`:
```tsx
const cls: Record<string, string> = { confirmed: "pending", scheduled: "pending", out: "pending", delivered: "warn", finalized: "ok", complete: "ok", cancelled: "bad", skipped: "bad" };
export function StatusBadge({ status }: { status: string }) {
  return <span className={`badge ${cls[status] ?? ""}`}>{status}</span>;
}
```

`web/src/routes/office/Orders.tsx`:
```tsx
import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router";
import { api, errorOf, type ApiError } from "../../api/client";
import { ErrorBanner, Field } from "../../components/Field";
import { StatusBadge } from "../../components/StatusBadge";
import { useApi } from "../../components/useApi";

const STATUSES = ["", "draft", "confirmed", "scheduled", "delivered", "finalized", "cancelled"];

export default function Orders() {
  const nav = useNavigate();
  const [date, setDate] = useState("");
  const [status, setStatus] = useState("");
  const list = useApi(() => api.GET("/api/office/orders", { params: { query: { date: date || undefined, status: status || undefined } } }), [date, status]);
  const customers = useApi(() => api.GET("/api/office/customers"), []);
  const [customerId, setCustomerId] = useState(0);
  const [newDate, setNewDate] = useState(new Date(Date.now() + 86400000).toISOString().slice(0, 10));
  const [err, setErr] = useState<ApiError | null>(null);

  async function create(e: FormEvent) {
    e.preventDefault();
    const res = await api.POST("/api/office/orders", { body: { customerId, requestedDeliveryDate: newDate } });
    setErr(errorOf(res));
    if (res.data) nav(`/office/orders/${res.data.id}`);
  }

  return (
    <>
      <h1>Orders</h1>
      <form onSubmit={create} className="row card">
        <Field label="Customer" error={err?.fields?.customerId}>
          <select value={customerId} onChange={(e) => setCustomerId(Number(e.target.value))}>
            <option value={0}>— choose —</option>
            {customers.data?.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
        </Field>
        <Field label="Delivery date" error={err?.fields?.requestedDeliveryDate}><input type="date" value={newDate} onChange={(e) => setNewDate(e.target.value)} /></Field>
        <button disabled={!customerId}>New order</button>
        <ErrorBanner error={err && !err.fields ? err : null} />
      </form>
      <div className="row">
        <label>Date<input type="date" value={date} onChange={(e) => setDate(e.target.value)} /></label>
        <label>Status<select value={status} onChange={(e) => setStatus(e.target.value)}>{STATUSES.map((s) => <option key={s} value={s}>{s || "any"}</option>)}</select></label>
      </div>
      <ErrorBanner error={list.error} />
      {list.data && (
        <table>
          <thead><tr><th>#</th><th>Customer</th><th>Delivery</th><th>Status</th><th className="num">Lines</th><th></th></tr></thead>
          <tbody>
            {list.data.map((o) => (
              <tr key={o.id}>
                <td><Link to={`/office/orders/${o.id}`}>{o.id}</Link></td>
                <td>{o.customerName}</td>
                <td>{o.requestedDeliveryDate}</td>
                <td><StatusBadge status={o.status} /></td>
                <td className="num">{o.lineCount}</td>
                <td>{o.needsReview && <span className="badge warn">needs review</span>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </>
  );
}
```

- [ ] **Step 3: Order detail with editable lines and actions**

`web/src/routes/office/OrderDetail.tsx`:
```tsx
import { useState, type FormEvent } from "react";
import { Link, useParams } from "react-router";
import { api, errorOf, type ApiError, type Schemas } from "../../api/client";
import { ErrorBanner, Field } from "../../components/Field";
import { StatusBadge } from "../../components/StatusBadge";
import { useApi } from "../../components/useApi";
import { hundredths, lb, money, qty, toCents, toHundredths } from "../../lib/format";

type Order = Schemas["Order"];
type Line = Schemas["OrderLine"];

function scope(o: Order) {
  const routeOut = o.routeStatus === "out";
  return {
    lines: ["draft", "confirmed", "scheduled"].includes(o.status) && !routeOut,
    shippedAndPrice: ["draft", "confirmed", "scheduled"].includes(o.status),
    delivered: o.status === "delivered",
  };
}

function LineRow({ o, l, onPatch, onDelete }: { o: Order; l: Line; onPatch: (lineId: number, body: Schemas["LinePatch"]) => Promise<ApiError | null>; onDelete: (lineId: number) => void }) {
  const s = scope(o);
  const [err, setErr] = useState<ApiError | null>(null);
  const [edit, setEdit] = useState<{ orderedQty: string; unitPrice: string; shippedWeight: string; deliveredQty: string; deliveredWeight: string; shortageNote: string }>({
    orderedQty: hundredths(l.orderedQty), unitPrice: (l.unitPriceCents / 100).toFixed(2), shippedWeight: l.shippedWeight != null ? hundredths(l.shippedWeight) : "",
    deliveredQty: l.deliveredQty != null ? hundredths(l.deliveredQty) : "", deliveredWeight: l.deliveredWeight != null ? hundredths(l.deliveredWeight) : "", shortageNote: l.shortageNote,
  });
  async function commit(field: keyof typeof edit) {
    const body: Schemas["LinePatch"] = {};
    const v = edit[field];
    if (field === "orderedQty") { const n = toHundredths(v); if (n == null) return setErr({ code: "invalid", message: "quantity must be a number" }); body.orderedQty = n; }
    if (field === "unitPrice") { const n = toCents(v); if (n == null) return setErr({ code: "invalid", message: "price must be a dollar amount" }); body.unitPriceCents = n; }
    if (field === "shippedWeight") { if (v === "") return; const n = toHundredths(v); if (n == null) return setErr({ code: "invalid", message: "weight must be a number" }); body.shippedWeight = n; }
    if (field === "deliveredQty") { if (v === "") return; const n = toHundredths(v); if (n == null) return setErr({ code: "invalid", message: "quantity must be a number" }); body.deliveredQty = n; }
    if (field === "deliveredWeight") { if (v === "") return; const n = toHundredths(v); if (n == null) return setErr({ code: "invalid", message: "weight must be a number" }); body.deliveredWeight = n; }
    if (field === "shortageNote") body.shortageNote = v;
    setErr(await onPatch(l.id, body));
  }
  const num = (field: keyof typeof edit, enabled: boolean) => (
    <input className="short" value={edit[field]} disabled={!enabled} onChange={(e) => setEdit({ ...edit, [field]: e.target.value })} onBlur={() => enabled && commit(field)} />
  );
  return (
    <tr>
      <td>{l.sku}<br /><span className="muted">{l.productName}</span></td>
      <td>{s.lines ? num("orderedQty", true) : qty(l.orderedQty, l.sellUnit)}{l.catchWeight && <div className="muted">est {lb(l.estWeight)}</div>}</td>
      <td>{s.shippedAndPrice ? num("unitPrice", true) : money(l.unitPriceCents)}{l.priceOverridden && <div className="muted">overridden</div>}<div className="muted">{l.catchWeight ? "/lb" : `/${l.sellUnit}`}</div></td>
      <td>{l.catchWeight ? (s.shippedAndPrice ? num("shippedWeight", true) : lb(l.shippedWeight)) : "—"}</td>
      <td>{s.delivered ? <>{num("deliveredQty", true)}{l.catchWeight && <> {num("deliveredWeight", true)}</>}</> : <>{l.deliveredQty != null ? qty(l.deliveredQty, l.sellUnit) : "—"}{l.catchWeight && <div className="muted">{lb(l.deliveredWeight)}</div>}</>}
        {(s.delivered || l.shortageNote) && <input value={edit.shortageNote} disabled={!s.delivered} placeholder="shortage note" onChange={(e) => setEdit({ ...edit, shortageNote: e.target.value })} onBlur={() => s.delivered && commit("shortageNote")} />}
      </td>
      <td className="num">{money(l.amountCents)}<div className="muted">{l.amountSource}</div></td>
      <td>{s.lines && <button className="link" onClick={() => onDelete(l.id)}>remove</button>}{err && <div className="error">{err.message}</div>}</td>
    </tr>
  );
}

export default function OrderDetail() {
  const id = Number(useParams().id);
  const ord = useApi(() => api.GET("/api/office/orders/{orderId}", { params: { path: { orderId: id } } }), [id]);
  const products = useApi(() => api.GET("/api/office/products"), []);
  const [err, setErr] = useState<ApiError | null>(null);
  const [productId, setProductId] = useState(0);
  const [qtyStr, setQtyStr] = useState("1");
  const [notes, setNotes] = useState<string | null>(null);

  const o = ord.data;
  if (ord.error) return <ErrorBanner error={ord.error} />;
  if (!o) return <p className="muted">Loading…</p>;
  const s = scope(o);
  const path = { params: { path: { orderId: id } } };

  const apply = async (p: Promise<{ data?: Order; error?: unknown; response: Response }>) => {
    const res = await p;
    const e = errorOf(res);
    setErr(e);
    if (res.data) ord.setData(res.data);
    return e;
  };

  async function addLine(e: FormEvent) {
    e.preventDefault();
    const n = toHundredths(qtyStr);
    if (n == null || n === 0) return setErr({ code: "invalid", message: "quantity must be a positive number" });
    if (!(await apply(api.POST("/api/office/orders/{orderId}/lines", { ...path, body: { productId, orderedQty: n } })))) setQtyStr("1");
  }

  const action = (name: "confirm" | "unconfirm" | "cancel" | "finalize") => () => {
    switch (name) {
      case "confirm": return apply(api.POST("/api/office/orders/{orderId}/confirm", path));
      case "unconfirm": return apply(api.POST("/api/office/orders/{orderId}/unconfirm", path));
      case "cancel": return apply(api.POST("/api/office/orders/{orderId}/cancel", path));
      case "finalize": return apply(api.POST("/api/office/orders/{orderId}/finalize", path));
    }
  };

  return (
    <>
      <p><Link to="/office/orders">← Orders</Link></p>
      <h1>Order #{o.id} <StatusBadge status={o.status} /> {o.needsReview && <span className="badge warn">needs review</span>}</h1>
      <div className="cols">
        <div className="card">
          <strong>{o.customer.name}</strong><br />
          {o.customer.deliveryAddress}<br />
          <span className="muted">{o.customer.contactName} {o.customer.phone}</span>
          {o.customer.deliveryNotes && <p className="muted">{o.customer.deliveryNotes}</p>}
        </div>
        <div className="card stack">
          <div>Delivery date: <strong>{o.requestedDeliveryDate}</strong>{o.routeId && <> · on route <Link to={`/office/day?date=${o.requestedDeliveryDate}`}>#{o.routeId}</Link> ({o.routeStatus})</>}</div>
          <Field label="Notes"><textarea value={notes ?? o.notes} disabled={!s.lines} onChange={(e) => setNotes(e.target.value)} onBlur={() => notes != null && notes !== o.notes && apply(api.PATCH("/api/office/orders/{orderId}", { ...path, body: { notes } }))} /></Field>
          <div className="row">
            {o.status === "draft" && <button onClick={action("confirm")}>Confirm</button>}
            {o.status === "confirmed" && <button className="secondary" onClick={action("unconfirm")}>Back to draft</button>}
            {o.status === "delivered" && <button onClick={action("finalize")}>Finalize</button>}
            {["draft", "confirmed", "scheduled"].includes(o.status) && <button className="danger" onClick={() => confirm("Cancel this order?") && action("cancel")()}>Cancel order</button>}
          </div>
        </div>
      </div>
      <ErrorBanner error={err} />
      {err?.fields && <ul className="error">{Object.entries(err.fields).map(([k, v]) => <li key={k}>{k}: {v}</li>)}</ul>}

      <h2>Lines</h2>
      <table>
        <thead><tr><th>Product</th><th>Ordered</th><th>Unit price</th><th>Shipped wt</th><th>Delivered</th><th className="num">Amount</th><th></th></tr></thead>
        <tbody>
          {o.lines.map((l) => (
            <LineRow key={l.id} o={o} l={l}
              onPatch={(lineId, body) => apply(api.PATCH("/api/office/orders/{orderId}/lines/{lineId}", { params: { path: { orderId: id, lineId } }, body }))}
              onDelete={(lineId) => apply(api.DELETE("/api/office/orders/{orderId}/lines/{lineId}", { params: { path: { orderId: id, lineId } } }))} />
          ))}
          <tr><td colSpan={5} className="num"><strong>Total</strong></td><td className="num"><strong>{money(o.totalCents)}</strong></td><td /></tr>
        </tbody>
      </table>
      {s.lines && (
        <form onSubmit={addLine} className="row card">
          <Field label="Product">
            <select value={productId} onChange={(e) => setProductId(Number(e.target.value))}>
              <option value={0}>— choose —</option>
              {products.data?.map((p) => <option key={p.id} value={p.id}>{p.sku} · {p.name}</option>)}
            </select>
          </Field>
          <Field label="Quantity"><input className="short" value={qtyStr} onChange={(e) => setQtyStr(e.target.value)} /></Field>
          <button disabled={!productId}>Add line</button>
        </form>
      )}
    </>
  );
}
```

Replace the `Orders` and `Order` `Todo` routes in `router.tsx`.

- [ ] **Step 4: Run tests, typecheck, verify in the browser, commit**

Run: `cd ~/deepcutsCRM/web && npx vitest run 2>&1 | tail -3 && npx tsc -b && npm run build 2>&1 | tail -2`

Verify against `make demo`: open a draft order from the list, add a brisket line (see estimated weight and amount), change the quantity (amount updates), confirm it, enter a shipped weight (amount source flips to `shipped`), open yesterday's delivered order with the review badge, adjust delivered weight, finalize.

```bash
cd ~/deepcutsCRM && git add -A && git commit -q -m "Add office orders list and order detail with line editing" && git push
```

---

### Task 18: Office day view: routes, stops, review

**Files:**
- Create: `web/src/routes/office/Day.tsx`, `web/src/routes/office/Day.test.tsx`
- Modify: `web/src/router.tsx`

**Interfaces:**
- Consumes: `GET /api/office/day`, `POST /api/office/routes`, `POST/DELETE .../stops`, `PUT .../sequence`, `POST .../out`, `POST .../complete`, `GET /api/office/drivers`, `GET /api/office/stops/{id}/proof`.
- Behavior: date from `?date=` query (default today); left column lists unscheduled confirmed orders each with an "add to route" select; right column lists routes with stops (sequence, customer, address, line count, status badge, review badge linking to the order, proof link when an image exists); per-route: up/down reorder buttons and remove while `planned`, "Mark out" button while planned, "Complete" while out; a "New route" form (driver select, truck label).

- [ ] **Step 1: Write the failing test**

`web/src/routes/office/Day.test.tsx`:
```tsx
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import Day from "./Day";

const day = {
  date: "2026-09-10",
  unscheduled: [{ id: 5, customerId: 1, customerName: "Corner Tavern", requestedDeliveryDate: "2026-09-10", status: "confirmed", needsReview: false, lineCount: 2, notes: "" }],
  routes: [{
    id: 2, routeDate: "2026-09-10", driverUserId: 9, driverName: "Sam", truckLabel: "Reefer 1", status: "out",
    stops: [
      { id: 11, routeId: 2, orderId: 3, sequence: 1, status: "delivered", hasProofImage: true, proofType: "signature", skipReason: "", driverNote: "", customerName: "Blue Plate", deliveryAddress: "1 Main St", phone: "", contactName: "", deliveryNotes: "", orderStatus: "delivered", needsReview: true, lineCount: 3 },
      { id: 12, routeId: 2, orderId: 4, sequence: 2, status: "pending", hasProofImage: false, skipReason: "", driverNote: "", customerName: "Red Barn", deliveryAddress: "4400 County Rd", phone: "", contactName: "", deliveryNotes: "", orderStatus: "scheduled", needsReview: false, lineCount: 1 },
    ],
  }],
};

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("Day", () => {
  afterEach(() => vi.restoreAllMocks());
  it("renders unscheduled orders and routes with stops", async () => {
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url = String(input);
      if (url.includes("/drivers")) return jsonResponse([{ id: 9, displayName: "Sam" }]);
      return jsonResponse(day);
    });
    render(<MemoryRouter initialEntries={["/office/day?date=2026-09-10"]}><Day /></MemoryRouter>);
    expect(await screen.findByText("Corner Tavern")).toBeInTheDocument();
    expect(screen.getAllByText(/Sam/).length).toBeGreaterThan(0); // route header and driver select
    expect(screen.getByText("Blue Plate")).toBeInTheDocument();
    expect(screen.getByText("needs review")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /proof/i })).toHaveAttribute("href", "/api/office/stops/11/proof");
    expect(screen.getByRole("button", { name: /complete/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /mark out/i })).not.toBeInTheDocument();
  });
});
```

Run: `cd ~/deepcutsCRM/web && npx vitest run src/routes/office/Day 2>&1 | tail -3` → fails.

- [ ] **Step 2: Implement Day.tsx**

```tsx
import { useState, type FormEvent } from "react";
import { Link, useSearchParams } from "react-router";
import { api, errorOf, type ApiError, type Schemas } from "../../api/client";
import { ErrorBanner, Field } from "../../components/Field";
import { StatusBadge } from "../../components/StatusBadge";
import { useApi } from "../../components/useApi";

type Route = Schemas["Route"];

function today(): string {
  const d = new Date();
  return new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 10);
}

export default function Day() {
  const [params, setParams] = useSearchParams();
  const date = params.get("date") || today();
  const day = useApi(() => api.GET("/api/office/day", { params: { query: { date } } }), [date]);
  const drivers = useApi(() => api.GET("/api/office/drivers"), []);
  const [err, setErr] = useState<ApiError | null>(null);
  const [driverId, setDriverId] = useState(0);
  const [truck, setTruck] = useState("");

  const run = async (p: Promise<{ error?: unknown; response: Response }>) => {
    const res = await p;
    setErr(errorOf(res));
    day.reload();
  };
  const rp = (routeId: number) => ({ params: { path: { routeId } } });

  async function newRoute(e: FormEvent) {
    e.preventDefault();
    await run(api.POST("/api/office/routes", { body: { routeDate: date, driverUserId: driverId, truckLabel: truck } }));
    setTruck("");
  }

  function move(r: Route, idx: number, dir: -1 | 1) {
    const ids = r.stops.map((s) => s.id);
    const j = idx + dir;
    if (j < 0 || j >= ids.length) return;
    [ids[idx], ids[j]] = [ids[j], ids[idx]];
    run(api.PUT("/api/office/routes/{routeId}/sequence", { ...rp(r.id), body: { stopIds: ids } }));
  }

  return (
    <>
      <div className="row">
        <h1>Day</h1>
        <input type="date" value={date} onChange={(e) => setParams({ date: e.target.value })} />
        <button className="secondary" onClick={() => setParams({ date: shift(date, -1) })}>‹ prev</button>
        <button className="secondary" onClick={() => setParams({ date: shift(date, 1) })}>next ›</button>
      </div>
      <ErrorBanner error={err} />
      <ErrorBanner error={day.error} />
      <div className="cols">
        <section>
          <h2>Unscheduled ({day.data?.unscheduled.length ?? 0})</h2>
          {day.data?.unscheduled.length === 0 && <p className="muted">Every confirmed order for this date is on a route.</p>}
          {day.data?.unscheduled.map((o) => (
            <div key={o.id} className="card row">
              <div style={{ flex: 1 }}>
                <Link to={`/office/orders/${o.id}`}>#{o.id}</Link> <strong>{o.customerName}</strong> <span className="muted">{o.lineCount} lines</span>
                {o.needsReview && <> <span className="badge warn">skipped earlier</span></>}
              </div>
              {day.data && day.data.routes.filter((r) => r.status === "planned").length > 0 && (
                <select defaultValue="" onChange={(e) => e.target.value && run(api.POST("/api/office/routes/{routeId}/stops", { ...rp(Number(e.target.value)), body: { orderId: o.id } }))}>
                  <option value="">add to route…</option>
                  {day.data.routes.filter((r) => r.status === "planned").map((r) => <option key={r.id} value={r.id}>#{r.id} {r.driverName} {r.truckLabel}</option>)}
                </select>
              )}
            </div>
          ))}
          <h2>New route</h2>
          <form onSubmit={newRoute} className="row card">
            <Field label="Driver"><select value={driverId} onChange={(e) => setDriverId(Number(e.target.value))}><option value={0}>— choose —</option>{drivers.data?.map((d) => <option key={d.id} value={d.id}>{d.displayName}</option>)}</select></Field>
            <Field label="Truck"><input className="short" value={truck} onChange={(e) => setTruck(e.target.value)} placeholder="Reefer 1" /></Field>
            <button disabled={!driverId}>Create route</button>
          </form>
        </section>
        <section>
          <h2>Routes ({day.data?.routes.length ?? 0})</h2>
          {day.data?.routes.map((r) => (
            <div key={r.id} className="card" style={{ marginBottom: "1rem" }}>
              <div className="row">
                <strong>Route #{r.id} · {r.driverName}</strong> <span className="muted">{r.truckLabel}</span> <StatusBadge status={r.status} />
                <span className="spacer" />
                {r.status === "planned" && <button onClick={() => run(api.POST("/api/office/routes/{routeId}/out", rp(r.id)))}>Mark out</button>}
                {r.status === "out" && <button className="secondary" onClick={() => run(api.POST("/api/office/routes/{routeId}/complete", rp(r.id)))}>Complete</button>}
              </div>
              {r.stops.length === 0 && <p className="muted">No stops yet.</p>}
              <table>
                <tbody>
                  {r.stops.map((s, i) => (
                    <tr key={s.id}>
                      <td className="num">{s.sequence}</td>
                      <td>
                        <Link to={`/office/orders/${s.orderId}`}>{s.customerName}</Link><br />
                        <span className="muted">{s.deliveryAddress}</span>
                        {s.deliveryNotes && <div className="muted">{s.deliveryNotes}</div>}
                        {s.skipReason && <div className="error">skipped: {s.skipReason}</div>}
                        {s.driverNote && <div className="muted">driver: {s.driverNote}</div>}
                      </td>
                      <td>
                        <StatusBadge status={s.status} />
                        {s.needsReview && <> <Link to={`/office/orders/${s.orderId}`} className="badge warn">needs review</Link></>}
                        {s.hasProofImage && <> <a href={`/api/office/stops/${s.id}/proof`} target="_blank" rel="noreferrer">proof</a></>}
                        {s.proofName && <div className="muted">received by {s.proofName}</div>}
                      </td>
                      <td>
                        {r.status === "planned" && (
                          <span className="row">
                            <button className="secondary" onClick={() => move(r, i, -1)} disabled={i === 0}>↑</button>
                            <button className="secondary" onClick={() => move(r, i, 1)} disabled={i === r.stops.length - 1}>↓</button>
                            <button className="link" onClick={() => run(api.DELETE("/api/office/routes/{routeId}/stops/{stopId}", { params: { path: { routeId: r.id, stopId: s.id } } }))}>remove</button>
                          </span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
        </section>
      </div>
    </>
  );
}

function shift(date: string, days: number): string {
  const d = new Date(date + "T12:00:00");
  d.setDate(d.getDate() + days);
  return d.toISOString().slice(0, 10);
}
```

Replace the `Day` `Todo` route in `router.tsx`.

- [ ] **Step 3: Run tests, typecheck, verify in the browser, commit**

Run: `cd ~/deepcutsCRM/web && npx vitest run 2>&1 | tail -3 && npx tsc -b && npm run build 2>&1 | tail -2`

Verify against `make demo`: today shows one route out with four stops and two unscheduled orders; create a route for tomorrow, add both tomorrow orders, reorder, remove one, mark out; yesterday's route shows a delivered stop with a review badge and the received-by name.

```bash
cd ~/deepcutsCRM && git add -A && git commit -q -m "Add office day view with routes, stops and review" && git push
```

---

### Task 19: Driver screens: login, route, stop, proof capture (online)

**Files:**
- Create: `web/src/routes/driver/Login.tsx`, `web/src/routes/driver/Route.tsx`, `web/src/routes/driver/Stop.tsx`, `web/src/routes/driver/Proof.tsx`, `web/src/offline/useRoute.ts`, `web/src/offline/types.ts`, `web/src/routes/driver/Stop.test.tsx`
- Modify: `web/src/router.tsx`

**Interfaces:**
- Produces:
  - `offline/types.ts`: `type DriverRoute = Schemas["DriverRoute"]`, `type DriverStop = Schemas["DriverStop"]`, `type DriverAction = Schemas["DriverAction"]`, `type QueueItem = { clientId: string; stopId: number; body: DriverAction; status: "pending" | "stuck"; attempts: number; error?: string; createdAt: number }`.
  - `offline/useRoute.ts` (online-only in this task; Task 20 swaps the internals without changing this signature): `useRoute(): { route: DriverRoute | null; loading: boolean; error: ApiError | null; reload: () => void; pending: Record<number, number> /* stopId → queued count */; stuck: QueueItem[]; act: (stopId: number, body: DriverAction) => Promise<ApiError | null>; complete: (routeId: number) => Promise<ApiError | null>; online: boolean }`.
  - `Proof.tsx`: `<ProofCapture onDone(proof: Schemas["Proof"]) onCancel>` with three tabs: signature (signature_pad on a canvas, exported as PNG data URL, base64 stripped of the prefix), photo (`<input type="file" accept="image/*" capture="environment">`, downscaled through a canvas to a JPEG under 300 KB by lowering quality then dimensions), name (text).
  - `newClientId(): string` using `crypto.randomUUID()`.

- [ ] **Step 1: Write the failing stop test**

`web/src/routes/driver/Stop.test.tsx`:
```tsx
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import Stop from "./Stop";

const route = {
  route: { id: 2, routeDate: "2026-09-10", driverUserId: 9, driverName: "Sam", truckLabel: "", status: "out", stops: [] },
  stops: [{
    stop: { id: 11, routeId: 2, orderId: 3, sequence: 1, status: "pending", hasProofImage: false, skipReason: "", driverNote: "", customerName: "Blue Plate", deliveryAddress: "1 Main St", phone: "555-0101", contactName: "Marcy", deliveryNotes: "Back door", orderStatus: "scheduled", needsReview: false, lineCount: 1 },
    order: {
      id: 3, customer: { id: 1, name: "Blue Plate", billingAddress: "", deliveryAddress: "1 Main St", contactName: "Marcy", phone: "555-0101", email: "", deliveryNotes: "Back door", deliveryDays: [], active: true },
      requestedDeliveryDate: "2026-09-10", status: "scheduled", notes: "", needsReview: false, totalCents: 70982, createdAt: "2026-09-10T15:00:00Z",
      lines: [{ id: 8, productId: 2, sku: "BRIS", productName: "Brisket", sellUnit: "case", catchWeight: true, orderedQty: 200, unitPriceCents: 599, priceOverridden: false, estWeight: 12000, shippedWeight: 11850, shortageNote: "", amountCents: 70982, amountSource: "shipped" }],
    },
  }],
};

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

describe("Stop", () => {
  afterEach(() => vi.restoreAllMocks());
  it("shows the stop, lines with shipped weight, and posts a skip with a reason", async () => {
    const calls: { url: string; body: unknown }[] = [];
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      const url = String(input);
      if (init?.method === "POST") {
        calls.push({ url, body: JSON.parse(String(init.body)) });
        return jsonResponse({ applied: true, stop: { ...route.stops[0], stop: { ...route.stops[0].stop, status: "skipped", skipReason: "closed" } } });
      }
      return jsonResponse(route);
    });
    render(<MemoryRouter initialEntries={["/driver/stops/11"]}><Routes><Route path="/driver/stops/:id" element={<Stop />} /></Routes></MemoryRouter>);
    expect(await screen.findByText("Blue Plate")).toBeInTheDocument();
    expect(screen.getByText(/118\.50 lb/)).toBeInTheDocument();
    expect(screen.getByText("Back door")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /skip/i }));
    await userEvent.type(screen.getByLabelText(/reason/i), "closed");
    await userEvent.click(screen.getByRole("button", { name: /confirm skip/i }));
    expect(calls).toHaveLength(1);
    expect(calls[0].url).toContain("/api/driver/stops/11/actions");
    const body = calls[0].body as { type: string; skipReason: string; clientId: string };
    expect(body.type).toBe("skip");
    expect(body.skipReason).toBe("closed");
    expect(body.clientId).toMatch(/^[0-9a-f-]{36}$/);
  });
});
```

Run: `cd ~/deepcutsCRM/web && npx vitest run src/routes/driver/Stop 2>&1 | tail -3` → fails.

- [ ] **Step 2: Types and the online useRoute**

`web/src/offline/types.ts`:
```ts
import type { Schemas } from "../api/client";

export type DriverRoute = Schemas["DriverRoute"];
export type DriverStop = Schemas["DriverStop"];
export type DriverAction = Schemas["DriverAction"];
export type QueueItem = { clientId: string; stopId: number; body: DriverAction; status: "pending" | "stuck"; attempts: number; error?: string; createdAt: number };

export function newClientId(): string {
  return crypto.randomUUID();
}
```

`web/src/offline/useRoute.ts` (online-only version):
```ts
import { useCallback, useEffect, useState } from "react";
import { api, errorOf, type ApiError } from "../api/client";
import type { DriverAction, DriverRoute, QueueItem } from "./types";

export function useRoute() {
  const [route, setRoute] = useState<DriverRoute | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ApiError | null>(null);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    api.GET("/api/driver/route").then((r) => {
      if (!alive) return;
      const e = errorOf(r);
      setError(e && e.code !== "not_found" ? e : null);
      setRoute(r.data ?? null);
    }).finally(() => alive && setLoading(false));
    return () => { alive = false; };
  }, [tick]);

  const reload = useCallback(() => setTick((t) => t + 1), []);

  const act = useCallback(async (stopId: number, body: DriverAction) => {
    const res = await api.POST("/api/driver/stops/{stopId}/actions", { params: { path: { stopId } }, body });
    const e = errorOf(res);
    if (!e && res.data) {
      setRoute((r) => r && { ...r, stops: r.stops.map((s) => (s.stop.id === stopId ? res.data!.stop : s)) });
    }
    return e;
  }, []);

  const complete = useCallback(async (routeId: number) => {
    const res = await api.POST("/api/driver/routes/{routeId}/complete", { params: { path: { routeId } } });
    const e = errorOf(res);
    if (res.data) setRoute(res.data);
    return e;
  }, []);

  const pending: Record<number, number> = {};
  const stuck: QueueItem[] = [];
  return { route, loading, error, reload, pending, stuck, act, complete, online: true };
}
```

- [ ] **Step 3: Driver login and route list**

`web/src/routes/driver/Login.tsx`:
```tsx
import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router";
import { api, errorOf } from "../../api/client";
import { useSession } from "../../auth/Session";
import { useApi } from "../../components/useApi";

export default function DriverLogin() {
  const { setUser } = useSession();
  const nav = useNavigate();
  const drivers = useApi(() => api.GET("/api/driver/drivers"), []);
  const [userId, setUserId] = useState(0);
  const [pin, setPin] = useState("");
  const [error, setError] = useState<string | null>(null);

  async function submit(e: FormEvent) {
    e.preventDefault();
    const res = await api.POST("/api/driver/login", { body: { userId, pin } });
    const err = errorOf(res);
    if (err) return setError(err.code === "rate_limited" ? err.message : "Wrong PIN");
    setUser(res.data!);
    nav("/driver/route", { replace: true });
  }

  return (
    <main className="narrow driver">
      <h1>Deep Cuts — Driver</h1>
      <form onSubmit={submit} className="stack">
        <label>Who are you?
          <select value={userId} onChange={(e) => setUserId(Number(e.target.value))} required>
            <option value={0}>— choose —</option>
            {drivers.data?.map((d) => <option key={d.id} value={d.id}>{d.displayName}</option>)}
          </select>
        </label>
        <label>PIN<input type="password" inputMode="numeric" pattern="[0-9]{6}" maxLength={6} value={pin} onChange={(e) => setPin(e.target.value)} autoComplete="off" required /></label>
        {error && <p className="error" role="alert">{error}</p>}
        <button disabled={!userId || pin.length !== 6}>Start</button>
      </form>
    </main>
  );
}
```

`web/src/routes/driver/Route.tsx`:
```tsx
import { Link } from "react-router";
import { ErrorBanner } from "../../components/Field";
import { StatusBadge } from "../../components/StatusBadge";
import { useRoute } from "../../offline/useRoute";

export default function DriverRoute() {
  const { route, loading, error, reload, pending, stuck, complete, online } = useRoute();
  if (loading && !route) return <p className="muted">Loading…</p>;
  if (error) return <><ErrorBanner error={error} /><button onClick={reload}>Retry</button></>;
  if (!route) return <><h1>No route today</h1><p className="muted">Nothing is assigned to you for today.</p><button className="secondary" onClick={reload}>Refresh</button></>;
  const done = route.stops.every((s) => s.stop.status !== "pending");
  return (
    <>
      <div className="row">
        <h1>Today · {route.route.truckLabel || "Route"} <StatusBadge status={route.route.status} /></h1>
        <span className="spacer" />
        {!online && <span className="badge warn">offline</span>}
        <button className="secondary" onClick={reload}>Refresh</button>
      </div>
      {stuck.length > 0 && <p className="error">{stuck.length} update(s) could not be sent: {stuck[0].error}</p>}
      {route.stops.map((s) => (
        <Link key={s.stop.id} to={`/driver/stops/${s.stop.id}`} className="stop card">
          <h3>{s.stop.sequence}. {s.stop.customerName} <StatusBadge status={s.stop.status} /> {pending[s.stop.id] > 0 && <span className="badge pending">pending sync</span>}</h3>
          <div className="muted">{s.stop.deliveryAddress}</div>
          <div className="muted">{s.order.lines.length} lines{s.stop.deliveryNotes ? ` · ${s.stop.deliveryNotes}` : ""}</div>
        </Link>
      ))}
      {route.route.status === "out" && (
        <div className="actions">
          <button disabled={!done} onClick={() => complete(route.route.id)}>{done ? "Finish route" : "Finish route (stops remaining)"}</button>
        </div>
      )}
    </>
  );
}
```

- [ ] **Step 4: Proof capture**

`web/src/routes/driver/Proof.tsx`:
```tsx
import { useEffect, useRef, useState } from "react";
import SignaturePad from "signature_pad";
import type { Schemas } from "../../api/client";

type Proof = Schemas["Proof"];
const MAX = 300 * 1024;

function stripDataUrl(u: string): string {
  return u.slice(u.indexOf(",") + 1);
}

/** Downscale/recompress an image file to a JPEG under 300 KB. */
export async function shrinkImage(file: File): Promise<string> {
  const bmp = await createImageBitmap(file);
  let w = bmp.width, h = bmp.height;
  for (let pass = 0; pass < 8; pass++) {
    const c = document.createElement("canvas");
    c.width = w; c.height = h;
    c.getContext("2d")!.drawImage(bmp, 0, 0, w, h);
    for (const q of [0.8, 0.6, 0.4]) {
      const url = c.toDataURL("image/jpeg", q);
      if (stripDataUrl(url).length * 3 / 4 < MAX) return stripDataUrl(url);
    }
    w = Math.round(w * 0.7); h = Math.round(h * 0.7);
  }
  throw new Error("photo could not be reduced under 300 KB");
}

export function ProofCapture({ onDone, onCancel }: { onDone: (p: Proof) => void; onCancel: () => void }) {
  const [tab, setTab] = useState<"signature" | "photo" | "name">("signature");
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const padRef = useRef<SignaturePad | null>(null);
  const [name, setName] = useState("");
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (tab !== "signature" || !canvasRef.current) return;
    const c = canvasRef.current;
    const ratio = Math.max(window.devicePixelRatio || 1, 1);
    c.width = c.offsetWidth * ratio; c.height = c.offsetHeight * ratio;
    c.getContext("2d")!.scale(ratio, ratio);
    padRef.current = new SignaturePad(c, { backgroundColor: "rgb(255,255,255)" });
    return () => padRef.current?.off();
  }, [tab]);

  function doneSignature() {
    const pad = padRef.current;
    if (!pad || pad.isEmpty()) return setErr("Please sign first");
    const data = stripDataUrl(pad.toDataURL("image/png"));
    if (data.length * 3 / 4 > MAX) return setErr("Signature too large, clear and try again");
    onDone({ type: "signature", dataBase64: data });
  }

  async function donePhoto(f: File | undefined) {
    if (!f) return;
    try { onDone({ type: "photo", dataBase64: await shrinkImage(f) }); } catch (e) { setErr((e as Error).message); }
  }

  return (
    <div className="card stack">
      <div className="row">
        {(["signature", "photo", "name"] as const).map((t) => <button key={t} type="button" className={tab === t ? "" : "secondary"} onClick={() => { setTab(t); setErr(null); }}>{t}</button>)}
      </div>
      {tab === "signature" && <>
        <canvas ref={canvasRef} className="sig" aria-label="signature area" />
        <div className="row"><button type="button" className="secondary" onClick={() => padRef.current?.clear()}>Clear</button><button type="button" onClick={doneSignature}>Use signature</button></div>
      </>}
      {tab === "photo" && <label>Take a photo of the delivery<input type="file" accept="image/*" capture="environment" onChange={(e) => donePhoto(e.target.files?.[0])} /></label>}
      {tab === "name" && <>
        <label>Received by<input value={name} onChange={(e) => setName(e.target.value)} autoFocus /></label>
        <button type="button" disabled={!name.trim()} onClick={() => onDone({ type: "name", name: name.trim() })}>Use name</button>
      </>}
      {err && <p className="error">{err}</p>}
      <button type="button" className="link" onClick={onCancel}>Cancel</button>
    </div>
  );
}
```

- [ ] **Step 5: Stop screen**

`web/src/routes/driver/Stop.tsx`:
```tsx
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router";
import type { ApiError, Schemas } from "../../api/client";
import { ErrorBanner } from "../../components/Field";
import { StatusBadge } from "../../components/StatusBadge";
import { hundredths, lb, qty, toHundredths } from "../../lib/format";
import { newClientId } from "../../offline/types";
import { useRoute } from "../../offline/useRoute";
import { ProofCapture } from "./Proof";

type Adj = { deliveredQty: string; deliveredWeight: string; shortageNote: string };

export default function Stop() {
  const id = Number(useParams().id);
  const nav = useNavigate();
  const { route, loading, act, pending } = useRoute();
  const [mode, setMode] = useState<"view" | "adjust" | "proof" | "skip">("view");
  const [adj, setAdj] = useState<Record<number, Adj>>({});
  const [note, setNote] = useState("");
  const [skipReason, setSkipReason] = useState("");
  const [err, setErr] = useState<ApiError | null>(null);
  const [busy, setBusy] = useState(false);

  const ds = route?.stops.find((s) => s.stop.id === id);
  if (loading && !route) return <p className="muted">Loading…</p>;
  if (!ds) return <><p><Link to="/driver/route">← Route</Link></p><p>Stop not found on today's route.</p></>;
  const { stop, order } = ds;

  function adjustments(): Schemas["LineAdjustment"][] | undefined {
    const out: Schemas["LineAdjustment"][] = [];
    for (const l of order.lines) {
      const a = adj[l.id];
      if (!a) continue;
      const item: Schemas["LineAdjustment"] = { lineId: l.id };
      if (a.deliveredQty !== "") { const n = toHundredths(a.deliveredQty); if (n == null) throw new Error(`${l.sku}: bad quantity`); item.deliveredQty = n; }
      if (a.deliveredWeight !== "") { const n = toHundredths(a.deliveredWeight); if (n == null) throw new Error(`${l.sku}: bad weight`); item.deliveredWeight = n; }
      if (a.shortageNote) item.shortageNote = a.shortageNote;
      if (item.deliveredQty != null || item.deliveredWeight != null || item.shortageNote) out.push(item);
    }
    return out.length ? out : undefined;
  }

  async function send(body: Omit<Schemas["DriverAction"], "clientId">) {
    setBusy(true);
    setErr(null);
    try {
      const e = await act(id, { clientId: newClientId(), ...body });
      setErr(e);
      if (!e) { setMode("view"); setAdj({}); if (body.type !== "adjust") nav("/driver/route"); }
    } catch (ex) {
      setErr({ code: "invalid", message: (ex as Error).message });
    } finally {
      setBusy(false);
    }
  }

  const setA = (lineId: number, k: keyof Adj, v: string) => setAdj({ ...adj, [lineId]: { deliveredQty: "", deliveredWeight: "", shortageNote: "", ...adj[lineId], [k]: v } });

  return (
    <>
      <p><Link to="/driver/route">← Route</Link></p>
      <h1>{stop.sequence}. {stop.customerName} <StatusBadge status={stop.status} /> {pending[id] > 0 && <span className="badge pending">pending sync</span>}</h1>
      <div className="card">
        <div><a href={`https://maps.google.com/?q=${encodeURIComponent(stop.deliveryAddress)}`}>{stop.deliveryAddress}</a></div>
        {stop.phone && <div><a href={`tel:${stop.phone}`}>{stop.contactName ? `${stop.contactName} · ` : ""}{stop.phone}</a></div>}
        {stop.deliveryNotes && <p><strong>{stop.deliveryNotes}</strong></p>}
        {order.notes && <p className="muted">Order note: {order.notes}</p>}
        {stop.skipReason && <p className="error">Skipped: {stop.skipReason}</p>}
      </div>
      <h2>On the truck</h2>
      <table>
        <tbody>
          {order.lines.map((l) => (
            <tr key={l.id}>
              <td><strong>{l.productName}</strong><br /><span className="muted">{l.sku}</span></td>
              <td>{qty(l.orderedQty, l.sellUnit)}{l.catchWeight && <div>{lb(l.shippedWeight ?? l.estWeight)}</div>}
                {l.deliveredQty != null && l.deliveredQty !== l.orderedQty && <div className="error">delivered {qty(l.deliveredQty, l.sellUnit)}</div>}
                {l.catchWeight && l.deliveredWeight != null && l.deliveredWeight !== l.shippedWeight && <div className="error">delivered {lb(l.deliveredWeight)}</div>}
                {l.shortageNote && <div className="muted">{l.shortageNote}</div>}
              </td>
              {mode === "adjust" && (
                <td className="stack">
                  <input inputMode="decimal" placeholder={`qty (${hundredths(l.orderedQty)})`} value={adj[l.id]?.deliveredQty ?? ""} onChange={(e) => setA(l.id, "deliveredQty", e.target.value)} />
                  {l.catchWeight && <input inputMode="decimal" placeholder={`lb (${hundredths(l.shippedWeight ?? l.estWeight ?? 0)})`} value={adj[l.id]?.deliveredWeight ?? ""} onChange={(e) => setA(l.id, "deliveredWeight", e.target.value)} />}
                  <input placeholder="note (rejected, short…)" value={adj[l.id]?.shortageNote ?? ""} onChange={(e) => setA(l.id, "shortageNote", e.target.value)} />
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
      <ErrorBanner error={err} />
      {mode === "view" && stop.status === "pending" && (
        <div className="actions">
          <button onClick={() => setMode("proof")} disabled={busy}>Delivered</button>
          <button className="secondary" onClick={() => setMode("adjust")} disabled={busy}>Adjust quantities</button>
          <button className="secondary" onClick={() => setMode("skip")} disabled={busy}>Skip stop</button>
        </div>
      )}
      {mode === "view" && stop.status === "delivered" && (
        <div className="actions"><button className="secondary" onClick={() => setMode("adjust")} disabled={busy}>Adjust quantities</button></div>
      )}
      {mode === "adjust" && (
        <div className="actions">
          <input placeholder="note for the office" value={note} onChange={(e) => setNote(e.target.value)} />
          <button onClick={() => { try { send({ type: "adjust", lines: adjustments(), note }); } catch (ex) { setErr({ code: "invalid", message: (ex as Error).message }); } }} disabled={busy}>Save adjustments</button>
          <button className="link" onClick={() => setMode("view")}>Cancel</button>
        </div>
      )}
      {mode === "proof" && (
        <>
          <h2>Proof of delivery</h2>
          <input placeholder="note for the office (optional)" value={note} onChange={(e) => setNote(e.target.value)} />
          <ProofCapture onCancel={() => setMode("view")} onDone={(proof) => { try { send({ type: "deliver", proof, lines: adjustments(), note }); } catch (ex) { setErr({ code: "invalid", message: (ex as Error).message }); } }} />
        </>
      )}
      {mode === "skip" && (
        <div className="actions card">
          <label>Reason<input value={skipReason} onChange={(e) => setSkipReason(e.target.value)} autoFocus /></label>
          <button className="danger" disabled={!skipReason.trim() || busy} onClick={() => send({ type: "skip", skipReason: skipReason.trim(), note })}>Confirm skip</button>
          <button className="link" onClick={() => setMode("view")}>Cancel</button>
        </div>
      )}
    </>
  );
}
```

Replace the driver `Todo` routes in `router.tsx` with `DriverLogin`, `DriverRoute`, `Stop`.

- [ ] **Step 6: Run tests, typecheck, verify on a phone-sized viewport, commit**

Run: `cd ~/deepcutsCRM/web && npx vitest run 2>&1 | tail -3 && npx tsc -b && npm run build 2>&1 | tail -2`

Verify against `make demo` in a 390 px wide viewport: log in as Sam / 111111, see 4 stops (one delivered), open a pending stop, adjust a weight, deliver with a drawn signature, confirm the office day view shows the proof link and review badge; skip another stop and confirm it lands back in the office's unscheduled list.

```bash
cd ~/deepcutsCRM && git add -A && git commit -q -m "Add driver login, route, stop and proof capture screens" && git push
```

---

### Task 20: Driver offline: IndexedDB cache, sync queue, PWA

**Files:**
- Create: `web/src/offline/db.ts`, `web/src/offline/queue.ts`, `web/src/offline/queue.test.ts`, `web/src/offline/register.ts`
- Modify: `web/src/offline/useRoute.ts` (offline-aware internals, same signature), `web/src/main.tsx` (register SW)

**Interfaces:**
- Produces:
  - `db.ts`: `openDB()` with stores `route` (key `"current"`, value `{ route: DriverRoute; savedAt: number }`) and `queue` (keyPath `clientId`, index `byStop`); helpers `saveRoute`, `loadRoute`, `putQueue`, `listQueue`, `deleteQueue`.
  - `queue.ts`: `enqueue(stopId, body): Promise<void>`, `flush(): Promise<void>` (serial, oldest first; stops at the first retryable failure), `subscribe(fn: (items: QueueItem[]) => void): () => void`, `startAutoFlush()` (on `online` event and every 15 s while items remain), `applyLocally(route, item): DriverRoute` (optimistic: `deliver` → stop `delivered`, `skip` → `skipped`, `adjust` → merges line values).
  - Retry policy (from the spec): HTTP 200 → done; 409 `client_id_reused` → stuck; other 409 → stuck with message; 5xx or network error → retry with backoff `min(2^attempts, 60)` seconds; 401 → stuck "please sign in again"; any other 4xx → stuck.
  - `register.ts`: `registerSW()` from `virtual:pwa-register`, only when `location.pathname.startsWith("/driver")`.

- [ ] **Step 1: Write the failing queue test**

`web/src/offline/queue.test.ts`:
```ts
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { deleteDB } from "idb";
import { applyLocally, enqueue, flush, listQueue } from "./queue";
import type { DriverRoute } from "./types";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
}

const stopBody = { clientId: "11111111-1111-4111-8111-111111111111", type: "deliver" as const, proof: { type: "name" as const, name: "Pat" } };

describe("sync queue", () => {
  beforeEach(async () => { await deleteDB("deepcuts-driver"); });
  afterEach(() => vi.restoreAllMocks());

  it("retries after a network failure, then completes", async () => {
    const f = vi.spyOn(globalThis, "fetch");
    f.mockRejectedValueOnce(new TypeError("Failed to fetch"));
    await enqueue(11, stopBody);
    await flush();
    let items = await listQueue();
    expect(items).toHaveLength(1);
    expect(items[0].attempts).toBe(1);
    expect(items[0].status).toBe("pending");
    f.mockResolvedValueOnce(jsonResponse({ applied: true, stop: {} }));
    await flush(true);
    items = await listQueue();
    expect(items).toHaveLength(0);
    expect(f).toHaveBeenCalledTimes(2);
  });

  it("treats a replay (applied:false) as done and a 422 as stuck", async () => {
    const f = vi.spyOn(globalThis, "fetch");
    f.mockResolvedValueOnce(jsonResponse({ applied: false, stop: {} }));
    await enqueue(11, stopBody);
    await flush();
    expect(await listQueue()).toHaveLength(0);
    f.mockResolvedValueOnce(jsonResponse({ code: "invalid", message: "proof required" }, 422));
    await enqueue(12, { ...stopBody, clientId: "22222222-2222-4222-8222-222222222222" });
    await flush();
    const items = await listQueue();
    expect(items[0].status).toBe("stuck");
    expect(items[0].error).toBe("proof required");
  });

  it("applies deliver and adjust optimistically", () => {
    const route = { route: { id: 1, routeDate: "", driverUserId: 1, driverName: "", truckLabel: "", status: "out", stops: [] }, stops: [{
      stop: { id: 11, routeId: 1, orderId: 3, sequence: 1, status: "pending", hasProofImage: false, skipReason: "", driverNote: "", customerName: "", deliveryAddress: "", phone: "", contactName: "", deliveryNotes: "", orderStatus: "scheduled", needsReview: false, lineCount: 1 },
      order: { id: 3, customer: { id: 1, name: "", billingAddress: "", deliveryAddress: "", contactName: "", phone: "", email: "", deliveryNotes: "", deliveryDays: [], active: true }, requestedDeliveryDate: "", status: "scheduled", notes: "", needsReview: false, totalCents: 0, createdAt: "", lines: [{ id: 8, productId: 1, sku: "", productName: "", sellUnit: "case", catchWeight: true, orderedQty: 200, unitPriceCents: 1, priceOverridden: false, shortageNote: "", amountCents: 0, amountSource: "shipped" }] },
    }] } as unknown as DriverRoute;
    const adjusted = applyLocally(route, { clientId: "x", stopId: 11, status: "pending", attempts: 0, createdAt: 0, body: { clientId: "x", type: "adjust", lines: [{ lineId: 8, deliveredWeight: 5000, shortageNote: "short" }] } });
    expect(adjusted.stops[0].order.lines[0].deliveredWeight).toBe(5000);
    expect(adjusted.stops[0].order.lines[0].shortageNote).toBe("short");
    const delivered = applyLocally(adjusted, { clientId: "y", stopId: 11, status: "pending", attempts: 0, createdAt: 0, body: { clientId: "y", type: "deliver", proof: { type: "name", name: "Pat" } } });
    expect(delivered.stops[0].stop.status).toBe("delivered");
    expect(delivered.stops[0].stop.proofName).toBe("Pat");
  });
});
```

Run: `cd ~/deepcutsCRM/web && npx vitest run src/offline 2>&1 | tail -3` → fails.

- [ ] **Step 2: Implement db.ts and queue.ts**

`web/src/offline/db.ts`:
```ts
import { openDB as open, type DBSchema, type IDBPDatabase } from "idb";
import type { DriverRoute, QueueItem } from "./types";

interface Schema extends DBSchema {
  route: { key: string; value: { route: DriverRoute; savedAt: number } };
  queue: { key: string; value: QueueItem; indexes: { byStop: number; byCreated: number } };
}

let dbp: Promise<IDBPDatabase<Schema>> | null = null;

export function openDB() {
  if (!dbp) {
    dbp = open<Schema>("deepcuts-driver", 1, {
      upgrade(db) {
        db.createObjectStore("route");
        const q = db.createObjectStore("queue", { keyPath: "clientId" });
        q.createIndex("byStop", "stopId");
        q.createIndex("byCreated", "createdAt");
      },
    });
  }
  return dbp;
}

export async function saveRoute(route: DriverRoute) { (await openDB()).put("route", { route, savedAt: Date.now() }, "current"); }
export async function loadRoute(): Promise<DriverRoute | null> { return (await (await openDB()).get("route", "current"))?.route ?? null; }
export async function putQueue(item: QueueItem) { await (await openDB()).put("queue", item); }
export async function listQueue(): Promise<QueueItem[]> { return (await openDB()).getAllFromIndex("queue", "byCreated"); }
export async function deleteQueue(clientId: string) { await (await openDB()).delete("queue", clientId); }
```

`web/src/offline/queue.ts`:
```ts
import { api, errorOf } from "../api/client";
import { deleteQueue, listQueue, putQueue } from "./db";
import type { DriverAction, DriverRoute, QueueItem } from "./types";

export { listQueue } from "./db";

type Listener = (items: QueueItem[]) => void;
const listeners = new Set<Listener>();
let flushing = false;
const nextTry = new Map<string, number>();

export function subscribe(fn: Listener): () => void {
  listeners.add(fn);
  listQueue().then(fn);
  return () => { listeners.delete(fn); };
}

async function notify() {
  const items = await listQueue();
  listeners.forEach((l) => l(items));
}

export async function enqueue(stopId: number, body: DriverAction): Promise<void> {
  await putQueue({ clientId: body.clientId, stopId, body, status: "pending", attempts: 0, createdAt: Date.now() });
  await notify();
}

/** Sends pending items oldest-first. Stops at the first retryable failure. `force` ignores backoff timers. */
export async function flush(force = false): Promise<void> {
  if (flushing) return;
  flushing = true;
  try {
    for (const item of await listQueue()) {
      if (item.status !== "pending") continue;
      if (!force && (nextTry.get(item.clientId) ?? 0) > Date.now()) continue;
      let res: { data?: unknown; error?: unknown; response: Response };
      try {
        res = await api.POST("/api/driver/stops/{stopId}/actions", { params: { path: { stopId: item.stopId } }, body: item.body });
      } catch {
        await retryLater(item, "network");
        break;
      }
      const err = errorOf(res);
      if (!err) { await deleteQueue(item.clientId); nextTry.delete(item.clientId); continue; }
      if (res.response.status >= 500) { await retryLater(item, err.message); break; }
      const msg = res.response.status === 401 ? "please sign in again" : err.message;
      await putQueue({ ...item, status: "stuck", attempts: item.attempts + 1, error: msg });
    }
  } finally {
    flushing = false;
    await notify();
  }
}

async function retryLater(item: QueueItem, reason: string) {
  const attempts = item.attempts + 1;
  nextTry.set(item.clientId, Date.now() + Math.min(2 ** attempts, 60) * 1000);
  await putQueue({ ...item, attempts, error: reason });
}

let started = false;
export function startAutoFlush() {
  if (started || typeof window === "undefined") return;
  started = true;
  window.addEventListener("online", () => flush(true));
  setInterval(() => { listQueue().then((items) => items.some((i) => i.status === "pending") && flush()); }, 15000);
}

/** Optimistically applies a queued action to the cached route. */
export function applyLocally(route: DriverRoute, item: QueueItem): DriverRoute {
  return {
    ...route,
    stops: route.stops.map((s) => {
      if (s.stop.id !== item.stopId) return s;
      const b = item.body;
      let stop = { ...s.stop };
      let lines = s.order.lines;
      if (b.lines) {
        lines = lines.map((l) => {
          const a = b.lines!.find((x) => x.lineId === l.id);
          return a ? { ...l, deliveredQty: a.deliveredQty ?? l.deliveredQty, deliveredWeight: a.deliveredWeight ?? l.deliveredWeight, shortageNote: a.shortageNote ?? l.shortageNote } : l;
        });
      }
      if (b.note) stop.driverNote = b.note;
      if (b.type === "deliver") {
        stop = { ...stop, status: "delivered", proofType: b.proof?.type, proofName: b.proof?.type === "name" ? b.proof.name : undefined, hasProofImage: b.proof?.type !== "name" };
      }
      if (b.type === "skip") stop = { ...stop, status: "skipped", skipReason: b.skipReason ?? "" };
      return { ...s, stop, order: { ...s.order, lines } };
    }),
  };
}
```

Run: `cd ~/deepcutsCRM/web && npx vitest run src/offline 2>&1 | tail -3` → passes.

- [ ] **Step 3: Offline-aware useRoute (same signature)**

Replace `web/src/offline/useRoute.ts`:
```ts
import { useCallback, useEffect, useMemo, useState } from "react";
import { api, errorOf, type ApiError } from "../api/client";
import { loadRoute, saveRoute } from "./db";
import { applyLocally, enqueue, flush, startAutoFlush, subscribe } from "./queue";
import type { DriverAction, DriverRoute, QueueItem } from "./types";

export function useRoute() {
  const [server, setServer] = useState<DriverRoute | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ApiError | null>(null);
  const [queue, setQueue] = useState<QueueItem[]>([]);
  const [online, setOnline] = useState(typeof navigator === "undefined" ? true : navigator.onLine);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    startAutoFlush();
    const on = () => setOnline(true), off = () => setOnline(false);
    window.addEventListener("online", on); window.addEventListener("offline", off);
    const unsub = subscribe(setQueue);
    return () => { unsub(); window.removeEventListener("online", on); window.removeEventListener("offline", off); };
  }, []);

  useEffect(() => {
    let alive = true;
    setLoading(true);
    (async () => {
      await flush(true);
      const cached = await loadRoute();
      if (cached && alive) setServer(cached);
      try {
        const r = await api.GET("/api/driver/route");
        if (!alive) return;
        const e = errorOf(r);
        if (r.data) { setServer(r.data); await saveRoute(r.data); setError(null); }
        else if (e?.code === "not_found") { setServer(null); setError(null); }
        else if (!cached) setError(e);
      } catch {
        if (!cached && alive) setError({ code: "offline", message: "No connection and no cached route" });
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => { alive = false; };
  }, [tick]);

  const route = useMemo(() => {
    if (!server) return null;
    return queue.filter((q) => q.status === "pending").reduce((r, q) => applyLocally(r, q), server);
  }, [server, queue]);

  const pending = useMemo(() => {
    const m: Record<number, number> = {};
    for (const q of queue) if (q.status === "pending") m[q.stopId] = (m[q.stopId] ?? 0) + 1;
    return m;
  }, [queue]);

  const reload = useCallback(() => setTick((t) => t + 1), []);

  const act = useCallback(async (stopId: number, body: DriverAction): Promise<ApiError | null> => {
    await enqueue(stopId, body);
    await flush(true);
    const items = await new Promise<QueueItem[]>((res) => { const un = subscribe((i) => { un(); res(i); }); });
    const mine = items.find((i) => i.clientId === body.clientId);
    if (mine?.status === "stuck") return { code: "stuck", message: mine.error ?? "could not send" };
    if (!mine) {
      // delivered to the server: refresh the cached copy in the background
      api.GET("/api/driver/route").then((r) => { if (r.data) { setServer(r.data); saveRoute(r.data); } });
    }
    return null;
  }, []);

  const complete = useCallback(async (routeId: number) => {
    const res = await api.POST("/api/driver/routes/{routeId}/complete", { params: { path: { routeId } } });
    const e = errorOf(res);
    if (res.data) { setServer(res.data); await saveRoute(res.data); }
    return e;
  }, []);

  return { route, loading, error, reload, pending, stuck: queue.filter((q) => q.status === "stuck"), act, complete, online };
}
```

- [ ] **Step 4: Service worker registration**

`web/src/offline/register.ts`:
```ts
import { registerSW } from "virtual:pwa-register";

export function registerDriverSW() {
  if (!location.pathname.startsWith("/driver")) return;
  registerSW({ immediate: true });
}
```

Add `/// <reference types="vite-plugin-pwa/client" />` to `web/src/vite-env.d.ts`, and in `main.tsx` call `registerDriverSW()` before render.

- [ ] **Step 5: Run everything, verify offline on a device-sized viewport, commit**

Run: `cd ~/deepcutsCRM/web && npx vitest run 2>&1 | tail -3 && npx tsc -b && npm run build 2>&1 | tail -3`
Expected: tests pass; build lists `sw.js`, `workbox-*.js`, `manifest.webmanifest`.

Verify against `make demo` in Chrome: log in as a driver, load the route, open DevTools → Network → Offline, open a pending stop, deliver with a received-by name: the stop shows "pending sync"; turn the network back on: the badge clears within 15 s and the office day view shows the stop delivered. Confirm `Application → Manifest` shows "Deep Cuts Driver" installable.

```bash
cd ~/deepcutsCRM && git add -A && git commit -q -m "Add driver offline cache, idempotent sync queue and PWA" && git push
```

---

### Task 21: End-to-end script, phone demo over TLS, CI, docs

**Files:**
- Create: `scripts/e2e.sh`, `scripts/demo-tls.sh`, `.github/workflows/ci.yml`
- Modify: `README.md`, `Makefile` (targets already reference the scripts)

**Interfaces:**
- `scripts/e2e.sh`: builds the binary, seeds a temp database, starts the server on a free port, walks one order from draft to finalized through the API with `curl`, and exits non-zero on any unexpected status. No test framework, plain bash with `set -euo pipefail`.
- `scripts/demo-tls.sh`: requires `mkcert` (`brew install mkcert`), creates `certs/` for `localhost`, `127.0.0.1` and the machine's LAN IP, builds, seeds `data/demo.sqlite` if absent, and runs `serve` with TLS on `:8443`, printing the `https://<lan-ip>:8443/driver` URL and the one-time phone steps.
- CI: Go tests, web tests, `make check-generated`, build.

- [ ] **Step 1: e2e script**

`scripts/e2e.sh`:
```bash
#!/usr/bin/env bash
# Builds the binary, seeds demo data into a temp dir, and walks an order through the API.
set -euo pipefail
export PATH=/opt/homebrew/bin:$PATH
cd "$(dirname "$0")/.."

T=$(mktemp -d)
PORT=${PORT:-8098}
BIN=bin/deepcuts
DB=$T/e2e.sqlite
JAR=$T/office.jar
DJAR=$T/driver.jar
API=http://localhost:$PORT/api
TODAY=$(date +%F)

make -s build >/dev/null
DEEPCUTS_DB_PATH=$DB DEEPCUTS_DATA_DIR=$T $BIN seed demo >/dev/null
DEEPCUTS_DB_PATH=$DB DEEPCUTS_DATA_DIR=$T DEEPCUTS_ADDR=:$PORT $BIN serve >$T/server.log 2>&1 &
SERVER=$!
trap 'kill $SERVER 2>/dev/null; rm -rf "$T"' EXIT
for i in $(seq 1 50); do curl -s -o /dev/null "$API/driver/drivers" && break; sleep 0.1; done

# call METHOD PATH [JSON] [JAR] -> prints body; sets $STATUS
call() {
  local m=$1 p=$2 body=${3:-} jar=${4:-$JAR}
  local out
  if [ -n "$body" ]; then
    out=$(curl -s -w '\n%{http_code}' -b "$jar" -c "$jar" -X "$m" "$API$p" -H 'Content-Type: application/json' -d "$body")
  else
    out=$(curl -s -w '\n%{http_code}' -b "$jar" -c "$jar" -X "$m" "$API$p")
  fi
  STATUS=$(echo "$out" | tail -n1)
  echo "$out" | sed '$d'
}
expect() { if [ "$STATUS" != "$1" ]; then echo "FAIL: $2 → HTTP $STATUS: $3" >&2; exit 1; fi; echo "ok: $2"; }
jq_() { python3 -c "import sys,json; d=json.load(sys.stdin); print($1)"; }

B=$(call POST /office/login '{"email":"office@demo.local","password":"demo1234"}'); expect 200 "office login" "$B"
CUST=$(call GET /office/customers | jq_ 'd[0]["id"]')
PROD=$(call GET /office/products | jq_ '[p for p in d if p["catchWeight"]][0]["id"]')
B=$(call POST /office/orders "{\"customerId\":$CUST,\"requestedDeliveryDate\":\"$TODAY\"}"); expect 201 "create order" "$B"
ORDER=$(echo "$B" | jq_ 'd["id"]')
B=$(call POST /office/orders/$ORDER/lines "{\"productId\":$PROD,\"orderedQty\":200}"); expect 200 "add line" "$B"
LINE=$(echo "$B" | jq_ 'd["lines"][0]["id"]')
[ "$(echo "$B" | jq_ 'd["lines"][0]["amountSource"]')" = "estimated" ] || { echo "FAIL: amount source"; exit 1; }
B=$(call POST /office/orders/$ORDER/confirm); expect 200 "confirm" "$B"
B=$(call PATCH /office/orders/$ORDER/lines/$LINE '{"shippedWeight":11850}'); expect 200 "shipped weight" "$B"
DRIVER=$(call GET /office/drivers | jq_ '[x for x in d if x["displayName"]=="Riley"][0]["id"]')
B=$(call POST /office/routes "{\"routeDate\":\"$TODAY\",\"driverUserId\":$DRIVER,\"truckLabel\":\"e2e\"}"); expect 201 "create route" "$B"
ROUTE=$(echo "$B" | jq_ 'd["id"]')
B=$(call POST /office/routes/$ROUTE/stops "{\"orderId\":$ORDER}"); expect 200 "add stop" "$B"
STOP=$(echo "$B" | jq_ 'd["stops"][0]["id"]')
B=$(call POST /office/routes/$ROUTE/out); expect 200 "route out" "$B"

B=$(call POST /driver/login "{\"userId\":$DRIVER,\"pin\":\"222222\"}" "$DJAR"); expect 200 "driver login" "$B"
B=$(call GET /driver/route "" "$DJAR"); expect 200 "driver route" "$B"
[ "$(echo "$B" | jq_ 'd["stops"][0]["stop"]["id"]')" = "$STOP" ] || { echo "FAIL: driver sees wrong stop"; exit 1; }
CID=$(python3 -c 'import uuid; print(uuid.uuid4())')
B=$(call POST /driver/stops/$STOP/actions "{\"clientId\":\"$CID\",\"type\":\"deliver\",\"proof\":{\"type\":\"name\",\"name\":\"Pat\"},\"lines\":[{\"lineId\":$LINE,\"deliveredWeight\":6000,\"shortageNote\":\"one case short\"}]}" "$DJAR"); expect 200 "deliver" "$B"
[ "$(echo "$B" | jq_ 'd["applied"]')" = "True" ] || { echo "FAIL: not applied"; exit 1; }
B=$(call POST /driver/stops/$STOP/actions "{\"clientId\":\"$CID\",\"type\":\"deliver\",\"proof\":{\"type\":\"name\",\"name\":\"Pat\"}}" "$DJAR"); expect 200 "replay" "$B"
[ "$(echo "$B" | jq_ 'd["applied"]')" = "False" ] || { echo "FAIL: replay applied twice"; exit 1; }
B=$(call POST /driver/routes/$ROUTE/complete "" "$DJAR"); expect 200 "complete route" "$B"

B=$(call GET /office/orders/$ORDER); expect 200 "order after delivery" "$B"
[ "$(echo "$B" | jq_ 'd["status"]')" = "delivered" ] || { echo "FAIL: status $(echo "$B" | jq_ 'd["status"]')"; exit 1; }
[ "$(echo "$B" | jq_ 'd["needsReview"]')" = "True" ] || { echo "FAIL: needsReview"; exit 1; }
[ "$(echo "$B" | jq_ 'd["lines"][0]["amountSource"]')" = "delivered" ] || { echo "FAIL: amount source after delivery"; exit 1; }
B=$(call POST /office/orders/$ORDER/finalize); expect 200 "finalize" "$B"
[ "$(echo "$B" | jq_ 'd["status"]')" = "finalized" ] || { echo "FAIL: not finalized"; exit 1; }
echo "E2E PASSED"
```

`chmod +x scripts/e2e.sh`. Run: `cd ~/deepcutsCRM && make e2e 2>&1 | tail -5` → `E2E PASSED`.

- [ ] **Step 2: TLS demo script**

`scripts/demo-tls.sh`:
```bash
#!/usr/bin/env bash
# Serves the demo over HTTPS so a phone on the same Wi-Fi can install the driver PWA.
set -euo pipefail
export PATH=/opt/homebrew/bin:$PATH
cd "$(dirname "$0")/.."

command -v mkcert >/dev/null || { echo "mkcert is required: brew install mkcert && mkcert -install" >&2; exit 1; }
LAN=$(ipconfig getifaddr en0 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}')
mkdir -p certs
if [ ! -f certs/cert.pem ]; then
  mkcert -cert-file certs/cert.pem -key-file certs/key.pem localhost 127.0.0.1 ${LAN:-}
fi
make -s build
if [ ! -f data/demo.sqlite ]; then
  DEEPCUTS_DB_PATH=data/demo.sqlite bin/deepcuts seed demo
fi
cat <<EOF

Phone setup (one time): install the mkcert root CA on the phone so the certificate is trusted.
  Root CA file: $(mkcert -CAROOT)/rootCA.pem  (AirDrop it, then Settings → General → VPN & Device Management → install, and enable full trust under Certificate Trust Settings on iOS)

Driver app: https://${LAN:-localhost}:8443/driver   (add to home screen to install)
Office:     https://${LAN:-localhost}:8443/office

EOF
DEEPCUTS_DB_PATH=data/demo.sqlite DEEPCUTS_ADDR=:8443 DEEPCUTS_TLS_CERT=certs/cert.pem DEEPCUTS_TLS_KEY=certs/key.pem bin/deepcuts serve
```

`chmod +x scripts/demo-tls.sh`. `mkcert` install is a one-time human step; if it is missing the script says so and exits 1. The coordinator installs it with `brew install mkcert && mkcert -install` (the `-install` step prompts for the macOS password: leave that to Chris if it prompts).

- [ ] **Step 3: CI workflow**

`.github/workflows/ci.yml`:
```yaml
name: ci
on:
  push:
    branches: [main]
  pull_request:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.27" }
      - uses: actions/setup-node@v4
        with: { node-version: "26", cache: npm, cache-dependency-path: web/package-lock.json }
      - run: cd web && npm ci --no-audit --no-fund
      - run: go tool sqlc generate && go tool oapi-codegen -config oapi-codegen.yaml api/openapi.yaml && (cd web && npm run generate)
      - run: git diff --exit-code -- internal/db/queries internal/api web/src/api/schema.d.ts
      - run: go vet ./... && go test ./...
      - run: cd web && npx vitest run && npx tsc -b && npm run build
      - run: go build -o bin/deepcuts ./cmd/deepcuts
      - run: ./scripts/e2e.sh
```
The Makefile's `export PATH := /opt/homebrew/bin:$(PATH)` is harmless on Linux. The scripts also prepend it; on Linux the directory simply does not exist.

- [ ] **Step 4: README**

Replace `README.md` with the final version:
```markdown
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
`DEEPCUTS_TIMEZONE` (America/New_York), `DEEPCUTS_TLS_CERT` / `DEEPCUTS_TLS_KEY`.

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
```

- [ ] **Step 5: Visual verification pass (coordinator, not a subagent)**

Per Chris's standing rule, mocks and green tests are not verification. With `make demo` running, screenshot and check:
- Desktop (1280 px): office login, day view for today, an order detail, customers list.
- Phone (390 px iframe or device emulation): driver login, route list, a stop, the proof signature pad.
Fix anything that looks broken (overflow, unreadable badges, buttons off-screen) before declaring done, then re-screenshot.

- [ ] **Step 6: Final commit**

```bash
cd ~/deepcutsCRM && make test 2>&1 | tail -4 && make e2e 2>&1 | tail -2 && git add -A && git commit -q -m "Add e2e script, TLS phone demo, CI workflow, README" && git push
```

---

## Not in this plan (follow-ons)

- **AWS deployment** (Terraform, Litestream, Caddy, SSM secrets, deploy workflow): separate plan, after a cost rundown.
- **QuickBooks Online live pull** for customers and items: needs an Intuit developer app; CSV import covers phase 1.
- **Phase 2** (finalize → QBO invoice), **phase 3** (lots), **phase 4** (portal), **phase 5** (inbound intake): separate specs.
