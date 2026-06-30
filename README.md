# Mango MDU Service

A repo-ready foundation for the Mango Cloud MDU backend service in the OpenWiFi environment. This service is intended to evolve into the MDU orchestration layer that integrates with OWSEC and PROV.

---

## Folder Structure

```text
├── .github/
│   └── workflows/
│       └── ci.yaml              # Continuous Integration workflow configuration
├── cmd/
│   └── main.go                  # Boilerplate entrypoint (Config load, Logger init, runs App, OS signals)
├── db/
│   └── schema/                  # SQL schema migrations directory
│       └── 0001_initial.sql     # Reserved migration baseline for approved MDU-owned schema
├── docs/                        # Specifications and API contracts
│   ├── requirement.md           # Master requirements document
│   ├── common-requirement.md    # Cross-phase engineering and security guardrails
│   ├── openapi.yaml             # OpenAPI (Swagger) API definition (master draft)
│   └── phase-1/                 # Phase 1 specification and workflow documents
│       ├── spec.md              # Phase 1 specification
│       ├── mango-mdu-openapi.yaml # Phase 1 OpenAPI API contract (authoritative for Phase 1)
│       └── phase-1-workflow.md  # Phase 1 runtime workflow description
├── configs/                     # Configurations for development/testing
│   └── local-dev.env            # Env configuration for local running (outside Docker)
├── deployments/                 # Deployment-related configurations
│   └── docker-compose/
│       ├── mango-mdu-service.env # Env template for Docker Compose execution
│       └── docker-compose.yaml  # Docker Compose deployment integration template
├── external/                    # Third-party API client integration wrappers
│   └── README.md                # Developer guide for external adapters
├── internal/
│   ├── app/                     # Application wiring and dependency injection
│   │   └── app.go               # Dynamic struct creation, DB pool, and module boot
│   ├── config/                  # caarlos0/env environment parsing
│   ├── db/                      # Connection pool (pgxpool) & migration engine
│   ├── http/                    # Routing, middleware, and Dual TLS engine
│   ├── models/                  # Domain-level request/response model package
│   └── services/                # Business logic service package
├── .dockerignore                # Exclusions for Docker build context
├── .gitignore                   # Exclusions for Git repository
├── Dockerfile                   # Multi-stage production container configuration
├── Makefile                     # Build, run, test, and containerize commands
├── README.md                    # This developer guide
```

---

## OpenAPI Contract Versioning

Because the MDU service is implemented phase-wise, each phase maintains its own dedicated and authoritative OpenAPI specification under `docs/phase-<N>/`. 

* **Phase 1 Active Contract:** `docs/phase-1/mango-mdu-openapi.yaml` is the authoritative contract for Phase 1 development, testing, and reviews.
* **Master Draft Contract:** The top-level `docs/openapi.yaml` acts as a master draft encompassing multi-phase proposals.
* **Alignment Strategy:** As each phase's implementation is completed and validated, the master draft `docs/openapi.yaml` will be aligned to match the finalized contract of that phase. Currently, the phase-specific OpenAPI spec takes priority for the code in that phase.

---

## Runtime Surface

The current checked-in runtime baseline exposes:

- public TLS interface on port `16010`
- private TLS interface on port `17010`
- unauthenticated `/livez` on both ports
- authenticated `/api/v1/system` diagnostics routes on both ports via the shared `system-routes` module

The MDU-specific Mango-facing `/api/v1/mdu/*` business APIs described in the Phase 1 docs are not yet implemented in this branch.

---

## Local Development (Outside Docker)

### Prerequisites
* Go 1.25+ installed
* PostgreSQL and Kafka running (or forwarded to `localhost`)

### Steps
1. Start the service locally:
   ```bash
   make run
   ```
   *(Note: The Makefile will automatically generate self-signed TLS certificates under `./certs/` if they do not exist).*

2. Alternatively, you can run it manually:
   ```bash
   # Make sure self-signed certs exist first
   make certs
   # Run with sourced configurations
   source configs/local-dev.env && go run ./cmd
   ```

---

## Docker Integration

### 1. Build the Image
```bash
make docker-build
```

### 2. Integrate with Mango Cloud Compose Stack
1. Copy the checked-in environment template into your deployment compose directory:
   ```bash
   cp deployments/docker-compose/mango-mdu-service.env /path_to/mango-cloud-deployment/docker-compose/
   ```

2. Paste the service block from `deployments/docker-compose/docker-compose.yaml` inside the `services:` block of your deployment's `docker-compose.yml`.

3. Re-launch the compose deployment:
   ```bash
   docker compose up -d --build mango-mdu-service
   ```

---

## Database Migrations
Migrations are managed dynamically. When the service boots up:
1. It validates the database connection.
2. It verifies the presence of the `schema_migrations` tracking table.
3. It scans the `db/schema/` directory for `.sql` files.
4. Any SQL files that have not been registered are executed sequentially in individual SQL transactions.
5. If a migration fails, the transaction is rolled back and the service blocks startup to prevent running on a broken schema.
