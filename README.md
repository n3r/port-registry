# port-server

A local port registry that prevents port conflicts across Docker containers and dev services on your machine.

## Problem

When you run multiple projects locally — each with their own Docker Compose stacks — port collisions are inevitable. Two projects both want port 5432 for Postgres, or 3000 for a web server. You discover the conflict only when `docker compose up` fails, then waste time grepping through YAML files to find a free port.

**port-server** solves this by maintaining a central registry of allocated ports. Services request a port (or ask for any available one), and the server guarantees no two services get the same port.

## How it works

```
┌──────────────┐         HTTP          ┌──────────────┐        ┌────────────┐
│   portctl    │ ───────────────────── │  port-server │ ────── │   SQLite   │
│   (CLI)      │   localhost:51234     │  (HTTP API)  │        │   (WAL)    │
└──────────────┘                       └──────────────┘        └────────────┘

• port-server runs as a background daemon on 127.0.0.1
• portctl is the CLI client — allocate, release, list, check ports
• SQLite with WAL mode stores allocations durably
• Auto-assign picks the first free port in 3000–9999
```

## Quick start

```bash
# Build both binaries
make build

# Start the server (background it however you like)
./bin/port-server &

# Allocate a port for your service
./bin/portctl allocate --app myapp --instance dev --service postgres
# → allocated port 3000 (id=1) for myapp/dev/postgres

# Allocate a specific port
./bin/portctl allocate --app myapp --instance dev --service web --port 8080
# → allocated port 8080 (id=2) for myapp/dev/web

# List everything
./bin/portctl list
# ID  APP    INSTANCE  SERVICE   PORT  CREATED
# 1   myapp  dev       postgres  3000  2025-02-08 15:04:05
# 2   myapp  dev       web       8080  2025-02-08 15:04:06

# Check if a port is free
./bin/portctl check --port 3000
# → port 3000 is allocated to myapp/dev/postgres (id=1)

# Release when done
./bin/portctl release --id 1
# → released allocation 1
```

## CLI reference

The CLI binary is `portctl`. Set `PORT_SERVER_ADDR` to override the default server address (`127.0.0.1:51234`).

### `portctl allocate`

Allocate a port for a service.

```
portctl allocate --app <name> --instance <name> --service <name> [--port <number>]
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--app` | yes | | Application name |
| `--instance` | yes | | Instance name |
| `--service` | yes | | Service name |
| `--port` | no | 0 (auto) | Specific port to allocate; 0 = auto-assign from 3000–9999 |

**Exit codes:** `0` success, `1` error (port taken, validation failure, server unreachable)

### `portctl release`

Release one or more port allocations.

```
portctl release --id <number>
portctl release --app <name> [--instance <name>] [--service <name>] [--port <number>]
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--id` | no | 0 | Release a specific allocation by ID |
| `--app` | no | | Filter by application name |
| `--instance` | no | | Filter by instance name |
| `--service` | no | | Filter by service name |
| `--port` | no | 0 | Filter by port number |

When `--id` is not provided, at least `--app` or `--port` is required. Filters are AND-ed together.

**Exit codes:** `0` success, `1` error (not found, validation failure)

### `portctl list`

List current allocations.

```
portctl list [--app <name>] [--instance <name>] [--service <name>] [--json]
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--app` | no | | Filter by application |
| `--instance` | no | | Filter by instance |
| `--service` | no | | Filter by service |
| `--json` | no | false | Output as JSON instead of table |

**Exit codes:** `0` success, `1` error

### `portctl check`

Check whether a port is available.

```
portctl check --port <number>
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--port` | yes | | Port number to check |

**Exit codes:** `0` port is available, `1` port is allocated or error

### `portctl health`

Check if the server is reachable.

```
portctl health
```

**Exit codes:** `0` healthy, `1` unhealthy or unreachable

## API reference

Base URL: `http://127.0.0.1:51234`

### `GET /healthz`

Health check.

**Response:** `200 OK`

```json
{"status": "ok"}
```

### `POST /v1/allocations`

Allocate a port.

**Request:**

```json
{
  "app": "myapp",
  "instance": "dev",
  "service": "postgres",
  "port": 5432
}
```

Omit `port` or set to `0` for auto-assignment.

**Responses:**

`201 Created`

