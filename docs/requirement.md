# Mango MDU Service Requirements

## Document Purpose

This document defines the **master requirements** for the full lifecycle of `mango-mdu-service`.

MDU Service is the Mango Cloud orchestration service for operator-facing workflows. Its role is to provide a stable Mango-facing API layer between the UI and the broader OpenWiFi / Mango service ecosystem, including identity, provisioning, billing, gateway runtime operations, topology, and analytics.

This document is intentionally broader than a Phase 1 implementation document. It defines:

- service mission and architectural role
- development phases
- ownership and source-of-truth boundaries
- downstream dependency model
- whole-service API-family roadmap
- data and persistence rules
- implementation constraints for the Go microservice

This document is the **whole-service roadmap and scope baseline**.

Cross-phase implementation, delivery, runtime, CI, and security guardrails are defined separately in `mango-mdu-common-requirements.md`. Detailed phase-specific requirements documents, beginning with Phase 1, shall inherit from both this master document and the common requirements document.

---

## Document Control

| Field | Value |
|---|---|
| Document Title | Mango MDU Service Requirements |
| Service Name | `mango-mdu-service` |
| Repository | `routerarchitects/mango-mdu-service` |
| Service Type | Internal orchestration microservice |
| Primary Language | Go (Golang) |
| Current Purpose | Master service requirements |
| Status | Draft master baseline |
| Last Updated | 2026-06-16 |
| Primary Consumers | Mango Operator UI and approved internal callers |
| Primary Downstream Services | OWSEC, PROV, Billing Service, OWGW, NW Topology Service, OWANALYTICS |

---

# 1. Executive Summary

MDU Service shall be the Mango-facing orchestration layer for the MDU product domain.

It shall expose stable `/api/v1/mdu/*` contracts for UI and internal consumers while coordinating workflows across downstream systems that own identity, hierarchy, RBAC, billing, live device operations, topology, and analytics.

MDU Service shall **not** become the source of truth for domains already owned by downstream services. Instead, it shall:

- validate and normalize requests
- enforce scope-aware behavior
- compose and shape UI-facing responses
- abstract away downstream route quirks and implementation details
- provide a consistent contract for customer, hierarchy, access, device, billing, topology, and analytics workflows
- evolve over phases from a foundational orchestrator into a broader cross-domain aggregation service

The service shall start with a strict verified baseline for OWSEC and PROV interactions and then expand phase by phase into customer workspaces, billing integration, live device operations, topology, analytics, and eventually durable workflow management and advanced admin/automation capabilities.

---

# 2. Development Phases

| Phase | Scope |
|---|---|
| Phase 1 | Orchestrator foundation, auth/session context, user lifecycle, operator/entity/venue/policy/role wrappers, access summary, and verified downstream contract normalization |
| Phase 2 | Customer/sub-operator lifecycle, hierarchy workspace APIs, subtree navigation, lazy-loading pagination, billing-facing workspace integration, and customer bootstrap orchestration |
| Phase 3 | Device inventory wrappers, venue-device assignment, live device state integration via OWGW, configuration wrappers, and operational device/config UX support |
| Phase 4 | Topology, analytics, maps, metrics, health/workspace aggregation, client visibility foundations, and cross-domain operational views |
| Phase 5 | Workflow hardening, idempotency, async job model, auditability, reconciliation, selective saga/forward-recovery for real multi-system workflows, rollout orchestration, AI hooks, and advanced admin/debug maturity |

The phases are cumulative. Each phase builds on the previous one and must preserve already-established ownership boundaries for downstream systems.

### Phase 1 intent
Establish the verified OWSEC and PROV integration baseline and the core service foundation.

### Phase 2 intent
Introduce the higher-level customer/sub-operator and hierarchy-facing APIs the UI actually needs, including billing-aware workspace summaries.

### Phase 3 intent
Make the service operationally useful for device and configuration management by combining PROV ownership data with OWGW live/runtime integrations.

### Phase 4 intent
Expand the product into richer operational visibility through topology, analytics, maps, metrics, health, and aggregated workspace views.

### Phase 5 intent
Introduce durable workflow safety, async processing, reconciliation, rollout-oriented orchestration, AI hooks, and advanced operational maturity features.

---

# 3. Source Priority and Conflict Resolution

This document is synthesized from multiple inputs, but not all inputs have equal authority.

The service requirements and implementation shall follow this priority order whenever there is a conflict.

## 3.1 Absolute Authority for Phase 1

The uploaded Phase 1 downstream mapping for PROV and UCENTRALSEC is the highest authority for Phase 1-specific ownership, endpoint mappings, and workflow rules.

For Phase 1, that means:

- OWSEC owns identity/session/admin-user lifecycle
- PROV owns operators, entities, venues, policies, roles, and persisted RBAC scope structure
- MDU must not become a separate RBAC source of truth
- MDU must obey verified non-standard downstream route patterns

## 3.2 Current Repository State

The current `mango-mdu-service` repository defines the implementation baseline, package structure, app bootstrap, migration pattern, and deployment shape.

It does **not** define domain truth. The repo README explicitly states that the sample CRUD, sample schema, and `/api/v1/items` endpoints are placeholder scaffold code only. The checked-in design doc says the same for the sample design flow and schema. fileciteturn14file0

