# Port Manager Workflow Reference

## Multi-Service Allocation

When a project needs multiple services (e.g., a typical web stack), allocate all ports upfront:

```bash
# Allocate ports for a full stack
portctl allocate --app myapp --instance dev --service postgres
# -> allocated port 3042 (id=1) for myapp/dev/postgres

portctl allocate --app myapp --instance dev --service redis
# -> allocated port 3043 (id=2) for myapp/dev/redis

portctl allocate --app myapp --instance dev --service web
# -> allocated port 3044 (id=3) for myapp/dev/web

portctl allocate --app myapp --instance dev --service api
# -> allocated port 3045 (id=4) for myapp/dev/api
```

Verify all allocations:

```bash
portctl list --app myapp
```

## Docker Compose Integration

After allocating ports, use them in `docker-compose.yml`:

```yaml
services:
  postgres:
    image: postgres:16
    ports:
      - "3042:5432"  # allocated by portctl

  redis:
    image: redis:7
    ports:
      - "3043:6379"  # allocated by portctl

  web:
    build: .
    ports:
      - "3044:3000"  # allocated by portctl
    environment:
      DATABASE_URL: postgres://postgres:postgres@postgres:5432/myapp
      REDIS_URL: redis://redis:6379
```

The host port (left side) comes from `portctl`. The container port (right side) is the service's default internal port.

### Using .env Files

For projects that use `.env` files with docker-compose:

```bash
# Allocate and capture ports
PG_PORT=$(portctl list --app myapp --service postgres --json | jq '.[0].port')
REDIS_PORT=$(portctl list --app myapp --service redis --json | jq '.[0].port')
WEB_PORT=$(portctl list --app myapp --service web --json | jq '.[0].port')

# Write to .env
cat > .env <<EOF
PG_PORT=${PG_PORT}
REDIS_PORT=${REDIS_PORT}
WEB_PORT=${WEB_PORT}
EOF
```

Then reference in `docker-compose.yml`:

```yaml
services:
  postgres:
    ports:
      - "${PG_PORT}:5432"
```

## Multiple Instances

Use the `--instance` flag to run parallel environments without conflicts:

```bash
# Developer A working on feature-auth
portctl allocate --app myapp --instance feature-auth --service postgres
portctl allocate --app myapp --instance feature-auth --service web

# Developer B working on feature-payments
portctl allocate --app myapp --instance feature-payments --service postgres
portctl allocate --app myapp --instance feature-payments --service web

# Each gets unique ports, no conflicts
portctl list --app myapp
```

## Cleanup Patterns

### Release all ports for a project

```bash
portctl release --app myapp
```

### Release a specific environment

```bash
portctl release --app myapp --instance dev
```

### Release a single service

```bash
portctl release --app myapp --instance dev --service postgres
```

### Release by allocation ID

```bash
portctl release --id 42
```

## Checking Before Hardcoding

If a user or config file specifies a particular port, check availability first:

```bash
portctl check --port 5432
# Exit code 0 -> available, safe to use
# Exit code 1 -> taken, pick another or auto-assign
```

## Troubleshooting

### "connection refused" from portctl

The port-server is not running. Start it:

```bash
port-server &
```

Or if not on PATH:

```bash
~/path/to/port_server/bin/port-server &
```

The server listens on `127.0.0.1:51234` by default.

### "port is already allocated"

The requested port is taken. Options:

1. Use auto-assign (omit `--port`) to get an available port
2. Check who holds it: `portctl check --port <N>`
3. Release it if it's stale: `portctl release --id <N>`

### "no ports available"

The port range (1-65535) is exhausted. Release unused allocations:

```bash
portctl list  # review all allocations
portctl release --app <stale-project>  # clean up old projects
```

### portctl not found

Build from the port_server repository:

```bash
cd /path/to/port_server
make build
```

Then either:
- Add `bin/` to your PATH: `export PATH="/path/to/port_server/bin:$PATH"`
- Install to a system path: `cp bin/portctl /usr/local/bin/`
- Run `make install-skill` to set up the skill with the correct paths
