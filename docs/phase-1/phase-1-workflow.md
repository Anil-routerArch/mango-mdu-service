# Mango MDU Service — Phase 1 Workflow: Policy Overview

## 1. Overview

This document describes the runtime execution flow for Phase 1 of `mango-mdu-service`.

The focus of this phase is delivering the **Policy Overview API** to power the `Users & Access -> Policies -> Overview` tab in `mango-operator-ui`, alongside core operational system routes (`/livez`, `/api/v1/system`).

---

## 2. Policy Overview Workflow (`GET /api/v1/policies/{policyId}/overview`)

### Step 1: Inbound Request Handling
- The Operator UI sends `GET /api/v1/policies/{policyId}/overview` with an `Authorization: Bearer <token>` header.
- Optional tracing headers (`X-Request-Id`, `X-Correlation-Id`) are parsed or generated if absent.
- The authentication middleware validates the bearer token against OWSEC (`AUTH_ENABLED=true`).
- If token validation fails, a normalized `401 Unauthorized` is returned immediately.

### Step 2: Policy Details Resolution
- MDU calls OWPROV: `GET /api/v1/managementPolicy/{policyId}`.
- Propagates downstream headers according to the master architecture contract:
  - `x-api: <mdu-service-api-key>` (identifies MDU Service as a trusted internal microservice)
  - `x-authorization: Bearer <owsec-token>` and `Authorization: Bearer <owsec-token>` (forwards user token for OWPROV RBAC evaluation)
  - `x-request-id` and `x-correlation-id` (preserves distributed trace context)
- If the policy does not exist in OWPROV, MDU returns `404 Not Found`.
- Policy metadata (Name, Description, Modified/Created timestamp) is extracted.

### Step 3: Management Roles & Scope Aggregation
- MDU queries OWPROV for management roles using the established service auth (`x-api`) and forwarded user context (`x-authorization` / `Authorization`):
  - Where supported downstream, MDU queries by policy filter: `GET /api/v1/managementRole?managementPolicy={policyId}`.
  - Alternatively, MDU retrieves roles exhaustively (either omitting `limit` or paginating via `offset` and `limit` until the full result set is exhausted) to prevent silent truncation.
- OWPROV evaluates the forwarded user token to authoritatively resolve caller permissions and user-scoped role visibility.
- MDU filters all matching management roles that reference `policyId` (by matching role's `managementPolicy` ID or name).
- For each matching role:
  - Collects `role.id`, `role.entity` (Property ID), and `role.venue` (Venue ID).
  - Collects all user IDs in `role.users[]`.
- Calculates the aggregate summary counts:
  - `usedByUsers`: count of unique user IDs across all matching roles.
  - `scopedAssignmentsCount`: total count of matching management roles (scoped assignments).
  - `propertiesCount`: count of unique non-empty entity IDs.
  - `venuesCount`: count of unique non-empty venue IDs.

### Step 4: User & Scope Entity Enrichment
- Resolves entity names from OWPROV (`GET /api/v1/entity`) and venue names from OWPROV (`GET /api/v1/venue`) using the authenticated downstream client.
- Resolves user profiles (display name and email) from OWSEC (`GET /api/v1/users`).
- Groups role assignments by unique user ID:
  - Each unique user entry contains `id`, `name`, `email`, and a `scopes[]` array.
  - Each item in `scopes[]` captures one assignment scope: `roleId`, `propertyId`, `propertyName`, `venueId`, and human-readable `venueScope` ("Whole property" if venue is empty/null, or the specific venue name).
  - If a user has multiple scoped assignments (e.g., across multiple towers or properties), each assignment is preserved as an entry in `scopes[]` without dropping scope details or duplicating top-level user metadata.
- Populates the `assignedUsers[]` list (`assignedUsers.length == summary.usedByUsers`).

### Step 5: Response Composition
- Formats the consolidated response according to `docs/phase-1/mango-mdu-openapi.yaml`.
- Returns `200 OK` with JSON payload.
