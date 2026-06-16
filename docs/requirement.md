# Mango MDU Service Phase 1 Requirements

## Document Purpose

This document defines the **Phase 1 requirements baseline** for `mango-mdu-service`.

Phase 1 is intentionally limited to:

- auth and session middleware
- current caller context
- user lifecycle via OWSEC
- operator wrappers via PROV
- entity wrappers via PROV
- venue wrappers via PROV
- management policy wrappers via PROV
- management role wrappers via PROV
- effective access summary
- user-to-role, policy, and entity assignment orchestration

This document does **not** define later-phase billing, topology, analytics, jobs, saga, rollout, or AI behavior.

---

## Document Control

| Field | Value |
|---|---|
| Document Title | Mango MDU Service Phase 1 Requirements |
| Service Name | `mango-mdu-service` |
| Repository | `routerarchitects/mango-mdu-service` |
| Service Type | Internal orchestration microservice |
| Primary Language | Go (Golang) |
| Current Purpose | Phase 1 requirements baseline |
| Status | Draft |
| Last Updated | 2026-06-16 |
| Primary Consumers | Mango Operator UI and approved internal callers |
| Primary Downstream Services | OWSEC, PROV |

---

## 1. Executive Summary

`mango-mdu-service` is the Phase 1 Mango-facing orchestration service for MDU workflows.

In Phase 1, MDU shall provide stable `/api/v1/mdu/*` APIs for session context, users, operators, entities, venues, management policies, management roles, and access assignment workflows, while keeping OWSEC and PROV as the systems of record.

MDU must not become a local RBAC source of truth. It shall validate, orchestrate, normalize, and compose.

---

## 2. Phase 1 Scope

### In Scope

- auth and session middleware
- current caller context
- user lifecycle via OWSEC
- operator wrappers via PROV
- entity wrappers via PROV
- venue wrappers via PROV
- management policy wrappers via PROV
- management role wrappers via PROV
- effective access summary
- user-to-role, policy, and entity assignment orchestration

### Out of Scope

- billing
- OWGW
- topology
- analytics
- jobs
- saga
- async workflows
- rollout
- AI

---

## 3. Verified Phase 1 Downstream Truth

- User create must call `POST /user/0` in OWSEC.
- Current caller must be resolved using `GET /oauth2?me=true`.
- Token validation uses OWSEC validation endpoints.
- Operator create must call `POST /operator/{uuid}`.
- Entity create must call `POST /entity/{uuid}` and **not** the tree import route.
- Venue create must call `POST /venue/{uuid}`.
- Policy create must call `POST /managementPolicy/{uuid}`.
- Role create must call `POST /managementRole/{id}`.
- MDU must not become a local RBAC source of truth.

---

## 4. Core Architecture Rules

1. MDU shall expose Phase 1 APIs under `/api/v1/mdu/*`.
2. MDU shall orchestrate across OWSEC and PROV only in Phase 1.
3. OWSEC shall remain the source of truth for user identity, current caller, token validation, and user CRUD.
4. PROV shall remain the source of truth for operators, entities, venues, management policies, management roles, and persisted RBAC structure.
5. MDU shall not create local RBAC truth.
6. MDU shall not create local hierarchy truth.
7. MDU shall not create local user truth.
8. MDU shall hide downstream route quirks from upstream consumers.
9. For direct wrappers, MDU should reuse downstream schema bodies rather than inventing new wrapper bodies.
10. MDU-specific bodies should exist only where MDU composes or enriches data beyond a direct downstream wrapper.

---

## 5. Verified Downstream Quirks That MDU Must Hide

- PROV create routes are compatibility detail routes.
- callers must not supply the compatibility path token.
- MDU injects it internally.
- `/user/0` is required for user creation.
- operator create auto-creates linked entity.
- normal entity create is not tree import.
- raw route irregularities must not leak into UI-facing contracts.

---

## 6. Local Persistence in Phase 1