## 3.3 Functional Scope Discovery

The operator UI Phase 1 documentation is authoritative for user workflows, screen expectations, navigation model, hierarchy behavior, visible actions, and the need for a backend orchestration layer that hides raw downstream complexity.

## 3.4 Format Reference

The Billing Service requirements document is used as the structural and stylistic reference for the depth, clarity, and phase-oriented format expected of this master specification.

## 3.5 Future Breadth Reference

The earlier `mdu_api_flow_rbac_saga_pagination(2).md` draft is used as a future-scope and API-breadth reference.

It is useful for:

- understanding likely future API families
- seeing how broad the service becomes over time
- identifying likely later-phase durability needs
- understanding customer/billing/device/configuration workflow breadth
- understanding lazy-loading and pagination direction

It is **not** authoritative for Phase 1 ownership or RBAC design.

## 3.6 Additional Whole-Service Downstream Inputs

The OWGW, NW Topology, and OWANALYTICS OpenAPI documents are authoritative for understanding the broader whole-service dependency ecosystem.

These inputs make clear that MDU is not only a PROV/OWSEC wrapper. It must ultimately orchestrate across:

- identity and security
- provisioning and RBAC
- billing
- live device operations
- topology
- telemetry and analytics

---

# 4. Repository Baseline and Implementation Starting Point

The current `mango-mdu-service` repository is a service scaffold with the correct implementation shape but placeholder business behavior. fileciteturn14file0

The repository already provides:

- Go service entrypoint and boot sequence
- app wiring and dependency injection structure
- HTTP routing and middleware area
- PostgreSQL integration and startup migrations
- service-layer and model-layer package structure
- an external integrations boundary
- Docker and Compose artifacts
- CI workflow
- docs placeholders

This is valuable because the service does **not** need a new architectural skeleton. It needs the placeholder logic replaced with the real orchestrator design defined in this document.

## 4.1 Repository Constraints

The implementation shall preserve the repository’s clean structure:

- `cmd/` for startup
- `internal/app/` for wiring
- `internal/http/` for transport and middleware
- `internal/services/` for orchestration/use-cases
- `internal/models/` for normalized contracts and DTOs
- `external/` for downstream clients
- `internal/db/` only for MDU-owned persistence concerns

## 4.2 Repository Non-Truth Rule

The existing scaffold docs and sample schema must not be treated as domain truth.

No final API, domain, or schema decision shall be derived from the sample items CRUD or sample tables.

---

# 5. Service Mission and Architectural Role

## 5.1 Mission

MDU Service shall provide a stable orchestration and aggregation layer for the Mango operator domain.

It shall allow the UI and approved internal consumers to interact with the Mango/OpenWiFi ecosystem through one coherent MDU-facing API surface instead of directly coupling the frontend to many backend services.

## 5.2 Core Role

MDU Service is a **backend orchestrator and domain aggregation service**.

That means it shall:

- validate incoming requests
- validate identity and access
- resolve hierarchy context
- orchestrate downstream calls
- compose responses from multiple systems
- hide backend service complexity
- normalize errors
- expose business-oriented API contracts
- evolve over time into a richer cross-domain workspace service

## 5.3 What MDU Service Is Not

MDU Service shall not:

- replace OWSEC as the identity or session owner
- replace PROV as the hierarchy or RBAC persistence owner
- replace Billing Service as the billing owner
- replace OWGW as the live command/runtime device operations owner
- replace topology service as the topology graph computation owner
- replace analytics as the telemetry/time-series owner
- expose raw downstream details when a business-facing contract is more appropriate
- force every workflow into persistent saga state when synchronous orchestration is sufficient

---

# 6. Core Architecture Rules

1. MDU Service shall expose all MDU business APIs under `/api/v1/mdu/*`.
2. MDU Service shall be the Mango-facing orchestration service for MDU workflows.
3. MDU Service shall hide downstream route quirks, compatibility tokens, and service-specific oddities from upstream consumers.
4. MDU Service shall preserve downstream systems as systems of record.
5. OWSEC shall remain the source of truth for authentication, token validation, current caller identity, API-key validation, and admin-user CRUD.
6. PROV shall remain the source of truth for operators, entities, venues, policies, roles, hierarchy persistence, RBAC persistence, inventory ownership, and configuration ownership.
7. Billing Service shall remain the source of truth for billing plans, subscriptions, and billing lifecycle.
8. OWGW shall remain the source of truth for live device runtime operations, command execution, and runtime diagnostics/actions.
9. NW Topology Service shall remain the topology graph computation service.
10. OWANALYTICS shall remain the source of truth for analytics/timepoints/historical telemetry.
11. MDU Service shall not become a separate RBAC source of truth.
12. MDU Service shall not create an alternate durable hierarchy ownership model unless explicitly approved in a future architecture revision.
13. MDU Service shall not create a second inventory or billing source of truth.
14. MDU Service shall use business-oriented API contracts for the UI wherever possible.
15. Admin and debug views may expose deeper downstream concepts where operationally justified.
16. Clean architecture and layered service boundaries shall be preserved.
17. Local durable workflow persistence shall be introduced only where the active phase justifies it.
18. Every added downstream dependency must be represented in OpenAPI, service design, observability, and error-mapping behavior.
19. Phase-specific docs shall inherit from this master and the common requirements document and add detail, not contradict them.

