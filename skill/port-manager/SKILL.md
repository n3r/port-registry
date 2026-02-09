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
portctl allocate --app <project> --instance <env> --service <service>
```

Output: `allocated port 4521 (id=3) for myapp/dev/postgres`

### Allocate a specific port

```bash
portctl allocate --app <project> --instance <env> --service <service> --port <N>
```

Fails if the port is already taken. Prefer auto-assign unless the user explicitly requests a specific port.

### Check if a port is available

```bash
portctl check --port <N>
```

Exit code 0 = available, exit code 1 = taken. Use this before hardcoding any port.

### List allocations

```bash
portctl list                          # all allocations
portctl list --app <project>          # filter by project
portctl list --app <project> --json   # JSON output for parsing
```

### Release ports

```bash
portctl release --app <project>                          # release all ports for a project
portctl release --app <project> --instance <env>         # release ports for a specific instance
portctl release --id <N>                                 # release a single allocation by ID
```

## Naming Conventions

| Field      | Value                        | Examples                          |
|------------|------------------------------|-----------------------------------|
| `--app`    | Project or repository name   | `myapp`, `backend`, `analytics`   |
| `--instance` | Branch, environment, or variant | `dev`, `test`, `feature-auth`  |
| `--service`  | Container or service name   | `postgres`, `redis`, `web`, `api` |

## Workflow

When setting up services that need ports:

1. **List existing allocations** for the project to avoid duplicates:
   ```bash
   portctl list --app <project>
   ```

2. **Allocate ports** for each service:
   ```bash
   portctl allocate --app myapp --instance dev --service postgres
   portctl allocate --app myapp --instance dev --service redis
   portctl allocate --app myapp --instance dev --service web
   ```

3. **Use the allocated ports** in configuration files (docker-compose.yml, .env, etc.)

4. **When tearing down**, release the ports:
   ```bash
   portctl release --app myapp --instance dev
   ```

## Port Range

Ports are allocated from the full range **1-65535**. Any valid port number can be requested.

## Reference

For detailed workflow examples including Docker Compose integration, multi-service patterns, and troubleshooting, see [references/WORKFLOW.md](references/WORKFLOW.md).