- no local RBAC truth
- no local hierarchy truth
- no local user truth
- no local ownership tables for assignment
- only standard service metadata, migrations, and support concerns if needed

---

## 7. Phase 1 Endpoint Inventory

| Domain    | MDU Endpoint                               | Method | Downstream Mapping                                       |
| --------- | ------------------------------------------ | ------ | -------------------------------------------------------- |
| Session   | `/api/v1/mdu/me`                           | GET    | validate token + `GET /oauth2?me=true` + PROV enrichment |
| Users     | `/api/v1/mdu/users`                        | POST   | `POST /user/0`                                           |
| Users     | `/api/v1/mdu/users`                        | GET    | `GET /users`                                             |
| Users     | `/api/v1/mdu/users/{userId}`               | GET    | `GET /user/{id}`                                         |
| Users     | `/api/v1/mdu/users/{userId}`               | PATCH  | `PUT /user/{id}`                                         |
| Users     | `/api/v1/mdu/users/{userId}`               | DELETE | `DELETE /user/{id}`                                      |
| Users     | `/api/v1/mdu/users/{userId}/access`        | GET    | composed OWSEC + PROV                                    |
| Operators | `/api/v1/mdu/operators`                    | POST   | `POST /operator/{uuid}`                                  |
| Operators | `/api/v1/mdu/operators`                    | GET    | `GET /operator`                                          |
| Operators | `/api/v1/mdu/operators/{operatorId}`       | GET    | `GET /operator/{uuid}`                                   |
| Operators | `/api/v1/mdu/operators/{operatorId}`       | PATCH  | `PUT /operator/{uuid}`                                   |
| Entities  | `/api/v1/mdu/entities`                     | POST   | `POST /entity/{uuid}`                                    |
| Entities  | `/api/v1/mdu/entities`                     | GET    | `GET /entity`                                            |
| Entities  | `/api/v1/mdu/entities/{entityId}`          | GET    | `GET /entity/{uuid}`                                     |
| Entities  | `/api/v1/mdu/entities/{entityId}`          | PATCH  | `PUT /entity/{uuid}`                                     |
| Entities  | `/api/v1/mdu/entities/{entityId}/children` | GET    | composed from PROV                                       |
| Venues    | `/api/v1/mdu/entities/{entityId}/venues`   | POST   | `POST /venue/{uuid}`                                     |
| Venues    | `/api/v1/mdu/entities/{entityId}/venues`   | GET    | `GET /venue?entity={entityId}`                           |
| Venues    | `/api/v1/mdu/venues/{venueId}`             | GET    | `GET /venue/{uuid}`                                      |
| Venues    | `/api/v1/mdu/venues/{venueId}`             | PATCH  | `PUT /venue/{uuid}`                                      |
| Venues    | `/api/v1/mdu/venues/{venueId}/children`    | GET    | composed from PROV                                       |
| Policies  | `/api/v1/mdu/policies`                     | POST   | `POST /managementPolicy/{uuid}`                          |
| Policies  | `/api/v1/mdu/policies`                     | GET    | `GET /managementPolicy`                                  |
| Policies  | `/api/v1/mdu/policies/{policyId}`          | GET    | `GET /managementPolicy/{uuid}`                           |
| Roles     | `/api/v1/mdu/roles`                        | POST   | `POST /managementRole/{id}`                              |
| Roles     | `/api/v1/mdu/roles`                        | GET    | `GET /managementRole`                                    |
| Roles     | `/api/v1/mdu/roles/{roleId}`               | GET    | `GET /managementRole/{id}`                               |

---

## 8. Downstream Schema Reuse Rule

For direct Phase 1 wrappers, MDU should reuse downstream body shapes instead of inventing new MDU schemas.

Direct wrapper request and response bodies should reuse:

- Users → `UserInfo`
- Operators → `Operator` / `OperatorList`
- Entities → `Entity` / `EntityList`
- Venues → `Venue` / `VenueList`
- Policies → `ManagementPolicy` / `ManagementPolicyList`
- Roles → `ManagementRole` / `ManagementRoleList`