---

# 7. Service Ownership and Boundaries

| Data / Workflow | Owner |
|---|---|
| Login / token issuance | OWSEC |
| Token validation | OWSEC |
| API-key validation | OWSEC |
| Current caller profile | OWSEC |
| Admin-user CRUD | OWSEC |
| Operators | PROV |
| Entities | PROV |
| Venues | PROV |
| Management policies | PROV |
| Management roles | PROV |
| Persisted RBAC scope structure | PROV |
| Inventory records | PROV |
| Configuration records | PROV |
| Billing plans | Billing Service |
| Billing subscriptions | Billing Service |
| Billing lifecycle state | Billing Service |
| Live device command/runtime state | OWGW |
| Topology graph computation | NW Topology Service |
| Device/client telemetry and historical analytics | OWANALYTICS |
| UI-facing orchestration across the above | MDU Service |
| Scope-aware workspaces and composed responses | MDU Service |
| Future approved durable workflow state | MDU Service |

## 7.1 Boundary Principle

MDU Service owns **orchestration**, not the raw domain systems listed above.

## 7.2 MDU-Owned Concerns

MDU shall own:

- API shaping
- request validation
- service-layer authorization
- hierarchy-aware response composition
- user-facing workflow orchestration
- future workflow state where explicitly approved

---

# 8. Whole-Service Domain Model

Over its full lifecycle, MDU Service is expected to expose or orchestrate the following domains:

- session and bootstrap
- current user context
- users
- access summary
- operators
- customers / sub-operators
- hierarchy nodes
- entities
- venues
- policies
- roles
- billing workspaces
- devices
- live device status and actions
- configurations
- topology
- analytics and time-series summaries
- maps and overlays
- clients
- metrics
- health summaries
- rollout workflows
- AI hooks
- admin/debug views

## 8.1 UI Terminology Rule

The UI docs make clear that business-oriented terminology should dominate the normal API surface.

Therefore, the MDU-facing model should prefer:

- customer
- sub-operator
- node
- site
- building
- tower
- floor
- venue
- user
- device
- configuration
- subscription
- workspace
- access summary

instead of directly exposing raw downstream internal terms in normal workflows.

### Exception
Admin/debug APIs may surface underlying downstream concepts such as `managementPolicy` or `managementRole` when operationally useful.

---

# 9. Downstream Service Ecosystem

## 9.1 OWSEC / UCENTRALSEC

OWSEC provides:

- login/session routes
- token validation
- sub-token validation
- API-key validation
- current caller profile
- admin-user CRUD

This is the security and identity anchor of the service.

## 9.2 PROV / OWPROV

PROV provides:

- operators
- entities
- venues
- management policies
- management roles
- hierarchy persistence
- RBAC persistence and scope structure
- inventory objects
- configuration objects

This is the hierarchy, access, inventory, and configuration anchor of the service.

## 9.3 Billing Service

Billing Service provides:

- billing plans
- subscriptions
- billing summaries and states
- plan-selection/assignment workflows from the billing domain side

This is the billing anchor of the service.

## 9.4 OWGW / uCentral Gateway

OWGW provides runtime device operations and command surfaces.

Its OpenAPI shows a broad live/runtime control domain including command execution records and device actions such as configure, ping, reboot, factory reset, script, LEDs, event queue, RRM, Wi-Fi scan, request, trace, upgrade, reenroll, certificate update, transfer, and package operations. That makes OWGW a key dependency for live device state, diagnostics, support workflows, and device command orchestration. fileciteturn8file1L47-L118

### MDU implication
MDU must distinguish between:

- inventory/source-of-truth device data from PROV
- live/runtime/action state from OWGW

## 9.5 NW Topology Service

Topology is a dedicated topology computation service.

Its `/topology` API explicitly builds topology using analytics board timepoints, board device inventory data, and Wi‑Fi client history, and can tolerate some historical-client lookup failures without failing the whole topology response. That means MDU should treat topology as its own downstream domain rather than attempting to re-implement topology graph assembly internally. fileciteturn9file0L4-L4

### MDU implication
MDU shall use topology service for topology workspaces and topology-oriented operational views.

## 9.6 OWANALYTICS

OWANALYTICS gathers statistics about devices used in OpenWiFi and groups them by provisioning entities or venues. Its OpenAPI defines timepoint-style device and Wi‑Fi client telemetry structures used for trends, history, and metrics views. fileciteturn11file0L12-L19 fileciteturn12file0L4-L4

### MDU implication
MDU shall use analytics for telemetry summaries, trends, historical views, KPI panels, and future overlays/maps/health experiences.

---

# 10. Verified Phase 1 Integration Rules

Phase 1 shall follow the exact verified downstream mapping for OWSEC and PROV.

## 10.1 Verified OWSEC Routes

Verified OWSEC routes include:

