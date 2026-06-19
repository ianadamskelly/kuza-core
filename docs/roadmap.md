# Kuza Core Roadmap

## Phase 0: Foundation

- API service skeleton. Done.
- Health and readiness endpoints. Done.
- Environment-backed configuration. Done.
- Docker Compose for API, PostgreSQL, and MinIO. Done.
- Embedded migration runner. Done.
- First owner bootstrap. Done.
- Architecture notes. Done.

## Phase 1: Identity And Tenancy

- Database connection pool. Done.
- Projects list/create API. Done.
- Users list/create API. Done.
- Project update/delete APIs.
- Password hashing. Done.
- Bearer session authentication. Done.
- Current user endpoint. Done.
- Project membership list/add APIs. Done.
- Membership checks for project member routes. Done.
- Seed script for first owner account. Done.

## Phase 2: Generic Project Data

- Project table definitions. Done.
- Project JSON records. Done.
- Project API keys. Done.
- Basic table read/write access policies. Done.
- Record update/delete APIs. Done.
- Basic field validation from table schema. Done.
- Project file metadata and upload intents. Done.
- S3/MinIO presigned URLs. Done.
- Optional object proxying for deployments that do not expose object storage directly.
- Audit events for sensitive actions.

## Phase 3: Operations

- Backup and restore scripts.
- Admin dashboard API.
- Structured request logging.
- Rate limiting.
- Production Dockerfile.
- Deployment guide for a single VPS.

## Phase 4: Platform Capabilities

- Reusable SDK.
- Webhooks.
- Background jobs.
- Optional realtime events.
- Policy engine.
- Multi-project admin console.