MDU-owned schemas should exist only for:

- `/api/v1/mdu/me`
- `/api/v1/mdu/users/{userId}/access`
- access-assignment endpoints

---

## 9. Main Schema Field Tables

### 9.1 `UserInfo`

| Field | Type | Required on Create | Mutable on Update | Notes |
|---|---|---:|---:|---|
| `id` | string | no | no | OWSEC user identifier |
| `name` | string | yes | yes | display name |
| `description` | string | no | yes | optional |
| `email` | string | yes | yes | unique user email |
| `avatar` | string | no | yes | optional |
| `validated` | boolean | no | limited | downstream-controlled lifecycle field |
| `notes` | array | no | yes | optional |
| `locale` | string | no | yes | optional |
| `location` | string | no | yes | optional |
| `owner` | string | no | limited | do not treat as MDU ownership |
| `suspended` | boolean | no | yes | admin update use case |
| `blacklisted` | boolean | no | yes | admin update use case |
| `userrole` | string | no | limited | raw downstream field; not MDU RBAC truth |
| `securitypolicy` | string | no | limited | downstream security field |

### 9.2 `Operator`

| Field | Type | Required on Create | Mutable on Update | Notes |
|---|---|---:|---:|---|
| `id` | string | no | no | operator identifier |
| `name` | string | yes | yes | operator display name |
| `description` | string | no | yes | optional |
| `notes` | array | no | yes | optional |
| `created` | integer | no | no | server-managed |
| `modified` | integer | no | no | server-managed |
| `managementPolicy` | string | no | yes | linked policy id or empty |
| `managementRoles` | array | no | limited | downstream-managed linkage |
| `deviceRules` | object | no | yes | downstream shape |
| `variables` | array | no | yes | downstream shape |
| `defaultOperator` | boolean | no | no | downstream-managed |
| `sourceIP` | array | no | yes | optional |
| `registrationId` | string | no | limited | create/use constraints apply |
| `entityId` | string | no | no | linked entity created by PROV |

### 9.3 `Entity`

| Field | Type | Required on Create | Mutable on Update | Notes |
|---|---|---:|---:|---|
| `id` | string | no | no | entity identifier |
| `name` | string | yes | yes | authoritative create field |
| `description` | string | no | yes | mutable |
| `notes` | array | no | yes | mutable |
| `parent` | string | no | limited | authoritative create field |
| `operatorId` | string | no | no | downstream-managed link |
| `children` | array | no | no | derived linkage |
| `venues` | array | no | no | derived linkage |
| `managementPolicy` | string | no | yes | linked policy |
| `managementPolicies` | array | no | no | downstream-managed linkage |
| `managementRoles` | array | no | no | downstream-managed linkage |

### 9.4 `Venue`

| Field | Type | Required on Create | Mutable on Update | Notes |
|---|---|---:|---:|---|
| `id` | string | no | no | venue identifier |
| `name` | string | yes | yes | required |
| `description` | string | no | yes | optional |
| `notes` | array | no | yes | optional |
| `entity` | string | conditional | yes | mutually exclusive with `parent` on create |
| `parent` | string | conditional | indirect | mutually exclusive with `entity` on create |
| `subscriber` | string | no | limited | downstream-owned |
| `children` | array | no | no | not valid create input |
| `managementPolicy` | string | no | yes | policy link |
| `devices` | array | no | no | derived linkage |
| `topology` | object | no | limited | do not treat as normal create input |
| `design` | string | no | limited | do not treat as normal create input |
| `deviceConfiguration` | array | no | yes | downstream shape |
| `contacts` | array | no | yes | downstream shape |
| `location` | string | no | yes | optional |
| `deviceRules` | object | no | yes | downstream shape |

### 9.5 `ManagementPolicy`

