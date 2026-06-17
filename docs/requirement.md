# Mango MDU Service Master Requirements

## Document Purpose

This document defines the master requirements for the full lifecycle of `mango-mdu-service`.

`mango-mdu-service` is the Mango Cloud orchestration service for operator-facing MDU workflows. It exposes stable Mango-facing APIs under `/api/v1/mdu/*` and coordinates calls to downstream systems that own authentication, provisioning, customers, users, RBAC, billing, device runtime operations, topology, and analytics.

This document is the whole-service roadmap and scope baseline. Phase-specific requirements documents shall inherit from this master document and from the common engineering requirements document.

---

## Document Control

| Field | Value |
|---|---|
| Document Title | Mango MDU Service Master Requirements |
| Service Name | `mango-mdu-service` |
| Repository | `routerarchitects/mango-mdu-service` |
| Service Type | Internal orchestration microservice |
| Primary Language | Go |
| Current Purpose | Master service requirements |
| Status | Draft final baseline |
| Last Updated | 2026-06-17 |
| Primary Consumers | Mango Operator UI and approved internal callers |
| Primary Downstream Services | OWSEC, PROV, Billing Service, OWGW, NW Topology Service, OWANALYTICS |

---

## 1. Executive Summary

MDU Service shall be the Mango-facing orchestration layer for the MDU product domain.

The UI shall authenticate directly with OWSEC. OWSEC owns login, session issuance, browser bearer tokens, and token validation. After login, the UI shall call MDU business APIs with the OWSEC-issued bearer token.

MDU shall validate inbound requests, normalize and shape UI-facing APIs, orchestrate downstream calls, and hide downstream route quirks. MDU shall not become a source of truth for domains already owned by downstream services.

The final private-call rule is:

1. MDU authenticates to downstream services using service-to-service credentials such as `x-api` or equivalent.
2. MDU forwards the end-user bearer token to downstream services using `x-authorization`.
3. The downstream service, especially PROV, is responsible for interpreting that forwarded user context and enforcing its own RBAC.
4. MDU does not resolve PROV RBAC locally and does not persist RBAC truth.

PROV owns users, customers, hierarchy, RBAC, policies, roles, inventory ownership, and configuration ownership. MDU exposes cleaner business APIs for those workflows, but PROV remains the system of record.

---

## 2. Core Architecture Rules

1. MDU shall expose Mango-facing APIs under `/api/v1/mdu/*`.
2. The UI shall authenticate directly with OWSEC.
3. MDU shall not expose a competing login or signup API unless a future approved architecture explicitly adds that requirement.
4. The UI shall call MDU using `Authorization: Bearer <owsec-token>`.
5. MDU shall validate the inbound bearer token using OWSEC-owned validation mechanisms before executing protected business APIs.
6. MDU shall call downstream private APIs using service credentials such as `x-api` or equivalent.
7. MDU shall forward the inbound user token to downstream private APIs using `x-authorization` when the downstream service needs user context.
8. PROV shall resolve user, customer, scope, role, policy, and RBAC decisions using its own source-of-truth data.
9. MDU shall not become a separate RBAC, hierarchy, customer, user, inventory, configuration, billing, topology, or analytics source of truth.
10. MDU shall normalize downstream responses and errors into stable UI-facing contracts.
11. MDU shall hide downstream route quirks and compatibility paths from the UI.
12. MDU shall preserve clean package boundaries in the Go service.
13. Local persistence in MDU shall be minimal and limited to MDU-owned operational concerns.
14. Durable workflow state shall be introduced only in later phases where explicitly justified.
15. Every new downstream dependency shall be represented in API design, service design, observability, and error mapping.

---

## 3. Trust Lanes

### 3.1 Browser-to-OWSEC Auth Lane

The browser/UI uses OWSEC for login and session issuance. OWSEC owns the public authentication boundary.

Normal login flow:

```text
UI -> OWSEC
OWSEC -> UI: bearer token
```

### 3.2 Browser-to-MDU Business Lane

The browser/UI calls MDU business APIs using the OWSEC-issued bearer token.

```http
Authorization: Bearer <owsec-token>
```

MDU validates the token with OWSEC before business orchestration.

### 3.3 MDU-to-Downstream Private Lane

MDU calls downstream services as a trusted internal service and forwards user context separately.

```http
x-api: <mdu-service-api-key>
x-authorization: Bearer <owsec-token>
x-request-id: <correlation-id>
```

The downstream service decides how to use `x-authorization`. For PROV, RBAC, scope, user, and customer checks are resolved inside PROV.

---

## 4. Ownership Boundaries

