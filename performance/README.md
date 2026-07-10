# Performance tests

## Prerequisites

Install [k6](https://grafana.com/docs/k6/latest/set-up/install-k6/).

```sh
brew install k6
```

## Random quiz API

Run a safe smoke test against the production route. The defaults generate one
request per second for 30 seconds. Before measurement, k6 sends three sequential
warm-up requests. These requests are excluded from `warm_random_quiz_duration`.

```sh
BASE_URL=https://main.d8lcxn6e5s253.amplifyapp.com \
  k6 run performance/k6/random-quiz.js
```

Run the portfolio measurement at five requests per second for five minutes.

```sh
BASE_URL=https://main.d8lcxn6e5s253.amplifyapp.com \
RATE=5 \
DURATION=5m \
PRE_ALLOCATED_VUS=10 \
MAX_VUS=30 \
  k6 run performance/k6/random-quiz.js
```

Record `warm_random_quiz_duration` p(50) and p(95), `http_req_failed`, the test
duration, request rate, execution date, and the AWS region with the result. Do
not use the aggregate `http_req_duration`, because it includes warm-up requests.

## Lambda duration

For the same test window, select the
`/aws/lambda/guess_the_celebrity_api` log group in CloudWatch Logs Insights and
run:

```sql
filter msg = "http request"
| filter route = "/quizzes/random" and status = 200
| stats
    count(*) as requests,
    pct(duration_ms, 50) as p50_ms,
    pct(duration_ms, 95) as p95_ms,
    pct(duration_ms, 99) as p99_ms,
    max(duration_ms) as max_ms
```

k6 measures the end-to-end response time through Amplify. Logs Insights measures
the time spent in the API application. Keep these results separate.

To aggregate Lambda runtime duration while excluding invocations that report a
cold start, use the same isolated test window and run:

```sql
filter @type = "REPORT"
| filter !strcontains(@message, "Init Duration")
| stats
    count(*) as warm_invocations,
    pct(@duration, 50) as p50_ms,
    pct(@duration, 95) as p95_ms,
    pct(@duration, 99) as p99_ms,
    max(@duration) as max_ms
```

Report these values as warm steady-state performance. They do not represent the
latency experienced by a request that creates a new Lambda execution environment.
