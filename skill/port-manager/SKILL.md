---
name: port-manager
description: Manage local port allocations to prevent conflicts between Docker containers, dev servers, and services across projects. Use when setting up docker-compose, starting dev servers, assigning ports, or when the user mentions port conflicts.
user-invocable: false
allowed-tools:
  - Bash(portctl *)
---

# Port Manager

You have access to `portctl`, a CLI that allocates and tracks local ports across projects. Use it to prevent port conflicts between Docker containers, dev servers, and other services.

## When to Use

- Setting up or modifying `docker-compose.yml` (always allocate ports instead of hardcoding)
- Starting dev servers that need a port
- Configuring services that bind to localhost ports
- User mentions port conflicts or "address already in use"
- Any time you would otherwise pick an arbitrary port number

## Prerequisites

The port-server must be running. Check with:

```bash
portctl health
```

If it's not running, start it:

```bash
port-server &
```

If `portctl` is not on PATH, build it first:

```bash
# From the port_server repo
make build
# Then either add bin/ to PATH or use the full path
```

## Commands

### Allocate a port (auto-assign)

```bash
portctl allocate --instance <env> --service <service>
```

Output: `allocated port 4521 (id=3) for myapp/dev/postgres`

`--app` defaults to the git repo name (or folder name if not in a repo). Pass `--app <name>` only to override.

### Allocate a specific port

```bash
portctl allocate --instance <env> --service <service> --port <N>
```

Fails if the port is already taken. Prefer auto-assign unless the user explicitly requests a specific port.

### Check if a port is available

```bash
portctl check --port <N>
```

Exit code 0 = available, exit code 1 = taken. Use this before hardcoding any port.

### List allocations

```bash
portctl list                          # current project (auto-detected --app)
portctl list --app <project>          # explicit project filter
portctl list --app <project> --json   # JSON output for parsing
```

### Release ports

```bash
portctl release                                          # release all ports for the current project
portctl release --instance <env>                         # release ports for a specific instance
portctl release --app <project>                          # release by explicit project name
portctl release --id <N>                                 # release a single allocation by ID
```

## Naming Conventions

| Field      | Value                        | Examples                          |
|------------|------------------------------|-----------------------------------|
| `--app`    | Project or repository name (auto-detected from git repo or folder name) | `myapp`, `backend`, `analytics` |
| `--instance` | Branch, environment, or variant | `dev`, `test`, `feature-auth`  |
| `--service`  | Container or service name   | `postgres`, `redis`, `web`, `api` |

Do not pass `--app` unless you need to override the auto-detected project name.

## What to Register

Only register ports that **actually bind on the host machine**. These are the ports that can conflict with other projects.

### Docker Compose ports

Read the `ports:` section carefully:

- **`"5432:5432"`** (host:container) — fixed host binding, **register the host port** (left side)
- **`"9100:8080"`** (host:container differ) — fixed host binding, **register 9100** (host port)
- **`"3001"`** (container-only) — Docker assigns a random ephemeral host port, **do NOT register**

### Non-Docker ports

Also register ports from other sources that bind on the host:

- **npm scripts** (`npm run dev` on port 3001, `npm run storybook` on port 6006)
- **dev servers** started outside Docker
- **Ports in .env files** used by services running on the host

### Register all port numbers

Any valid port (1-65535) can be registered — including well-known low ports like 80, 443, 1080, etc.

## Workflow

When setting up services that need ports:

1. **List existing allocations** for the project to avoid duplicates:
   ```bash
   portctl list
   ```

2. **Allocate ports** for each service:
   ```bash
   portctl allocate --instance dev --service postgres
   portctl allocate --instance dev --service redis
   portctl allocate --instance dev --service web
   ```

3. **Use the allocated ports** in configuration files (docker-compose.yml, .env, etc.)

4. **When tearing down**, release the ports:
   ```bash
   portctl release --instance dev
   ```

When registering ports from an existing project:

1. **Scan all port sources**: docker-compose.yml, .env, package.json scripts, Makefile, etc.
2. **Filter to host-bound ports only**: skip container-only Docker port mappings
3. **Register with specific port**: use `--port <N>` for each known host port (--app is auto-detected)

## Port Range

Ports are allocated from the full range **1-65535**. Any valid port number can be requested.

## Reference

For detailed workflow examples including Docker Compose integration, multi-service patterns, and troubleshooting, see [references/WORKFLOW.md](references/WORKFLOW.md).
