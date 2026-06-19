# Kuza Core

Kuza Core is a self-hostable backend platform for building many kinds of apps without rebuilding backend plumbing each time.

Create a project, define the data tables that project needs, connect a frontend, and run the backend yourself. A CV builder, school app, jobs board, directory, or internal tool should all be able to use the same Kuza Core foundation.

## Goals

- Own the core data and deployment path.
- Run locally or on a small VPS with Docker.
- Keep PostgreSQL as the source of truth.
- Make projects, auth, roles, files, audits, and operational tools first-class.
- Provide generic project data APIs so each frontend can bring its own domain model.

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

With PostgreSQL configured:

```sh
curl -X POST http://localhost:8080/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"owner@example.com","password":"change-me-before-production"}'

curl http://localhost:8080/v1/projects

curl -X POST http://localhost:8080/v1/projects \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"name":"CV Builder","slug":"cv-builder","template":"blank"}'

curl http://localhost:8080/v1/auth/me \
  -H 'Authorization: Bearer <token>'

curl http://localhost:8080/v1/users \
  -H 'Authorization: Bearer <token>'

curl -X POST http://localhost:8080/v1/users \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"email":"builder@example.com","display_name":"Builder","password":"change-me"}'

curl http://localhost:8080/v1/projects/<project-id>/members \
  -H 'Authorization: Bearer <token>'

curl -X POST http://localhost:8080/v1/projects/<project-id>/members \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"user_id":"<user-id>","role":"developer"}'

curl -X POST http://localhost:8080/v1/projects/<project-id>/api-keys \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"name":"Frontend client"}'

curl -X POST http://localhost:8080/v1/projects/<project-id>/tables \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{"name":"profiles","read_access":"api_key","write_access":"api_key","schema":{"required":["name"],"fields":{"name":"text","headline":"text"}}}'

curl -X POST http://localhost:8080/v1/projects/<project-id>/tables/profiles/records \
  -H 'X-Kuza-API-Key: <api-key>' \
  -H 'Content-Type: application/json' \
  -d '{"data":{"name":"Ian","headline":"Educator"}}'

curl http://localhost:8080/v1/projects/<project-id>/tables/profiles/records \
  -H 'X-Kuza-API-Key: <api-key>'

curl -X PATCH http://localhost:8080/v1/projects/<project-id>/tables/profiles/records/<record-id> \
  -H 'X-Kuza-API-Key: <api-key>' \
  -H 'Content-Type: application/json' \
  -d '{"data":{"name":"Ian","headline":"Founder"}}'

curl -X DELETE http://localhost:8080/v1/projects/<project-id>/tables/profiles/records/<record-id> \
  -H 'X-Kuza-API-Key: <api-key>'
```

Project table access policies are:

- `project_members`: project members only.
- `api_key`: project members or a project API key.
- `public`: no token required.

Project table schemas currently support:

- `required`: list of required field names.
- `fields`: map of field names to simple types: `text`, `number`, `boolean`, `object`, or `array`.

If `KUZA_CORE_DATABASE_URL` is set, the API connects to PostgreSQL, runs embedded migrations, and can bootstrap the first owner account from:

- `KUZA_CORE_BOOTSTRAP_PROJECT_NAME`
- `KUZA_CORE_BOOTSTRAP_PROJECT_SLUG`
- `KUZA_CORE_BOOTSTRAP_OWNER_EMAIL`
- `KUZA_CORE_BOOTSTRAP_OWNER_PASSWORD`

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

This is the foundation slice: API skeleton, health/readiness routes, deployment shape, PostgreSQL connection, embedded migrations, first-owner bootstrap, bearer sessions, project APIs, users, memberships, API keys, table policies, generic project data tables/records, record update/delete, and basic schema validation. Storage and richer permissions come next.
