---
name: golang-error-handling
description: Design, implement, and review idiomatic Go 1.22 error creation, wrapping, inspection, translation, and structured logging. Use when changing repository or service error paths, mapping errors to HTTP responses, handling AWS SDK failures, reviewing swallowed or duplicate errors, or improving Lambda and Gin diagnostics.
---

# Go Error Handling

Preserve useful error chains internally, translate errors at system boundaries, and log each failure once at the layer that owns the final outcome.

## Workflow

1. Trace the error from its source to the HTTP or Lambda boundary.
2. Decide whether each layer can recover, translate, or only add context.
3. Preserve programmatic identity with `%w` when callers need `errors.Is` or `errors.As`.
4. Classify failures with `errors.Is` or `errors.As` and map them to fixed public status codes and messages. Never return a wrapped error's `Error()` text directly.
5. Log at the boundary with stable messages and structured attributes.

## Rules

- Check every returned error. Discard one only when the API contract makes the result irrelevant, and document non-obvious cases.
- Add concise operation context: `fmt.Errorf("query published quizzes: %w", err)`.
- Keep error strings lowercase and without terminal punctuation.
- Use `errors.Is` for sentinel errors and `errors.As` for typed errors. Stay within Go 1.22 APIs.
- Inspect AWS failures with service-specific error types or `smithy.APIError`; do not branch on error strings. Map unknown AWS failures to a safe internal-error response.
- Use sentinel errors for stable expected categories and custom error types only when callers need structured data.
- Use `errors.Join` when multiple independent cleanup or validation failures must be retained.
- Avoid `panic` for expected failures. Recover only at a deliberate process or request boundary.
- Do not emit the same diagnostic error log and then return it when an upper layer will log it again. A separate access log for every HTTP request is not a duplicate diagnostic log.
- Use `slog` for structured logging. Keep messages stable and attach IDs, counts, durations, and operations as attributes.
- Never expose AWS errors, stack traces, credentials, tokens, table names, or internal identifiers to users.
- Do not introduce third-party error frameworks unless explicitly requested.

## Review checklist

- Is any error ignored, overwritten, or converted to a string too early?
- Does wrapping preserve the original error where inspection matters?
- Does the public response use the intended status and safe message?
- Is the same failure logged more than once?
- Do logs contain enough operation context without sensitive data?
- Are error paths covered by focused tests?

Follow the complete validation commands in `AGENTS.md` for every affected module. In particular, run `go test -race ./...` for Feed Worker changes rather than substituting the non-race command.
