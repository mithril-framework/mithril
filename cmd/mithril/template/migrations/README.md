Goose SQL migrations for this app (`NNNN_name.sql` with `-- +goose Up` / `-- +goose Down`).
They are embedded in the binary; apply them with `go run . migrate up`.
Mithril's own tables (users, roles, permissions) are created by the framework's core migrations.