| Field | Type | Required on Create | Mutable on Update | Notes |
|---|---|---:|---:|---|
| `id` | string | no | no | policy identifier |
| `name` | string | yes | yes | required |
| `description` | string | no | yes | optional |
| `notes` | array | no | yes | optional |
| `entries` | array | no | yes | effective access entries |
| `inUse` | array | no | no | runtime linkage |
| `tags` | array | no | yes | optional |
| `entity` | string | yes | yes | owning entity scope |
| `venue` | string | no | yes | optional venue scope |

### 9.6 `ManagementRole`

| Field | Type | Required on Create | Mutable on Update | Notes |
|---|---|---:|---:|---|
| `id` | string | no | no | role identifier |
| `name` | string | yes | yes | required |
| `description` | string | no | yes | optional |
| `notes` | array | no | yes | optional |
| `managementPolicy` | string | no | yes | linked policy id |
| `users` | array | no | yes | OWSEC user ids |
| `inUse` | array | no | no | downstream linkage |
| `tags` | array | no | yes | optional |
| `entity` | string | yes | yes | linked entity scope |
| `venue` | string | no | yes | optional venue scope |

---

## 10. Phase 1 Resource Validation Rules

### Entity create

Only treat these as authoritative create fields:

- `name`
- `description`
- `notes`
- `parent`

### Entity update

Normal mutable core fields:

- `name`
- `description`
- `notes`

### Venue create

- `entity` and `parent` are mutually exclusive
- children cannot be present at create
- topology and design should not be treated as normal create inputs

### Operator create

- automatically creates linked entity
- MDU must not duplicate that work

---

## 11. Session and Auth APIs

### Endpoint

- `GET /api/v1/mdu/me`

### Purpose

Return the current caller context after OWSEC validation and MDU composition.

### Downstream mapping

- token validation through OWSEC validation endpoints
- `GET /oauth2?me=true`
- PROV enrichment for effective access summary

### Path parameters

None.

### Query parameters

None.

### Request body schema

None.

### Response schema

MDU-owned composed `/me` schema containing:

- current caller identity from OWSEC
- effective role and policy summary from PROV
- effective scope summary

### Validation rules

- bearer token required on authenticated public flow
- current caller must resolve from OWSEC
- MDU shall follow the verified Phase 1 API-key validation rule from the integration baseline, even where legacy or raw downstream OpenAPI naming differs

### Orchestration notes

- validate token first
- resolve current caller with `GET /oauth2?me=true`
- enrich with effective access from PROV
- do not persist local session truth

### Auth and Session Acceptance

- current caller is resolved via `GET /oauth2?me=true`
- token validation uses OWSEC validation endpoints
- response includes composed effective access context
- errors are normalized into the MDU error envelope

---

## 12. User APIs

### Endpoint

- `POST /api/v1/mdu/users`
- `GET /api/v1/mdu/users`
- `GET /api/v1/mdu/users/{userId}`
- `PATCH /api/v1/mdu/users/{userId}`
- `DELETE /api/v1/mdu/users/{userId}`

### Purpose

Provide direct user lifecycle wrappers over OWSEC admin-user APIs.

### Downstream mapping

- create → `POST /user/0`
- list → `GET /users`
- detail → `GET /user/{id}`
- update → `PUT /user/{id}`
- delete → `DELETE /user/{id}`

### Path parameters

| Name | Required | Notes |
|---|---:|---|
| `userId` | yes for detail/update/delete | OWSEC user id |

### Query parameters

#### Users list

| Name | Notes |
|---|---|
| `offset` | downstream pagination |
| `limit` | downstream pagination |
| `filter` | downstream filter support |
| `idOnly` | downstream compact response mode |
| `select` | downstream field selection |
| `nameSearch` | name search |
| `emailSearch` | email search |

#### User detail

| Name | Notes |
|---|---|
| `byEmail` | interpret identifier as email when supported |

#### User update

