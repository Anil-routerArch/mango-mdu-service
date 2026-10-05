# Mango MDU Service — Phase 1 Test Cases: Policy Overview

## 1. Scope

This document specifies test cases for Phase 1 of `mango-mdu-service`:
1. Operational & System Endpoints (`/livez`, `/api/v1/system`)
2. Policy Overview Endpoint (`GET /api/v1/policy/{id}/overview`)

---

## 2. Test Cases Matrix

| Test ID | Description | Input / Precondition | Expected Status | Expected Behavior |
|:---|:---|:---|:---|:---|
| **TC-SYS-001** | Public Liveness probe | `GET /livez` (no auth) | `200 OK` | Returns empty body or OK status |
| **TC-SYS-002** | System diagnostics info | `GET /api/v1/system?command=info` with valid token | `200 OK` | Returns subsystem info JSON |
| **TC-SYS-003** | System diagnostics unauthorized | `GET /api/v1/system?command=info` without token | `401 Unauthorized` | Standard `ApiError` envelope |
| **TC-POL-001** | Get Policy Overview — Policy with active assignments across multiple scopes | `GET /api/v1/policy/{id}/overview` with valid token; policy has 2 unique users across 3 roles, 1 entity, and 2 venues (User 1 assigned to 2 venues, User 2 assigned to whole property) | `200 OK` | `totalUsers == 2`, `totalScopedAssignments == 3`, `totalProperties == 1`, `totalVenues == 2`, `policy.name` populated, `usersWithPolicy` has 2 unique user entries; User 1 has `scopes` length 2 ("Tower A" and "Tower B"), User 2 has `scopes` length 1 ("Whole property"); total scope items across users equals 3 |
| **TC-POL-002** | Get Policy Overview — Unassigned policy | `GET /api/v1/policy/{id}/overview` with valid token; policy has no roles | `200 OK` | All summary counters are `0` (`totalUsers == 0`, `totalScopedAssignments == 0`, `totalProperties == 0`, `totalVenues == 0`), `usersWithPolicy` is empty `[]` |
| **TC-POL-003** | Get Policy Overview — Target policy not found (valid UUID) | `GET /api/v1/policy/00000000-0000-4000-8000-000000000001/overview` with valid token | `404 Not Found` | Valid UUID not present in OWPROV returns `ApiError` with `ErrorCode: 404` |
| **TC-POL-004** | Get Policy Overview — Missing bearer token | `GET /api/v1/policy/{id}/overview` without `Authorization` header | `401 Unauthorized` | Rejection before downstream calls |
| **TC-POL-005** | Get Policy Overview — Invalid bearer token | `GET /api/v1/policy/{id}/overview` with invalid/expired token | `401 Unauthorized` | Rejection from OWSEC token validation |
| **TC-POL-006** | Get Policy Overview — Downstream PROV unreachable | `GET /api/v1/policy/{id}/overview` with valid token; PROV mock down | `503 Service Unavailable` | Graceful failure in `ApiError` envelope with `ErrorCode: 503` |
| **TC-POL-007** | Downstream header, tracing, and filter propagation | `GET /api/v1/policy/{id}/overview` with `X-Request-Id: req-1` and `X-Correlation-Id: corr-1` | `200 OK` | Outbound downstream requests to PROV & SEC propagate `Authorization: Bearer <owsec-token>`, `X-API-KEY`, and tracing headers (`X-Request-Id`, `X-Correlation-Id`); role query specifies `?policyId={id}` |
| **TC-POL-008** | CORS Preflight OPTIONS (Phase 1 Acceptance) | `OPTIONS /api/v1/policy/{id}/overview` with `Access-Control-Request-Headers: Authorization, X-Request-Id, X-Correlation-Id` | `204 No Content` or `200 OK` | Bypasses bearer auth middleware; `Access-Control-Allow-Headers` includes `Authorization`, `X-Request-Id`, and `X-Correlation-Id` |
| **TC-POL-009** | Get Policy Overview — Malformed policy UUID | `GET /api/v1/policy/not-a-uuid/overview` with valid token | `400 Bad Request` | Immediate validation failure before downstream call; returns `ApiError` with `ErrorCode: 400` |
| **TC-POL-010** | Get Policy Overview — Orphaned user in OWPROV role not found in OWSEC | `GET /api/v1/policy/{id}/overview` with valid token; role references user UUID deleted from OWSEC | `200 OK` | Graceful fallback: entry in `usersWithPolicy` populated with `name: "Deleted User"`, `email: "deleted@example.invalid"`, `userRole: "unknown"`; scoped assignments retained; `usersWithPolicy.length == totalUsers` maintained |

