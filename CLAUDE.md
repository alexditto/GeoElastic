# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repository is

GeoElastic is a Go API that accepts an address and runs fuzzy searches against an Elasticsearch
index of business/geo records (address, geolocation, name, type, phone, square footage) to find
exact or probable matches. It's a learning project (first Go project, new to Elastic) for a
developer with a MySQL/Laravel background — see the note on Go vs. Laravel conventions below
before assuming framework-style structure that Go doesn't have.

Local Elasticsearch + Kibana are provisioned via Elastic's official `start-local` scaffold
(`elastic-start-local/`, see https://github.com/elastic/start-local); the Go application is a
separate, hand-built API that talks to that local stack.

## Commands

Application (run from repo root):

- `go build ./...` — build all packages.
- `go run ./cmd/api` — run the API server (reads config from `.env`, listens on `:8080` by
  default).
- `go run ./cmd/migrate` — create/update the Elasticsearch `businesses` index and alias from the
  mapping in `internal/store/mappings/`. Safe to re-run (idempotent).
- `go build -o /tmp/geoelastic-api ./cmd/api` — build a standalone binary, useful for a quick
  manual smoke test (`./that-binary & curl localhost:8080/health; kill %1`).
- `gofmt -l .` — list files needing formatting; `gofmt -w <file>` to fix.
- `go mod tidy` — sync `go.mod`/`go.sum` after adding/removing imports.
- No test suite yet.

Local Elastic Stack (run from `elastic-start-local/`):

- `./start.sh` — starts Elasticsearch + Kibana via `docker compose up --wait`. On first run past
  the trial expiry date it auto-converts the license to Basic via the ES API.
- `./stop.sh` — stops the containers (`docker compose stop`), preserving data volumes.
- `./uninstall.sh` — destructive: prompts for confirmation, then removes containers, volumes, and
  the compose/env/script files themselves, and optionally the Docker images. Do not run this
  without explicit user confirmation.

## Architecture

### Go application layout

Go has no framework and no MVC convention (no bundled ORM, no service container) — structure here
is hand-rolled, not enforced by tooling:

- `cmd/api/main.go` — entry point: loads config, constructs the Elasticsearch client, wires up
  routes (stdlib `net/http`, using Go 1.22+'s method+pattern mux syntax, e.g. `"GET /health"`),
  starts the server.
- `cmd/migrate/main.go` — entry point for `EnsureBusinessIndex` (see below); no generic migration
  runner exists, just this one explicit command.
- `internal/config` — reads settings from environment variables (`.env` loaded via
  `github.com/joho/godotenv`, since unlike Laravel, Go does not auto-load `.env` files).
- `internal/model` — plain structs (e.g. `Business`, `Address`, `OpeningHours`, `GeoPoint`).
  These are just data, not Active Record-style objects — there's no `business.Save()`;
  persistence logic lives in `store`. `Address` is embedded directly in `Business` rather than
  referenced by ID: Elasticsearch has no real joins, so a 1:1 relationship like this is
  denormalized into one document instead of split across a separate index.
- `internal/store` — hand-written code that talks to Elasticsearch via
  `github.com/elastic/go-elasticsearch/v9` (matched to the local server's major version, 9.x).
  This is the layer Eloquent's query builder would cover in Laravel, but there's no ES-flavored
  Go ORM, so queries are built and issued by hand here.
  - `internal/store/mappings/businesses_v1.json` — the mapping for the `businesses` index,
    `//go:embed`-ed into the binary. Elasticsearch mappings are largely immutable once created
    (most field type changes can't be applied in place), so the convention here is: a mapping
    change means authoring `businesses_v2.json`, reindexing into a new `businesses_v2` index, and
    swapping the `businesses` **alias** to point at it — never editing `businesses_v1` in place.
    Application code should always query the `businesses` alias, never a versioned index name
    directly.
  - `internal/store/index.go` — `EnsureBusinessIndex` creates the versioned index + alias if
    they don't already exist (idempotent; this is the closest thing this repo has to a Laravel
    migration).
- `internal/handler` — HTTP handlers (Laravel's "controllers," different name): parse the
  request, call into `store`/future service logic, write the response.

Fuzzy-match decision logic and the auth middleware don't exist yet; when added, auth should
follow an opaque-token pattern (hash-on-issue, hash-and-lookup, modeled on Laravel Sanctum's
personal access tokens) backed by an Elasticsearch index rather than JWTs — see rationale in
memory if unclear.

### Local Elastic Stack

`elastic-start-local/docker-compose.yml` defines three services:

- `elasticsearch` — single-node ES cluster, security enabled, HTTP (not HTTPS), exposed on
  `127.0.0.1:${ES_LOCAL_PORT}` (default 9200).
- `kibana_settings` — a one-shot init container that runs after ES is healthy to set the
  `kibana_system` user's password via the ES security API, then exits.
- `kibana` — depends on `kibana_settings` completing successfully; exposed on
  `127.0.0.1:${KIBANA_LOCAL_PORT}` (default 5601).

`elastic-start-local/.env` holds the Docker Compose config (versions, ports, container names,
ES/Kibana passwords, Kibana encryption key, ES API key). The Go app reads its own `.env` at the
repo root (gitignored) — currently `USERNAME`, `PASSWORD`, `API_KEY`, `SERVER_PORT`, `ES_URL` —
via `internal/config`. Both files contain live credentials for the local stack — do not print
their contents into commits, logs, or committed docs, and treat them as secrets even though the
stack is local-only.

The Go app connects to Elasticsearch at `http://localhost:9200` using `API_KEY` (see
`internal/store/elasticsearch.go`); Kibana is reachable separately at `http://localhost:5601` for
manual inspection.
