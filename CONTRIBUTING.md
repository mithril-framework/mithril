# Contributing to Mithril

Thank you for your interest in contributing to Mithril!

## Getting Started

1. Fork the repository on GitHub.
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/mithril.git
   cd mithril
   ```
3. Install dependencies:
   ```bash
   make install
   # or
   ./mithril install
   ```
4. Copy environment file and start Postgres (optional):
   ```bash
   cp env.example .env
   make dc-up-postgres
   make migrate-up
   ```

## Development

- Run the app: `make run` or `make run-dev` (live reload with Air)
- Run tests: `make test`
- List routes: `make routes`
- Build: `make build`

## Pull Requests

1. Create a feature branch from `main`.
2. Make your changes with clear commit messages.
3. Ensure tests pass: `go test ./...`
4. Open a PR with a description of what changed and why.
5. Link any related issues.

## Code Style

- Follow standard Go conventions (`gofmt`, idiomatic naming).
- Match existing patterns in `internal/`, `routes/`, and `cmd/`.
- Keep changes focused — one logical change per PR when possible.

## Reporting Issues

Use GitHub Issues for bugs and feature requests. Include:

- Go version (`go version`)
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs or stack traces

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
