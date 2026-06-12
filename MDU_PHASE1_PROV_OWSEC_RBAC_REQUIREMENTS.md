# MDU Phase 1 PROV and OWSEC RBAC Requirements

## Status

This is the working Markdown document for ongoing edits.

Current verification basis:

- confirmed against the provided OWPROV root OpenAPI file where possible
- exact HTTP methods are still partially unverified where the root spec delegates path definitions to external handler files

## 1. Purpose

Define the Phase 1 implementation requirements for MDU backend and UI integration focused on:

- `PROV`
- `OWSEC`
- RBAC
- entity and venue management
- user, role, policy, and assignment workflows

In this document, customer and operator are treated as entity-level business objects. The difference is driven by assigned roles and policies, not by a separate ownership model in MDU.

This document is intentionally implementation-ready but not language- or framework-specific.

## 2. Phase 1 Scope

### In Scope

- PROV integration
- OWSEC integration
- authentication and identity resolution
- RBAC enforcement
- entity or operator management
- venue management
- user management
- policy management
- role management
- role and policy assignment
- installer-restricted access
- root, system, admin, noc, csr, installer, and accounting role modeling
- MDU API endpoints required by UI
- MDU to PROV and OWSEC endpoint mapping
- detailed access and workflow definitions

### Out of Scope

- saga
- billing
- payments
- analytics
- notifications
- unrelated `mdu-ui` areas
- inventory workflows, except where referenced as future RBAC resources
- configuration workflows, except where referenced as future RBAC resources
- technology-specific implementation details

## 3. Source Inputs

### `mango-cloud-yaml/prov`

- Repository: `https://github.com/routerarchitects/mango-cloud-yaml/tree/main/prov`
- Purpose: source of PROV API and schema definitions
- Note: the user indicated there are three relevant folders under `prov`
- Constraint: some schemas may still be under development

### `ow-common-mods`

- Repository: `https://github.com/routerarchitects/ow-common-mods/tree/main`
- Purpose: shared reusable utilities and helper functions
- Constraint: this is not a middleware layer and must not be modified for Phase 1

### `mdu-ui`

- Repository: `https://github.com/Anil-routerArch/mdu-ui`
- Purpose: reference for UI direction only
- Constraint: billing and unrelated UI areas must be ignored in Phase 1

### Reference Format

- Reference file: `mdu_api_flow_rbac_saga_pagination (1)(1).md`
- Usage: structure and formatting reference only
- Exclusion: do not carry forward saga content

## 4. Architecture Rules

### Required Integration Rule

MDU directly orchestrates calls to PROV and OWSEC. `common-mods` may be used only for existing shared helper functions. `common-mods` must not be modified, extended, or treated as a PROV or OWSEC integration layer.

Customer ownership, customer hierarchy, and RBAC source-of-truth remain in PROV. MDU must not become the owning system for customer access models, policy models, or role models.

Users are stored in OWSEC. Operators and entities are stored in PROV. User access is established by creating the user in OWSEC and then assigning the correct entity, role, and policy through PROV-managed RBAC.

### Correct Architecture

```text
MDU UI
  -> MDU Backend
      -> Use common-mods only if an existing helper is already available
      -> Otherwise call PROV directly
      -> Otherwise call OWSEC directly
```

### Architecture to Avoid

```text
MDU -> common-mods -> PROV/OWSEC
```

## 5. Service Responsibilities

### MDU Backend

- validate or introspect OWSEC-backed identity context
- normalize UI-facing API contracts
- forward customer, entity, venue, role, and policy operations to PROV
- rely on PROV as the RBAC authority for scope and access decisions
- translate downstream responses into MDU domain responses

### PROV

- customer ownership and hierarchy
- operator records
- entity management
- venue management
- management policy management
- management role management
- RBAC source of truth

### OWSEC

- authentication
- token validation or identity resolution
- user identity lifecycle
- user record storage

### `common-mods`