| Name | Notes |
|---|---|
| `email_verification` | downstream update flag |
| `forgotPassword` | downstream reset flow flag |
| `resetMFA` | downstream reset flag |

### Request body schema

- create request body → `UserInfo` create-compatible downstream body
- update request body → `UserInfo` update-compatible downstream body

### Response schema

- single-user responses → `UserInfo`
- list response → downstream users list response shape

### Validation rules

- create must use `POST /user/0`
- Phase 1 supports normal update semantics first
- MDU should not invent a second user schema for direct wrappers

### Orchestration notes

- MDU is a thin OWSEC wrapper for direct user CRUD
- do not persist local user truth
- normalize query parameter names and error envelope

### User API Acceptance

- create uses `/user/0`
- list supports downstream query filters
- detail supports `byEmail`
- update maps to OWSEC update
- delete maps to OWSEC delete
- errors normalized into MDU error envelope

---

## 13. Operator APIs

### Endpoint

- `POST /api/v1/mdu/operators`
- `GET /api/v1/mdu/operators`
- `GET /api/v1/mdu/operators/{operatorId}`
- `PATCH /api/v1/mdu/operators/{operatorId}`

### Purpose

Expose direct operator wrappers over PROV.

### Downstream mapping

- create → `POST /operator/{uuid}`
- list → `GET /operator`
- detail → `GET /operator/{uuid}`
- update → `PUT /operator/{uuid}`

### Path parameters

| Name | Required | Notes |
|---|---:|---|
| `operatorId` | yes for detail/update | PROV operator id |

### Query parameters

#### Operator list

| Name | Notes |
|---|---|
| `offset` | downstream pagination |
| `limit` | downstream pagination |
| `select` | field selection |
| `countOnly` | count-only mode |

### Request body schema

- create → `Operator`
- update → `Operator`

### Response schema

- detail/create/update → `Operator`
- list → `OperatorList`

### Validation rules

- operator create uses compatibility create route `POST /operator/{uuid}`
- operator create auto-creates linked entity
- MDU must not create that entity separately

### Orchestration notes

- inject compatibility path token internally
- do not leak the raw create-route quirk to callers

### Operator API Acceptance

- create uses `POST /operator/{uuid}`
- list supports downstream pagination/query options
- detail maps to `GET /operator/{uuid}`
- update maps to `PUT /operator/{uuid}`
- operator create does not trigger a duplicate entity create flow

---

## 14. Entity APIs

### Endpoint

- `POST /api/v1/mdu/entities`
- `GET /api/v1/mdu/entities`
- `GET /api/v1/mdu/entities/{entityId}`
- `PATCH /api/v1/mdu/entities/{entityId}`
- `GET /api/v1/mdu/entities/{entityId}/children`

### Purpose

Expose direct entity wrappers and a composed children helper.

### Downstream mapping

- create → `POST /entity/{uuid}`
- list → `GET /entity`
- detail → `GET /entity/{uuid}`
- update → `PUT /entity/{uuid}`
- children → composed from PROV

### Path parameters

| Name | Required | Notes |
|---|---:|---|
| `entityId` | yes for detail/update/children | PROV entity id |

### Query parameters

#### Entity list

| Name | Notes |
|---|---|
| `offset` | downstream pagination |
| `limit` | downstream pagination |
| `select` | field selection |
| `countOnly` | count-only mode |

### Request body schema

- create → `Entity`
- update → `Entity`

### Response schema

- detail/create/update → `Entity`
- list → `EntityList`
- children → MDU-composed entity children response

### Validation rules

- use `POST /entity/{uuid}`
- do not use the tree import route for normal Phase 1 create
- authoritative create fields are `name`, `description`, `notes`, `parent`
- authoritative update fields are `name`, `description`, `notes`

### Orchestration notes

- inject compatibility path token internally
- children endpoint may compose from parent-child linkage in PROV responses

### Entity API Acceptance

- create uses `POST /entity/{uuid}`
- tree import route is not used for normal create
- list supports downstream pagination/query options
- children response is composed from PROV
- errors are normalized

