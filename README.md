# port-registry

[![CI](https://github.com/n3r/port-registry/actions/workflows/ci.yml/badge.svg)](https://github.com/n3r/port-registry/actions/workflows/ci.yml)
[![Release](https://github.com/n3r/port-registry/actions/workflows/release.yml/badge.svg)](https://github.com/n3r/port-registry/actions/workflows/release.yml)
[![Go](https://img.shields.io/github/go-mod/go-version/n3r/port-registry)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> A local port registry that eliminates port conflicts across your dev services.

<p align="center">
  <img src="demo.gif" alt="port-registry demo" width="600">
</p>

## The problem

When you run multiple projects locally — each with Docker Compose stacks — port collisions are inevitable. Two projects both want 5432 for Postgres, and you only find out when `docker compose up` fails. **port-registry** maintains a central registry so no two services get the same port.

## Features

- **Central registry** — no more grepping YAML files for free ports
- **Auto-assign** — request any free port, or claim a specific one
- **Conflict detection** — 409 with the current holder on collision
- **Smart defaults** — auto-detects app from git repo, instance from branch/worktree
- **AI agent integration** — built-in skill teaches Claude Code and Codex to use `portctl`
- **Pure Go** — single binary, no CGO, SQLite with WAL mode
- **REST API** — scriptable HTTP interface under `/v1/`
- **Homebrew** — `brew install n3r/tap/port-registry`

## Quick start

```bash
brew install n3r/tap/port-registry

portctl start

portctl allocate --service postgres
# → allocated port 3000 for myapp/main/postgres

portctl allocate --service web --port 8080
# → allocated port 8080 for myapp/main/web

portctl list
# ID  APP    INSTANCE  SERVICE   PORT   CREATED
# 1   myapp  main      postgres  3000   2025-02-08 15:04:05
# 2   myapp  main      web       8080   2025-02-08 15:04:06

portctl check --port 8080
# → port 8080 is allocated to myapp/main/web
```

## Installation

### Homebrew (recommended)

```bash
brew install n3r/tap/port-registry
```

### From source

```bash
git clone https://github.com/n3r/port-registry.git
cd port-registry
make build    # produces bin/port-registry and bin/portctl
```

### Binary download

Grab the latest release from [GitHub Releases](https://github.com/n3r/port-registry/releases).

## Usage

### Allocating ports

```bash
# Auto-assign a port
portctl allocate --service postgres

# Claim a specific port
portctl allocate --service web --port 8080

# Specify app and instance explicitly
portctl allocate --app myapi --instance feature-x --service redis --port 6379
```

### Auto-detection

`--app` defaults to the git repo name (or current folder). `--instance` defaults to the git worktree or branch name. In most cases you only need `--service`:

```bash
portctl allocate --service postgres
# → allocated port 4521 for my-project/feature-branch/postgres
```

### Checking & releasing

```bash
# Check if a port is available
portctl check --port 5432

# Release by ID
portctl release --id 1

# Release by filter
portctl release --service postgres
```

### JSON output for scripting

```bash
portctl list --json
portctl list --json | jq '.[].port'
```

## AI agent integration

port-registry ships with an agent skill that teaches AI coding agents (Claude Code, OpenAI Codex) to use `portctl` automatically. Instead of hardcoding ports, agents allocate from the registry.

### Install the skill

```bash
# Project-local (recommended)
portctl skill install

# Global — all projects, all platforms
portctl skill install --global
```

### Supported platforms

| Platform | Directory |
|----------|-----------|
| Claude Code | `~/.claude/skills/port-registry/` |
| OpenAI Codex | `~/.codex/skills/port-registry/` |
| Generic Agents | `~/.agents/skills/port-registry/` |

### Example prompts

> Set up docker-compose for this project with Postgres, Redis, and a Node.js web server. Allocate ports through portctl.

> Scan this project's docker-compose.yml and register all host-bound ports with portctl.

> Port 5432 is already in use. Find out what's using it and allocate a different port for Postgres.

> Check which ports are allocated and whether any conflict with my other projects.

## Configuration

| Setting | Flag / Env | Default | Description |
|---------|-----------|---------|-------------|
| Server port | `--port` | `51234` | Port the HTTP server listens on |
| Database path | `--db` | `~/.port-registry/ports.db` | SQLite database file location |
| PID file | `--pidfile` | `~/.port-registry/port-registry.pid` | PID file for the server process |
| Log file | — | `~/.port-registry/port-registry.log` | Server log output (when started via `portctl start`) |
| Server address (client) | `PORT_REGISTRY_ADDR` | `127.0.0.1:51234` | Address `portctl` connects to |
| Auto-assign range | — | `1–65535` | Port range for auto-assignment |

<details>
<summary><strong>CLI reference</strong></summary>

The CLI binary is `portctl`. Set `PORT_REGISTRY_ADDR` to override the default server address (`127.0.0.1:51234`).

### `portctl start`

Start the port-registry daemon in the background.

```
portctl start
```

Locates the `port-registry` binary next to the `portctl` executable, starts it as a detached process, and waits for the health check to pass. Logs are written to `~/.port-registry/port-registry.log`.

**Exit codes:** `0` started successfully, `1` already running or startup failed

### `portctl stop`

Stop the running port-registry daemon.

```
portctl stop
```

Reads the PID file, sends SIGTERM, and waits up to 5 seconds for the process to exit.

**Exit codes:** `0` stopped successfully, `1` not running or failed to stop

### `portctl restart`

Stop and start the port-registry daemon.

```
portctl restart
```

**Exit codes:** `0` restarted successfully, `1` error during stop or start

### `portctl status`

Show whether the port-registry daemon is running.

```
portctl status
```

Reports the PID and health status. Cleans up stale PID files automatically.

**Exit codes:** `0` always

### `portctl allocate`

Allocate a port for a service.

```
portctl allocate [--app <name>] [--instance <name>] --service <name> [--port <number>]
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--app` | no | git repo or folder name | Application name |
| `--instance` | no | worktree or branch name | Instance name |
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
| `--instance` | no | worktree or branch name | Filter by instance name |
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
| `--instance` | no | worktree or branch name | Filter by instance |
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

Install the port-registry agent skill.

```
portctl skill install            # install to project-local .claude/
portctl skill install --global   # install to global platforms (~/.claude, ~/.codex, ~/.agents)
```

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--global` | no | false | Install to global platforms instead of project-local |

By default, installs to `.claude/skills/port-registry/` in the current directory. With `--global`, installs to all detected global agent platforms. Skill files are embedded in the binary — no source tree needed.

**Exit codes:** `0` always

</details>

<details>
<summary><strong>API reference</strong></summary>

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

</details>

<details>
<summary><strong>Project structure</strong></summary>

```
port-registry/
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
│   └── port-registry/
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

</details>

## Development

```bash
make build         # Build bin/port-registry and bin/portctl
make test          # Run all tests (go test ./...)
go test -race ./...  # Race detector
make clean         # Remove bin/
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
