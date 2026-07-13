# AGENTS.md

## Project overview

Guess the Celebrity is a four-choice celebrity quiz application. The frontend is an Astro site with React islands, and the backend is split into an HTTP API and two Lambda workers.

## Repository layout

- `frontend/`: Astro 7 and React 19 frontend. Authentication uses Amazon Cognito through AWS Amplify.
- `server/api/`: Go 1.22 HTTP API running on Gin and deployed to AWS Lambda.
- `server/feed-worker/`: Go 1.22 Lambda that builds the precomputed public quiz feed in DynamoDB.
- `server/image-worker/`: Python 3.14 Lambda that processes uploaded quiz images from SQS.
- `performance/`: k6 performance tests and measurement notes.
- `docs/`: specifications and architecture assets.
- `.github/workflows/`: validation and deployment workflows for each service.

## Local development

Create local environment files from the provided examples when needed. Do not commit `.env.local` files or credentials.

Start the frontend and API from the repository root:

```sh
docker compose up --build frontend api
```

- Frontend: `http://localhost:4321`
- API: `http://localhost:8080`
- The frontend proxies `/api/*` to the API container.
- Local API authentication is disabled by the Docker Compose configuration.

## Validation

Run the checks for every area you change. Run all affected suites when a change crosses service boundaries.

### Frontend

```sh
cd frontend
npm ci
npm run lint
npm run build
```

`npm run build` includes `astro check`. Use the committed `package-lock.json`; do not introduce another package manager or lockfile.

### API

```sh
cd server/api
gofmt -w .
go vet ./...
go test ./...
```

### Feed worker

```sh
cd server/feed-worker
gofmt -w .
go vet ./...
go test -race ./...
```

### Image worker

```sh
cd server/image-worker
python -m pip install -r requirements-dev.txt
ruff check .
ruff format --check .
pytest
```

## Implementation guidelines

- Keep changes scoped to the service responsible for the behavior.
- Preserve the asynchronous architecture: clients upload images directly to S3, image processing runs through SQS, and public quiz reads use the precomputed feed.
- Do not expose answer data or private creator data in public quiz responses or feed records.
- Keep Cognito configuration and authentication error handling consistent between signup, confirmation, login, and password reset flows.
- Use existing CSS variables and component patterns for frontend styling. Check both desktop and mobile layouts.
- Add or update tests for backend behavior changes. Keep tests deterministic and avoid real AWS calls.
- Treat AWS resource names, account IDs, environment variables, and workflow permissions carefully. Never commit secrets.
- Deployment is performed by GitHub Actions after changes reach `main`; do not deploy Lambda functions manually unless explicitly requested.

## Project skills

Reusable Go workflows are available under `.agents/skills/` for testing, error handling, context propagation, and performance work. Use the relevant skill when a task matches its description; `AGENTS.md` remains authoritative for repository-wide commands and conventions.

## Git and pull requests

- Start branches from the latest `origin/main`.
- Use conventional branch prefixes such as `feature/`, `fix/`, `docs/`, `refactor/`, `test/`, or `chore/`.
- Keep commits focused and use concise English commit messages.
- Use an English pull request title and a Japanese pull request description unless instructed otherwise.
- In the pull request description, include the change, motivation, impact, and validation performed.
- Do not include generated artifacts, local environment files, or unrelated working-tree changes.