- shared utilities only
- no new PROV orchestration logic
- no new OWSEC orchestration logic
- no repository changes required for Phase 1

## 6. Authentication and Identity Context

### Core Rule

All Phase 1 requests must be authorized using OWSEC-backed identity context.

### Required Flow

1. UI sends `Authorization: Bearer <token>`.
2. MDU validates the token with OWSEC or equivalent approved validation path.
3. MDU resolves the caller identity.
4. MDU passes identity context to PROV where RBAC scope and access are evaluated.
5. MDU returns the normalized response from PROV-backed access evaluation.

### `GET /api/v1/mdu/me`

Purpose:

- validate current session
- return normalized user context
- return PROV-backed effective roles
- return PROV-backed effective scopes

Response should include:

- user identity
- assigned roles
- assigned policies
- computed allowed entity or venue scope

## 7. RBAC Model

### Access Model

- access is denied by default
- a role without policy gives actions with no usable scope
- a policy without role gives scope with no permitted actions
- a user needs both role and policy for effective access
- PROV is the authority that stores and evaluates these assignments

### RBAC Interpretation

- role defines allowed actions
- policy defines allowed scope
- MDU does not own RBAC state
- MDU should not implement a separate customer-ownership model
- user-to-entity assignment is part of access control

### Actual PROV RBAC Shape

- `managementPolicy` is the policy object stored in PROV
- `managementPolicy.entries[]` carries the effective access entries
- each policy entry contains `users`, `resources`, `access`, and a serialized `policy` scope document
- `managementRole` is the role-assignment object stored in PROV
- `managementRole.managementPolicy` links a role to a management policy
- `managementRole.users[]` links one or more OWSEC user IDs to that role
- `managementRole.entity` and `managementRole.venue` bind the role to entity or venue scope

## 8. Domain Model

### Core Business Model

- customer and operator are treated as the same entity-level concept for Phase 1
- differences between customer and operator behavior come from assigned roles and policies
- the canonical business scope object is `entity`
- users are assigned to entities
- venues belong under entities or parent venues as supported by PROV

### Storage Boundary

- OWSEC `users` table stores user identity records
- PROV stores operator records
- PROV stores entity records
- PROV stores management roles and management policies
- PROV is the source of truth for which entity scope a user can manage or access

### Naming Rules for This Document

- use `managementPolicy` for provisioning policy objects
- use `managementRole` for provisioning role objects
- use `entity` as the main business scope object
- treat `operator.entityId` as the link from operator record to entity record
- do not describe policy and role storage as MDU-native tables

## 9. Policy Scope Model

### Entity-Scoped Example

```json
{
  "type": "entity",
  "entityId": "<entity-id>",
  "includeVenues": true,
  "includeChildEntities": true
}
```

### Actual `managementPolicy` Entry Pattern

```json
{
  "name": "Operator Admin Full Access Policy",
  "entries": [
    {
      "users": ["<owsec-user-id>"],
      "resources": ["entity"],
      "access": ["FULL"],
      "policy": "{\"type\":\"entity\",\"entityId\":\"<entity-id>\",\"includeVenues\":true,\"includeChildEntities\":true}"
    },
    {
      "users": ["<owsec-user-id>"],
      "resources": ["venue"],
      "access": ["FULL"],
      "policy": "{\"type\":\"entity\",\"entityId\":\"<entity-id>\",\"includeVenues\":true,\"includeChildEntities\":true}"
    }
  ],
  "entity": "<entity-id>",
  "venue": ""
}
```

### Policy Interpretation Rules

- the outer object is the `managementPolicy`
- the inner `entries[]` list is where resource access is actually granted
- the `policy` field inside each entry is serialized JSON, not a nested object
- the same user can appear in multiple entries for different resources
- resource names currently observed include `entity`, `venue`, `inventory`, `configuration`, `managementPolicy`, and `managementRole`
- a support-style policy can grant `READ` and `LIST` for multiple resources in a single entry

