# Contributing to eyrie

Thanks for considering a contribution. eyrie is the LLM client + skill router
that powers the hawk-eco ecosystem. It is built for **solo developers** —
small surface area, zero magic, fast feedback.

## Quick start

```bash
git clone https://github.com/GrayCodeAI/eyrie.git
cd eyrie
make build      # CGO_ENABLED=0 go build to bin/eyrie
make test       # race detector, -count=1, -timeout=120s
make lint       # golangci-lint v2
```

Go 1.26.1 is the targeted toolchain.

## Branch flow

- The default branch is `dev`. `main` is reserved for tagged releases.
- Branch from `dev`: `git checkout -b feat/<scope>-<short-description>`.
- Open the PR against `dev`. Do **not** push directly to `dev` or `main`.
- One PR per logical change. Do not mix unrelated changes in a single PR.

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(client): add Vertex AI streaming support
fix(router): correct fallback selection under context cancellation
perf(catalog): cache embedded model lookup
docs(readme): document EYRIE_PROFILE env var
test(client): add OpenAI batch reconnect coverage
```

Allowed types: `feat`, `fix`, `perf`, `refactor`, `test`, `docs`, `chore`,
`build`, `ci`, `style`. Add a scope when it clarifies the change. Do not add
`Co-authored-by:` trailers — this is solo-developer work.

## Code standards

- `gofmt -l .` must be empty for files you touch.
- `go vet ./...` must be clean.
- `golangci-lint run ./...` must surface no new findings. The repo enables
  `errcheck`, `staticcheck`, `gocritic`, `unused`, `ineffassign`, `misspell`,
  `noctx`, `bodyclose`, `unconvert`, `whitespace`. Use `//nolint:<linter>`
  only with a one-line justification.
- Public APIs must have godoc comments.
- Prefer table-driven tests with `t.Parallel()` where independent.
- Wrap errors with context: `fmt.Errorf("loading config from %s: %w", path, err)`.
- Propagate `context.Context` everywhere; never call out to a network without it.
- Keep packages small and feature-named (`client`, `router`, `catalog`).
  Banned package names: `util`, `helpers`, `common`, `misc`.

## Testing

```bash
make test          # full suite with race detector
make test-coverage # generates coverage.out and prints the totals
make bench         # benchmarks
```

When adding a new provider client, add tests for: success, retry on 429/5xx,
non-retriable error pass-through, context cancellation, and (where it
applies) streaming.

## Updating CHANGELOG

User-visible changes go under `## [Unreleased]` in `CHANGELOG.md`. Group by
`Added` / `Changed` / `Fixed` / `Removed` / `Security`. Keep entries
short and actionable.

## Reporting bugs / requesting features

- Bug: open an issue using the bug report template.
- Feature: open an issue using the feature request template.
- Security: do **not** file a public issue. Use a GitHub Security Advisory
  per `SECURITY.md`.
