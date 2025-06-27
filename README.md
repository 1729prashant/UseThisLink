# UseThisLink

A scalable, production-ready URL shortening service with click tracking, analytics, and user authentication, built in Go with a distributed microservices architecture and PostgreSQL.

---

## Prerequisites

- **Docker** and **Docker Compose** (for local development and orchestration)
- **Go 1.21+** (for local development, optional if using Docker only)
- **Make** (optional, for local test/dev convenience)

---

## Architecture Overview

- **API Gateway**: Entry point for all client requests, routes to internal services, serves static files and HTML.
- **Link Service**: Handles URL shortening and redirection.
- **Analytics Service**: Tracks clicks, stores analytics, exposes stats endpoints.
- **User Service**: Handles registration, login, session management, and user info.
- **PostgreSQL**: Shared DB instance, each service uses its own schema for isolation.

---

## Quickstart (Docker Compose)

1. **Clone the repo**

2. **Run all services**
   ```sh
   docker-compose up --build
   ```
   This will build and start all services and a Postgres database.

3. **Access the app**
   - Web UI: [http://localhost:8080](http://localhost:8080)
   - API Gateway: [http://localhost:8080](http://localhost:8080)
   - Link Service: [http://localhost:8081](http://localhost:8081)
   - Analytics Service: [http://localhost:8082](http://localhost:8082)
   - User Service: [http://localhost:8083](http://localhost:8083)

4. **Run tests** (from project root):
   ```sh
   docker-compose run link go test ./...
   docker-compose run analytics go test ./...
   docker-compose run user go test ./...
   ```

---

## Configuration & Environment Variables

All configuration is managed via environment variables (see `docker-compose.yml`).

| Service      | Key Variables (default)                                                                 |
|--------------|----------------------------------------------------------------------------------------|
| Gateway      | PORT=8080, LINK_SERVICE_URL, ANALYTICS_SERVICE_URL, USER_SERVICE_URL                   |
| Link         | PORT=8081, PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE, PGSCHEMA=link, BASE_URL     |
| Analytics    | PORT=8082, PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE, PGSCHEMA=analytics, BASE_URL|
| User         | PORT=8083, PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE, PGSCHEMA=user, BASE_URL, SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS |
| Postgres     | POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB                                          |

- **BASE_URL** should be set to the Gateway's public URL (e.g., `http://localhost:8080`).
- **SMTP_*** variables are required for email/OTP in the User service.

---

## API Examples

```
curl -X POST -d '{"original_url": "https://example.com"}' http://localhost:8080/shorten
curl http://localhost:8080/r/Ab1XyZ # Redirects
curl http://localhost:8080/stats/Ab1XyZ
```

---

## Directory Structure

```
UseThisLink/
  services/
    gateway/
    link/
    analytics/
    user/
  static/
  templates/
  docker-compose.yml
  README.md
  DesignDocument.md
```

---

## Development

- Each service is a standalone Go app with its own Dockerfile.
- All inter-service communication is via HTTP (never direct Go calls or shared DB access).
- Each service manages its own DB schema and tables.
- To run a service locally (with Docker Compose running Postgres):
  ```sh
  cd services/link
  go run cmd/main.go
  # or build and run the Docker image
  ```

---

## Production Notes

- Use a managed Postgres instance for production.
- Set strong, unique secrets for all environment variables.
- Use HTTPS and secure cookies in production.
- Add monitoring, logging, and CI/CD as needed.

---

## License

GPLv3. See LICENSE for details.
