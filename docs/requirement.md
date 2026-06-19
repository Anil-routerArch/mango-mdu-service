# Mango MDU Service Master Requirements

## Document Purpose

This document defines the master requirements for the full lifecycle of `mango-mdu-service`.

`mango-mdu-service` is the MangoCloud operator-domain orchestration service for MDU workflows. It exposes stable Mango-facing APIs under `/api/v1/mdu/*`, validates inbound access, shapes UI-facing business contracts, and coordinates calls to downstream systems that own authentication, users, customers, hierarchy, RBAC, inventory, configuration, billing, live device operations, topology, and analytics.

This document is the whole-service master specification and roadmap baseline. Phase-specific requirements documents shall inherit from this document and from the common engineering requirements document.

This document is intentionally a commit-ready service requirements baseline, not only an architecture note. It defines service boundaries, trust rules, phased scope, workflow expectations, validation direction, and operational behavior while keeping the content lean enough to maintain.

---

## Document Control

| Field | Value |
|---|---|
| Document Title | Mango MDU Service Master Requirements |
| Service Name | `mango-mdu-service` |
| Repository | `routerarchitects/mango-mdu-service` |
| Service Type | Internal orchestration microservice |
| Primary Language | Go |
| Runtime | Dockerized Go service |
| Status | Master baseline |
| Last Updated | 2026-06-18 |
| Primary Consumers | Mango Operator UI and approved internal callers |
| Primary Downstream Services | OWSEC, PROV, Billing Service, OWGW, NW Topology Service, OWANALYTICS |
| Base API Namespace | `/api/v1/mdu/*` |

---

# 1. Executive Summary

MDU Service shall be the Mango-facing orchestration layer for the MDU product domain.

The browser/UI authenticates directly with OWSEC. OWSEC owns login, session issuance, browser bearer tokens, and token validation primitives. After login, the UI calls MDU business APIs using the OWSEC-issued bearer token.

MDU validates inbound requests, normalizes business-facing contracts, composes multi-service payloads, forwards approved user context to downstream private APIs, hides downstream route quirks, and returns stable UI-facing responses. MDU shall not become a second source of truth for domains already owned by downstream services.

The downstream trust model is:

1. MDU authenticates to downstream services using service-to-service credentials such as `x-api` or equivalent.
2. MDU forwards the end-user bearer token to downstream services using `x-authorization` when the downstream service requires user context.
3. The downstream service, especially PROV, interprets that forwarded user context and enforces its own authorization and RBAC.
4. MDU does not resolve PROV RBAC locally and does not persist RBAC truth.

PROV remains the system of record for users, customers, hierarchy, roles, policies, inventory ownership, and configuration ownership. Billing Service remains the system of record for billing. OWGW remains the system of record for live runtime operations. NW Topology Service remains the system of record for topology computation. OWANALYTICS remains the system of record for telemetry and historical analytics. MDU owns orchestration, shaping, composition, and approved workflow state only.

---

# 2. Core Architecture Rules

1. MDU shall expose Mango-facing APIs under `/api/v1/mdu/*`.
2. The UI shall authenticate directly with OWSEC.
3. Protected MDU business APIs shall accept `Authorization: Bearer <owsec-token>`.
4. MDU shall validate the inbound bearer token using OWSEC-owned validation mechanisms before executing protected business APIs.
5. MDU shall call downstream private APIs using trusted service credentials such as `x-api` or equivalent.
6. MDU shall forward the inbound user token to downstream private APIs using `x-authorization` when the downstream service needs user context.
7. PROV shall resolve user, customer, scope, role, policy, and RBAC decisions using its own source-of-truth data.
8. MDU shall not become a separate RBAC, hierarchy, customer, user, inventory, configuration, billing, topology, or analytics source of truth.
9. MDU shall normalize downstream responses and errors into stable UI-facing contracts.
10. MDU shall hide downstream route quirks and compatibility paths from the UI.
11. Local persistence in MDU shall be minimal and limited to MDU-owned operational concerns.
12. MDU shall remain an orchestration service, not a replacement for downstream domain services.

---

# 3. Common Cross-Phase Requirements

## 3.1 Delivery and Engineering Rules

1. Each phase requires automated CI/CD tests passing.
2. A phase is not complete unless required CI jobs pass.
3. Every PR shall rely on automated CI execution as primary test evidence.
4. New route families shall not be merged without corresponding handler, service, and adapter-level coverage.
5. Docker image build and container startup smoke test shall run in CI.
6. Placeholder scaffold APIs shall not remain exposed in production route groups.