| Domain / Workflow | System of Record | MDU Role |
|---|---|---|
| Login and token issuance | OWSEC | Not owned by MDU |
| Browser auth boundary | OWSEC | UI redirects/calls OWSEC directly |
| Token validation | OWSEC | MDU consumes validation |
| Users | PROV | MDU exposes user workflow APIs and forwards to PROV |
| Customers / sub-operators | PROV | MDU exposes customer workflow APIs and forwards to PROV |
| Operators | PROV | MDU exposes wrapper APIs |
| Entities / hierarchy | PROV | MDU exposes normalized hierarchy APIs |
| Venues | PROV | MDU exposes normalized venue APIs |
| Roles and policies | PROV | MDU exposes wrapper/access APIs only |
| RBAC and scope resolution | PROV | MDU does not resolve or persist RBAC |
| Inventory ownership | PROV | MDU wraps and composes device inventory APIs |
| Configuration ownership | PROV | MDU wraps and composes configuration APIs |
| Billing plans/subscriptions | Billing Service | MDU composes billing summaries |
| Live device runtime/actions | OWGW | MDU exposes approved action wrappers |
| Topology graph computation | NW Topology Service | MDU composes topology views |
| Telemetry and historical analytics | OWANALYTICS | MDU composes analytics views |
| UI-facing orchestration | MDU | MDU owns API shaping and workflow composition |
| Workflow/job state where approved | MDU | MDU owns only approved operational state |

---

## 5. Repository and Implementation Baseline

The current `mango-mdu-service` repository provides the implementation skeleton. The scaffold behavior and sample item CRUD are not domain truth.

The implementation shall preserve this structure:

- `cmd/` for startup
- `internal/app/` for application wiring
- `internal/http/` for routing, middleware, and transport concerns
- `internal/services/` for orchestration and use cases
- `internal/models/` for normalized API and domain DTOs
- `external/` for downstream clients
- `internal/db/` only for MDU-owned persistence concerns

The production route surface shall remove or isolate placeholder scaffold APIs.

---

## 6. Downstream Dependency Model

### 6.1 OWSEC

OWSEC owns authentication and session security.

MDU shall use OWSEC for:

- validating inbound bearer tokens
- validating API keys where required
- resolving token/session validity

Verified OWSEC routes for Phase 1 include:

- `POST /oauth2` for login/session issuance, called directly by the UI
- `DELETE /oauth2/{token}` for logout/session removal where applicable
- `GET /validateToken?token=...` for token validation
- `GET /validateSubToken?token=...` where sub-token validation is required
- `GET /validateApiKey?apikey=...` where API-key validation is required
- `GET /oauth2?me=true` if current caller profile is required from OWSEC

### 6.2 PROV

PROV owns users, customers, operators, entities, venues, roles, policies, RBAC, hierarchy, inventory ownership, and configuration ownership.

MDU shall call PROV using service authentication and forwarded user context:

```http
x-api: <mdu-service-api-key>
x-authorization: Bearer <owsec-token>
```

PROV is responsible for resolving RBAC, scope, customer, and user permissions.

Verified PROV route families for Phase 1 include:

- `/operator`
- `/operator/{uuid}`
- `/entity`
- `/entity/{uuid}`
- `/entity?setTree=true`
- `/venue`
- `/venue/{uuid}`
- `/managementPolicy`
- `/managementPolicy/{uuid}`
- `/managementRole`
- `/managementRole/{id}`
- PROV-owned user/customer routes as required by the implementation baseline

Critical PROV quirks:

1. Create routes may use compatibility detail paths, not collection POST paths.
2. The create-route path token may not be the created resource ID.
3. Operator creation may auto-create linked hierarchy objects; MDU must not duplicate those objects.
4. Normal entity creation uses PROV-owned route semantics and shall not be replaced by local MDU hierarchy persistence.
5. PROV remains responsible for RBAC enforcement.

### 6.3 Billing Service

Billing Service owns plans, subscriptions, billing lifecycle, and billing state. MDU may compose billing summaries but shall not persist billing truth.

### 6.4 OWGW

OWGW owns live device runtime operations, command execution, and diagnostics. MDU shall expose only approved and validated action wrappers rather than raw unfiltered command access.

### 6.5 NW Topology Service

NW Topology Service owns topology graph computation. MDU shall consume topology outputs and shape them into workspace-friendly responses.

### 6.6 OWANALYTICS

OWANALYTICS owns telemetry, historical timepoints, client history, and analytics data. MDU shall consume analytics outputs for summaries, trends, health views, and dashboards.

---

## 7. MDU-Owned Responsibilities

