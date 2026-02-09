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
• Auto-assign picks the first free port in 1–65535
```

## Agent skill

port-server ships with an agent skill that teaches AI coding agents (Claude Code, OpenAI Codex, etc.) to use `portctl` automatically when managing ports. Instead of hardcoding ports, agents will allocate from the registry.

### Install locally (recommended)

Run from your project directory:

```bash
portctl skill install
```

This creates `.claude/skills/port-manager/` in the current directory. The skill is scoped to that project.

### Install globally

To make the skill available across all projects:

```bash
portctl skill install --global
```

This installs to all detected global platforms:

| Platform | Directory |
|----------|-----------|
| Claude Code | `~/.claude/skills/port-manager/` |
| OpenAI Codex | `~/.codex/skills/port-manager/` |
| Generic Agents | `~/.agents/skills/port-manager/` |

Platforms that don't exist on your system are skipped.

### What the skill does

Once installed, agents will automatically:

- **Allocate ports** via `portctl` instead of picking arbitrary numbers
- **Register existing ports** from docker-compose.yml, .env files, and npm scripts
- **Only track host-bound ports** — distinguishes `"5432:5432"` (register) from bare `"3001"` (skip)
- **Check for conflicts** before using any port
- **Release ports** when tearing down services

### Example prompts

**New project** — setting up a Docker Compose stack from scratch:

> Set up docker-compose for this project with Postgres, Redis, and a Node.js web server. Allocate ports through portctl.

> Add a Minio S3 service to our docker-compose. Use portctl to get a port for it.

> Create a docker-compose.yml for local development. I need Postgres, Elasticsearch, and a Rails API server.

**Existing project** — registering ports that are already in use:

> Scan this project's docker-compose.yml and register all host-bound ports with portctl.

> Register all the ports used by this project — check docker-compose, .env files, and npm scripts.

> I'm getting port conflicts with another project. Register all ports from this repo so other projects know what's taken.

**Troubleshooting:**

> Check which ports are allocated and whether any of them conflict with my other projects.

> Port 5432 is already in use. Find out what's using it and allocate a different port for Postgres.

## Installation

### Homebrew

```bash
brew install n3r/tap/port-registry
```

### From source

```bash
make build    # produces bin/port-server and bin/portctl
```

## Quick start

```bash
# Build both binaries (skip if installed via Homebrew)
make build

# Start the server
./bin/portctl start
# → port-server started (pid 12345)

# Allocate a port for your service (--app defaults to repo/folder name)
./bin/portctl allocate --instance dev --service postgres
# → allocated port 3000 (id=1) for myapp/dev/postgres

# Allocate a specific port
./bin/portctl allocate --instance dev --service web --port 8080
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

### `portctl start`

Start the port-server daemon in the background.

```
portctl start
```

Locates the `port-server` binary next to the `portctl` executable, starts it as a detached process, and waits for the health check to pass. Logs are written to `~/.port_server/port-server.log`.

**Exit codes:** `0` started successfully, `1` already running or startup failed

### `portctl stop`

Stop the running port-server daemon.

```
portctl stop
```

Reads the PID file, sends SIGTERM, and waits up to 5 seconds for the process to exit.

**Exit codes:** `0` stopped successfully, `1` not running or failed to stop

### `portctl restart`

Stop and start the port-server daemon.

```
portctl restart
```

**Exit codes:** `0` restarted successfully, `1` error during stop or start

### `portctl status`

Show whether the port-server daemon is running.

```
portctl status
```

Reports the PID and health status. Cleans up stale PID files automatically.

**Exit codes:** `0` always

### `portctl allocate`

Allocate a port for a service.

```
portctl allocate [--app <name>] --instance <name> --service <name> [--port <number>]
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--app` | no | git repo or folder name | Application name |
| `--instance` | yes | | Instance name |
| `--service` | yes | | Service name |
| `--port` | no | 0 (auto) | Specific port to allocate; 0 = auto-assign from 1–65535 |

**Exit codes:** `0` success, `1` error (port taken, validation failure, server unreachable)