- `GET /oauth2?me=true`
- `POST /oauth2`
- `DELETE /oauth2/{token}`
- `GET /validateToken?token=...`
- `GET /validateSubToken?token=...`
- `GET /validateApiKey?apikey=...`
- `GET /users`
- `GET /user/{id}`
- `POST /user/0`
- `PUT /user/{id}`
- `DELETE /user/{id}`

## 10.2 Verified PROV Routes

Verified PROV routes include:

- `GET /operator`
- `GET /operator/{uuid}`
- `POST /operator/{uuid}`
- `PUT /operator/{uuid}`

- `GET /entity`
- `GET /entity/{uuid}`
- `POST /entity/{uuid}`
- `PUT /entity/{uuid}`
- `POST /entity?setTree=true`

- `GET /venue`
- `GET /venue/{uuid}`
- `POST /venue/{uuid}`
- `PUT /venue/{uuid}`

- `GET /managementPolicy`
- `GET /managementPolicy/{uuid}`
- `POST /managementPolicy/{uuid}`

- `GET /managementRole`
- `GET /managementRole/{id}`
- `POST /managementRole/{id}`

## 10.3 Critical Verified Route Quirks

The following quirks are mandatory for Phase 1:

1. PROV create routes are compatibility detail routes, not collection POST routes.
2. The path token used for those create routes is not the created resource ID.
3. OWSEC admin-user create uses `POST /user/0`.
4. Operator create auto-creates the linked entity and MDU must not create a second entity for that workflow.
5. Normal entity creation uses `POST /entity/{uuid}` and not the tree-import route.
6. Current caller profile uses `GET /oauth2?me=true`.
7. API-key validation must use `/validateApiKey?apikey=...`.

---

# 11. Phase 1 Requirements

Phase 1 is the verified orchestrator foundation.

## 11.1 Objectives

Phase 1 shall:

- establish auth/session middleware
- provide current-user/session context
- provide user lifecycle through OWSEC
- provide operator/entity/venue/policy/role wrappers through PROV
- provide access-summary and access-assignment orchestration
- normalize verified downstream contracts into clean MDU-facing APIs

## 11.2 Scope

Phase 1 includes:

### Session and Auth-Aware APIs
- token validation middleware
- API-key validation middleware where required
- `GET /api/v1/mdu/me`

### User APIs
- create user
- list users
- get user detail
- update user
- delete user
- effective access summary

### Operator APIs
- create operator
- list operators
- get operator
- update operator

### Entity APIs
- create entity
- list entities
- get entity
- update entity
- list child entities

### Venue APIs
- create venue under entity
- list entity venues
- get venue
- update venue
- list child venues

### Policy APIs
- create policy
- list policies
- get policy

### Role APIs
- create role
- list roles
- get role

### Access Assignment APIs
- assign entity-scoped access
- assign role-based access
- assign policy-linked access
- get user access summary

## 11.3 Functional Requirements

### 11.3.1 Session and Current Caller Context
MDU shall provide a composed `/me` endpoint that validates the token, resolves the current caller through OWSEC, and enriches the result with PROV-derived access and scope information.

### 11.3.2 User Lifecycle
MDU shall expose stable user APIs while translating those calls to OWSEC admin-user endpoints and preserving downstream restrictions such as the use of `/user/0` for create.

### 11.3.3 Provisioning Wrappers
MDU shall expose clean wrapper APIs for operators, entities, venues, policies, and roles while hiding PROV’s compatibility-detail create routes from upstream consumers.

### 11.3.4 Access Assignment
MDU shall compose user access workflows using PROV role and policy objects instead of creating local RBAC truth.

### 11.3.5 Error and Observability Baseline
Every Phase 1 endpoint shall participate in the standardized error envelope and observability model defined by the common requirements document.

## 11.4 Phase 1 API Inventory

### Session
- `GET /api/v1/mdu/me`

### Users
- `POST /api/v1/mdu/users`
- `GET /api/v1/mdu/users`
- `GET /api/v1/mdu/users/{userId}`
- `PATCH /api/v1/mdu/users/{userId}`
- `DELETE /api/v1/mdu/users/{userId}`
- `GET /api/v1/mdu/users/{userId}/access`

### Operators
- `POST /api/v1/mdu/operators`
- `GET /api/v1/mdu/operators`
- `GET /api/v1/mdu/operators/{operatorId}`
- `PATCH /api/v1/mdu/operators/{operatorId}`

### Entities
- `POST /api/v1/mdu/entities`
- `GET /api/v1/mdu/entities`
- `GET /api/v1/mdu/entities/{entityId}`
- `PATCH /api/v1/mdu/entities/{entityId}`
- `GET /api/v1/mdu/entities/{entityId}/children`

### Venues
- `POST /api/v1/mdu/entities/{entityId}/venues`
- `GET /api/v1/mdu/entities/{entityId}/venues`
- `GET /api/v1/mdu/venues/{venueId}`
- `PATCH /api/v1/mdu/venues/{venueId}`
- `GET /api/v1/mdu/venues/{venueId}/children`

### Policies
- `POST /api/v1/mdu/policies`
- `GET /api/v1/mdu/policies`
- `GET /api/v1/mdu/policies/{policyId}`

### Roles
- `POST /api/v1/mdu/roles`
- `GET /api/v1/mdu/roles`
- `GET /api/v1/mdu/roles/{roleId}`

