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
go test ./internal/skill -run TestInstallDetectsAndWritesToPlatforms
```

## Architecture

Layered design: **CLI (`portctl`) -> HTTP client -> server -> handler -> store interface -> SQLite**

```
cmd/server/main.go       # HTTP server entry point (flag: -port, -db, -pidfile)
cmd/portctl/main.go      # CLI client (subcommands: start, stop, restart, status, allocate, release, list, check, health, skill, version)
internal/config/          # Constants: DefaultServerPort=51234, port range 3000-9999
internal/model/           # Shared types: Allocation, AllocateRequest, PortStatus, etc.
internal/store/store.go   # Store interface (Allocate, List, GetByPort, DeleteByID, DeleteByFilter)
internal/store/sqlite.go  # SQLite implementation (WAL mode, modernc.org/sqlite driver)
internal/handler/         # Chi router, REST API under /v1/allocations and /v1/ports/{port}
internal/client/          # Go HTTP client wrapping the REST API
internal/ui/              # CLI output styling (lipgloss-based colors, tables, symbols)
internal/skill/           # Agent skill install logic (platform detection, file writing)
internal/version/         # Version info injected via ldflags at build time
skill/embed.go            # go:embed package exposing SKILL.md and WORKFLOW.md as []byte
```

## Key Patterns

- **Store interface** (`internal/store/store.go`) — all data access goes through this; SQLite is the only implementation.
- **In-memory SQLite for tests** — `NewSQLite(":memory:")` used in both store and handler tests; no test fixtures or external DB needed.
- **Chi router** — routes defined in `handler.Routes()`, versioned under `/v1`.
- **Uniqueness constraints** — `UNIQUE(port)` prevents port conflicts; `UNIQUE(app, instance, service)` prevents duplicate service allocations. Both return `409 Conflict` with the existing holder.
- **go:embed for skill files** — `skill/embed.go` embeds `SKILL.md` and `WORKFLOW.md` so `portctl skill install` works from the binary without needing the source tree.
- **Styled CLI output** — `internal/ui/` wraps lipgloss for consistent colored output (success/error/warning/info/subtle). All CLI output goes through `ui.*` helpers.
- **Version injection** — `internal/version/` has `Version`, `Commit`, `Date` vars set via `-ldflags` at build time. Both binaries support `--version`.

## Config Defaults

| Setting | Default |
|---------|---------|
| Server port | `51234` (listens on `127.0.0.1`) |
| Port range | `1–65535` |
| DB path | `~/.port_server/ports.db` |
| PID file | `~/.port_server/port-server.pid` |
| Log file | `~/.port_server/port-server.log` |
| Client env var | `PORT_SERVER_ADDR` (default `127.0.0.1:51234`) |

## Agent Skill

The `skill/port-manager/` directory contains an agent skill that teaches AI agents to use `portctl` automatically when managing ports. Install with:

```bash
portctl skill install   # auto-detects platforms (~/.claude, ~/.codex, ~/.agents) and project-local .claude/
make install-skill      # alternative: hard-copy to ~/.claude/skills/ and ~/.agents/skills/
```
