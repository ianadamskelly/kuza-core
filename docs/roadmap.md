# Kuza Core Roadmap

## Phase 0: Foundation

- API service skeleton.
- Health and readiness endpoints.
- Environment-backed configuration.
- Docker Compose for API, PostgreSQL, and MinIO.
- Initial PostgreSQL schema.
- Architecture notes.

## Phase 1: Identity And Tenancy

- Migration runner.
- Database connection pool.
- Users and organizations CRUD.
- Password hashing.
- Session or JWT authentication.
- Membership checks for every organization-scoped route.
- Seed script for first owner account.

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
