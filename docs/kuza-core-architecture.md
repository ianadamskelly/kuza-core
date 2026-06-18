# Kuza Core Architecture

## Backend Platform First

Kuza Core is a self-hostable backend platform. It should let a frontend spin up a backend instance, create a project, define project data, and start building without writing a custom backend for every app.

The first platform modules are:

- Identity: users, sessions, roles, memberships.
- Projects: the app and tenant boundary for data ownership.
- Project data: generic tables and JSON records owned by a project.
- Storage: S3-compatible file metadata and object storage.
- Audit: trace important changes and admin activity.

## Product Templates Later

School systems, CV builders, jobs boards, directories, and other apps should become templates or starter kits on top of Kuza Core. They should not be hardcoded into the core schema.

The platform path is protected by a few early choices:

- PostgreSQL is the source of truth.
- Every app record belongs to a project boundary.
- Generic project APIs sit in front of the database.
- Storage is S3-compatible so local MinIO and hosted object stores are interchangeable.
- Operational concerns like audit logs, backups, and health checks are built early.

## Non-Goals For The First Version

- Full Supabase-compatible APIs.
- Realtime subscriptions for every table.
- Plugin marketplace.
- Multi-cloud orchestration.
- Complex policy UI.
- App-specific tables in the core schema.

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
- Prefer explicit project APIs over raw database exposure.
- Make local development and self-hosting obvious.
- Keep deployment small enough for one VPS.
- Add platform abstractions when multiple app types need the same capability.
