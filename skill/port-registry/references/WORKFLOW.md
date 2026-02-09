# Port Registry Workflow Reference

## Multi-Service Allocation

When a project needs multiple services (e.g., a typical web stack), allocate all ports upfront:

```bash
# Allocate ports for a full stack (--app and --instance auto-detected)
portctl allocate --service postgres
# -> allocated port 3042 (id=1) for myapp/main/postgres

portctl allocate --service redis
# -> allocated port 3043 (id=2) for myapp/main/redis

portctl allocate --service web
# -> allocated port 3044 (id=3) for myapp/main/web

portctl allocate --service api
# -> allocated port 3045 (id=4) for myapp/main/api
```

Verify all allocations:

```bash
portctl list
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

### Host-Bound vs Container-Only Ports

When reading an existing `docker-compose.yml`, distinguish between port mapping styles:

```yaml
services:
  # FIXED HOST BINDING — register these with portctl
  postgres:
    ports:
      - "5432:5432"   # host 5432 -> container 5432: REGISTER 5432
  proxy:
    ports:
      - "80:80"       # host 80 -> container 80: REGISTER 80
  idp:
    ports:
      - "9100:8080"   # host 9100 -> container 8080: REGISTER 9100 (host port)

  # CONTAINER-ONLY — do NOT register these
  mailcatcher:
    ports:
      - "1025"         # random ephemeral host port -> container 1025: SKIP
  swapper:
    ports:
      - "8080"         # random ephemeral host port -> container 8080: SKIP
```

**Rule**: only `"host:container"` format binds a fixed host port. A bare `"port"` lets Docker pick a random host port, so there is no conflict risk to track.

### Using .env Files

For projects that use `.env` files with docker-compose:

```bash
# Allocate and capture ports (--app and --instance auto-detected)
PG_PORT=$(portctl list --service postgres --json | jq '.[0].port')
REDIS_PORT=$(portctl list --service redis --json | jq '.[0].port')
WEB_PORT=$(portctl list --service web --json | jq '.[0].port')

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

Use the `--instance` flag to run parallel environments without conflicts. When using git worktrees, `--instance` is auto-detected from the worktree directory name. On the main worktree, it defaults to the branch name.

```bash
# Developer A working on feature-auth worktree (--instance auto-detected as "feature-auth")
portctl allocate --service postgres
portctl allocate --service web

# Developer B working on feature-payments worktree (--instance auto-detected as "feature-payments")
portctl allocate --service postgres
portctl allocate --service web

# Override --instance explicitly if needed
portctl allocate --instance staging --service postgres

# Each gets unique ports, no conflicts
portctl list
```

## Registering Ports From an Existing Project

When a user asks you to register ports for an existing project, scan all port sources:

1. **docker-compose.yml / docker-compose.*.yml** — look for `ports:` with `"host:container"` format only
2. **package.json / npm scripts** — look for `--port`, `-p`, or hardcoded ports in dev/start/test scripts
3. **.env files** — look for `*_PORT` variables used by host-side services
4. **Makefile / scripts/** — look for port bindings in dev tooling

Register each host-bound port with `--port <N>`:

```bash
# Docker-exposed ports (--app and --instance auto-detected)
portctl allocate --service postgres --port 5432
portctl allocate --service proxy-nginx --port 80

# npm script ports
portctl allocate --service storybook --port 6006
portctl allocate --service nodejs-dev --port 3001
```

Do **not** register container-only ports (bare `"3001"` in docker-compose) unless the same port is also used by a host-side process like `npm run dev`.

## Cleanup Patterns

### Release all ports for a project

```bash
portctl release
```

### Release a specific instance

```bash
portctl release --instance dev
```

### Release a single service (--instance auto-detected)

```bash
portctl release --service postgres
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

The port-registry is not running. Start it:

```bash
port-registry &
```

Or if not on PATH:

```bash
~/path/to/port-registry/bin/port-registry &
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

Build from the port-registry repository:

```bash
cd /path/to/port-registry
make build
```

Then either:
- Add `bin/` to your PATH: `export PATH="/path/to/port-registry/bin:$PATH"`
- Install to a system path: `cp bin/portctl /usr/local/bin/`
- Run `make install-skill` to set up the skill with the correct paths
