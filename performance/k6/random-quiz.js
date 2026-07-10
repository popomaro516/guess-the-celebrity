import http from "k6/http";
import { check } from "k6";
import { Trend } from "k6/metrics";

const baseUrl = (__ENV.BASE_URL || "").replace(/\/$/, "");

if (!baseUrl) {
  throw new Error("BASE_URL is required");
}

const rate = Number.parseInt(__ENV.RATE || "1", 10);
const preAllocatedVUs = Number.parseInt(__ENV.PRE_ALLOCATED_VUS || "2", 10);
const maxVUs = Number.parseInt(__ENV.MAX_VUS || "10", 10);
const warmupRequests = Number.parseInt(__ENV.WARMUP_REQUESTS || "3", 10);

if (![rate, preAllocatedVUs, maxVUs, warmupRequests].every(Number.isInteger)) {
  throw new Error("RATE, PRE_ALLOCATED_VUS, MAX_VUS, and WARMUP_REQUESTS must be integers");
}

const warmRequestDuration = new Trend("warm_random_quiz_duration", true);

export const options = {
  scenarios: {
    randomQuiz: {
      executor: "constant-arrival-rate",
      rate,
      timeUnit: "1s",
      duration: __ENV.DURATION || "30s",
      preAllocatedVUs,
      maxVUs,
    },
  },
  summaryTrendStats: ["min", "avg", "p(50)", "p(95)", "p(99)", "max"],
  thresholds: {
    "http_req_failed{endpoint:random-quiz}": ["rate<0.01"],
    "checks{endpoint:random-quiz}": ["rate>0.99"],
  },
};

export function setup() {
  for (let index = 0; index < warmupRequests; index += 1) {
    const response = http.get(`${baseUrl}/api/quizzes/random`, {
      tags: { endpoint: "random-quiz", phase: "warmup" },
    });
    if (response.status !== 200) {
      throw new Error(`warm-up request failed with status ${response.status}`);
    }
  }
}

export default function () {
  const response = http.get(`${baseUrl}/api/quizzes/random`, {
    tags: { endpoint: "random-quiz", phase: "measurement" },
  });
  warmRequestDuration.add(response.timings.duration);

  check(
    response,
    {
      "status is 200": (result) => result.status === 200,
      "response is a quiz": (result) => {
        try {
          const body = result.json();
          return (
            Array.isArray(body.quizzes) &&
            body.quizzes.length > 0 &&
            body.quizzes.every(
              (quiz) =>
                typeof quiz.quiz_id === "string" &&
                typeof quiz.question === "string" &&
                Array.isArray(quiz.choices) &&
                quiz.choices.length === 4,
            )
          );
        } catch {
          return false;
        }
      },
    },
    { endpoint: "random-quiz" },
  );
}