### Access Assignment
- `POST /api/v1/mdu/users/{userId}/entities/{entityId}`
- `POST /api/v1/mdu/users/{userId}/roles`
- `POST /api/v1/mdu/users/{userId}/policies`

## 11.5 Deliverables

Phase 1 shall deliver:

- production-grade service bootstrap
- auth/session middleware
- normalized user/provisioning/access wrapper APIs
- OpenAPI for implemented endpoints
- integration adapters for OWSEC and PROV
- removal of placeholder scaffold APIs from the production route surface

## 11.6 Explicit Non-Goals for Phase 1

Phase 1 does not yet require:

- customer/sub-operator business APIs as first-class concepts
- billing workspace aggregation
- full device runtime/live-state aggregation
- topology aggregation
- analytics aggregation
- async workflow persistence
- saga/compensation model for normal CRUD flows
- rollout orchestration
- AI hooks

---

# 12. Phase 2 Requirements

Phase 2 makes the service product-facing for customer/sub-operator and hierarchy workspaces.

## 12.1 Objectives

Phase 2 shall:

- introduce customer/sub-operator lifecycle APIs
- implement hierarchy browsing and subtree navigation
- support lazy-loading hierarchy APIs for the UI
- support customer bootstrap flows
- expose customer workspace APIs
- integrate billing-facing workspace summaries
- make the service useful for the actual operator portal experience

## 12.2 Scope

Phase 2 includes:

### Customer / Sub-Operator Lifecycle APIs
- create child customer/sub-operator
- list permitted customers/sub-operators
- get customer detail
- update customer metadata and status
- expose customer workspace summary
- expose child-scope and parent-scope relationships needed by the UI

### Customer Bootstrap Orchestration
- create first admin user through OWSEC
- create or resolve required hierarchy scope through PROV
- create or resolve required access structures through PROV
- attach customer bootstrap metadata required by the UI flow
- optionally initiate or attach billing-selection context where product-approved

### Hierarchy Workspace APIs
- subtree roots
- node detail
- node children
- lazy child expansion with pagination metadata
- breadcrumb/context payloads
- workspace tab support payloads
- scope switching support for the UI shell

### Billing-Facing Workspace Integration
- current subscription summary
- available plans summary
- direct-child billing visibility
- customer billing summary panels
- operator-facing child billing list summaries where approved

## 12.3 Functional Requirements

### 12.3.1 Customer/Sub-Operator Domain Abstraction
MDU shall introduce the MDU-facing business concepts of customer and sub-operator even when the underlying implementation uses downstream hierarchy objects. This abstraction shall remain stable for the UI.

### 12.3.2 Customer Create Workflow
A customer/sub-operator create workflow shall be orchestrated by MDU using downstream services. At minimum the workflow must support:
- customer details
- first admin user details
- scoped hierarchy placement
- initial access setup
- optional billing context linkage

### 12.3.3 Customer Workspace
MDU shall expose customer workspace payloads that aggregate the minimum data needed for:
- overview
- users
- hierarchy
- billing summary
- operational summary placeholders for later phases

### 12.3.4 Hierarchy Navigation
MDU shall support hierarchy-first navigation with APIs that allow:
- subtree root loading
- child expansion
- context switching
- stable node identifiers
- breadcrumb-friendly path information
- tab/workspace composition around the selected scope

### 12.3.5 Lazy-Loading and Pagination
Recursive hierarchy APIs shall avoid full unbounded tree responses by default. They shall support child counts, paged child loading, and stable continuation semantics suitable for tree UIs.

### 12.3.6 Billing Summary Composition
MDU shall aggregate Billing Service data into customer and operator workspaces while keeping Billing Service as the system of record for plan/subscription truth.

## 12.4 API Families Expected in Phase 2

Examples of expected Phase 2 API families include:

- `/api/v1/mdu/customers`
- `/api/v1/mdu/customers/{customerId}`
- `/api/v1/mdu/customers/{customerId}/workspace`
- `/api/v1/mdu/customers/{customerId}/users`
- `/api/v1/mdu/customers/{customerId}/billing`
- `/api/v1/mdu/hierarchy`
- `/api/v1/mdu/hierarchy/{nodeId}`
- `/api/v1/mdu/hierarchy/{nodeId}/children`
- `/api/v1/mdu/workspaces/{nodeId}/context`
- `/api/v1/mdu/bootstrap/*` style APIs if approved

## 12.5 Deliverables

Phase 2 shall deliver:

- stable customer/sub-operator API contracts
- hierarchy APIs suitable for the UI shell and hierarchy tree
- billing summary integration contracts
- first-admin bootstrap orchestration
- paginated hierarchy child loading
- customer workspace payloads for the UI

## 12.6 Design Rule

Even when the UI uses the language “customer” and “sub-operator,” MDU should still treat these as orchestrated concepts over underlying systems unless a later architecture explicitly assigns durable local ownership.

## 12.7 Explicit Non-Goals for Phase 2

Phase 2 does not yet require:

- live device runtime aggregation
- device command orchestration
- full topology APIs
- analytics/time-series views
- durable workflow engine
- async jobs
- large-batch recovery models
- maps or AI features