### Venue-Scoped Installer Example

```json
{
  "type": "venue",
  "entityId": "<entity-id>",
  "venueId": "<venue-id>",
  "includeVenues": true,
  "includeChildEntities": false
}
```

### Policy Rules

- do not imply parent or child inheritance unless explicitly encoded in policy
- installer policies must remain narrow
- root or system bootstrap paths must be explicitly controlled

## 10. Role Model

Phase 1 must model all of the following roles:

- `root`
- `system`
- `admin`
- `noc`
- `csr`
- `installer`
- `accounting`

In PROV terms, these business roles are expected to be represented through `managementRole` objects linked to `managementPolicy` records.

## 11. Role Intent

### `root`

- highest platform-level role
- manage all entities
- manage all users
- manage all policies
- manage all roles
- assign any non-protected role
- view all scoped PROV and OWSEC data

### `system`

- reserved for internal service operations
- provision default roles and policies where required
- execute backend automation
- not a standard UI role

### `admin`

- manage assigned entity scope
- manage users within allowed scope
- manage venues within allowed scope
- create or update policies within allowed scope
- assign allowed roles within allowed scope
- cannot escalate to `root` or `system`
- typically backed by a `managementRole` linked to a full-access `managementPolicy` for the target entity

### `noc`

- operational visibility into allowed scope
- read or monitor entity and venue data
- limited operational updates only if explicitly allowed
- no billing scope in Phase 1

### `csr`

- support-focused read access within allowed scope
- limited user-assistance actions if explicitly allowed
- no policy management
- no role management
- no privileged user creation

### `installer`

- venue-scoped access only
- read venue details needed for installation
- perform limited installation-related updates if supported
- no user management
- no role management
- no policy management
- no sibling venue or broad entity access

### `accounting`

- modeled in Phase 1 for future compatibility
- minimal or reserved access only
- no billing workflows in Phase 1
- no role or policy administration

## 12. Role Permission Matrix

| Role | Entity | Venue | Users | Policies | Roles | Assignments | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `root` | full | full | full | full | full | full | global or root-scoped authority |
| `system` | controlled | controlled | controlled | controlled | controlled | controlled | service-only role |
| `admin` | scoped manage | scoped manage | scoped manage | scoped manage | scoped read or manage | scoped assign | cannot assign `root` or `system` |
| `noc` | scoped read | scoped read or limited ops | limited read | no | no | no | operations-focused |
| `csr` | scoped read | scoped read | limited support actions | no | no | no | support-focused |
| `installer` | no broad access | venue-scoped | no | no | no | no | installation-only |
| `accounting` | limited read | limited read | limited read if needed | no | no | no | future billing placeholder |

## 13. Resource Model

### Active Phase 1 Resources

- `entity`
- `operator`
- `venue`
- `managementPolicy`
- `managementRole`
- `user`
- `userEntityAssignment`
- `roleAssignment`
- `policyAssignment`

### Future or Referenced-Only Resources

- `inventory`
- `configuration`
- `billing`
- `analytics`

### Observed Provisioning Resource Names

From the provided provisioning policy examples, these resource names are already used inside `managementPolicy.entries[].resources`:

- `entity`
- `venue`
- `inventory`
- `configuration`
- `managementPolicy`
- `managementRole`

## 14. MDU API Surface

These are the required UI-facing MDU endpoints for Phase 1. Where possible, they should map closely to existing PROV resources instead of introducing an independent MDU ownership or RBAC model.

### Session

```http
GET /api/v1/mdu/me
```

### Operators

```http
POST /api/v1/mdu/operators
GET /api/v1/mdu/operators
GET /api/v1/mdu/operators/{operatorId}
PATCH /api/v1/mdu/operators/{operatorId}
DELETE /api/v1/mdu/operators/{operatorId}
```

### Entities