---

## 15. Venue APIs

### Endpoint

- `POST /api/v1/mdu/entities/{entityId}/venues`
- `GET /api/v1/mdu/entities/{entityId}/venues`
- `GET /api/v1/mdu/venues/{venueId}`
- `PATCH /api/v1/mdu/venues/{venueId}`
- `GET /api/v1/mdu/venues/{venueId}/children`

### Purpose

Expose venue wrappers and composed venue-children behavior.

### Downstream mapping

- create → `POST /venue/{uuid}`
- list by entity → `GET /venue?entity={entityId}`
- detail → `GET /venue/{uuid}`
- update → `PUT /venue/{uuid}`
- children → composed from PROV

### Path parameters

| Name | Required | Notes |
|---|---:|---|
| `entityId` | yes for entity venue list/create | parent entity scope |
| `venueId` | yes for detail/update/children | PROV venue id |

### Query parameters

#### Venue list

| Name | Notes |
|---|---|
| `entity` | entity scope filter |
| `venue` | parent venue filter |
| `offset` | downstream pagination |
| `limit` | downstream pagination |
| `select` | field selection |
| `countOnly` | count-only mode |
| `RRMvendor` | downstream query option |

#### Venue detail

| Name | Notes |
|---|---|
| `getDevices` | include devices when supported |
| `getChildren` | include children when supported |

#### Venue update

| Name | Notes |
|---|---|
| `updateAllDevices` | downstream update option |
| `rebootAllDevices` | downstream update option |
| `testUpdateOnly` | downstream update option |
| `upgradeAllDevices` | downstream update option |
| `revisionsAvailable` | downstream update option |
| `revision` | downstream update option |

### Request body schema

- create → `Venue`
- update → `Venue`

### Response schema

- detail/create/update → `Venue`
- list → `VenueList`
- children → MDU-composed venue children response

### Validation rules

- create uses `POST /venue/{uuid}`
- `entity` and `parent` are mutually exclusive
- `children` cannot be present at create
- `topology` and `design` are not normal create inputs

### Orchestration notes

- entity-scoped create route should still map to PROV venue create shape
- list-by-entity should normalize the entity filter for callers

### Venue API Acceptance

- create uses `POST /venue/{uuid}`
- list supports downstream venue query parameters
- detail supports `getDevices` and `getChildren`
- update supports documented downstream query flags
- create rejects invalid `entity`/`parent` combinations

---

## 16. Management Policy APIs

### Endpoint

- `POST /api/v1/mdu/policies`
- `GET /api/v1/mdu/policies`
- `GET /api/v1/mdu/policies/{policyId}`

### Purpose

Expose direct management policy wrappers over PROV.

### Downstream mapping

- create → `POST /managementPolicy/{uuid}`
- list → `GET /managementPolicy`
- detail → `GET /managementPolicy/{uuid}`

### Path parameters

| Name | Required | Notes |
|---|---:|---|
| `policyId` | yes for detail | PROV policy id |

### Query parameters

#### Policy list

| Name | Notes |
|---|---|
| `entity` | entity scope filter |
| `venue` | venue scope filter |
| `offset` | downstream pagination |
| `limit` | downstream pagination |
| `select` | field selection |
| `countOnly` | count-only mode |

#### Policy detail

| Name | Notes |
|---|---|
| `expandInUse` | expand in-use references |

### Request body schema

- create → `ManagementPolicy`

### Response schema

- detail/create → `ManagementPolicy`
- list → `ManagementPolicyList`

### Validation rules

- create uses `POST /managementPolicy/{uuid}`
- direct wrapper must preserve downstream `entries[]` semantics

### Orchestration notes

- MDU should preserve the `managementPolicy.entries[]` structure
- MDU should not reshape the serialized inner `policy` document for direct wrappers

### Policy API Acceptance

