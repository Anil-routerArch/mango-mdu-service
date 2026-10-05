# Mango MDU Service — Phase 1 Workflow: Policy Overview

## 1. Overview

This document describes the runtime execution flow for Phase 1 of `mango-mdu-service`.

The focus of this phase is delivering the **Policy Overview API** to power the `Users & Access -> Policies -> Overview` tab in `mango-operator-ui`, alongside core operational system routes (`/livez`, `/api/v1/system`).

---

## 2. Policy Overview Workflow (`GET /api/v1/policy/{id}/overview`)

### Step 1: Inbound Request Handling
- The Operator UI sends `GET /api/v1/policy/{id}/overview` with an `Authorization: Bearer <token>` header.
- Optional tracing headers (`X-Request-Id`, `X-Correlation-Id`) are parsed or generated if absent.
- The authentication middleware validates the bearer token against OWSEC (`AUTH_ENABLED=true`).
- If token validation fails, a normalized `401 Unauthorized` is returned immediately.

### Step 2: Policy Details Resolution
- MDU calls OWPROV: `GET /api/v1/managementPolicy/{id}`.
- Propagates downstream headers:
  - `Authorization: Bearer <owsec-token>` (forwards caller token for OWPROV RBAC evaluation)
  - `X-API-KEY: <mdu-service-api-key>` (identifies MDU Service as a trusted internal microservice)
  - `X-Request-Id` and `X-Correlation-Id` (preserves distributed trace context)
- If the policy does not exist in OWPROV, MDU returns `404 Not Found`.
- Policy metadata (`id`, `name`, `description`, `entity`, `venue`, `created`, `modified`) is extracted.

### Step 3: Management Roles & Scope Aggregation
- MDU queries OWPROV for management roles using the service authentication (`X-API-KEY`) and forwarded user context (`Authorization: Bearer <owsec-token>`).
- Leverages OWPROV's native `policyId` query filter to push role filtering directly to the database:
  ```http
  GET /api/v1/managementRole?policyId={id}&offset={offset}&limit={limit}
  ```
- OWPROV evaluates the forwarded user token to authoritatively resolve caller permissions and user-scoped role visibility, returning only roles matching `managementPolicy == id`.
- For policies with many assignments exceeding page limit (OWPROV defaults to `limit=100` when omitted), MDU uses chunked pagination (`limit=500`) starting at `offset=0` and continuing while `returned_count == limit` to ensure complete retrieval without truncation.
- Aggregation is performed only after the complete visible role set has been retrieved.
- For each matching role:
  - Collects `role.id`, `role.entity` (Property ID), and `role.venue` (Venue ID).
  - Collects all user IDs in `role.users[]`.
- Calculates the aggregate summary counts:
  - `totalUsers`: count of unique user IDs across all matching roles.
  - `totalScopedAssignments`: total count of matching management roles (scoped assignments).
  - `totalProperties`: count of unique non-empty entity IDs.
  - `totalVenues`: count of unique non-empty venue IDs.

### Step 4: User & Scope Entity Enrichment
- Resolves entity names from OWPROV (`GET /api/v1/entity`) and venue names from OWPROV (`GET /api/v1/venue`) using the authenticated downstream client.
- Resolves user profiles (display name, email, userRole, avatar) from OWSEC (`GET /api/v1/users`) forwarding `Authorization: Bearer <owsec-token>` and distributed tracing headers.
- Handles orphaned users (users deleted from OWSEC whose UUIDs remain in OWPROV roles):
  - Instead of failing the request or dropping the user, MDU populates the user record with fallback values: `name: "Deleted User"`, `email: "deleted@example.invalid"`, `userRole: "unknown"`, `avatar: ""`.
  - Preserves the user's role assignments in `scopes[]`, enabling operators to identify and clean up orphaned assignments.
  - Ensures the invariant `usersWithPolicy.length == totalUsers` is strictly maintained.
  - (Note: Future lifecycle enhancements will implement cascaded role cleanup in OWPROV upon user deletion).
- Groups role assignments by unique user ID:
  - Each unique user entry contains `id`, `name`, `email`, `userRole`, `avatar`, `scopedAssignmentsCount`, and a `scopes[]` array.
  - Each item in `scopes[]` captures one assignment scope: `entityId`, `entityName`, `venueId`, and human-readable `venueName` ("All venues" if venue is empty/null, or the specific venue name).
  - If a user has multiple scoped assignments (e.g., across multiple towers or properties), each assignment is preserved as an entry in `scopes[]` without dropping scope details or duplicating top-level user metadata.
- Populates the `usersWithPolicy[]` list (`usersWithPolicy.length == totalUsers`).

### Step 5: Response Composition
- Formats the consolidated response according to `docs/phase-1/mango-mdu-openapi.yaml` with the `policy` object, KPI counts (`totalUsers`, `totalScopedAssignments`, `totalProperties`, `totalVenues`), and `usersWithPolicy[]`.
- Returns `200 OK` with JSON payload.
