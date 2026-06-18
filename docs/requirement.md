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
3. MDU shall not expose a competing login or signup API unless a future approved architecture explicitly adds that requirement.
4. Protected MDU business APIs shall accept `Authorization: Bearer <owsec-token>`.
5. MDU shall validate the inbound bearer token using OWSEC-owned validation mechanisms before executing protected business APIs.
6. MDU shall call downstream private APIs using trusted service credentials such as `x-api` or equivalent.
7. MDU shall forward the inbound user token to downstream private APIs using `x-authorization` when the downstream service needs user context.
8. PROV shall resolve user, customer, scope, role, policy, and RBAC decisions using its own source-of-truth data.
9. MDU shall not become a separate RBAC, hierarchy, customer, user, inventory, configuration, billing, topology, or analytics source of truth.
10. MDU shall normalize downstream responses and errors into stable UI-facing contracts.
11. MDU shall hide downstream route quirks and compatibility paths from the UI.
12. Local persistence in MDU shall be minimal and limited to MDU-owned operational concerns.
13. Durable workflow or job state shall be introduced only in later phases where explicitly justified.
14. Wildcard route families in this master document represent planned namespaces, not final endpoint-level contracts, unless a route is explicitly enumerated.
15. MDU shall remain an orchestration service, not a replacement for downstream domain services.

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
4. Protected business APIs require a valid OWSEC-issued bearer token unless a route is explicitly documented as internal-only machine access.
5. Internal service authentication is mandatory for MDU-to-downstream private calls.
6. Frontend or end-user tokens alone shall not be used as the service-authentication credential for private downstream calls.
7. Logs shall not contain raw bearer tokens, raw `x-api` values, or raw `x-authorization` values.

## 3.3 Common Module Usage

MDU should use approved shared Router Architects and OpenWiFi common modules for logging, build metadata, service discovery, and common client or RPC wrapper behavior where applicable.

MDU should not duplicate shared transport, logging, discovery, or internal-auth infrastructure without approved justification.

## 3.4 API Contract Requirement

1. Every stable MDU API should be represented in OpenAPI 3.x or an approved machine-readable contract format.
2. CI should validate handlers and route behavior against the declared contract as the API surface matures.

## 3.5 Required CI Checks

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

## 5.4 Security Rules

1. OWSEC owns login, token issuance, and token/session validation primitives.
2. MDU validates inbound protected requests before orchestration.
3. MDU authenticates to downstreams using service credentials.
4. MDU forwards user context only where needed by downstream authorization or business logic.
5. Downstream systems retain their own authorization rules.
6. MDU shall not bypass downstream authorization by using only machine credentials for user-sensitive workflows.
7. MDU shall not persist browser bearer tokens.

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

Critical PROV quirks:

1. Create routes may use compatibility detail paths, not collection POST paths.
2. The create-route path token may not be the created resource ID.
3. Operator creation may auto-create linked hierarchy objects; MDU must not duplicate those objects.
4. PROV remains responsible for RBAC enforcement.

## 8.3 Billing Service

Billing Service owns plans, subscriptions, billing lifecycle, invoices, and billing state. MDU may compose billing summaries but shall not persist billing truth.

## 8.4 OWGW

OWGW owns live device runtime operations, command execution, and diagnostics. MDU shall expose only approved and validated action wrappers rather than raw unfiltered command access.

## 8.5 NW Topology Service

NW Topology Service owns topology graph computation. MDU shall consume topology outputs and shape them into workspace-friendly responses.

## 8.6 OWANALYTICS

OWANALYTICS owns telemetry, historical timepoints, client history, and analytics data. MDU shall consume analytics outputs for summaries, trends, health views, and dashboards.

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
8. MDU does not require a local audit database in early phases.

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

