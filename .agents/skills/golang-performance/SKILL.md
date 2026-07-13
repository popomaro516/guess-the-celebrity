---
name: golang-performance
description: Investigate, review, and improve Go 1.22 performance with measurement-first methods. Use when API or Lambda latency, throughput, allocations, memory, DynamoDB access, cold starts, or hot paths are under investigation, or when a performance-sensitive change needs benchmark evidence.
---

# Go Performance

Do not optimize from intuition alone. Define the user-visible metric, measure a baseline, locate the bottleneck, change one relevant factor, and compare results.

## Workflow

1. Define the target metric and success threshold: HTTP latency, Lambda duration, allocation count, memory, or throughput.
2. Reproduce with the narrowest representative workload.
3. Separate application CPU and allocation costs from DynamoDB, network, logging, and cold-start time.
4. Capture a baseline using an appropriate source:
   - Package benchmark with `go test -bench ... -benchmem`.
   - CPU or heap profile with `go test` and `go tool pprof`.
   - HTTP behavior with the repository's k6 scripts.
   - Lambda duration with CloudWatch `REPORT` logs.
5. Form one hypothesis and make one focused change.
6. Repeat the same measurement under comparable conditions.
7. Keep the change only when evidence shows a meaningful improvement without correctness or readability regressions.

Before running k6 against production or querying CloudWatch, read `performance/README.md` and obtain explicit user authorization for the external workload or production-data access. For the random-quiz test, report `warm_random_quiz_duration` and do not substitute aggregate `http_req_duration`, which includes warm-up requests.

## Optimization order

Prefer high-leverage work in this order:

1. Remove unnecessary network or DynamoDB operations.
2. Fix inappropriate algorithms or repeated work.
3. Batch, cache, or precompute only with clear invalidation and memory bounds.
4. Reduce hot-path allocations using justified preallocation or reuse.
5. Tune concurrency only after checking downstream limits and cancellation.
6. Consider GC or runtime tuning only with production-like measurements.

## Guardrails

- Preserve behavior with tests before optimizing risky code.
- Do not introduce `unsafe`, pooling, reflection tricks, or custom serialization without benchmark proof.
- Keep caches and goroutine counts bounded.
- Include data size and environment in benchmark comparisons.
- Run benchmark variants serially; concurrent runs contaminate CPU results.
- Use the Go 1.22 `b.N` benchmark loop, not newer `b.Loop` APIs.
- Document non-obvious optimizations with the measured reason, not a generic claim that they are faster.
- Do not treat microbenchmarks as proof of end-to-end Lambda or HTTP improvement.
- After changing code, run the complete formatting, vet, and test commands from `AGENTS.md` for every affected module.

## Commands

```sh
go test -run '^$' -bench=. -benchmem -count=6 ./...
go test -run '^$' -bench BenchmarkName -benchmem -cpuprofile cpu.out -memprofile mem.out ./path/to/package
go tool pprof cpu.out
```

When `benchstat` is available, capture comparable before and after runs and use it to evaluate noise and statistical significance. Use `performance/k6/random-quiz.js` for the existing random-quiz HTTP workload only after the authorization step above. Report before/after results and any uncertainty.