---

# 13. Phase 3 Requirements

Phase 3 expands MDU into device and configuration orchestration.

## 13.1 Objectives

Phase 3 shall:

- make the service operationally useful for device and configuration workflows
- integrate live device data and actions from OWGW
- wrap inventory/configuration data from PROV
- provide device and configuration contracts suitable for the UI

## 13.2 Scope

Phase 3 includes:

### Device Inventory Wrappers
- device inventory list/detail/create/update/delete
- bulk device import
- serial-based lookup flows
- entity and venue scoped inventory filtering
- inventory status normalization for UI-facing tables and detail pages

### Venue Device Assignment
- assign devices to venues
- unassign devices
- list devices by venue
- move devices between compatible scopes where approved
- batch assignment and import summaries

### Live Device State via OWGW
- current live status snapshot
- online/offline and last-contact style state surfaces
- support-safe diagnostics summaries
- safe action wrappers for approved commands
- command submission and result tracking where needed by the UI
- runtime failure normalization

### Configuration Wrappers
- configuration list/detail/create/update/delete
- venue configuration assignment
- AP-specific configuration assignment
- effective configuration preview
- apply configuration workflows
- configuration validation/error surfacing

## 13.3 Functional Requirements

### 13.3.1 Inventory vs Live State Separation
MDU shall keep inventory and live state separate in the orchestration layer. PROV-backed inventory data and OWGW-backed runtime state shall be composed deliberately rather than merged into an uncontrolled shadow model.

### 13.3.2 Device Detail Composition
Device detail APIs shall progressively combine:
- inventory metadata
- venue and hierarchy placement
- live runtime state
- action eligibility
- configuration linkage

### 13.3.3 Device Actions
MDU shall expose only approved device actions through validated MDU contracts. Each action must include:
- authorization checks
- request validation
- safe command mapping to OWGW
- result/status normalization
- audit visibility

### 13.3.4 Batch and Import Flows
Batch device imports and assignment flows shall return structured success/failure breakdowns that allow the UI and operators to understand what happened without inspecting raw downstream responses.

### 13.3.5 Configuration Management
MDU shall expose stable configuration APIs that abstract away the underlying complexity of configuration records, venue assignments, and AP-specific overrides.

### 13.3.6 Effective Configuration Preview
MDU shall support “effective configuration” views that help the UI show what is actually applied or would be applied to a device or venue after inheritance and assignment are considered.

## 13.4 Data Composition Rule for Devices

MDU shall not flatten all device concerns into one fake source of truth.

It shall explicitly recognize:

1. **Inventory truth** from PROV
2. **Live/runtime truth** from OWGW
3. **Historical telemetry** from analytics
4. **Topology relationships** from topology service

The MDU-facing device APIs shall progressively compose these concerns into coherent UI-facing responses.

## 13.5 API Families Expected in Phase 3

Examples of expected Phase 3 API families include:

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

## 13.6 Deliverables

Phase 3 shall deliver:

- device inventory API surface
- venue assignment API surface
- live status and diagnostics composition
- configuration management API surface
- effective configuration preview/apply endpoints
- command-safe action wrappers for approved OWGW workflows
- device/configuration UI support contracts

## 13.7 Safety Rule

MDU shall not expose every OWGW command raw and unfiltered to end users. It shall expose only approved device actions behind validated MDU contracts with authorization, auditability, and response normalization.

## 13.8 Explicit Non-Goals for Phase 3

Phase 3 does not yet require:

- topology graph APIs as a first-class user feature
- analytics trend/history dashboards
- map overlays
- async workflow engine
- rollout orchestration
- AI recommendations
- generalized saga-based recovery

---

# 14. Phase 4 Requirements

Phase 4 expands the service into topology, analytics, maps, metrics, health, and richer operational aggregation.

## 14.1 Objectives

Phase 4 shall:

- provide rich operational visibility views
- integrate topology and analytics as first-class downstream dependencies
- support health, KPI, trend, and historical workspace views
- make MDU the true cross-domain operational workspace backend

## 14.2 Scope

Phase 4 includes:

### Topology
- topology workspace APIs
- node/floor/venue/site/building topology summaries
- device connectivity views
- overlay-ready responses for topology UIs
- topology detail composition with hierarchy context and permissions

### Analytics and Time-Series
- telemetry summaries
- trend/timepoint APIs
- historical device summaries
- historical client summaries
- KPI panels and dashboard widgets
- health scoring / summary models where approved
- scoped analytics for customer and hierarchy workspaces

### Maps and Metrics Foundations
- map/overlay payloads
- venue/floor-level visualization support
- node health summaries
- metrics workspace summaries
- signal/coverage/association density support payloads where approved

### Client Visibility Foundations
- scoped client summaries
- client visibility panels inside workspaces
- client history integration through analytics sources
- client-centric diagnostics support payloads for future phases

## 14.3 Functional Requirements

### 14.3.1 Topology Aggregation
MDU shall consume topology-service outputs and shape them into workspace-friendly responses that include scope, permissions, UI metadata, and optional enrichment from inventory and analytics domains.

### 14.3.2 Analytics Aggregation
MDU shall consume analytics-service outputs and expose summaries, KPIs, historical trends, and scoped operational data without duplicating analytics ownership.

