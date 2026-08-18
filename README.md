# AeroXe LaundryPro

> Phase 1 | AeroXe Ecosystem

---

## Table of Contents

- [What Is This Project?](#what-is-this-project)
- [Why Was It Built?](#why-was-it-built)
- [When Should You Use It?](#when-should-you-use-it)
- [Where Does It Run?](#where-does-it-run)
- [How Does It Work?](#how-does-it-work)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Service Modules](#service-modules)
- [Saga Orchestrator](#saga-orchestrator)
- [WebSocket Events](#websocket-events)
- [Database Schema](#database-schema)
- [Setup & Installation](#setup--installation)
- [Environment Variables](#environment-variables)
- [Development](#development)
- [Testing](#testing)
- [Deployment](#deployment)

---

## What Is This Project?

AeroXe LaundryPro is a laundry service management platform that handles booking, pickup scheduling, order processing, garment tracking, delivery management, and payment collection for laundry businesses.

---

## Why Was It Built?

Laundry businesses struggle with order management, delivery scheduling, and customer communication. LaundryPro digitizes the entire workflow — from booking and pickup to washing, ironing, and delivery — improving operational efficiency and customer satisfaction.

---

## When Should You Use It?

Use LaundryPro for laundry businesses, dry cleaners, garment care services, or any on-demand laundry service requiring end-to-end order and delivery management.

---

## Where Does It Run?

Backend runs Go/Hertz with PostgreSQL for orders and customer data, Redis for caching/sessions/saga state, and NATS for event delivery (durability via the transactional outbox). WebSocket pushes real-time order status updates. The web client ships in this repo: a React 19 admin console (operations) and a public marketing landing page, both in `./frontend`. Android/iOS clients are planned but not yet in this repo.

---

## How Does It Work?

LaundryPro implements BookingService, PickupService, DeliveryService, OrderService, PaymentService, InventoryService, AnalyticsService, and NotificationService. BookingFlow saga handles: booking → pickup scheduling → pickup confirmation → order processing → quality check → delivery scheduling → delivery confirmation. PaymentCollection saga processes payment on delivery or prepayment.

---

## Architecture

### High-Level Architecture

```
+---------------------------------------------------------------------+
|                          CLIENT LAYER                               |
|  +----------+  +------------------+  +-----------------------+     |
|  | React    |  | Android          |  | iOS                   |     |
|  | (Web)    |  | (Kotlin+Compose) |  | (SwiftUI)             |     |
|  +----+-----+  +--------+---------+  +----------+------------+     |
|       |                 |                        |                  |
+-------+-----------------+------------------------+------------------+
        |                 |                        |
        v                 v                        v
+---------------------------------------------------------------------+
|                       API GATEWAY (Hertz)                           |
|  +-------------+  +----------------------------+                  |
|  | HTTP REST   |  | WebSocket Hub              |                  |
|  | Routes      |  | (coder/websocket)          |                  |
|  +------+------+  +----------------------------+                  |
|         |                        |                                |
|  +------v------------------------v-------------------------+      |
|  |  Auth | Rate Limit | Circuit Breaker | Logging          |      |
|  +---------------------------------------------------------+      |
+----------------------+---------------------+---------------------+
                       |                     |
     +------------------------+  +--------v--------+
     |  NATS (async)          |  |  WebSocket      |
     |  pub/sub + outbox      |  |  real-time      |
     +------------+-----------+  +--------+--------+
                  |                      |
+-----------------v----------------------v---------------+
|                  MODULAR MONOLITH BACKEND                          |
|  +------------+ +------------+ +------------+ +------------+     |
|  | Module A   | | Module B   | | Module C   | | Module D   |     |
|  +-----+------+ +-----+------+ +-----+------+ +-----+------+     |
|        |              |              |              |              |
|  +-----v--------------v--------------v--------------v------+      |
|  |  +----------+ +-------+ +--------+ +--------------+    |      |
|  |  | Postgres | | Redis | |  NATS  | | Saga Engine  |    |      |
|  |  +----------+ +-------+ +--------+ +--------------+    |      |
|  +---------------------------------------------------------+      |
+--------------------------------------------------------------------+
```

### Tech Stack

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **HTTP Framework** | [Hertz](https://github.com/cloudwego/hertz) | High-performance HTTP server |
| **Messaging** | [NATS](https://nats.io/) core pub/sub | Async event delivery; durability comes from the transactional outbox table |
| **Database** | [PostgreSQL](https://www.postgresql.org/) 15+ | Primary data store |
| **Cache** | [Redis](https://redis.io/) 7+ | Caching, sessions, saga state, rate limiting |
| **WebSocket** | [coder/websocket](https://github.com/coder/websocket) | Real-time communication |
| **Frontend** | React 19 + Vite + Tailwind v4 | Web app (`./frontend`): public landing at `/`, admin console at `/app`, TanStack Query + Redux |
| **Android** | Kotlin + Jetpack Compose + Hilt | Android application |
| **iOS** | Swift 5.9+ + SwiftUI | iOS application |

---

## Service Modules

| Module | Description | Protocol |
|--------|-------------|----------|
| BookingService | Core booking operations | HTTP + NATS |
| PickupService | Core pickup operations | HTTP + NATS |
| DeliveryService | Core delivery operations | HTTP + NATS |
| OrderService | Core order operations | HTTP + NATS |
| PaymentService | Core payment operations | HTTP + NATS |
| InventoryService | Core inventory operations | HTTP + NATS |
| AnalyticsService | Core analytics operations | HTTP |
| NotificationService | Core notification operations | HTTP + WebSocket |

---

## Saga Orchestrator

| Saga | Pattern |
|------|---------|
| BookingFlow | Orchestrated via NATS + Redis state |
| PickupDelivery | Orchestrated via NATS + Redis state |
| OrderProcessing | Orchestrated via NATS + Redis state |
| PaymentCollection | Orchestrated via NATS + Redis state |

---

## WebSocket Events

| Event | Description |
|-------|-------------|
| `booking_confirmed` | Real-time updates for booking confirmed |
| `pickup_scheduled` | Real-time updates for pickup scheduled |
| `order_ready` | Real-time updates for order ready |
| `delivery_update` | Real-time updates for delivery update |

---

## Database Schema

| Table | Description | Key Fields |
|-------|-------------|------------|
| Booking | `bookings` table | UUID, timestamps, soft delete |
| Pickup | `pickups` table | UUID, timestamps, soft delete |
| Delivery | `deliverys` table | UUID, timestamps, soft delete |
| Order | `orders` table | UUID, timestamps, soft delete |
| Payment | `payments` table | UUID, timestamps, soft delete |
| ServiceType | `servicetypes` table | UUID, timestamps, soft delete |
| Garment | `garments` table | UUID, timestamps, soft delete |
| Invoice | `invoices` table | UUID, timestamps, soft delete |
| Rating | `ratings` table | UUID, timestamps, soft delete |
| AuditLog | `auditlogs` table | UUID, timestamps, soft delete |

### Redis Usage

| Key Pattern | Purpose | TTL |
|------------|---------|-----|
| `session:<user_id>` | User session | 24h |
| `cache:<slug>:<id>` | Entity cache | 15m |
| `saga:<saga_id>` | Saga state | Until completion |
| `ratelimit:<ip>` | Rate limiting | 1m |

---

## Setup & Installation

### Prerequisites

- Go 1.22+
- PostgreSQL 15+
- Redis 7+
- NATS Server 2.10+
- Docker & Docker Compose (optional)

### Quick Start

```bash
# Backend lives in ./backend
cd backend

# Start infrastructure services (postgres, redis, nats)
make up

# Apply schema migrations (idempotent; also runs on container boot)
make migrate

# Start the Go server
make run

# Admin is bootstrapped automatically on first boot (see ADMIN_* env vars).
# OpenAPI docs: http://localhost:8080/swagger/index.html
```

### Docker Compose

The backend ships a production-grade compose stack (`backend/docker-compose.yml`):
Postgres 16, Redis 7, NATS, the app (migrations run on boot, unprivileged
user, readiness HEALTHCHECK), the **web SPA** (built from `../frontend`, nginx
serving `dist/` with immutable asset caching and SPA fallback), a Caddy
reverse proxy that terminates TLS (automatic Let's Encrypt when `APP_DOMAIN`
is set), and Prometheus scraping `/metrics`. Caddy routes `/api/*` and
`/swagger/*` to the app, everything else to the SPA, and keeps `/metrics` off
the public edge. Run it with:

```bash
cd backend
export JWT_SECRET=... ENCRYPTION_KEY=... ADMIN_PASSWORD=...
docker compose up -d --build
```

> **Local access**: Caddy auto-redirects HTTP → HTTPS. With the default
> `APP_DOMAIN=localhost` it terminates TLS with its own internal CA, so use
> `https://localhost` and accept the certificate (or install Caddy's local
> CA). In production set `APP_DOMAIN` to your real domain for Let's Encrypt.
> For day-to-day UI work prefer `npm run dev` (Vite on `:5173`).

### Frontend (React SPA)

The web app lives in `./frontend` and ships two surfaces on one SPA:

- `/` — public marketing landing page (hero, services, pricing, FAQ, CTA)
- `/app` — the authenticated admin console (bookings, orders, payments, RBAC, …)
- `/login` — staff sign-in (redirects to `/app`)

```bash
cd frontend
npm install
npm run dev        # http://localhost:5173 (proxies /api -> http://localhost:9090)
npm run typecheck  # tsc -b --noEmit
npm run build      # production bundle -> dist/
```

The SPA talks to the backend through a same-origin `/api` prefix (the Vite dev
proxy, or Caddy/nginx in production). `VITE_API_BASE_URL` defaults to `/api/v1`;
see `frontend/.env.example`.

> **Production serving**: `docker compose up -d --build` builds and serves the
> SPA (`web` service) behind Caddy alongside the API — `/api/*` and
> `/swagger/*` proxy to the backend, everything else is the SPA with client
> routes falling back to `index.html`. Same-origin requests need no CORS; set
> `CORS_ORIGINS` only if other browser origins call the API directly.

---

## Environment Variables

See `backend/.env.example` for the full annotated list and
`backend/.env.production.example` for the production template. Key groups:

```bash
# Server
PORT=8080
ENV=development            # set to production to enable fail-fast validation
LOG_LEVEL=debug
LOG_FORMAT=text            # json in production

# PostgreSQL
DATABASE_URL=postgres://aeroxe:secret@localhost:5433/laundry_pro?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379
REDIS_PREFIX=laundry-pro

# NATS (optional; the app degrades to a no-op publisher when down)
NATS_URL=nats://localhost:4222
NATS_ENABLED=false

# Security (>=32 chars in production; weak values fail fast when ENV=production)
JWT_SECRET=your-secret-key
JWT_EXPIRY=24h
ENCRYPTION_KEY=your-encryption-key
PASSWORD_COST=12

# Bootstrap admin (created on first boot)
ADMIN_EMAIL=admin@laundrypro.local
ADMIN_PASSWORD=Admin@12345

# Operations
RATE_LIMIT_PER_MIN=120
OUTBOX_POLL_INTERVAL=2s
OUTBOX_BATCH_SIZE=50
OUTBOX_MAX_RETRIES=5
MAX_BODY_BYTES=5242880
CORS_ORIGINS=
SHUTDOWN_TIMEOUT=15s
```

---

## Development

### Makefile Targets (in `backend/`)

```makefile
make run          # go run cmd/server/main.go
make build        # build bin/server
make test         # go test ./... -cover
make test-unit    # infra packages only
make test-integration  # ./tests/... -tags=integration (needs a running server)
make vet          # go vet ./...
make migrate      # apply schema migrations only
make swagger      # regenerate OpenAPI docs
make up           # docker compose up postgres redis nats
make up-all       # full stack incl. caddy (TLS) + prometheus + blackbox
make observability # caddy + prometheus + blackbox only
make load-test    # k6 booking-flow load test (via Docker)
make backup / make restore BACKUP=backups/<dir>
```

---

## Testing

```bash
# Unit tests (default; 12 packages incl. crypto, lock, pagination, outbox,
# migrations, rate limiting, auth, booking, order, delivery, payment, flows)
go test ./... -count=1

# Integration tests against a running server (docker compose stack up):
BASE_URL=http://localhost:9090/api/v1 \
  go test ./tests/... -tags=integration -count=1 -v

# Coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Deployment

### Production Checklist (backend)

- **Secrets**: `ENV=production` fails fast if `JWT_SECRET` / `ENCRYPTION_KEY` are
  weaker than 32 chars, if the admin password is a default, or if
  `PASSWORD_COST < 12`. Provide unique values (secrets manager recommended).
- **CORS**: set `CORS_ORIGINS=https://app.example.com` (comma-separated) for
  browser clients. Empty in production denies all browser origins.
- **Probes**: `/api/v1/health` is liveness; `/api/v1/health/ready` pings
  Postgres + Redis and returns `503` when a dependency is down. Wire these to
  your orchestrator/load balancer.
- **Logging**: `LOG_FORMAT=json` emits structured lines; `LOG_LEVEL=info` is
  the production default.
- **Graceful shutdown**: SIGTERM drains in-flight requests for up to
  `SHUTDOWN_TIMEOUT` (default 15s) before exiting; the outbox relay and NATS
  publisher are closed during shutdown.
- **Migrations**: `./bin/server -migrate-only` applies schema changes without
  serving traffic (run as a job before rolling out new replicas).
- **Backups**: snapshot the `pgdata` volume (or use managed Postgres PITR) and
  export Redis with `SAVE`/RDB; restore order: Postgres -> Redis.

### Container

```bash
cd backend
export JWT_SECRET=<unique 32+ chars>
export ENCRYPTION_KEY=<unique 32+ chars>
export ADMIN_PASSWORD=<strong password>
export APP_DOMAIN=api.example.com   # for automatic HTTPS via Caddy
export ACME_EMAIL=ops@example.com
docker compose up -d --build   # postgres, redis, nats, app, caddy, prometheus
```

- The image runs as an unprivileged user and includes a Docker `HEALTHCHECK`
  that calls the readiness probe.
- **Migrations on boot**: the container entrypoint applies pending schema
  migrations (`server -migrate-only`) before serving traffic. The runner is
  versioned and idempotent (see `schema_migrations`), so it is safe across
  replicas; for zero-downtime rollouts run `make migrate` as a separate job
  before scaling out.
- **TLS termination**: the `caddy` service terminates HTTPS in front of the
  app. With a real `APP_DOMAIN`, Caddy obtains Let's Encrypt certificates
  automatically. The app container is no longer published to the host — all
  traffic (including health probes and swagger) goes through the proxy on
  port 80/443.
- **Metrics**: the `prometheus` service scrapes `app:8080/metrics` (UI at
  `http://localhost:9091`). Prometheus includes HTTP latency/request
  counters, outbox gauges and the NATS circuit-breaker histogram.

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: laundry-pro
spec:
  replicas: 3
  selector:
    matchLabels:
      app: laundry-pro
  template:
    metadata:
      labels:
        app: laundry-pro
    spec:
      containers:
      - name: app
        image: aeroxe/laundry-pro:latest
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

---

## License

Copyright (c) 2026 AeroXe Enterprises Private Limited. All rights reserved.

---

*Built with love by the AeroXe Team*
