# Shopify OAuth Authentication

A Shopify OAuth 2.0 demo written in Go. A single `/login` endpoint starts both the installation and login flows, stores the offline access token in PostgreSQL, and returns a simple HTML dashboard.

## Features

- Single `/login`: if the shop exists in the DB **and the request is Shopify-signed with HMAC**, it redirects to `/dashboard`; otherwise it starts the OAuth flow (install / re-authorization).
- Offline access token: a long-lived token is stored in the DB; upsert is used on reinstall.
- Security: Shopify HMAC validation, CSRF protection with nonce (state), `*.myshopify.com` domain validation.
- Simple demo UI: `/dashboard` returns plain HTML.

## Stack

- Go (go.mod: `1.24.4`)
- Gin
- PostgreSQL
- pgx/v5

## Project Structure

    .
    +-- cmd/
    |   +-- server/
    |       +-- main.go
    +-- internal/
    |   +-- config/
    |   |   +-- config.go
    |   +-- db/
    |   |   +-- db.go
    |   +-- httpapi/
    |   |   +-- handlers.go
    |   |   +-- router.go
    |   +-- repository/
    |   |   +-- shop_repository.go
    |   |   +-- state_repository.go
    |   +-- shopify/
    |       +-- authorize.go
    |       +-- hmac.go
    |       +-- token.go
    +-- migrations/
    |   +-- 001_create_shops.sql
    |   +-- 002_create_oauth_states.sql
    +-- docker-compose.yml
    +-- .env.example
    +-- go.mod

## Requirements

- Go (compatible with the version in go.mod)
- Docker + Docker Compose
- Shopify Partners account and a development store
- ngrok (or a similar HTTPS tunnel)

## Setup

1. Start PostgreSQL

   docker compose up -d

2. Run migrations

macOS / Linux:

    cat migrations/001_create_shops.sql | docker exec -i shopify_auth_db psql -U app -d shopify_auth
    cat migrations/002_create_oauth_states.sql | docker exec -i shopify_auth_db psql -U app -d shopify_auth

Windows (PowerShell):

    type migrations\001_create_shops.sql | docker exec -i shopify_auth_db psql -U app -d shopify_auth
    type migrations\002_create_oauth_states.sql | docker exec -i shopify_auth_db psql -U app -d shopify_auth

3. Configure environment variables

   cp .env.example .env

Example `.env`:

    APP_PORT=8080
    DATABASE_URL=postgres://app:app@localhost:5433/shopify_auth?sslmode=disable
    SHOPIFY_API_KEY=your_api_key_here
    SHOPIFY_API_SECRET=your_api_secret_here
    SHOPIFY_SCOPES=read_products
    OAUTH_CALLBACK_URL=https://your-subdomain.ngrok-free.dev/auth/callback

4. Start the ngrok tunnel

   ngrok http 8080

Update the HTTPS URL in `.env` (`OAUTH_CALLBACK_URL`) and in your Shopify app settings.

5. Shopify app settings

- App URL: `https://<ngrok-host>/login`
- Allowed redirection URL(s): `https://<ngrok-host>/auth/callback`

6. Run the server

   go run cmd/server/main.go

The server listens on `http://localhost:8080`.

## Endpoints

- `GET /health` -> `{ "ok": true }`

- `GET /login?shop=<shop-domain>`

  - `shop` is required and must match the `*.myshopify.com` format.
  - If the shop exists in the DB and the request includes `hmac` (Shopify-signed request), it redirects to `/dashboard`.
  - Otherwise it generates a nonce, stores it with a TTL in the DB, and redirects to Shopify OAuth.

- `GET /auth/callback`

  - Parameters: `shop`, `code`, `state`, `hmac` (`timestamp` is typically present but not used).
  - HMAC and nonce are validated; the nonce is **deleted from the DB after successful validation** (hard delete).
  - The offline token is obtained using `code`, the shop is stored (upsert), then it redirects to `/dashboard`.

- `GET /dashboard?shop=<shop-domain>`
  - Returns shop info from the DB as simple HTML (demo, no session protection).

## OAuth Flow (summary)

1. `/login?shop=store.myshopify.com`
2. If the shop is not in the DB (or the request is not Shopify-signed), generate a nonce and store it in the DB with a TTL (10 minutes).
3. Redirect to Shopify authorize URL with `grant_options[]=offline`.
4. Shopify returns to `/auth/callback`: HMAC and nonce are validated.
5. After nonce validation it is deleted from the DB; `code` -> offline token; the shop is upserted.
6. Redirect to `/dashboard`.

## Database

- `shops`: `shop_domain` UNIQUE; stores offline token and scopes; upsert on reinstall.
- `oauth_states`: `nonce` UNIQUE; `expires_at` TTL. When the nonce is validated in callback, the row is **deleted** (hard delete).

  - Note: the migration includes a `used_at` column, but it is not used by the current implementation.
  - Optional cleanup (for expired nonces when no callback happens):

        DELETE FROM oauth_states WHERE expires_at < NOW();

## Security

- **HMAC**: Shopify-signed requests are verified using `SHOPIFY_API_SECRET` (`internal/shopify/hmac.go`).
- **Nonce/State**: cryptographically random nonce with a 10-minute TTL; validated on callback and deleted from the DB to enforce single-use.
- **Domain**: `*.myshopify.com` validation via regex (`internal/httpapi/handlers.go`).

## Manual Test

- First install: `/login?shop=...` -> approve in Shopify -> `/dashboard`.
- Existing shop: open the app from Shopify Admin (HMAC-signed request) -> expect direct `/dashboard`.
- Reinstall: uninstall and install again; token/scope are updated.