### `portctl release`

Release one or more port allocations.

```
portctl release --id <number>
portctl release [--app <name>] [--instance <name>] [--service <name>] [--port <number>]
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--id` | no | 0 | Release a specific allocation by ID |
| `--app` | no | git repo or folder name | Filter by application name |
| `--instance` | no | | Filter by instance name |
| `--service` | no | | Filter by service name |
| `--port` | no | 0 | Filter by port number |

When `--id` is not provided, at least `--app` or `--port` is required (--app is auto-detected if not specified). Filters are AND-ed together.

**Exit codes:** `0` success, `1` error (not found, validation failure)

### `portctl list`

List current allocations.

```
portctl list [--app <name>] [--instance <name>] [--service <name>] [--json]
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--app` | no | git repo or folder name | Filter by application |
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

### `portctl version`

Print the version, commit, and build date.

```
portctl version
```

### `portctl skill install`

Install the port-manager agent skill.

```
portctl skill install            # install to project-local .claude/
portctl skill install --global   # install to global platforms (~/.claude, ~/.codex, ~/.agents)
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--global` | no | false | Install to global platforms instead of project-local |

By default, installs to `.claude/skills/port-manager/` in the current directory. With `--global`, installs to all detected global agent platforms. Skill files are embedded in the binary — no source tree needed.

**Exit codes:** `0` always

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
| PID file | `--pidfile` | `~/.port_server/port-server.pid` | PID file for the server process |
| Log file | — | `~/.port_server/port-server.log` | Server log output (when started via `portctl start`) |
| Server address (client) | `PORT_SERVER_ADDR` | `127.0.0.1:51234` | Address `portctl` connects to |
| Auto-assign range | — | `1–65535` | Port range for auto-assignment |

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
│   │   └── config.go            # Defaults: port 51234, range 1–65535, DB path
│   ├── handler/
│   │   ├── handler.go           # HTTP route handlers (chi router)
│   │   └── handler_test.go      # Handler integration tests
│   ├── model/
│   │   └── model.go             # Request/response JSON structs
│   ├── skill/
│   │   ├── install.go           # Agent skill installer (platform detection)
│   │   └── install_test.go      # Install logic tests
│   ├── store/
│   │   ├── store.go             # Store interface
│   │   ├── sqlite.go            # SQLite implementation (WAL, auto-migrate)
│   │   └── sqlite_test.go       # Store unit tests
│   ├── ui/
│   │   ├── ui.go                # CLI output styling (lipgloss)
│   │   └── ui_test.go           # UI helper tests
│   └── version/
│       └── version.go           # Version info (injected via ldflags)
├── skill/
│   ├── embed.go                 # go:embed for skill files
│   └── port-manager/
│       ├── SKILL.md             # Agent skill definition
│       └── references/
│           └── WORKFLOW.md      # Agent workflow reference
├── .github/workflows/
│   ├── ci.yml                   # CI: test + build on push/PR
│   └── release.yml              # Release: GoReleaser on tag push
├── .goreleaser.yml              # Cross-platform builds + Homebrew tap
├── Makefile                     # build, test, clean, install-skill
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

**Flat schema.** One `allocations` table with two uniqueness constraints: `UNIQUE(port)` prevents port conflicts, and `UNIQUE(app, instance, service)` prevents duplicate service allocations. Both return `409 Conflict` with the existing holder.

**Delete safety.** `DeleteByFilter` requires at least one filter criterion, preventing accidental deletion of all allocations.

**Conflict reporting.** A `409 Conflict` response includes the existing holder so the caller knows who owns the port without a second request.

**API versioning.** All endpoints are under `/v1/` so the API can evolve without breaking existing clients.

**Embedded skill files.** Agent skill markdown files are compiled into the `portctl` binary via `go:embed`, so `portctl skill install` works without the source tree.

**GoReleaser + Homebrew.** Tagged releases build cross-platform binaries (darwin/linux, amd64/arm64) and publish to a Homebrew tap automatically via GitHub Actions.

## License

MIT
