---
name: golang-testing
description: Write, review, and debug reliable Go 1.22 tests for the API and Lambda workers. Use when changing Go behavior, adding regression coverage, reviewing test quality, diagnosing flaky tests, testing HTTP handlers or AWS-facing services, or deciding whether race, fuzz, benchmark, or integration coverage is appropriate.
---

# Go Testing

Treat tests as executable behavior specifications. Match the repository's existing standard-library test style and avoid new test dependencies unless they materially improve the suite.

## Workflow

1. Read the production path, its consumers, and adjacent tests before choosing cases.
2. Identify observable behavior, boundaries, error paths, and prior regressions.
3. Prefer the smallest test level that proves the behavior:
   - Pure logic: package unit test.
   - Gin handler or router: `httptest` request/response test.
   - Service with AWS SDK: consumer-defined interface and deterministic fake.
   - Real AWS or network behavior: separate integration test only when explicitly required.
4. Implement focused tests without changing production design solely to satisfy a mock framework.
5. Run the narrow test first, then the affected module suite.

## Test design

- Use table-driven tests when multiple inputs share the same setup and assertion structure.
- Give every table case a meaningful `name` and run it with `t.Run`.
- Test observable results and contracts, not implementation details or call order unless ordering is the contract.
- Cover success, validation boundaries, dependency failures, authorization decisions, and malformed data as applicable.
- Keep tests deterministic: inject clocks and dependencies; do not call real AWS services.
- Use `t.Parallel()` only when the test and shared fixtures are concurrency-safe.
- Define small interfaces at the consuming package and fake those interfaces. Do not add a mocking library by default.
- Use fuzz tests for parsers, decoders, and other input-heavy functions when they add durable value.
- Do not use APIs introduced after Go 1.22, including `testing/synctest` and `b.Loop`.

## Verification

For API changes:

```sh
cd server/api
gofmt -w .
go vet ./...
go test ./...
```

For Feed Worker changes:

```sh
cd server/feed-worker
gofmt -w .
go vet ./...
go test -race ./...
```

Run `go test -race ./...` for concurrency-sensitive API changes. Report the exact commands and results; do not claim coverage that was not executed.
