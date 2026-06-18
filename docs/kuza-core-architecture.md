# Kuza Core Architecture

## Product-Specific First

Kuza Core starts as the backend for Kuza Kizazi, not as a generic backend-as-a-service clone. The platform boundary should emerge from repeated product needs.

The first domain modules are:

- Identity: users, sessions, roles, memberships.
- Schools and organizations: the tenant boundary for data ownership.
- Learners and guardians: education-specific relationships.
- Content: lessons, resources, media, and publishing workflows.
- Storage: S3-compatible file metadata and object storage.
- Audit: trace important changes and admin activity.

## Platform Later

The platform path is still protected by a few early choices:

- PostgreSQL is the source of truth.
- Every product record belongs to a clear organization boundary where appropriate.
- Domain APIs sit in front of the database instead of exposing raw tables directly.
- Storage is S3-compatible so local MinIO and hosted object stores are interchangeable.
- Operational concerns like audit logs, backups, and health checks are built early.

## Non-Goals For The First Version

- Generic table editor.
- Full Supabase-compatible APIs.
- Realtime subscriptions for every table.
- Plugin marketplace.
- Multi-cloud orchestration.
- Complex policy UI.

## Initial Runtime

```text
client apps
   |
Kuza Core API
   |
PostgreSQL
   |
MinIO / S3
```

Background workers, queues, search, and realtime can be added as the workflows demand them.

## Design Principles

- Keep business rules in application code until a database policy clearly earns its place.
- Prefer explicit product APIs over generic database exposure.
- Make local development and self-hosting obvious.
- Keep deployment small enough for one VPS.
- Add platform abstractions only when the product has repeated the same need twice.