MDU owns:

- stable Mango-facing API contracts
- request validation and normalization
- token validation orchestration with OWSEC
- downstream adapter calls
- response shaping
- error normalization
- correlation ID propagation
- UI bootstrap/session payload composition
- service-level observability
- approved workflow/job state in later phases

MDU does not own:

- login/session issuance
- users or customers as persisted truth
- RBAC or scope truth
- hierarchy truth
- inventory truth
- configuration truth
- billing truth
- topology computation
- telemetry truth

---

## 8. Whole-Service Domain Roadmap

The full MDU service may eventually cover these API families:

1. session and bootstrap
2. users and access
3. customers and sub-operators
4. hierarchy and workspaces
5. operators, entities, venues, policies, and roles
6. billing workspaces
7. devices and inventory
8. configurations
9. live device state and actions
10. topology
11. analytics, metrics, and health
12. maps and overlays
13. clients
14. workflow/job status
15. audit, reconciliation, rollout, AI, and admin/debug APIs

Business-facing terminology should be used in MDU APIs wherever possible: customer, sub-operator, workspace, node, site, building, floor, venue, user, device, configuration, subscription, access summary.

Raw downstream names may be exposed only in admin/debug APIs or where operationally necessary.

---

# 9. Development Phases

The phases are cumulative. Each phase shall preserve ownership boundaries from previous phases.

---

## Phase 1: Orchestrator Foundation

### Objective

Establish the service foundation, auth boundary, downstream trust model, and core PROV/OWSEC integration.

### Scope

Phase 1 includes:

- service bootstrap and configuration
- inbound OWSEC bearer-token validation
- direct OWSEC login boundary
- service-authenticated downstream calls
- `x-authorization` forwarding to downstream services
- session/bootstrap APIs
- PROV-backed users, operators, entities, venues, roles, policies, customers where needed for foundation
- access-summary wrappers where PROV provides the RBAC result
- normalized errors and observability
- removal of production placeholder routes

### Auth Requirements

- UI login remains direct to OWSEC.
- MDU accepts `Authorization: Bearer <owsec-token>` on protected business APIs.
- MDU validates the bearer token with OWSEC.
- MDU calls PROV using `x-api` plus `x-authorization`.
- PROV resolves RBAC and user/customer permissions.

### Expected API Families

Session:

- `GET /api/v1/mdu/me`
- `GET /api/v1/mdu/session`

Users, backed by PROV:

- `POST /api/v1/mdu/users`
- `GET /api/v1/mdu/users`
- `GET /api/v1/mdu/users/{userId}`
- `PATCH /api/v1/mdu/users/{userId}`
- `DELETE /api/v1/mdu/users/{userId}`
- `GET /api/v1/mdu/users/{userId}/access`

Operators:

- `POST /api/v1/mdu/operators`
- `GET /api/v1/mdu/operators`
- `GET /api/v1/mdu/operators/{operatorId}`
- `PATCH /api/v1/mdu/operators/{operatorId}`

Entities:

- `POST /api/v1/mdu/entities`
- `GET /api/v1/mdu/entities`
- `GET /api/v1/mdu/entities/{entityId}`
- `PATCH /api/v1/mdu/entities/{entityId}`
- `GET /api/v1/mdu/entities/{entityId}/children`

Venues:

- `POST /api/v1/mdu/entities/{entityId}/venues`
- `GET /api/v1/mdu/entities/{entityId}/venues`
- `GET /api/v1/mdu/venues/{venueId}`
- `PATCH /api/v1/mdu/venues/{venueId}`
- `GET /api/v1/mdu/venues/{venueId}/children`

Policies and roles:

- `POST /api/v1/mdu/policies`
- `GET /api/v1/mdu/policies`
- `GET /api/v1/mdu/policies/{policyId}`
- `POST /api/v1/mdu/roles`
- `GET /api/v1/mdu/roles`
- `GET /api/v1/mdu/roles/{roleId}`

Access wrappers:

- `POST /api/v1/mdu/users/{userId}/entities/{entityId}`
- `POST /api/v1/mdu/users/{userId}/roles`
- `POST /api/v1/mdu/users/{userId}/policies`

Customers where required for foundation:

- `POST /api/v1/mdu/customers`
- `GET /api/v1/mdu/customers`
- `GET /api/v1/mdu/customers/{customerId}`
- `PATCH /api/v1/mdu/customers/{customerId}`

### Non-Goals

Phase 1 does not require:

- MDU-owned login/signup
- local RBAC storage
- local customer/user storage
- billing workspace aggregation
- live device aggregation
- topology aggregation
- analytics dashboards
- async workflow persistence