1. MDU is primarily a stateless orchestration service and shall not introduce database-backed idempotency, workflow, or webhook ledgers unless a later approved phase requires durable workflow state.
2. MDU shall not automatically retry unsafe write operations unless the downstream operation is explicitly confirmed to be idempotent.
3. Read-oriented downstream calls may use limited retries for transient failures where safe.
4. If the caller provides an `Idempotency-Key`, MDU should forward it to downstream services that support idempotent handling.
5. MDU shall not become the source of truth for idempotency state in early phases.
6. Every downstream call shall use a bounded timeout.
7. As a default guideline, read-oriented downstream calls should use short timeouts, and write or action-oriented downstream calls may use slightly longer bounded timeouts.
8. MDU shall return normalized timeout and dependency-failure errors.
9. Workspace, dashboard, and other composed read APIs shall use bounded fan-out and shall not perform unbounded hierarchy, venue, device, or child-node expansion within a single request.
10. If an optional read-composition dependency such as Billing, Analytics, or Topology is unavailable, MDU may return a degraded or partial read response only where that route contract explicitly allows it.
11. Write APIs shall fail cleanly and shall not return partial success unless the workflow explicitly supports that behavior.

## 10.4 Device Action Safety Rules

1. Device action APIs shall use an approved allow-list of supported actions.
2. MDU shall not expose raw command passthrough or unrestricted runtime command execution.
3. Device action routes shall define clear timeout, validation, and normalized error-mapping behavior.

## 10.5 Operational and NFR Rules

1. MDU shall be observable, traceable, and safe to operate as a dependency-orchestration service.
2. MDU shall expose health and readiness endpoints according to platform conventions.
3. Structured logs shall include request ID, correlation ID, route or operation name, downstream service name where applicable, and normalized result code.

---

# 11. Data and Persistence Rules

## 11.1 Early-Phase Persistence

Early MDU persistence shall be minimal.

Allowed early uses:

- migration tracking
- operational metadata
- correlation/request tracing metadata where needed
- bounded audit support where approved
- feature-flag or local service configuration where justified

Not allowed in early phases:

- local RBAC truth
- local hierarchy truth
- local user/customer truth
- local inventory truth
- local configuration truth
- local billing truth

## 11.2 Later-Phase Persistence

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

## 11.3 Caching

Any cache in MDU shall be:

- non-authoritative
- TTL-bounded
- safe to lose
- observable
- explicitly scoped to performance optimization

---

# 12. Core Workflow Specifications

These workflows define service behavior at master-spec level. Detailed request/response schemas belong in phase-specific docs or API specs.

## 12.1 Login and Session Bootstrap

**Trigger:** UI login and initial app load.

**Systems:** UI, OWSEC, MDU, optionally PROV.

**Flow:**
1. UI authenticates directly with OWSEC.
2. OWSEC returns bearer token to UI.
3. UI calls `GET /api/v1/mdu/session` or `GET /api/v1/mdu/me`.
4. MDU validates the token through OWSEC validation mechanisms.
5. MDU retrieves caller/profile/bootstrap context as required.
6. MDU returns normalized session/bootstrap payload.

## 12.2 Create Customer or Sub-Operator

**Trigger:** authorized UI action to create customer.

**Systems:** UI, MDU, PROV.

**Flow:**
1. UI calls `POST /api/v1/mdu/customers`.
2. MDU validates token and payload.
3. MDU calls PROV using service auth and `x-authorization`.
4. PROV enforces RBAC and creates the source-of-truth object.
5. MDU returns normalized business-facing customer payload.

## 12.3 Assign User Access

**Trigger:** authorized UI action to assign user to entity, role, or policy.

**Systems:** UI, MDU, PROV.

**Flow:**
1. UI calls assignment route.
2. MDU validates token, route scope, and payload.
3. MDU forwards to PROV using machine auth and user context.
4. PROV resolves authorization and persists the access truth.
5. MDU returns normalized access result.

## 12.4 Browse Hierarchy and Load Workspace Context

**Trigger:** user expands tree or opens workspace.