- create uses `POST /managementPolicy/{uuid}`
- list supports entity/venue/pagination query parameters
- detail supports `expandInUse`
- wrapper preserves downstream `ManagementPolicy` body shape

---

## 17. Management Role APIs

### Endpoint

- `POST /api/v1/mdu/roles`
- `GET /api/v1/mdu/roles`
- `GET /api/v1/mdu/roles/{roleId}`

### Purpose

Expose direct management role wrappers over PROV.

### Downstream mapping

- create → `POST /managementRole/{id}`
- list → `GET /managementRole`
- detail → `GET /managementRole/{id}`

### Path parameters

| Name | Required | Notes |
|---|---:|---|
| `roleId` | yes for detail | PROV role id |

### Query parameters

#### Role list

| Name | Notes |
|---|---|
| `entity` | entity scope filter |
| `offset` | downstream pagination |
| `limit` | downstream pagination |
| `select` | field selection |
| `countOnly` | count-only mode |

#### Role detail

| Name | Notes |
|---|---|
| `expandInUse` | expand in-use references |

### Request body schema

- create → `ManagementRole`

### Response schema

- detail/create → `ManagementRole`
- list → `ManagementRoleList`

### Validation rules

- create uses `POST /managementRole/{id}`
- direct wrapper must preserve downstream users/policy/entity linkage shape

### Orchestration notes

- `managementRole.users[]` links OWSEC user ids
- `managementRole.managementPolicy` links the downstream policy object

### Role API Acceptance

- create uses `POST /managementRole/{id}`
- list supports entity/pagination query parameters
- detail supports `expandInUse`
- wrapper preserves downstream `ManagementRole` body shape

---

## 18. Access Assignment APIs

### Endpoint

- `POST /api/v1/mdu/users/{userId}/entities/{entityId}`
- `POST /api/v1/mdu/users/{userId}/roles`
- `POST /api/v1/mdu/users/{userId}/policies`
- `GET /api/v1/mdu/users/{userId}/access`

### Purpose

Provide MDU-owned orchestration for user-to-entity, role, and policy assignment plus effective access summary.

### Downstream mapping

- composed OWSEC + PROV workflow
- no single direct downstream wrapper for effective access summary

### Path parameters

| Name | Required | Notes |
|---|---:|---|
| `userId` | yes | OWSEC user id |
| `entityId` | yes for entity assignment | target entity scope |

### Query parameters

None required by default for the assignment endpoints.

### Request body schema

MDU-owned assignment request schemas only, for:

- role assignment inputs
- policy assignment inputs
- entity assignment inputs

### Response schema

MDU-owned access summary schemas for:

- `/users/{userId}/access`
- assignment operation results where composed output is needed

### Validation rules

- no local ownership table
- no local RBAC table
- assignments must resolve to PROV `managementPolicy` and `managementRole` objects

### Orchestration notes

- user identity comes from OWSEC
- access structure comes from PROV
- entity assignment, role assignment, and policy assignment must end in PROV-managed linkage

### Access Assignment Acceptance

- user access summary composes OWSEC + PROV
- assignment flows do not create local RBAC truth
- errors are normalized into MDU envelope
- direct wrapper schemas are not invented where downstream shapes already exist

---

## 19. `/validateApiKey` Note

MDU shall follow the verified Phase 1 API-key validation rule from the integration baseline, even where legacy or raw downstream OpenAPI naming differs.

---

## 20. Phase 1 Deliverables

- auth and session middleware
- current caller context endpoint
- user lifecycle wrappers over OWSEC
- operator, entity, venue, management policy, and management role wrappers over PROV
- effective access summary
- user-to-role, policy, and entity assignment orchestration
- normalized MDU error envelope
- OpenAPI and requirements alignment for implemented Phase 1 endpoints

---

## 21. Phase 1 Explicit Non-Goals

- billing
- topology
- analytics
- OWGW runtime operations
- jobs
- saga
- async workflow persistence
- rollout orchestration
- AI hooks

