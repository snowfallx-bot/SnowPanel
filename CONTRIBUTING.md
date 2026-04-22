# Contributing to SnowPanel

Thanks for your interest in contributing.
This guide keeps contributions consistent and easy to review.

## Development Setup

1. Fork and clone the repository.
2. Create your env file:
   - macOS/Linux: `cp .env.example .env`
   - PowerShell: `Copy-Item .env.example .env`
3. Start the stack:
   - `make up`
4. Optional local (non-container) workflow:
   - `docker compose up -d postgres redis`
   - `make agent`
   - `make backend`
   - `make frontend`

## Branch And Commit Conventions

- Create a feature branch from `main`.
- Keep one concern per branch and per pull request.
- Prefer clear conventional commit prefixes, for example:
  - `feat:`
  - `fix:`
  - `chore:`
  - `docs:`
  - `refactor:`
  - `test:`

## Quality Checks

Before opening a pull request, run:

```bash
make lint
make test
```

If you only changed one service, you can run a scoped check:

- Backend: `cd backend && go test ./...`
- Core agent: `cd core-agent && cargo fmt --all -- --check && cargo test`
- Frontend: `cd frontend && npm run test && npm run build`

## Pull Request Expectations

- Explain the problem and the solution.
- Link related issues (`Fixes #123`).
- Include screenshots for UI changes.
- Keep docs and examples updated with code changes.
- Ensure CI is green.

## Security Reports

Please do not open public issues for vulnerabilities.
Use GitHub Security Advisories:
`https://github.com/snowfallx-bot/SnowPanel/security/advisories/new`