**Systems:** UI, MDU, PROV, optionally Billing, Topology, or Analytics depending on phase.

**Flow:**
1. UI requests hierarchy node or workspace context.
2. MDU validates token and route parameters.
3. MDU loads source hierarchy context from PROV.
4. MDU optionally enriches with billing, topology, or analytics data depending on phase.
5. MDU returns normalized workspace context.

## 12.5 Load Billing Summary

**Trigger:** UI opens customer billing tab.

**Systems:** UI, MDU, Billing Service.

**Flow:**
1. UI calls MDU billing-summary route.
2. MDU validates token and scope.
3. MDU calls Billing Service through approved private service auth.
4. MDU normalizes billing payload and returns UI-facing summary.

## 12.6 Get Device Status and Trigger Approved Device Action

**Trigger:** UI opens device view or runs an approved device action.

**Systems:** UI, MDU, PROV, OWGW.

**Flow:**
1. UI requests device detail, status, diagnostics, or approved action.
2. MDU validates token, identifiers, and action allow-list.
3. MDU retrieves inventory identity from PROV where needed.
4. MDU retrieves runtime state or forwards approved action to OWGW.
5. MDU returns a normalized device-facing response.

Rules:

- Inventory truth remains downstream-owned.
- Runtime truth remains downstream-owned.
- Raw arbitrary command execution is not exposed.

---

# 13. Development Phases

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

# 14. Recommended Build and Delivery Sequence

1. Remove or isolate scaffold placeholder APIs.
2. Finalize configuration, routing, middleware, and downstream adapter structure.
3. Implement OWSEC token-validation middleware.
4. Implement outbound service credentials and `x-authorization` forwarding.
5. Implement Phase 1 session, user, customer, provisioning, and access wrappers backed by PROV/OWSEC.
6. Implement Phase 2 customer, hierarchy, workspace, and billing APIs.
7. Implement Phase 3 device and configuration APIs.
8. Implement Phase 4 topology, analytics, metrics, health, and dashboard APIs.
9. Implement Phase 5 workflow durability, audit, reconciliation, rollout, AI, and admin/debug APIs.

---

# 15. Acceptance Criteria

This master document is acceptable only if:

1. it defines the whole service, not only Phase 1
2. it preserves direct UI login through OWSEC
3. it makes PROV the source of truth for users, customers, hierarchy, roles, policies, and RBAC
4. it makes MDU an orchestration and API-shaping service, not a duplicate source of truth
5. it uses `x-api` or equivalent service credentials for downstream private calls
6. it uses `x-authorization` to forward the user token to downstream services that need user context
7. it keeps RBAC resolution inside PROV for PROV-owned domains
8. it is compatible with the existing Go repository structure
9. it provides a clear base for detailed phase-specific documents and later machine-readable contract generation

---

# 16. YAML Readiness Appendix

This appendix keeps only the metadata needed later for YAML generation.

## 16.1 Service Metadata

| Field | Value |
|---|---|
| service_name | `mango-mdu-service` |
| repository | `routerarchitects/mango-mdu-service` |
| runtime | Go |
| packaging | Docker |
| base_api_path | `/api/v1/mdu` |
| exposure_type | internal Mango orchestration service |

## 16.2 Auth Metadata

- inbound_user_auth: OWSEC bearer token
- downstream_machine_auth: `x-api` or equivalent
- downstream_user_context_header: `x-authorization`
- required_trace_headers: `X-Request-Id`, `X-Correlation-Id`

## 16.3 Dependencies

- OWSEC
- PROV
- Billing Service
- OWGW
- NW Topology Service
- OWANALYTICS

## 16.4 State Rules

- local_domain_truth: not allowed for downstream-owned domains
- workflow_state: later phase only
- job_state: later phase only
- idempotency_records: later phase only
- cache_usage: optional, non-authoritative, TTL-bounded

---

# 17. Final Architectural Rules

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