## 3.2 Runtime and Security Baseline

1. MDU shall run as a Dockerized Go microservice.
2. Service configuration shall be provided through environment variables, config files, or approved secret-management mechanisms.
3. Secrets shall not be hardcoded.
4. Internal service authentication is mandatory for MDU-to-downstream private calls.
5. Frontend or end-user tokens alone shall not be used as the service-authentication credential for private downstream calls.
6. Logs shall not contain raw bearer tokens, raw `x-api` values, or raw `x-authorization` values.

## 3.3 Common Module Usage

MDU should use approved shared Router Architects and OpenWiFi common modules for logging, build metadata, service discovery, and common client or RPC wrapper behavior where applicable.
Repo:https://github.com/routerarchitects/ow-common-mods
Repo:https://github.com/routerarchitects/ra-common-mods

MDU should not duplicate shared transport, logging, discovery, or internal-auth infrastructure without approved justification.


## 3.4 Required CI Checks

Required CI checks should include:

- formatting check
- static analysis or linting
- unit tests
- handler or API tests
- service-layer orchestration tests
- downstream adapter tests
- authentication and token-validation tests
- error-envelope tests
- request-context propagation tests
- OpenAPI or contract validation where applicable
- Docker build
- container startup smoke test

---

# 4. Canonical Domain Vocabulary

MDU shall use the following business-facing terms unless a phase-specific contract explicitly narrows or extends them.

| Term | Meaning | Ownership |
|---|---|---|
| operator | top-level Mango tenant or operator scope exposed to the MDU product | PROV |
| customer | customer or sub-operator business scope visible in Mango workflows | PROV |
| entity | hierarchy object used for ownership and tree structure | PROV |
| venue | location or venue object within the hierarchy | PROV |
| workspace | Mango UI-facing composed context for a selected node or tenant | MDU composed view |
| hierarchy node | normalized tree node exposed by MDU | PROV truth, MDU shape |
| user | operator or customer-facing account represented through PROV and secured by OWSEC | PROV / OWSEC |
| access summary | composed view of effective access visible to UI | PROV truth, MDU shape |
| device | managed inventory device | PROV truth, OWGW runtime |
| configuration | assignable or viewable config object | PROV |
| subscription | plan or billing relationship | Billing Service |
| billing summary | composed billing-facing view for UI | Billing Service truth, MDU shape |
| topology node | graph or computed topology object | NW Topology Service |
| analytics summary | KPI or historical analytics payload | OWANALYTICS |

Rules:

1. Business-facing terms should be preferred over raw downstream route names.
2. Raw downstream naming may remain in internal adapters and approved debug/admin APIs.
3. MDU shall not rename source-of-truth identifiers unless it also preserves a stable mapping rule.

---

# 5. Trust Lanes and Security Model

## 5.1 Browser-to-OWSEC Auth Lane

The browser/UI uses OWSEC for login and session issuance. OWSEC owns the public authentication boundary.

```text
UI -> OWSEC
OWSEC -> UI: bearer token
```

## 5.2 Browser-to-MDU Business Lane

The browser/UI calls MDU business APIs using the OWSEC-issued bearer token.

```http
Authorization: Bearer <owsec-token>
```

MDU validates the token with OWSEC before business orchestration.

## 5.3 MDU-to-Downstream Private Lane

MDU calls downstream services as a trusted internal service and forwards user context separately when required.

```http
x-api: <mdu-service-api-key>
x-authorization: Bearer <owsec-token>
x-request-id: <request-id>
x-correlation-id: <correlation-id>
```

The downstream service decides how to use `x-authorization`. For PROV, RBAC, scope, user, and customer checks are resolved inside PROV.

## 5.4 MDU-to-Billing Private Lane

MDU calls Billing Service as a trusted internal service, but does not forward end-user bearer context for Phase 1 billing calls.

```http
x-api: <mdu-service-api-key>
x-request-id: <request-id>
x-correlation-id: <correlation-id>
```

Billing Service is the Phase 1 exception to the standard downstream forwarding rule. MDU shall authenticate using the trusted internal service credential only and shall not forward the end-user bearer token unless a later approved integration contract explicitly requires it.

## 5.5 Security Rules

1. OWSEC owns login, token issuance, and token/session validation primitives.
2. MDU validates inbound protected requests before orchestration.
3. MDU authenticates to downstreams using service credentials.
4. MDU forwards user context only where needed by downstream authorization or business logic.
5. Billing-service calls shall use service credentials only unless a later approved billing integration contract explicitly requires forwarded user context.
6. Downstream systems retain their own authorization rules.
7. MDU shall not bypass downstream authorization by using only machine credentials for user-sensitive workflows.
8. MDU shall not persist browser bearer tokens.