---

## Phase 2: Customers, Hierarchy, and Billing Workspaces

### Objective

Make MDU useful for customer/sub-operator workspace workflows and hierarchy navigation while keeping PROV and Billing as systems of record.

### Scope

Phase 2 includes:

- customer/sub-operator lifecycle APIs backed by PROV
- first-admin/customer bootstrap orchestration backed by PROV and OWSEC as applicable
- hierarchy browsing and lazy child loading backed by PROV
- workspace context APIs
- billing summary integration backed by Billing Service
- customer workspace payloads for UI tabs

### Expected API Families

- `/api/v1/mdu/customers`
- `/api/v1/mdu/customers/{customerId}`
- `/api/v1/mdu/customers/{customerId}/workspace`
- `/api/v1/mdu/customers/{customerId}/users`
- `/api/v1/mdu/customers/{customerId}/billing`
- `/api/v1/mdu/hierarchy`
- `/api/v1/mdu/hierarchy/{nodeId}`
- `/api/v1/mdu/hierarchy/{nodeId}/children`
- `/api/v1/mdu/workspaces/{nodeId}/context`
- `/api/v1/mdu/bootstrap/*` where approved

### Non-Goals

Phase 2 does not require:

- live device actions
- topology graph APIs
- analytics dashboards
- durable workflow engine
- broad async jobs

---

## Phase 3: Devices and Configurations

### Objective

Add operational device and configuration workflows by composing PROV inventory/configuration truth with OWGW live runtime data.

### Scope

Phase 3 includes:

- device inventory wrappers backed by PROV
- venue-device assignment backed by PROV
- configuration CRUD and assignment backed by PROV
- live status and diagnostics backed by OWGW
- approved device action wrappers backed by OWGW
- effective configuration preview where supported

### Expected API Families

- `/api/v1/mdu/devices`
- `/api/v1/mdu/devices/{serialNumber}`
- `/api/v1/mdu/devices/import`
- `/api/v1/mdu/venues/{venueId}/devices`
- `/api/v1/mdu/devices/{serialNumber}/status`
- `/api/v1/mdu/devices/{serialNumber}/diagnostics`
- `/api/v1/mdu/devices/{serialNumber}/actions/*`
- `/api/v1/mdu/configurations`
- `/api/v1/mdu/configurations/{configurationId}`
- `/api/v1/mdu/venues/{venueId}/configuration/*`
- `/api/v1/mdu/access-points/{serialNumber}/configuration/*`

### Non-Goals

Phase 3 does not require:

- full topology workspaces
- historical analytics dashboards
- rollout engine
- AI hooks

---

## Phase 4: Topology, Analytics, Metrics, and Health

### Objective

Add cross-domain operational visibility using topology and analytics as first-class downstream dependencies.

### Scope

Phase 4 includes:

- topology workspace APIs backed by NW Topology Service
- analytics summaries backed by OWANALYTICS
- KPI and health panels
- map/overlay payloads
- scoped client visibility foundations
- workspace overview aggregation

### Expected API Families

- `/api/v1/mdu/topology/*`
- `/api/v1/mdu/analytics/*`
- `/api/v1/mdu/maps/*`
- `/api/v1/mdu/metrics/*`
- `/api/v1/mdu/workspaces/{nodeId}/health`
- `/api/v1/mdu/workspaces/{nodeId}/overview`
- `/api/v1/mdu/clients/*`
- `/api/v1/mdu/dashboard/*`

### Non-Goals

Phase 4 does not require:

- generalized durable workflow engine
- full reconciliation system
- rollout orchestration engine
- AI execution hooks

---

## Phase 5: Workflow Durability and Advanced Operations

### Objective

Introduce durable workflow safety and advanced operational maturity for long-running, multi-system, or recovery-sensitive workflows.

### Scope

Phase 5 includes:

- idempotency keys for selected write APIs
- async jobs and operation status
- workflow tracking where needed
- audit and reconciliation support
- selective forward recovery or compensation
- rollout orchestration hooks
- AI recommendation hooks where approved
- advanced admin/debug APIs

### Expected API Families

- `/api/v1/mdu/operations/*`
- `/api/v1/mdu/jobs/*`
- `/api/v1/mdu/reconciliation/*`
- `/api/v1/mdu/rollouts/*`
- `/api/v1/mdu/ai/*`
- `/api/v1/mdu/admin/debug/*`
- `/api/v1/mdu/audit/*`

### Non-Goals

Phase 5 does not make MDU the source of truth for users, customers, RBAC, billing, device runtime, topology, or analytics.

---

