# Kuza Core

Kuza Core is the self-hostable backend foundation for Kuza Kizazi.

The first version is intentionally product-specific: schools, communities, learners, guardians, staff, content, files, and audit trails. The foundations are kept clean so it can later grow into a reusable education platform without forcing platform complexity into day one.

## Goals

- Own the core data and deployment path.
- Run locally or on a small VPS with Docker.
- Keep PostgreSQL as the source of truth.
- Make auth, roles, files, audits, and operational tools first-class.
- Grow toward platform capabilities only after the product workflows are stable.

## First Services

- `api`: Kuza Core HTTP API.
- `postgres`: relational database.
- `minio`: S3-compatible local object storage.

## Local Development

```sh
cp .env.example .env
go run ./cmd/kuzacore
```

The API listens on `http://localhost:8080` by default.

```sh
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:8080/v1
```

## Self-Hosting Shape

```sh
docker compose up
```

This starts the API, PostgreSQL, and MinIO. The compose file is a development baseline, not yet a hardened production profile.

## Project Layout

```text
cmd/kuzacore/          API entrypoint
internal/config/      environment-backed configuration
internal/httpapi/     HTTP routes and handlers
db/migrations/        PostgreSQL schema migrations
docs/                 architecture and roadmap notes
```

## Current Status

This is the foundation slice: API skeleton, health/readiness routes, deployment shape, and initial domain schema. Authentication, migrations runner, persistence code, and admin workflows come next.
