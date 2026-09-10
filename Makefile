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
	cd web && ([ -d node_modules ] || npm ci --no-audit --no-fund) && npm run build

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
