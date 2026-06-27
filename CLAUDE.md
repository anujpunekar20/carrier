# Carrier

Job aggregation and application management platform. Scrapes jobs from multiple websites, filters them, and provides a dashboard to search, track, and manage applications.

## Stack

- **Backend**: Go 1.25, Fiber v3, Ent ORM, PostgreSQL
- **Frontend**: Svelte (not yet started)
- **Module**: `github.com/anujpunekar20/carrier`

## Project Structure

```
cmd/api/main.go          — server entry point, graceful shutdown
internal/
  config/config.go       — loads .env into Config struct
  database/postgres.go   — opens *ent.Client (runs auto-migration on startup)
  ent/                   — generated Ent ORM code (do not edit manually)
    schema/job.go        — Job entity schema definition (edit this, then regenerate)
  handlers/jobs.go       — Fiber HTTP handlers
  services/jobs.go       — business logic; injected with *ent.Client
  routes/routes.go       — route registration
```

## Environment

Copy and fill in `.env` at the project root:

```
DB_HOST=localhost
DB_PORT=5432
DB_USER=anuj
DB_NAME=carrier
DB_PASSWORD=password
```

## Build & Run

```bash
# Build
go build ./...

# Run (requires PostgreSQL running and .env configured)
go run ./cmd/api/

# Server starts on :3000
# Schema migrations run automatically on startup
```

## Regenerate Ent Code

After editing `internal/ent/schema/job.go`:

```bash
go generate ./internal/ent/...
```

## Git Workflow

All changes go on a feature branch. Open a PR and wait for explicit review approval before merging to main.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/jobs/list` | List jobs (supports `q`, `company`, `location`, `employment_type`, `source`, `page`, `limit`) |
| GET | `/api/v1/jobs/:id` | Get job by ID |
| POST | `/api/v1/jobs` | Create a job |
| DELETE | `/api/v1/jobs/:id` | Delete a job |

## Job Schema Fields

| Field | Type | Notes |
|-------|------|-------|
| id | int | auto-increment PK |
| title | string | required |
| company | string | required |
| url | string | required, unique |
| source | string | required (scraper origin) |
| scraped_at | time | required |
| location | string | optional |
| salary | string | optional, stored as string (formats vary) |
| employment_type | string | optional |
| description | text | optional |
| posted_on | time | optional |

## What's Implemented

- [x] Fiber server with graceful shutdown
- [x] PostgreSQL connection via Ent ORM
- [x] Auto-migration on startup
- [x] Job schema with 10 fields
- [x] Full Job CRUD API (list with filter/search/paginate, get by ID, create, delete)

## What's Planned (in order)

- [ ] Middleware (error handling, logging, CORS, panic recovery)
- [ ] Application tracking (`status` + `notes` fields on Job: saved → applied → interview → offer → rejected)
- [ ] Scraper (interface + runner + per-site implementations)
- [ ] Docker + docker-compose
- [ ] Tests
- [ ] Svelte frontend dashboard

## Dependency Injection Pattern

All layers follow constructor injection — no globals. The chain is:

```
*ent.Client → JobService → JobHandler → routes.Register → fiber.App
```

New services and handlers should follow the same pattern.