```http
POST /api/v1/mdu/entities
GET /api/v1/mdu/entities
GET /api/v1/mdu/entities/{entityId}
PATCH /api/v1/mdu/entities/{entityId}
DELETE /api/v1/mdu/entities/{entityId}
GET /api/v1/mdu/entities/{entityId}/children
```

### Venues

```http
POST /api/v1/mdu/entities/{entityId}/venues
GET /api/v1/mdu/entities/{entityId}/venues
GET /api/v1/mdu/venues/{venueId}
PATCH /api/v1/mdu/venues/{venueId}
DELETE /api/v1/mdu/venues/{venueId}
GET /api/v1/mdu/venues/{venueId}/children
```

### Users

```http
POST /api/v1/mdu/users
GET /api/v1/mdu/users
GET /api/v1/mdu/users/{userId}
PATCH /api/v1/mdu/users/{userId}
DELETE /api/v1/mdu/users/{userId}
```

### User Entity Assignment

```http
POST /api/v1/mdu/users/{userId}/entities/{entityId}
DELETE /api/v1/mdu/users/{userId}/entities/{entityId}
GET /api/v1/mdu/users/{userId}/entities
```

This assignment should be implemented by creating or updating the appropriate PROV `managementRole` and `managementPolicy` linkage for that OWSEC user.

### Policies

```http
POST /api/v1/mdu/policies
GET /api/v1/mdu/policies
GET /api/v1/mdu/policies/{policyId}
PATCH /api/v1/mdu/policies/{policyId}
DELETE /api/v1/mdu/policies/{policyId}
```

### Roles

```http
POST /api/v1/mdu/roles
GET /api/v1/mdu/roles
GET /api/v1/mdu/roles/{roleId}
PATCH /api/v1/mdu/roles/{roleId}
DELETE /api/v1/mdu/roles/{roleId}
```

### Assignments

```http
POST /api/v1/mdu/users/{userId}/entities/{entityId}
DELETE /api/v1/mdu/users/{userId}/entities/{entityId}
POST /api/v1/mdu/users/{userId}/roles
DELETE /api/v1/mdu/users/{userId}/roles/{roleId}
POST /api/v1/mdu/users/{userId}/policies
DELETE /api/v1/mdu/users/{userId}/policies/{policyId}
GET /api/v1/mdu/users/{userId}/access
```

All entity-assignment, role, policy, and access endpoints above are MDU-facing façades over PROV-owned RBAC operations. They must not create a separate RBAC datastore in MDU.

In the current provisioning shape, these façade operations likely translate to:

- create or update a `managementPolicy`
- create or update a `managementRole`
- attach OWSEC user IDs in `managementRole.users`
- link the role to the policy through `managementRole.managementPolicy`
- bind the role to scope through `managementRole.entity` or `managementRole.venue`

## 15. Downstream Mapping Direction

The provided root OpenAPI confirms which PROV resource paths exist. It does not fully expose every HTTP method in this pasted file because the path bodies are delegated to external handler refs.

### Confirmed PROV Resource Paths

Under the PROV server base path `/api/v1`, the following paths are confirmed:

```http
/entity
/entity/{uuid}
/operator
/operator/{uuid}
/managementPolicy
/managementPolicy/{uuid}
/managementRole
/managementRole/{id}
/venue
/venue/{uuid}
```

### Confirmed PROV Schema Support

The provided spec confirms schema support for these resources and payloads:

- `Operator`
- `OperatorCreateRequest`
- `OperatorUpdateRequest`
- `Entity`
- `EntityList`
- `Venue`
- `VenueCreateRequest`
- `VenueUpdateRequest`
- `VenueList`
- `ManagementPolicy`
- `ManagementPolicyCreateRequest`
- `ManagementPolicyUpdateRequest`
- `ManagementRole`
- `ManagementRoleCreateRequest`
- `ManagementRoleUpdateRequest`

This means `managementPolicy` and `managementRole` are already present in the PROV contract. They are not absent from the YAML currently provided.

