# raterlog-go

## Operating Rules

- Use `mise` for tool versions and tasks.
- Keep the Go API as the owner of authentication, sessions, companies, members, invites, channels, and API tokens.
- Store relational product and authorization data in Postgres.
- Store append-heavy log events in Cassandra or ScyllaDB.
- Keep API changes documented in `README.md` and user-visible changes in `CHANGELOG.md`.
- Run `gofmt` and `go test ./...` before handing off backend changes.
