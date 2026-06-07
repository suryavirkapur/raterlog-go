# Changelog

## 0.1.0 - 2026-06-08

- Rewrote the Raterlog backend in Go.
- Moved authentication, sessions, company/team membership, invites, channels, and API token management into the backend.
- Kept Cassandra/ScyllaDB for event logs and Postgres for relational authorization and team data.
- Moved startup schema setup into embedded file-based migrations for Postgres and Scylla/Cassandra.