### MDU to PROV

Confirmed read-path direction:

```http
GET /api/v1/entity
GET /api/v1/entity/{entityId}
GET /api/v1/operator
GET /api/v1/operator/{operatorId}
GET /api/v1/venue
GET /api/v1/venue/{venueId}
GET /api/v1/managementPolicy
GET /api/v1/managementPolicy/{policyId}
GET /api/v1/managementRole
GET /api/v1/managementRole/{roleId}
```

Confirmed write-shape support exists for:

- operator create and update
- venue create and update
- management policy create and update
- management role create and update

Entity write behavior likely exists through `/entity`, but the exact request schema and verb contract are not explicit in the pasted root file and should be confirmed from `RESTAPI_entity_handler.yaml` before the MDU contract is frozen.

### Ownership and RBAC Boundary

- customer operations are provisioning operations
- customer and operator are the same entity-class concept with different RBAC
- customer ownership remains in PROV
- entity assignment remains in PROV-managed access control
- role and policy persistence remains in PROV
- RBAC evaluation remains in PROV
- MDU should only adapt UI requests and responses around PROV APIs
- `managementPolicy` is the provisioning policy schema
- `managementRole` is the provisioning role schema

### MDU to OWSEC

OWSEC-backed user and token paths must be confirmed from OWSEC definitions before finalizing endpoint names.

## 16. Core Workflows

### Login and Session Validation

1. UI sends OWSEC token.
2. MDU validates token.
3. MDU resolves identity.
4. MDU calls PROV-backed access and resource APIs.
5. MDU returns normalized access context.

### Create Operator or Entity

1. Validate caller token.
2. Determine whether the request is creating the business entity record, the operator record, or both, based on the PROV model.
3. Send identity context to PROV.
4. Let PROV enforce caller scope and creation rights.
5. Call the matching PROV create endpoint.
6. Return normalized response.

### Create Child Operator

1. Validate caller token.
2. Send identity context and parent reference to PROV.
3. Let PROV enforce parent ownership and allowed scope.
4. Call PROV create entity with parent reference.
5. Return created child operator.

### Create Venue

1. Validate caller token.
2. Send identity context and target entity to PROV.
3. Let PROV enforce customer ownership and allowed scope.
4. Call PROV venue create endpoint.
5. Return normalized venue response.

### Create User

1. Validate caller token.
2. Create the user in OWSEC where required.
3. Resolve the target entity in PROV.
4. Create or resolve the correct `managementPolicy` in PROV.
5. Create or resolve the correct `managementRole` in PROV.
6. Attach the OWSEC user ID to `managementRole.users`.
7. Link the role to the policy using `managementRole.managementPolicy`.
8. Bind the role to the target `entity` or `venue`.
9. Let PROV persist and evaluate RBAC state.
10. Return normalized user access summary.

### Assign Entity to User

1. Validate caller token.
2. Resolve the target user in OWSEC.
3. Resolve the target entity in PROV.
4. Create or update the matching `managementPolicy`.
5. Create or update the matching `managementRole`.
6. Attach the OWSEC user ID to `managementRole.users`.
7. Set `managementRole.entity` to the target entity.
8. Return normalized access summary for that user.

### Assign Role to User

1. Validate caller token.
2. Resolve the target user in OWSEC.
3. Resolve the target entity or venue scope in PROV.
4. Resolve the requested business role type such as `admin`, `noc`, `csr`, `installer`, or `accounting`.
5. Let PROV enforce assignment permissions and scope ownership.
6. Create or resolve the required `managementPolicy`.
7. Create or resolve the required `managementRole`.
8. Attach the OWSEC user ID to `managementRole.users`.
9. Link the role to the policy and bind it to `entity` or `venue` scope.
10. Return effective access summary.

### Installer Role Assignment Example

