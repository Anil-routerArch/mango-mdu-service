# Mango MDU Service

A repo-ready foundation for the Mango Cloud MDU backend service in the OpenWiFi environment. This service is intended to evolve into the MDU orchestration layer that integrates with OWSEC and PROV.

Important: the currently checked-in sample CRUD, sample database schema, and `/api/v1/items` endpoints are temporary scaffold placeholder code only. They are starter examples carried over from the foundation template and must not be interpreted as actual MDU domain, API, or schema decisions.

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
│       └── 0001_initial.sql     # Placeholder SQL table setup
├── docs/                        # Specifications and API contracts templates
│   ├── requirements.md          # Requirements template
│   ├── design.md                # Technical design doc template
│   └── openapi.yaml             # OpenAPI (Swagger) api definition
├── configs/                     # Configurations for development/testing
│   └── local-dev.env            # Env configuration for local running (outside Docker)
├── deployments/                 # Deployment-related configurations
│   └── docker-compose/
│       ├── docker-compose.env   # Env template for Docker Compose execution
│       └── docker-compose.yaml  # Docker Compose deployment integration template
├── external/                    # Third-party API client integration wrappers
│   └── README.md                # Developer guide for external adapters
├── internal/
│   ├── app/                     # Application wiring and dependency injection
│   │   └── app.go               # Dynamic struct creation, DB pool, and module boot
│   ├── config/                  # caarlos0/env environment parsing
│   ├── db/                      # Connection pool (pgxpool) & migration engine
│   ├── http/                    # Routing, middleware, and Dual TLS engine
│   ├── models/                  # Domain-level request/response model structs
│   └── services/                # Business logic interfaces and services
├── .dockerignore                # Exclusions for Docker build context
├── .gitignore                   # Exclusions for Git repository
├── Dockerfile                   # Multi-stage production container configuration
├── init-service.sh              # Scaffolding helper script to rename/configure
├── Makefile                     # Build, run, test, and containerize commands
├── README.md                    # This developer guide
```

---

## Bootstrap Reference

This repository started from a Go service foundation template. The example scaffold command is kept only as a reference for how the repo was initialized.

1. Execute the `init-service.sh` script, providing your new service name, public API port, private API port, and target directory:
   ```bash
   ./init-service.sh <new-service-name> <public-port> <private-port> [target-directory]
   ```

2. **Example**:
   ```bash
   ./init-service.sh mango-mdu-service 16010 17010 ../mango-mdu-service
   ```

3. Navigate to the generated directory and start customizing.

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
1. When you run `./init-service.sh`, prompt for the path of your `docker-compose` directory. The script will automatically copy the generated `.env` configuration file to that directory.

2. Alternatively, copy it manually:
   ```bash
   cp deployments/docker-compose/<your-service-name>.env /path_to/mango-cloud-deployment/docker-compose/
   ```

3. Paste the service block displayed on the screen (or from `deployments/docker-compose/docker-compose.yaml`) inside the `services:` block of your deployment's `docker-compose.yml`.

4. Re-launch the compose deployment:
   ```bash
   docker compose up -d --build <your-service-name>
   ```

---

## Database Migrations
Migrations are managed dynamically. When the service boots up:
1. It validates the database connection.
2. It verifies the presence of the `schema_migrations` tracking table.
3. It scans the `db/schema/` directory for `.sql` files.
4. Any SQL files that have not been registered are executed sequentially in individual SQL transactions.
5. If a migration fails, the transaction is rolled back and the service blocks startup to prevent running on a broken schema.
