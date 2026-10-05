# Mango MDU Service — Phase 1 Specification

## 1. Purpose

This document defines the Phase 1 specification for `mango-mdu-service`.

Phase 1 establishes MDU as the Mango-facing authenticated orchestration layer for the Mango Operator UI.
Its immediate focus is providing the live **Policy Overview API** required by the Operator UI (`Users & Access -> Policies -> Overview`), alongside standard operational system routes.

---

## 2. Phase 1 Goal & Scope

### In Scope for Phase 1:
1. **Policy Overview Orchestration**:
   - `GET /api/v1/policy/{id}/overview`
   - Orchestrates data across **OWPROV** (management policies, management roles, entities, venues) and **OWSEC** (user identity).
   - Computes usage summaries:
     - `totalUsers`: count of unique users assigned to the policy.
     - `totalScopedAssignments`: count of management role bindings using the policy.
     - `totalProperties`: count of unique properties (entities) linked to the policy.
     - `totalVenues`: count of unique venues linked to the policy.
   - Computes the itemized list of assigned users (`usersWithPolicy`):
     - Unique user identity: `id`, `name`, `email`, `userRole`, optional `avatar`.
     - `scopedAssignmentsCount`: count of assignments for this user under this policy.
     - Grouped list of scoped assignments (`scopes[]`): each containing `entityId`, `entityName`, `venueId`, and `venueName` ("All venues" or specific venue name).
2. **Operational Support & Diagnostics**:
   - `GET /livez`: Unauthenticated liveness probe on port `16010`.
   - `GET /api/v1/system`: System diagnostics with Bearer token authentication.
   - `POST /api/v1/system`: Runtime log level manipulation.
3. **Security & Transport**:
   - Inbound bearer-token validation through OWSEC (`AUTH_ENABLED=true`) via `Authorization: Bearer <owsec-token>`.
   - Outbound service-to-service calls: authenticates using service credentials (`X-API-KEY: <mdu-service-api-key>`) and forwards the caller's bearer token (`Authorization: Bearer <owsec-token>`) for downstream RBAC.
   - CORS support with automatic `OPTIONS` preflight bypass for browser compatibility. (Implementation requirement: `RegisterPublicCORS` in `internal/http/middleware/middleware.go` must configure `AllowHeaders` to include `"X-Request-Id"` and `"X-Correlation-Id"` alongside standard headers to allow browser tracing preflights).
   - Distributed request tracing: `X-Request-Id` and `X-Correlation-Id`.

### Deferred Scope (Later Milestones):
- Global Dashboard metrics & fleet telemetry (deferred).
- Property & Venue overview aggregation (deferred).
- Device operations & configuration management (deferred).
- Billing, subscriber management, and client troubleshooting (Phase 2).

---

## 3. Downstream Systems Integration

### Downstream Authentication & Propagation Contract
Downstream calls from MDU to OWPROV and OWSEC maintain explicit separation between service identity and end-user context:

```http
Authorization: Bearer <owsec-token>
X-API-KEY: <mdu-service-api-key>
X-Request-Id: <request-id>
X-Correlation-Id: <correlation-id>
```

1. **`Authorization: Bearer <owsec-token>`**: Forwards the end-user's OWSEC bearer token. Downstream services (particularly OWPROV) evaluate this token to authoritatively resolve caller identity, tenant boundaries, and RBAC / entity / venue scope permissions.
2. **`X-API-KEY` (or `x-api`)**: Service-to-service authentication credential establishing MDU Service as a trusted internal caller when accessing downstream microservices.
3. **Traceability**: Propagates `X-Request-Id` and `X-Correlation-Id` across all downstream interactions to maintain end-to-end distributed observability.

### OWSEC
- Validates bearer tokens before processing protected requests.
- Provides user directory lookups (`GET /api/v1/users` or `GET /api/v1/user/{id}`) to resolve user names and email addresses.
- **Orphaned User Handling**: Because OWPROV role assignments reference user UUIDs without cross-database foreign key constraints, an assigned user may have been deleted from OWSEC. If a user UUID in a role cannot be resolved in OWSEC, MDU handles this gracefully by generating a placeholder entry (`name: "Deleted User"`, `email: "deleted@example.invalid"`, `userRole: "unknown"`, `avatar: ""`) and retaining their scoped assignments. This ensures metrics remain consistent (`usersWithPolicy.length == totalUsers`) and operators can inspect orphaned assignments. In a later lifecycle milestone, user deletion workflows will be enhanced to automatically cascade and remove role bindings across OWPROV upon user deletion.

### OWPROV
- Provides policy definitions (`GET /api/v1/managementPolicy/{id}`).
- Provides management roles (`GET /api/v1/managementRole`). Evaluates user RBAC and entity/venue scoping based on the forwarded user token. Leverages the native `policyId` query filter (`GET /api/v1/managementRole?policyId={id}`) to perform database-level filtering directly in OWPROV. If a policy has a large volume of roles exceeding OWPROV's page size, MDU paginates using `limit=500` and `offset` until all policy-scoped roles are retrieved, preventing silent metric truncation while avoiding full-table scans.
- Provides entity and venue names (`GET /api/v1/entity`, `GET /api/v1/venue`) for resolving human-readable scope labels.

---

## 4. API Contract & Authority

The authoritative OpenAPI contract for this phase is:
**`docs/phase-1/mango-mdu-openapi.yaml`**

### Target Endpoint:
`GET /api/v1/policy/{id}/overview`

#### Parameters:
- `id` (path, string, required): UUID of the target management policy.
- `X-Request-Id` (header, string, optional): Request tracking UUID.
- `X-Correlation-Id` (header, string, optional): Correlation tracking UUID.

#### Response Envelope:
```json
{
  "policy": {
    "id": "523e4567-e89b-12d3-a456-426614174000",
    "name": "Network Operator",
    "description": "Monitor devices and manage network configuration.",
    "entity": "",
    "venue": "",
    "created": 1725000000,
    "modified": 1725500000
  },
  "totalUsers": 9,
  "totalScopedAssignments": 14,
  "totalProperties": 4,
  "totalVenues": 12,
  "usersWithPolicy": [
    {
      "id": "user-uuid-1",
      "name": "Anita Sharma",
      "email": "anita@ipnx.example",
      "userRole": "noc",
      "avatar": "1",
      "scopedAssignmentsCount": 2,
      "scopes": [
        {
          "entityId": "entity-uuid-1",
          "entityName": "Sunrise Apartments",
          "venueId": "",
          "venueName": "All venues"
        },
        {
          "entityId": "entity-uuid-2",
          "entityName": "Oakwood Housing",
          "venueId": "venue-uuid-001",
          "venueName": "Building A"
        }
      ]
    }
  ]
}
```

---

## 5. Error Handling

Normalized error responses adhere to the standard OpenWiFi/MDU PascalCase `ApiError` envelope:

```json
{
  "ErrorCode": 404,
  "ErrorDescription": "Not Found",
  "ErrorDetails": "Management policy not found"
}
```

- `400 Bad Request`: Malformed `id` parameter (`ErrorCode: 400`).
- `401 Unauthorized`: Missing or invalid bearer token (`ErrorCode: 401`).
- `403 Forbidden`: Caller lacks permission (`ErrorCode: 403`).
- `404 Not Found`: Target policy does not exist (`ErrorCode: 404`).
- `503 Service Unavailable`: Downstream dependency (OWPROV / OWSEC) unreachable (`ErrorCode: 503`).
