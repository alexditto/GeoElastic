# GeoElastic

## What this is

GeoElastic is a Go API that takes a business's identifying details — name, address, phone
number, location — and searches an Elasticsearch index of business/geo records to figure out
whether it's an **exact** match, a **probable ("fuzzy")** match, or **no** match at all. The goal
is a reusable building block for cross-referencing and de-duplicating business records that live
in two different systems, where the same business might be recorded with slightly different
spellings, formatting, or partial information in each.

It's also a personal project for learning Go and Elasticsearch from a MySQL/Laravel background,
so the implementation deliberately stays idiomatic to each rather than porting over
framework-style patterns that Go and Elasticsearch don't really have (no ORM, no Active Record,
no built-in migrations — see `CLAUDE.md` for the details on those tradeoffs).

There's no authentication yet — every endpoint below is open. The plan is an opaque,
Sanctum-style token issued on login and stored in its own Elasticsearch index, not JWTs, but
that isn't built yet.

## API

All requests and responses are JSON.

### `GET /health`

Confirms the API can reach Elasticsearch. No parameters.

```
200 {"status": "ok"}
503 {"status": "down", "error": "..."}
```

### `POST /businesses`

Creates a business. Every field is optional except in the sense that an empty document isn't
very useful — none are currently required by the server.

| Field | Type | Notes |
|---|---|---|
| `name` | string | |
| `display_name` | string | |
| `business_status` | string | e.g. `active`, `closed` |
| `primary_type` | string | e.g. `restaurant`, `retail` |
| `address` | object | `{street, city, state, zip}`, all strings |
| `location` | object | `{lat, lon}`, both numbers |
| `phone_number` | string | |
| `square_footage` | integer | |
| `rating` | number | |
| `opening_date` | string | RFC3339 timestamp, e.g. `2020-01-01T00:00:00Z` |
| `opening_hours` | array | `[{day, open, close}]` |

The response is the same shape with an added `id` — Elasticsearch's assigned document ID. Any
`id` sent in the request is ignored; it's always server-assigned.

```
POST /businesses
{"name": "Joes Pizza", "phone_number": "217-555-0100", "address": {"street": "123 Main St"}}

201 {"id": "AtPm058BZ7PV7IFzZMMO", "name": "Joes Pizza", ...}
```

### `GET /businesses`

Lists businesses. No parameters yet — capped at 1000 results, meant for smoke-testing the
connection rather than real pagination.

```
200 [{"id": "...", "name": "...", ...}, ...]
```

### `GET /business/{id}`

Fetches one business by its Elasticsearch document ID (the `id` returned from `POST /businesses`
or `GET /businesses`).

```
200 {"id": "...", "name": "...", ...}
404 {"error": "business not found"}
```

### `POST /businesses/match`

The actual fuzzy matcher. Every field is optional, but at least one must be present.

| Field | Type | Notes |
|---|---|---|
| `name` | string | |
| `display_name` | string | |
| `address` | object | `{street, city, state, zip}` — any subset |
| `phone_number` | string | |
| `location` | object | `{lat, lon}` — used as a proximity boost, not an exact filter |

Matching logic, in order:
1. **Exact** — every field you sent must match exactly (case-sensitive) against the corresponding
   business. If exactly one business satisfies that, it's returned alone with `"score": 1`.
2. **Fuzzy** — otherwise, a ranked search runs instead: `name`, `display_name`, and
   `address.street` tolerate typos; `address.city`/`state`/`zip` and `phone_number` currently
   still require an exact value to contribute (no typo tolerance there yet); `location` boosts
   businesses that are geographically closer, without excluding ones that aren't. Up to 5 results
   come back, each with a `score` normalized against the top result in that response (best match
   is always `1`, others proportionally lower) — this is a *relative* ranking within one search,
   not an absolute confidence percentage.
3. **None** — if nothing scores at all, `business` comes back empty.

```
POST /businesses/match
{"name": "Joe Pizzza", "address": {"street": "123 Main St"}}

200 {"match": "fuzzy", "business": [{"score": 1, "name": "Joes Pizza", ...}, {"score": 0.43, ...}]}
```

```
POST /businesses/match
{}

400 {"error": "at least one field must be provided to match on"}
```

## Getting started

**Prerequisites:** Go 1.22+ (developed against 1.26.5) and Docker (for the local Elasticsearch +
Kibana stack).

1. **Clone the repo.**

2. **Start the local Elastic stack:**
   ```
   cd elastic-start-local
   ./start.sh
   ```
   This brings up Elasticsearch and Kibana via Docker Compose and prints the credentials it
   generated into `elastic-start-local/.env`.

3. **Create a `.env` file at the repo root** (gitignored) with the app's own configuration,
   using the credentials from step 2:
   ```
   USERNAME=elastic
   PASSWORD=<ES_LOCAL_PASSWORD from elastic-start-local/.env>
   API_KEY=<ES_LOCAL_API_KEY from elastic-start-local/.env>
   SERVER_PORT=8080
   ES_URL=http://localhost:9200
   ```

4. **Create the Elasticsearch index and alias:**
   ```
   go run ./cmd/migrate
   ```

5. **(Optional) Seed some fake businesses to test against:**
   ```
   go run ./cmd/seed
   ```

6. **Run the API:**
   ```
   go run ./cmd/api
   ```

7. **Smoke-test it:**
   ```
   curl localhost:8080/health
   ```
