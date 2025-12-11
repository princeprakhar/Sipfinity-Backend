# Sipfinity-Backend

Production-ready Go e‑commerce backend providing admin and public product APIs, authentication (JWT + refresh tokens), S3 image storage, CSV bulk upload, review system, and optional FastAPI integration for CSV/image processing.

## Key features
- Admin product management: create, update, delete products; manage images, categories and services.
- CSV bulk upload with server-side parsing and optional external FastAPI processing.
- Product images stored on Amazon S3 (upload, delete, validation).
- Authentication & authorization: JWT access + refresh tokens, token revocation, role-based guards (admin vs customer).
- Public product API: paginated listing, search, filtering, categories, single-product endpoints.
- Reviews & moderation: create reviews, like/dislike, flagging, admin moderation.
- Password reset & email workflows via SMTP.
- Structured logging and environment-driven configuration for production readiness.

## Architecture / Core components
- HTTP layer: Gin handlers for Auth, Admin, Products, Reviews and CSV endpoints.
- Services (business logic): AuthService, AdminService, ProductService, ReviewService, S3Service, EmailService, FastAPIService.
- Persistence: GORM models and auto-migrations for PostgreSQL.
- External integrations: Amazon S3 (images), SMTP (emails), optional FastAPI for advanced CSV/image processing.
- Utilities: JWT/token helpers, input validation, response helpers and logger.

## Tech stack
- Go (Gin + GORM)
- PostgreSQL
- Amazon S3
- SMTP (email)
- Optional: FastAPI (Python) for CSV/image processing
- Docker-friendly

## Quick start (local)
Prerequisites: Go >=1.20, Docker (Postgres or local DB), AWS credentials for S3 if used.

1. Copy example env and set values:
   - `cp .env.example .env`
   - Required vars: DB_DSN, JWT_SECRET, JWT_REFRESH_SECRET, SMTP_HOST, SMTP_USER, SMTP_PASS, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION, S3_BUCKET, FASTAPI_URL (optional)
2. Start Postgres (example using Docker):
   - docker run --name sip-postgres -e POSTGRES_PASSWORD=pass -e POSTGRES_DB=sipfinity -p 5432:5432 -d postgres:15
3. Run the app:
   - go run ./cmd/server
   - or build: go build -o bin/server ./cmd/server && ./bin/server
4. Apply DB migrations / auto-migrate are performed on start (GORM auto-migrate).


## Common env variables
- DB_DSN — Postgres DSN (postgres://user:pass@host:port/dbname?sslmode=disable)
- JWT_SECRET, JWT_REFRESH_SECRET
- ACCESS_TOKEN_EXP, REFRESH_TOKEN_EXP
- SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS
- AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION, S3_BUCKET
- FASTAPI_URL (optional)

## Development notes
- Handlers live under internal/api/handlers, routes in internal/api/routes.
- Business logic placed in internal/services.
- Models and migrations under internal/models and internal/database.
- Logger in pkg/logger.
- Use the FastAPI integration for heavy CSV or image extraction tasks (configurable via FASTAPI_URL).

## Useful commands
- Run lint: go vet && golangci-lint run
- Run tests: go test ./... -v
- Run with env: env $(cat .env | xargs) go run ./cmd/server
