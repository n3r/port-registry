# CLAUDE.md

## Build & Run

```bash
make build      # builds bin/port-server and bin/portctl
make test       # go test ./...
make clean      # rm -rf bin/
```

Run a single test:
```bash
go test ./internal/store -run TestAllocateAutoAssign
go test ./internal/handler -run TestAllocateConflict
```

## Architecture

Layered design: **CLI (`portctl`) -> HTTP client -> server -> handler -> store interface -> SQLite**

```
cmd/server/main.go       # HTTP server entry point (flag: -port, -db)
cmd/portctl/main.go      # CLI client (subcommands: allocate, release, list, check, health)
internal/config/          # Constants: DefaultServerPort=51234, port range 3000-9999
internal/model/           # Shared types: Allocation, AllocateRequest, PortStatus, etc.
internal/store/store.go   # Store interface (Allocate, List, GetByPort, DeleteByID, DeleteByFilter)
internal/store/sqlite.go  # SQLite implementation (WAL mode, modernc.org/sqlite driver)
internal/handler/         # Chi router, REST API under /v1/allocations and /v1/ports/{port}
internal/client/          # Go HTTP client wrapping the REST API
```

## Key Patterns

- **Store interface** (`internal/store/store.go`) — all data access goes through this; SQLite is the only implementation.
- **In-memory SQLite for tests** — `NewSQLite(":memory:")` used in both store and handler tests; no test fixtures or external DB needed.
- **Chi router** — routes defined in `handler.Routes()`, versioned under `/v1`.

## Config Defaults

| Setting | Default |
|---------|---------|
| Server port | `51234` (listens on `127.0.0.1`) |
| Port range | `3000–9999` |
| DB path | `~/.port_server/ports.db` |
| Client env var | `PORT_SERVER_ADDR` (default `127.0.0.1:51234`) |
