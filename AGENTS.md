# Repository Guidelines

## Project Structure & Module Organization
The CLI entry lives in `cmd/run-all/main.go`; keep new commands behind feature flags or additional subcommands there so the flag parsing remains simple. Shared logic belongs in `pkg/dirutils`, where existing helpers and tests already model directory filtering and command orchestration. Top-level files supply tooling (`Makefile`, `go.mod`, `go.sum`) and docs (`README.md`, `LICENSE`); add new assets or scripts beside the Makefile only when they are part of the build pipeline.

## Build, Test, and Development Commands
- `go build -o run-all ./cmd/run-all`: quick local binary for smoke-testing new options.
- `make build`: runs formatting, vetting, the security scanners, and `golangci-lint` before compiling; use this before publishing or tagging a release.
- `make unit`: executes `go test --count=1 ./...`; rely on it for fast pre-commit feedback.
- `make cov`: generates `coverage.out` plus an HTML report; open `coverage.html` when debugging missed paths.

## Coding Style & Naming Conventions
This repo depends on standard Go formatting (tabs, go fmt) plus `golines` for line wrapping, so never hand-format files—run `make fmt` or let your editor call `goimports`. Exported identifiers describe behavior (`RunAll`, `DirectoryMatcher`), while package-private helpers should stay concise and lowerCamelCase. Keep flags and JSON keys kebab-case to match current CLI usage. Let `golangci-lint` be the arbiter for style disputes; do not merge lint failures.

## Testing Guidelines
Unit tests sit next to their packages (see `pkg/dirutils/dirutils_test.go`) and follow the standard `TestThing` naming convention. Table-driven tests are preferred for permutations of directory filters or command flags. Always cover success and failure paths, particularly for concurrency and exclusion logic, and add regression tests whenever bugs mention specific patterns. Run `make unit` before pushing and `make cov` when altering traversal logic so you can inspect updated coverage.

## Commit & Pull Request Guidelines
Recent history favors short, imperative subject lines such as `switch to assert` or `added regex to exclude directories`; mirror that style and mention the touched subsystem when possible (`dirutils: guard nil pattern`). Each pull request should include: a concise summary of why the change is needed, reproduction or testing notes (commands run), and links to tracking issues. Attach CLI output or screenshots if behavior changes. Ensure CI (lint, tests, security) is green before requesting review.

## Security & Dependency Hygiene
Run `make sec` when bumping dependencies—it chains `gosec`, `govulncheck`, and `trivy` to keep the multi-directory execution model safe. Add new third-party modules via `go get` followed by `make mod` so `go.mod` stays minimal and verified. Never store credentials in the repo; prefer environment variables that the CLI reads at runtime.
