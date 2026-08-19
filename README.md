# GeoElastic

## What this is

GeoElastic is a Go API that takes a business's identifying details — name, address, phone
number, location — and searches an Elasticsearch index of business/geo records to figure out
whether it's an **exact** match, a **probable ("fuzzy")** match, or **no** match at all. The goal
is a reusable building block for cross-referencing and de-duplicating business records that live
in two different systems, where the same business might be recorded with slightly different
spellings, formatting, or partial information in each.

Authentication is an opaque, Sanctum-style bearer token: register a user, exchange credentials
for a token via `POST /tokens`, then send that token as `Authorization: Bearer <token>` on every
`/businesses*` request. Tokens are stored hashed (SHA-256) in their own Elasticsearch index, not
JWTs — see `POST /users` / `POST /tokens` below.

## API

All requests and responses are JSON.

### `GET /health`

Confirms the API can reach Elasticsearch. No parameters. Not authenticated.

```
200 {"status": "ok"}
503 {"status": "down", "error": "..."}
```

### `POST /users`

Registers a new user. Open to anyone — there's no invite/approval step for this project. Not
authenticated (you need this to get your first token).

| Field | Type | Notes |
|---|---|---|
| `username` | string | required, must be unique |
| `password` | string | required; stored only as a bcrypt hash |

```
POST /users
{"username": "alex", "password": "correct-horse-battery-staple"}

201 {"id": "alex", "username": "alex"}
400 {"error": "username and password are required"}
409 {"error": "username is already taken"}
```

### `POST /tokens`

Exchanges a username/password for a new access token, valid for 30 days. Not authenticated
(that's the point). The raw token is only ever shown here, once — the server only ever stores its
SHA-256 hash, so losing it means issuing a new one.

```
POST /tokens
{"username": "alex", "password": "correct-horse-battery-staple"}

201 {"access_token": "9f1c...  (64 hex chars)"}
401 {"error": "invalid username or password"}
```

Use it on every request below:
```
curl -H "Authorization: Bearer <access_token>" localhost:8080/businesses
```
Every `/businesses*` route returns `401 {"error": "..."}` if the header is missing, malformed, or
the token is unknown, expired, or revoked — those three cases are deliberately indistinguishable
to the caller.

### `POST /businesses`

*Requires auth.* Creates a business, unless one already exists with the exact same `name`,
`phone_number`, and full `address` (street, city, state, zip) — in which case the existing
business is returned instead of creating a duplicate. This identity check is atomic against
Elasticsearch (see `CLAUDE.md`/code comments on `businessDedupeKey`), so two near-simultaneous
identical requests can't both create a duplicate.

| Field | Type | Notes |
|---|---|---|
| `name` | string | **required** |
| `phone_number` | string | **required**; any format, normalized to digits on write |
| `address` | object | **required**: `{street, city, state, zip}`, all four required |
| `display_name` | string | |
| `business_status` | string | e.g. `active`, `closed` |
| `primary_type` | string | e.g. `restaurant`, `retail` |
| `location` | object | `{lat, lon}`, both numbers |
| `square_footage` | integer | |
| `rating` | number | |
| `opening_date` | string | RFC3339 timestamp, e.g. `2020-01-01T00:00:00Z` |
| `opening_hours` | array | `[{day, open, close}]` |

The response is the same shape with an added `id` — Elasticsearch's assigned document ID. Any
`id` sent in the request is ignored; it's always server-assigned. `phone_number` in the response
is always the normalized (digits-only) form actually stored, even if you sent it formatted.

```
POST /businesses
{"name": "Joes Pizza", "phone_number": "217-555-0100", "address": {"street": "123 Main St", "city": "Springfield", "state": "IL", "zip": "62704"}}

201 {"id": "3f9c...", "name": "Joes Pizza", "phone_number": "2175550100", ...}
```

Submitting the same name/phone/address again returns the existing business instead of creating a
second one:

```
200 {"id": "3f9c...", "name": "Joes Pizza", "phone_number": "2175550100", ...}
```

Missing any required field:
```
400 {"error": "name, phone_number, and a complete address (street, city, state, zip) are required"}
```

### `GET /businesses`

*Requires auth.* Lists businesses. No parameters yet — capped at 1000 results, meant for
smoke-testing the connection rather than real pagination.

```
200 [{"id": "...", "name": "...", ...}, ...]
```

### `GET /business/{id}`

*Requires auth.* Fetches one business by its Elasticsearch document ID (the `id` returned from
`POST /businesses` or `GET /businesses`).

```
200 {"id": "...", "name": "...", ...}
404 {"error": "business not found"}
```

### `POST /businesses/match`

*Requires auth.* The actual fuzzy matcher. Every field is optional, but at least one must be
present.

| Field | Type | Notes |
|---|---|---|
| `name` | string | |
| `display_name` | string | |
| `address` | object | `{street, city, state, zip}` — any subset |
| `phone_number` | string | any format |
| `location` | object | `{lat, lon}` — used as a proximity boost, not an exact filter |

Matching logic, in order:
1. **Exact** — every field you sent must match exactly (case-sensitive) against the corresponding
   business (`phone_number` is normalized to digits before comparing, so formatting doesn't
   matter). If exactly one business satisfies that, it's returned alone with `"score": 1`.
2. **Fuzzy** — otherwise, a ranked search runs instead: `name`, `display_name`, and
   `address.street` tolerate typos; `address.city`/`state`/`zip` still require an exact value to
   contribute; `phone_number` is compared on its last 7 digits only (prefix + line number, area
   code excluded) and heavily boosted when they match, since two businesses sharing just an area
   code isn't a meaningful signal on its own; `location` boosts businesses that are geographically
   closer, without excluding ones that aren't. Up to 5 results come back, each with a `score`
   normalized against the top result in that response (best match is always `1`, others
   proportionally lower) — this is a *relative* ranking within one search, not an absolute
   confidence percentage.
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

4. **Create the Elasticsearch indices and aliases** (`businesses`, `users`, `tokens`):
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

   curl -X POST localhost:8080/users -d '{"username": "alex", "password": "correct-horse-battery-staple"}'

   TOKEN=$(curl -X POST localhost:8080/tokens \
     -d '{"username": "alex", "password": "correct-horse-battery-staple"}' \
     | python3 -c 'import sys, json; print(json.load(sys.stdin)["access_token"])')

   curl -H "Authorization: Bearer $TOKEN" localhost:8080/businesses
   ```
