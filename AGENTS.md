# AGENTS.md

## Project purpose

Mappe consolidates the map and world generators inventoried in `README.md`.
The repository is still in its review phase. Keep changes focused on evaluating
source projects, recording decisions, and migrating only approved capabilities.

## Source-project workflow

- Start with the source project's linked review issue in `README.md`.
- Do not import code until the issue records an explicit **Include** decision.
- For an included project, preserve its license, attribution, deterministic
  behavior, useful tests, fixtures, and documentation.
- Record the migration boundary and any intentionally omitted behavior in the
  review issue or a linked follow-up issue.
- Update the README inventory when a decision or migration changes a project's
  status.
- Do not publish private repository names or metadata in this public repository
  without explicit authorization.

## Go conventions

- Use the Go version declared in `go.mod`.
- Format changed Go files with `gofmt`.
- Prefer the standard library and keep new dependencies narrowly justified.
- Run `go mod tidy` after dependency changes and review both module files.
- Keep generators deterministic: make seeds and configuration explicit, avoid
  wall-clock or process-global randomness, and keep serialized output stable.
- Always use `math/rand/v2`; do not add new uses of `math/rand`.
- Avoid mutable package-level state. Return errors rather than logging or
  exiting from library code.
- Keep generator-specific code and tests together behind a clear package API;
  share code only after two accepted generators demonstrate the same need.

## Tests and validation

- Run `go test ./...` before completing a change.
- Add tests that distinguish the intended behavior from plausible incorrect
  implementations, especially at coordinate, wrapping, and range boundaries.
- Use fixed seeds for deterministic tests. Add golden fixtures only when the
  rendered or serialized output is itself part of the contract.
- When changing generated images or other visual output, inspect representative
  results in addition to running automated tests.

## Delivery workflow

- Work directly on `main` unless the user requests a different branch.
- Assign every repository issue and pull request to `mdhender` when it is
  created or first handled.
- After the required tests pass, commit the task's changes to `main` and push
  them to `origin/main` unless the user explicitly asks not to push.

## Repository hygiene

- Keep generated artifacts and local environment files out of version control.
- Do not commit credentials, private source snapshots, or third-party assets
  whose redistribution terms have not been reviewed.
- Keep `README.md` accurate as the project inventory and decisions evolve.