---

# 6. Ownership Boundaries

| Domain / Workflow | System of Record | MDU Role |
|---|---|---|
| Login and token issuance | OWSEC | Not owned by MDU |
| Token validation | OWSEC | MDU consumes validation |
| Users | PROV | MDU exposes user workflow APIs and forwards to PROV |
| Customers / sub-operators | PROV | MDU exposes customer workflow APIs and forwards to PROV |
| Operators | PROV | MDU exposes normalized wrapper APIs |
| Entities / hierarchy | PROV | MDU exposes normalized hierarchy APIs |
| Venues | PROV | MDU exposes normalized venue APIs |
| Roles and policies | PROV | MDU exposes wrapper/access APIs only |
| RBAC and scope resolution | PROV | MDU does not resolve or persist RBAC |
| Inventory ownership | PROV | MDU wraps and composes inventory APIs |
| Configuration ownership | PROV | MDU wraps and composes configuration APIs |
| Billing plans/subscriptions | Billing Service | MDU composes billing summaries |
| Live device runtime/actions | OWGW | MDU exposes approved action wrappers |
| Topology graph computation | NW Topology Service | MDU composes topology views |
| Telemetry and historical analytics | OWANALYTICS | MDU composes analytics views |
| UI-facing orchestration | MDU | MDU owns API shaping and workflow composition |
| Workflow/job state where approved | MDU | MDU owns only approved operational state |

Rules:

1. MDU owns API shaping, request validation, response composition, error normalization, and approved orchestration logic.
2. MDU does not own persistent truth for downstream-owned domains.
3. MDU may maintain minimal operational state only where explicitly approved by phase scope.

---

# 7. Repository and Implementation Baseline

The current `mango-mdu-service` repository provides the implementation skeleton. Scaffold behavior and sample item CRUD are not domain truth.

The implementation shall preserve this structure:

- `cmd/` for startup
- `internal/app/` for application wiring
- `internal/http/` for routing, middleware, and transport concerns
- `internal/services/` for orchestration and use cases
- `internal/models/` for normalized API and domain DTOs
- `external/` for downstream clients
- `internal/db/` only for MDU-owned persistence concerns

Rules:

1. Production route groups shall remove or isolate placeholder scaffold APIs.
2. Downstream-specific code shall remain in adapters/clients, not leak into business-facing handlers.
3. Local DB usage shall be confined to MDU-owned operational concerns.

---

# 8. Downstream Dependency Model

## 8.1 OWSEC

OWSEC owns authentication and session security.

MDU shall use OWSEC for:

- validating inbound bearer tokens
- validating API keys where required
- resolving token/session validity
- retrieving caller profile where explicitly needed

Verified OWSEC routes for early phases include:

- `POST /oauth2` for login/session issuance, called directly by the UI
- `DELETE /oauth2/{token}` for logout/session removal where applicable
- `GET /validateToken?token=...` for token validation
- `GET /validateSubToken?token=...` where sub-token validation is required
- `GET /validateApiKey?apikey=...` where API-key validation is required
- `GET /oauth2?me=true` if current caller profile is required from OWSEC

## 8.2 PROV

PROV owns users, customers, operators, entities, venues, roles, policies, RBAC, hierarchy, inventory ownership, and configuration ownership.

MDU shall call PROV using service authentication and forwarded user context:

```http
x-api: <mdu-service-api-key>
x-authorization: Bearer <owsec-token>
```

PROV is responsible for resolving RBAC, scope, customer, and user permissions.

Verified PROV route families for early phases include:

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


## 8.3 Billing Service

Billing Service owns plans, subscriptions, billing lifecycle, invoices, and billing state. MDU may compose billing summaries but shall not persist billing truth.

Billing integration is not implemented in the current MDU baseline. This section is present to document the intended ownership boundary only.

## 8.4 OWGW

OWGW owns live device runtime operations, command execution, and diagnostics. MDU shall expose only approved and validated action wrappers rather than raw unfiltered command access.

MDU shall call OWGW using service authentication and only for approved operational workflows.

Verified OWGW route families include:

- `/device/{serialNumber}`
- `/device/{serialNumber}/logs`
- `/device/{serialNumber}/healthchecks`
- `/device/{serialNumber}/capabilities`
- `/device/{serialNumber}/statistics`
- `/device/{serialNumber}/status`
- `/device/{serialNumber}/configure`
- `/device/{serialNumber}/upgrade`
- `/device/{serialNumber}/reboot`
- `/device/{serialNumber}/trace`
- `/device/{serialNumber}/telemetry`
- selected device action routes required by approved MDU workflows only

