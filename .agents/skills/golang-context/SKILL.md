---
name: golang-context
description: Design and review context.Context propagation, cancellation, timeouts, deadlines, and request-scoped values in the Go 1.22 API and Lambda workers. Use when a request crosses handlers, services, repositories, AWS SDK calls, HTTP clients, or goroutines, or when diagnosing leaked work and ignored cancellation.
---

# Go Context

Propagate the caller's context through the complete operation so cancellation and deadlines reach every downstream dependency.

## Rules

- Accept `ctx context.Context` as the first parameter when a function performs cancellable work.
- Pass `c.Request.Context()` from Gin handlers through services and repositories to AWS SDK calls. Do not pass or store `*gin.Context` outside the handler and middleware layer.
- Pass the Lambda invocation context through the worker operation and DynamoDB calls.
- Do not replace an active request context with `context.Background()` or `context.TODO()`.
- Use `context.Background()` only at process entry points, setup code, and tests that have no parent context.
- Never pass a nil context or store a context in a long-lived struct.
- Call the returned cancel function from `context.WithCancel`, `WithTimeout`, or `WithDeadline` on every owned path, normally with `defer cancel()` immediately after creation.
- Do not add arbitrary timeouts at every layer. Establish them at the boundary that owns the operation budget and allow stricter child deadlines only when justified.
- Store only request-scoped metadata in context values. Prefer explicit parameters for business data and use an unexported key type.
- Start goroutines only with a clear owner, lifetime, cancellation path, and error strategy. Do not detach work from a request unless the product behavior explicitly requires it.
- Use `context.WithoutCancel` only for intentional Go 1.21+ detached work, with an independent timeout and documented ownership.

## Review workflow

1. Identify the entry context: Gin request, Lambda invocation, or top-level process context.
2. Trace every call through service and platform layers.
3. Check AWS SDK and HTTP operations receive the propagated context.
4. Inspect goroutines and blocking operations for cancellation handling.
5. Verify timeout ownership and cleanup.
6. Add deterministic cancellation tests where behavior depends on context.

Coordinate cancellation tests with channels or other explicit signals and cancel the context directly. Avoid `time.Sleep` and wall-clock deadline assertions; use a bounded timeout only as a test fail-safe.

Run the affected module tests and use `go vet ./...`. Use `go test -race ./...` when goroutines or shared state change.