```json
{
  "id": 1,
  "app": "myapp",
  "instance": "dev",
  "service": "postgres",
  "port": 5432,
  "created_at": "2025-02-08T15:04:05Z"
}
```

`409 Conflict` — port already allocated, includes the current holder:

```json
{
  "error": "port already allocated",
  "holder": {
    "id": 3,
    "app": "other",
    "instance": "dev",
    "service": "db",
    "port": 5432,
    "created_at": "2025-02-08T14:00:00Z"
  }
}
```

`400 Bad Request` — missing required fields or invalid JSON.

### `GET /v1/allocations`

List allocations. All query parameters are optional filters.

```
GET /v1/allocations?app=myapp&instance=dev&service=postgres
```

**Response:** `200 OK`

```json
[
  {
    "id": 1,
    "app": "myapp",
    "instance": "dev",
    "service": "postgres",
    "port": 5432,
    "created_at": "2025-02-08T15:04:05Z"
  }
]
```

Returns `[]` when no allocations match.

### `GET /v1/ports/{port}`

Check if a specific port is available.

**Response:** `200 OK`

```json
{
  "port": 5432,
  "available": false,
  "holder": {
    "id": 1,
    "app": "myapp",
    "instance": "dev",
    "service": "postgres",
    "port": 5432,
    "created_at": "2025-02-08T15:04:05Z"
  }
}
```

When available, `holder` is omitted from the response.

### `DELETE /v1/allocations/{id}`

Release a single allocation by ID.

**Responses:**

`200 OK`

```json
{"status": "deleted"}
```

`404 Not Found`

```json
{"error": "allocation not found"}
```

### `DELETE /v1/allocations`

Release allocations matching a filter.

**Request:**

```json
{
  "app": "myapp",
  "instance": "dev"
}
```

At least one filter field is required.

**Response:** `200 OK`

```json
{"deleted": 2}
```

## Configuration

| Setting | Flag / Env | Default | Description |
|---------|-----------|---------|-------------|
| Server port | `--port` | `51234` | Port the HTTP server listens on |
| Database path | `--db` | `~/.port_server/ports.db` | SQLite database file location |
| Server address (client) | `PORT_SERVER_ADDR` | `127.0.0.1:51234` | Address `portctl` connects to |
| Auto-assign range | — | `3000–9999` | Port range for auto-assignment |

## Project structure

```
port_server/
├── cmd/
│   ├── server/
│   │   └── main.go              # HTTP server entry point
│   └── portctl/
│       └── main.go              # CLI client entry point
├── internal/
│   ├── client/
│   │   └── client.go            # HTTP client library used by portctl
│   ├── config/
│   │   └── config.go            # Defaults: port 51234, range 3000–9999, DB path
│   ├── handler/
│   │   ├── handler.go           # HTTP route handlers (chi router)
│   │   └── handler_test.go      # Handler integration tests
│   ├── model/
│   │   └── model.go             # Request/response JSON structs
│   └── store/
│       ├── store.go             # Store interface
│       ├── sqlite.go            # SQLite implementation (WAL, auto-migrate)
│       └── sqlite_test.go       # Store unit tests
├── Makefile                     # build, test, clean
├── go.mod
└── go.sum
```

## Development

```bash
make build    # Build bin/port-server and bin/portctl
make test     # Run all tests (go test ./...)
make clean    # Remove bin/
```

Tests use in-memory SQLite — no external dependencies needed.

## Design decisions

**Pure-Go SQLite (`modernc.org/sqlite`).** No CGO required. Builds anywhere Go runs without a C toolchain.

**Localhost-only binding.** The server binds to `127.0.0.1`, not `0.0.0.0`. This is a local development tool — there's no reason to expose it to the network.

**WAL journal mode.** Enabled on every connection for better concurrent read/write performance across multiple CLI invocations.

**Flat schema.** One `allocations` table with a `UNIQUE` constraint on `port`. The composite unique constraint on `(app, instance, service, port)` models the hierarchy without extra tables.

**Delete safety.** `DeleteByFilter` requires at least one filter criterion, preventing accidental deletion of all allocations.

**Conflict reporting.** A `409 Conflict` response includes the existing holder so the caller knows who owns the port without a second request.

**API versioning.** All endpoints are under `/v1/` so the API can evolve without breaking existing clients.

## License

MIT