1. Validate caller token.
2. Resolve the installer user in OWSEC.
3. Resolve the target entity or venue in PROV.
4. Let PROV enforce restricted installer scope.
5. Create or resolve an installer-scoped `managementPolicy`.
6. Create or resolve an installer `managementRole`.
7. Bind the installer access to narrow venue or entity scope.
8. Return effective installer access summary.

### Assign NOC Role

1. Validate caller token.
2. Resolve the target user in OWSEC.
3. Resolve the target entity scope in PROV.
4. Let PROV enforce privileged assignment permissions.
5. Create or resolve a NOC-scoped `managementPolicy`.
6. Create or resolve a NOC `managementRole`.
7. Attach the OWSEC user ID and bind the entity scope.
8. Return access summary.

### Assign CSR Role

1. Validate caller token.
2. Resolve the target user in OWSEC.
3. Resolve the target entity scope in PROV.
4. Create or resolve a CSR-scoped `managementPolicy`.
5. Create or resolve a CSR `managementRole`.
6. Attach limited support scope and exclude privileged capabilities.
7. Return access summary.

### Assign Accounting Role

1. Validate caller token.
2. Resolve the target user in OWSEC.
3. Resolve the target entity scope in PROV.
4. Create or resolve an accounting-scoped `managementPolicy`.
5. Create or resolve an accounting `managementRole`.
6. Keep billing capabilities disabled in Phase 1.
7. Return access summary.

### Root and System Bootstrap

1. `root` and `system` must be provisioned only through controlled bootstrap flows.
2. Standard admins must not assign `root` or `system`.
3. `system` is reserved for internal operations.
4. `root` retains platform-wide authority.

## 17. Read, Update, and Delete Rules

### Read

- reads must be filtered by effective policy scope
- list endpoints must not leak out-of-scope objects
- detail endpoints must return denied responses for out-of-scope resources

### Update

- updates require both action permission and matching scope
- update payloads must not allow privilege escalation through hidden fields

### Delete or Deactivate

- destructive operations must require explicit allowed roles
- delete behavior must be clarified as hard delete versus deactivate during implementation
- downstream system constraints must be surfaced consistently by MDU

## 18. Error Handling Requirements

- unauthorized when token is missing or invalid
- forbidden when scope or role is insufficient
- not found when the resource is absent or intentionally hidden by policy
- conflict when downstream uniqueness or assignment constraints fail
- bad request when scope, parent linkage, or payload is invalid

Responses should be normalized so UI behavior does not depend on raw PROV or OWSEC formats.

## 19. UI Requirements

### Phase 1 Screens to Support

- login and session context
- operator list and hierarchy
- operator create and update
- venue list and hierarchy
- venue create and update
- user list
- user create and edit
- user-to-entity assignment
- role assignment
- policy assignment
- role-specific assignment flows
- role and policy management for authorized users

### UI Areas to Exclude or Hide

- billing
- payments
- analytics
- unrelated dashboards
- features outside PROV and OWSEC Phase 1 scope

## 20. Phase 1 Checklist

- confirm exact PROV flow for linking OWSEC users to PROV entity scope
- confirm whether operator creation and entity creation are separate or coupled in PROV implementation
- confirm exact downstream API for entity assignment to a user

- confirm PROV entity endpoints from `mango-cloud-yaml/prov`
- confirm PROV venue endpoints from `mango-cloud-yaml/prov`
- confirm PROV management policy endpoints
- confirm PROV management role endpoints
- confirm OWSEC token and user endpoints
- confirm exact role-assignment storage model
- confirm exact policy-assignment storage model
- define normalized MDU response shapes
- define bootstrap path for `root` and `system`
- define final denied-response behavior for scoped resources
- validate UI screens against Phase 1-only scope

## 21. Main Rule for Phase 1

Only `PROV` and `OWSEC` are active integration systems for Phase 1. Billing and all other services are excluded. All roles must be modeled now, even if some role capabilities are minimal or reserved for future features.