OWGW-backed MDU work should focus on device runtime reads, diagnostics, and approved actions only. 


## 8.5 NW Topology Service

NW Topology Service owns topology graph computation. MDU shall consume topology outputs and shape them into workspace-friendly responses.

MDU shall call Topology Service using the approved downstream auth model for topology reads.

Verified Topology Service route families include:

- `/livez`
- `/topology`
- `/system`

`GET /topology` is the primary business route in the attached contract. It accepts `boardId` and optional `interval` query parameters. The attached OpenAPI also states that topology data is assembled from board timepoints, board device inventory data, and Wi-Fi client history data.


## 8.6 OWANALYTICS

OWANALYTICS owns telemetry, historical timepoints, client history, and analytics data. MDU shall consume analytics outputs for summaries, trends, health views, and dashboards.

MDU shall call OWANALYTICS using service authentication for approved read-only analytics workflows.

Verified OWANALYTICS route families include:

- `/board/{id}/devices`
- `/board/{id}/timepoints`
- `/wifiClientHistory`
- `/wifiClientHistory/{client}`



---

# 9. Request Context and Header Propagation Rules

## 9.1 Inbound Headers

Protected business APIs shall accept or generate the following context:

- `Authorization: Bearer <owsec-token>` for end-user auth
- `X-Request-Id` when supplied by caller or gateway
- `X-Correlation-Id` when supplied by caller or gateway
- additional approved tracing headers where supported

## 9.2 Generated Context

If `X-Request-Id` is absent, MDU shall generate one.

If `X-Correlation-Id` is absent, MDU shall use `X-Request-Id` as the correlation ID or generate both when needed.

## 9.3 Outbound Headers to Downstreams

MDU shall propagate:

- `x-api: <service credential>` or equivalent
- `x-authorization: Bearer <owsec-token>` when user context is required by downstream
- `x-request-id`
- `x-correlation-id`
- approved trace headers where supported

## 9.4 Propagation Rules

1. `Authorization` is the inbound UI-facing auth header.
2. `x-authorization` is the downstream private user-context forwarding header.
3. `x-api` or equivalent is the machine-to-machine auth header.
4. Downstream service credentials and end-user tokens serve different purposes and shall not be conflated.
5. MDU shall not log raw token or secret values.
6. MDU shall preserve traceability across all orchestrated downstream calls.
7. MDU should propagate actor context, request ID, correlation ID, and user context needed for downstream auditing.

---

# 10. Validation, Error Handling, and NFR Rules

## 10.1 Validation Rules

1. Path parameters shall be validated for presence and basic format.
2. Query parameters shall be validated for type, bounds, and allowed enumerations where defined.
3. Request bodies shall be validated for required fields, field types, and illegal combinations before downstream orchestration.
4. Unsupported enum values shall return a validation error.
5. MDU shall validate allow-listed action names for OWGW-backed action routes.
6. MDU shall not accept write requests that implicitly create local truth for downstream-owned domains.
7. For list APIs such as users, customers, entities, venues, and devices, MDU should normalize pagination, filtering, and sorting behavior even when downstream implementations differ.

## 10.2 Error Rules

MDU shall expose a consistent error envelope for UI-facing APIs.

Normalized error categories should include:

- `validation_error`
- `unauthorized`
- `forbidden`
- `not_found`
- `conflict`
- `downstream_timeout`
- `downstream_unavailable`
- `partial_data`
- `internal_error`

Rules:

1. MDU shall not expose raw downstream stack traces, secrets, or internal implementation details.
2. MDU shall map downstream failures into stable UI-facing semantics.
3. Pure write APIs should not return partial success unless the workflow is explicitly modeled to do so.
4. Composed read APIs may return partial data only if the route contract explicitly allows it.

## 10.3 Retry, Timeout, and Failure-Isolation Rules

1. MDU shall remain a mostly stateless orchestration service in early phases and shall not add durable idempotency or workflow state unless a later approved phase requires it.
2. MDU shall not automatically retry write or action calls unless the downstream operation is explicitly safe to retry.
3. MDU may use limited retries for safe read calls when failures are transient.
4. Every downstream call shall use a bounded timeout, and MDU shall return normalized timeout or dependency-failure errors.
5. Composed read APIs shall use bounded fan-out and shall not expand hierarchy, venue, device, or child-node data without limits in a single request.
6. If an optional read dependency such as Billing, Analytics, or Topology is unavailable, MDU may return partial data only where the route contract explicitly allows it.
7. Write APIs shall fail cleanly and shall not return partial success unless the workflow is explicitly designed for it.