### 14.3.3 Dashboard and Workspace Health Views
MDU shall expose health and overview APIs that combine data from multiple domains to support:
- operator overview dashboards
- customer workspaces
- hierarchy-node workspaces
- device and venue summaries

### 14.3.4 Map and Overlay Support
MDU shall expose payloads that support visual and spatial UI layers while keeping map/topology/analytics responsibilities separated under the orchestration model.

### 14.3.5 Client Visibility
MDU shall introduce client visibility foundations sufficient for UI-level summaries and analytics-backed historical views without yet becoming a full standalone client-management system.

## 14.4 Service-Composition Rule

For topology/analytics/maps/metrics domains, MDU shall aggregate and normalize, but it shall not attempt to replace the specialized downstream services that compute or own those domains.

## 14.5 API Families Expected in Phase 4

Examples include:

- `/api/v1/mdu/topology/*`
- `/api/v1/mdu/analytics/*`
- `/api/v1/mdu/maps/*`
- `/api/v1/mdu/metrics/*`
- `/api/v1/mdu/workspaces/{nodeId}/health`
- `/api/v1/mdu/workspaces/{nodeId}/overview`
- `/api/v1/mdu/clients/*`
- `/api/v1/mdu/dashboard/*`

## 14.6 Deliverables

Phase 4 shall deliver:

- topology workspace API contracts
- analytics/time-series summary API contracts
- KPI and health panels for core workspaces
- map/overlay support payloads
- client visibility foundations
- dashboard-ready operational aggregation contracts

## 14.7 Explicit Non-Goals for Phase 4

Phase 4 does not yet require:

- generalized durable workflow engine
- broad async job framework
- full reconciliation system
- rollout orchestration engine
- AI execution hooks
- advanced admin/debug toolchain beyond what is needed to support topology/analytics operations

---

# 15. Phase 5 Requirements

Phase 5 introduces workflow durability and advanced operational maturity.

## 15.1 Objectives

Phase 5 shall:

- provide durable workflow safety where truly needed
- support idempotency
- support async jobs
- support reconciliation and forward recovery
- support rollout orchestration
- support AI hooks and advanced admin/debug maturity
- make the service safe for large and long-running workflows

## 15.2 Scope

Phase 5 includes:

### Workflow Hardening
- idempotency keys
- mutation deduplication
- stored workflow/result status where approved
- retry-safe write APIs
- support-safe replay semantics

### Async Job Model
- background jobs
- job status visibility
- long-running operation tracking
- worker health and supportability
- operator-facing operation status endpoints where needed

### Auditability and Reconciliation
- mutation audit logs
- failed-workflow visibility
- reconcile-needed statuses
- operational repair/admin tooling
- support/debug visibility for orchestration outcomes

### Selective Saga / Forward Recovery
- compensation only for real multi-system workflows that justify it
- forward-recovery models where rollback is not correct
- import/onboarding/delete teardown durability for selected workflows
- persisted workflow step tracking where approved

### Rollouts / AI / Advanced Admin
- rollout orchestration hooks
- AI recommendation/explainability hooks
- advanced debug/admin APIs
- operational diagnostics endpoints
- support workflows for manual intervention and recovery

## 15.3 Functional Requirements

### 15.3.1 Idempotency
MDU shall support idempotent write behavior for selected mutation APIs using stable keys, request hashing, stored outcome references, and safe duplicate handling semantics.

### 15.3.2 Long-Running Workflow Visibility
MDU shall expose workflow and operation status models for imports, batch operations, customer bootstrap workflows, rollout-like workflows, and other long-running orchestrations.

### 15.3.3 Recovery and Reconciliation
When a workflow spans multiple systems of record and partial completion matters, MDU shall support either:
- forward recovery
- selective compensation
- reconcile-needed state
- manual intervention visibility

### 15.3.4 Rollout-Oriented Orchestration
Where the product introduces staged rollout workflows, MDU shall coordinate them as higher-level orchestrations rather than leaking raw downstream command complexity.

### 15.3.5 AI and Recommendation Hooks
AI-related features shall remain:
- scope-aware
- authorization-aware
- explainable
- approval-gated for side effects

### 15.3.6 Advanced Admin/Debug
MDU shall expose support- and admin-oriented APIs for operational visibility, tracing, repair, and inspection without forcing standard end-user APIs to expose raw backend internals.

## 15.4 Design Rule

Saga/compensation shall not be a universal default. It shall be introduced only where:

1. multiple systems of record are involved
2. partial completion matters
3. retry is insufficient
4. workflow persistence is explicitly justified

## 15.5 API Families Expected in Phase 5

Examples include:

- `/api/v1/mdu/operations/*`
- `/api/v1/mdu/jobs/*`
- `/api/v1/mdu/reconciliation/*`
- `/api/v1/mdu/rollouts/*`
- `/api/v1/mdu/ai/*`
- `/api/v1/mdu/admin/debug/*`
- `/api/v1/mdu/audit/*`

## 15.6 Deliverables

Phase 5 shall deliver:

