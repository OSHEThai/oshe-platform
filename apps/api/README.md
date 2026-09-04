# OSHE API Host & Local Container

## Overview

The `apps/api` package hosts the modular OSHE Platform HTTP API kernel and local container service. It provides the central HTTP ingress routing for safety inspection workflows, tenant enforcement, and walking-skeleton verification.

## Architecture & Governance

- **Trusted Tenant Enforcement:** All tenant-scoped routes are governed by `TenantMiddleware` (`tenant_middleware.go`). Tenant context is strictly derived from verified server-side claims; client-supplied tenant headers or query parameter overrides (`X-Tenant-ID`, etc.) are rejected with `403 Forbidden`.
- **Default-Deny Ingress:** All tenant API routes fail closed with `401 Unauthorized` unless an explicit local-synthetic authorization resolver is active (`OSHE_AUTH_MODE=synthetic`).
- **Health Endpoint:** Unauthenticated `/health` and `/healthz` endpoints provide service readiness and environment diagnostics without exposing protected tenant data.
- **In-Memory Walking Skeleton:** Implements the end-to-end synthetic inspection lifecycle (`walking_skeleton.go`) covering template publication, checklist instantiation, answer completion, photographic evidence hashing (`SHA-256`), corrective action tracking, report summaries, and audit trail reconstruction.

## Local Container Service (`deploy/local/compose.dev.yaml`)

- **Container Image:** Built via multi-stage Dockerfile (`apps/api/Dockerfile`) using digest-pinned Go toolchain builder and digest-pinned Alpine runtime (`alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d`).
- **Bound Loopback:** Exposed on `127.0.0.1:8080:8080` strictly for local developer testing. Public port binding is prohibited.
- **Healthcheck:** Configured via `wget` probe against `http://127.0.0.1:8080/health`.

## Operational Boundaries & Non-Claims

- **Synthetic-Only Profile:** This container operates strictly in a local-synthetic development profile.
- **No Live Persistence / Broker:** Real PostgreSQL clustering, NATS JetStream message brokers, and SeaweedFS S3 storage remain unexecuted and unverified in this slice.
- **No Production Credentials:** Zero customer data, production secrets, or public cloud endpoints are wired or claimed.