## 10.4 Device Action Safety Rules

1. Device action APIs shall use an approved allow-list of supported actions.
2. MDU shall not expose raw command passthrough or unrestricted runtime command execution.
3. Device action routes shall define clear timeout, validation, and normalized error-mapping behavior.

## 10.5 Operational and NFR Rules

1. MDU shall be observable, traceable, and safe to operate as a dependency-orchestration service.
2. MDU shall expose health and readiness endpoints according to platform conventions.
3. Structured logs shall include request ID, correlation ID, route or operation name, downstream service name where applicable, and normalized result code.

---

# 11. Development Phases

The phases are cumulative. Each phase shall preserve ownership boundaries and security rules from previous phases.

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

### Expected API Families

- `GET /api/v1/mdu/me`
- `GET /api/v1/mdu/session`
- `/api/v1/mdu/users/*`
- `/api/v1/mdu/operators/*`
- `/api/v1/mdu/entities/*`
- `/api/v1/mdu/venues/*`
- `/api/v1/mdu/policies/*`
- `/api/v1/mdu/roles/*`
- `/api/v1/mdu/customers/*`

### Completion Criteria

Phase 1 is complete only when:

1. protected business APIs validate OWSEC bearer tokens
2. downstream private calls use service auth plus forwarded user context where required
3. scaffold placeholder APIs are removed or isolated from production routes
4. session/bootstrap and core PROV wrapper APIs are available
5. normalized error handling and structured correlation are implemented
6. source-of-truth ownership rules remain preserved

## Phase 2: Customers, Hierarchy, and Billing Workspaces

### Objective

Make MDU useful for customer/sub-operator workspace workflows and hierarchy navigation while keeping PROV and Billing as systems of record.

### Scope

Phase 2 includes:

- customer lifecycle APIs backed by PROV
- customer bootstrap orchestration backed by PROV and OWSEC as applicable
- hierarchy browsing and lazy child loading backed by PROV
- workspace context APIs
- billing summary integration backed by Billing Service
- customer workspace payloads for UI tabs

### Non-Goals

Phase 2 does not require:

- live device actions
- topology graph APIs
- analytics dashboards
- durable workflow engine
- broad async jobs

### Expected API Families

- `/api/v1/mdu/customers/*`
- `/api/v1/mdu/hierarchy/*`
- `/api/v1/mdu/workspaces/*`
- `/api/v1/mdu/bootstrap/*` where approved

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

### Non-Goals

Phase 3 does not require:

- full topology workspaces
- historical analytics dashboards
- rollout engine
- AI hooks

### Expected API Families

- `/api/v1/mdu/devices/*`
- `/api/v1/mdu/configurations/*`
- `/api/v1/mdu/venues/{venueId}/devices`
- `/api/v1/mdu/devices/{serialNumber}/actions/*`

## Phase 4: Topology, Analytics, Metrics, and Health

### Objective

Add cross-domain operational visibility using topology and analytics as first-class downstream dependencies.

### Scope

Phase 4 includes:

- topology workspace APIs backed by NW Topology Service
- analytics summaries backed by OWANALYTICS
- KPI and health panels
- map/overlay payloads
- workspace overview aggregation

### Expected API Families

- `/api/v1/mdu/topology/*`
- `/api/v1/mdu/analytics/*`
- `/api/v1/mdu/maps/*`
- `/api/v1/mdu/metrics/*`
- `/api/v1/mdu/dashboard/*`

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

---

# 12. Final Architectural Rules

1. `mango-mdu-service` is the Mango operator-domain orchestration service.
2. The UI authenticates directly with OWSEC.
3. OWSEC owns login, session issuance, bearer token validity, and token validation primitives.
4. The UI calls MDU with `Authorization: Bearer <owsec-token>`.
5. MDU validates the inbound token before protected business workflows.
6. MDU calls downstream services with service authentication such as `x-api`.
7. MDU forwards user context to downstream services using `x-authorization` where required.
8. PROV owns users, customers, hierarchy, policies, roles, RBAC, inventory ownership, and configuration ownership.
9. Billing Service owns billing truth.
10. OWGW owns live device runtime and command execution.
11. NW Topology Service owns topology graph computation.
12. OWANALYTICS owns telemetry and historical analytics.
13. MDU owns orchestration, request validation, API shaping, response composition, error normalization, and future approved workflow state.
14. MDU shall not become a second source of truth for downstream-owned domains.
