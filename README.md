# Raterlog Go

Raterlog Go is a backend rewrite of `../raterlog`. It keeps the original split-storage intent, but makes ownership explicit:

- Postgres stores users, sessions, companies, team memberships, invites, channels, and API tokens.
- Cassandra or ScyllaDB stores append-heavy event logs.

The original Rust API and Next.js implementation lives at [suryavirkapur/raterlog](https://github.com/suryavirkapur/raterlog).

## Run

```sh
cp .env.example .env
docker compose up --build
```

For local development without Docker:

```sh
mise run dev
```

The API listens on `http://localhost:18080` through Docker Compose, and the web UI runs on `http://localhost:13000`. The API applies its Postgres and Scylla/Cassandra schema on startup. When run directly with `mise run dev`, it listens on `http://localhost:8080` unless `HTTP_ADDR` is changed.

## Main Endpoints

Authentication:

- `POST /api/auth/signup`
- `POST /api/auth/signin`
- `POST /api/auth/signout`
- `GET /api/auth/me`

Companies and teams:

- `GET /api/companies`
- `POST /api/companies`
- `GET /api/companies/{companyID}`
- `PATCH /api/companies/{companyID}`
- `GET /api/companies/{companyID}/members`
- `POST /api/companies/{companyID}/invites`
- `DELETE /api/companies/{companyID}/invites/{inviteID}`
- `GET /api/invites/{token}`
- `POST /api/invites/{token}/accept`

Channels and API tokens:

- `POST /api/companies/{companyID}/channels`
- `POST /api/companies/{companyID}/tokens`
- `DELETE /api/companies/{companyID}/tokens/{tokenID}`

Logs:

- `POST /api/logs`
- `GET /api/logs/{channelID}`
- Compatibility aliases: `POST /log`, `GET /log/{channelID}`

Log ingestion accepts an API token in either `Authorization: Bearer <token>` or `Authorization: Basic <token>`. Session-backed dashboard requests use the `raterlog_session` cookie or `X-Session-Token`.

## Storage Boundary

Use Postgres when the data needs joins, uniqueness constraints, membership checks, ownership, or transactional updates. Use Cassandra/ScyllaDB for time-ordered channel events where queries are scoped to a channel and ordered by timestamp.

## Migrations

Migrations are embedded and applied on API startup:

- Postgres migrations live in `internal/postgres/migrations/*.sql` and are tracked in `schema_migrations`.
- Scylla/Cassandra migrations live in `internal/scylla/migrations/*.cql` and are tracked in `schema_migrations` inside the configured keyspace.

Add migrations with a zero-padded version prefix, for example `0002_add_plan_limits.sql` or `0004_add_log_ttl.cql`.

## Verification

SEASNOKE_MULTITURN_EXISTING_FILE_20260612201850
