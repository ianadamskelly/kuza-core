# Kuza Core Roadmap

## Phase 0: Foundation

- API service skeleton. Done.
- Health and readiness endpoints. Done.
- Environment-backed configuration. Done.
- Docker Compose for API, PostgreSQL, and MinIO. Done.
- Initial PostgreSQL schema. Done.
- Embedded migration runner. Done.
- First owner bootstrap. Done.
- Architecture notes. Done.

## Phase 1: Identity And Tenancy

- Database connection pool. Done.
- Organizations list/create API. Done.
- Users list/create API. Done.
- Organization update/delete APIs.
- Password hashing. Done.
- Bearer session authentication. Done.
- Current user endpoint. Done.
- Organization membership list/add APIs. Done.
- Membership checks for organization member routes. Done.
- Seed script for first owner account. Done.

## Phase 2: Product Workflows

- Learner profiles.
- Guardian relationships.
- Staff and teacher roles.
- Content item publishing workflow.
- File upload flow using S3-compatible storage.
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
- Multi-organization admin console.