- idempotency model and storage
- async workflow execution model
- workflow/job status APIs
- audit/reconciliation support
- selective compensation/forward-recovery support
- rollout/AI/admin maturity surfaces
- support tooling visibility for operational incidents

---

# 16. Whole-Service API Family Roadmap

Across all phases, the service is expected to expose these API families.

## 16.1 Session and Bootstrap
- current session
- current scope
- role/access bootstrap
- backend bootstrap support

## 16.2 Users and Access
- user CRUD
- access summary
- policy/role/entity assignment
- future profile abstractions

## 16.3 Customers and Sub-Operators
- create/list/detail/update/status
- first-admin bootstrap
- workspace APIs
- child-scope lifecycle flows

## 16.4 Hierarchy
- roots
- node detail
- node children
- subtree expansion
- breadcrumbs/context payloads
- workspace-tab support

## 16.5 Provisioning Wrappers
- operator/entity/venue wrappers
- role/policy wrappers
- access workflows

## 16.6 Billing Workspaces
- current subscription
- available plans
- child billing summary
- billing visibility by scope

## 16.7 Devices
- inventory CRUD wrappers
- imports
- venue assignment
- live status
- diagnostics
- device actions
- firmware/support hooks

## 16.8 Configurations
- configuration CRUD wrappers
- venue assignment
- AP-specific assignment
- preview/apply

## 16.9 Topology
- topology summaries
- contextual topology workspaces
- connectivity views
- overlay support payloads

## 16.10 Analytics and Metrics
- timepoint summaries
- trend APIs
- historical views
- KPI/health panels
- map/overlay support

## 16.11 Advanced Operational Domains
- maps
- clients
- rollouts
- AI hooks
- admin/debug APIs
- workflow status APIs

---

# 17. Data and Persistence Requirements

## 17.1 Early-Phase Persistence Rule

In early phases, local persistence shall remain minimal and justified.

Valid early uses:

- migration tracking
- operational metadata
- bounded audit support
- request correlation support
- configuration for feature flags if needed
- idempotency support once introduced

Invalid early uses:

- authoritative RBAC persistence
- second hierarchy store
- second inventory store
- second billing store
- speculative ownership tables that conflict with downstream truth

## 17.2 Later-Phase Persistence

Later phases may add MDU-owned persistence for:

- workflow executions
- job state
- compensation/recovery metadata
- import tracking
- audit logs
- reconciliation status
- support/debug metadata
- bounded cached summaries

Every such table must correspond to an approved phase requirement and be documented in migrations and design docs.

## 17.3 Caching Rule

Any cache used by MDU shall be:

- non-authoritative
- TTL-bounded
- safe to lose
- observable
- explicitly scoped to performance optimization rather than ownership transfer

---

# 18. Whole-Service Non-Functional Expectations

Cross-phase implementation, runtime, observability, CI, and security guardrails are defined in `mango-mdu-common-requirements.md`.

At the master level, MDU Service is expected to support:

- interactive UI performance
- bounded downstream composition
- scoped aggregation at hierarchy/customer/device/workspace levels
- phased introduction of durable workflow state only when justified
- stable error envelopes
- production-grade observability and supportability

---

# 19. Recommended Build Sequence

Recommended implementation order:

1. remove scaffold placeholder behavior
2. finalize app wiring, config, downstream adapters, and middleware
3. implement Phase 1 session/user/provisioning/access foundation
4. implement customer/hierarchy/billing workspace APIs
5. implement device/configuration wrappers and safe live-state composition
6. integrate topology and analytics aggregation
7. introduce durable workflow features where justified
8. expand into rollouts, AI hooks, advanced admin/debug

---

# 20. Acceptance Criteria for the Master Requirements

This master document is acceptable only if:

1. It defines the whole service, not only Phase 1.
2. It preserves the exact Phase 1 ownership and downstream rules.
3. It uses the earlier draft as future-breadth guidance without importing outdated ownership assumptions into Phase 1.
4. It incorporates the broader downstream ecosystem: OWSEC, PROV, Billing Service, OWGW, NW Topology Service, and OWANALYTICS.
5. It clearly separates whole-service roadmap concerns from cross-phase engineering guardrails, which now live in the common requirements document.
6. It is compatible with the current repository scaffold while being far more complete than the scaffold docs.
7. It provides a solid base from which detailed phase documents can be written.

---

# 21. Final Architectural Rules

1. `mango-mdu-service` is the Mango operator-domain orchestration service.
2. This document defines the whole-service roadmap and architecture.
3. Phase 1 is governed by the exact verified downstream mapping for OWSEC and PROV.
4. OWSEC owns identity and session truth.
5. PROV owns hierarchy, roles, policies, inventory ownership, configuration ownership, and RBAC persistence truth.
6. Billing Service owns billing truth.
7. OWGW owns live device runtime and command execution surfaces.
8. NW Topology Service owns topology graph computation.
9. OWANALYTICS owns telemetry/timepoint analytics.
10. MDU owns orchestration, API shaping, scope-aware workflow composition, and future cross-domain workspace aggregation.
11. MDU shall not become a second system of record for domains already owned elsewhere unless a future approved architecture explicitly grants that role.
12. Detailed phase specifications shall be produced from this master requirements document, beginning with a detailed Phase 1 document.