# 10. Data and Persistence Rules

## 10.1 Early-Phase Persistence

Early MDU persistence shall be minimal.

Allowed early uses:

- migration tracking
- operational metadata
- correlation and request tracing metadata where needed
- bounded audit support where approved
- feature-flag or local service configuration where justified

Not allowed in early phases:

- local RBAC truth
- local hierarchy truth
- local user/customer truth
- local inventory truth
- local configuration truth
- local billing truth

## 10.2 Later-Phase Persistence

Later phases may add MDU-owned persistence for:

- workflow executions
- job state
- idempotency records
- import tracking
- audit logs
- reconciliation status
- support/debug metadata
- bounded non-authoritative caches

Every such table must map to an approved phase requirement.

## 10.3 Caching

Any cache in MDU shall be:

- non-authoritative
- TTL-bounded
- safe to lose
- observable
- explicitly scoped to performance optimization

---

# 11. Error Handling and Observability

MDU shall provide a consistent error envelope for UI-facing APIs.

Each downstream adapter shall map downstream failures into stable MDU-facing errors while preserving enough correlation metadata for support and debugging.

MDU shall propagate or generate:

- request ID
- trace ID where supported
- downstream service name
- normalized error code
- safe user-facing message
- internal diagnostic metadata in logs only

MDU shall not expose raw downstream secrets, tokens, stack traces, or implementation-specific failures to the UI.

---

# 12. Security Requirements

1. UI login stays with OWSEC.
2. MDU validates inbound bearer tokens before protected business APIs.
3. MDU uses service credentials for downstream calls.
4. MDU forwards user context to downstream using `x-authorization` when required.
5. MDU shall not persist browser bearer tokens.
6. Logs shall not include raw bearer tokens, `x-api` values, or `x-authorization` values.
7. PROV is responsible for RBAC enforcement for PROV-owned domains.
8. Billing, OWGW, Topology, and Analytics services remain responsible for their own authorization rules where applicable.
9. MDU shall enforce request-level validation and coarse service-level protections before orchestration.
10. MDU shall not bypass downstream authorization by using only machine credentials for user-sensitive actions.

---

# 13. Recommended Build Sequence

1. Remove or isolate scaffold placeholder APIs.
2. Finalize configuration, routing, middleware, and downstream adapter structure.
3. Implement OWSEC token-validation middleware.
4. Implement outbound service credentials and `x-authorization` forwarding.
5. Implement Phase 1 session/user/customer/provisioning/access wrappers backed by PROV/OWSEC.
6. Implement Phase 2 customer, hierarchy, workspace, and billing APIs.
7. Implement Phase 3 device and configuration APIs.
8. Implement Phase 4 topology, analytics, metrics, health, and dashboard APIs.
9. Implement Phase 5 workflow durability, audit, reconciliation, rollout, AI, and admin/debug APIs.

---

# 14. Acceptance Criteria

This master document is acceptable only if:

1. It defines the whole service, not only Phase 1.
2. It preserves direct UI login through OWSEC.
3. It makes PROV the source of truth for users, customers, hierarchy, roles, policies, and RBAC.
4. It makes MDU an orchestration and API-shaping service, not a duplicate source of truth.
5. It uses `x-api` or equivalent service credentials for downstream private calls.
6. It uses `x-authorization` to forward the user token to downstream services that need user context.
7. It keeps RBAC resolution inside PROV for PROV-owned domains.
8. It cleanly separates phase scopes and avoids duplicated requirements.
9. It is compatible with the existing Go repository structure.
10. It provides a clear base for detailed phase-specific documents.

---

# 15. Final Architectural Rules

1. `mango-mdu-service` is the Mango operator-domain orchestration service.
2. The UI authenticates directly with OWSEC.
3. OWSEC owns login, session issuance, bearer token validity, and token validation.
4. The UI calls MDU with `Authorization: Bearer <owsec-token>`.
5. MDU validates the inbound token before protected business workflows.
6. MDU calls downstream services with service authentication such as `x-api`.
7. MDU forwards user context to downstream services using `x-authorization`.
8. PROV owns users, customers, hierarchy, policies, roles, RBAC, inventory ownership, and configuration ownership.
9. PROV resolves RBAC and user/customer permissions for PROV-owned domains.
10. Billing Service owns billing truth.
11. OWGW owns live device runtime and command execution.
12. NW Topology Service owns topology graph computation.
13. OWANALYTICS owns telemetry and historical analytics.
14. MDU owns orchestration, request validation, API shaping, response composition, error normalization, and future approved workflow state.
15. MDU shall not become a second source of truth for downstream-owned domains.