# Domain-First Refactor Plan

## Goal

Restructure `gowatch` so package boundaries follow business domains instead of generic technical buckets. Keep behavior stable while reducing coupling between `internal/services`, `internal/models`, and `db.DB`.

This plan follows ideas from Redowan's article: domain packages at top, transport and persistence at edges, composition in startup wiring.

## Ground Rules

- No big-bang rewrite.
- Keep app working after each phase.
- Prefer moving one bounded slice at a time.
- Keep `internal/server/run.go` as composition root during migration.
- Keep `db/` as edge adapter for now; split interfaces before renaming packages.
- Refactor handlers last, after domain boundaries are stable.

## Current Problems

- `internal/services/` is one horizontal package for all business logic.
- `internal/models/` mixes domain entities, UI DTOs, import/export DTOs, and TMDB mapping.
- `db/interface.go` is one giant cross-domain interface.
- HTTP handlers are grouped by transport (`pages`, `htmx`, `api`) instead of feature/domain.
- Many services depend on implicit user state via `common.GetUser(ctx)`.

## Target Shape

Initial target:

```txt
internal/
  account/
  analytics/
  catalog/
  dashboard/
  libraryio/
  lists/
  watchlog/
  http/
  middleware/
  common/
  ui/
db/
cmd/
```

Notes:

- `account`, `catalog`, `lists`, `watchlog`, `analytics` are core domains.
- `dashboard` and `libraryio` are app-layer orchestrators across domains.
- `http` and `db` remain edge adapters.

## Domain Mapping

- `account`: users, sessions, login, registration, password reset, admin user ops.
- `catalog`: movies, people, search, TMDB fetch, TMDB-to-domain mapping, movie cache.
- `lists`: custom lists, watchlist, ordering, list display helpers.
- `watchlog`: watched CRUD, watch activity, watched history.
- `analytics`: stats and reporting derived from watched data.
- `dashboard`: home page aggregation.
- `libraryio`: import/export orchestration across watched and lists.

## Phase 0: Baseline And Safety

Status: [x] Completed

### Tasks

- [x] Confirm current tests pass with `make test`.
- [x] Record current package dependency shape for reference.
- [x] Identify any active worktree changes that overlap with refactor files.
- [x] Create this file and keep it updated as source of truth.

### Exit Criteria

- We have a stable baseline before moving code.

## Phase 1: Define Domain Boundaries Without Moving Everything

Status: [~] In progress

### Tasks

- [x] Create new domain packages with minimal starter files:
  - [x] `internal/account`
  - [x] `internal/catalog`
  - [x] `internal/lists`
  - [x] `internal/watchlog`
  - [x] `internal/analytics`
  - [x] `internal/dashboard`
  - [x] `internal/libraryio`
- [x] Add package docs describing each package's responsibility.
- [x] Decide which existing files belong to each domain before moving code.
- [x] Keep all wiring in existing `routes` and `server` packages for now.

### Exit Criteria

- New domain package layout exists.
- Each existing service/model file has clear destination.

## Phase 2: Split Giant DB Dependency Into Local Ports

Status: [~] In progress

### Why

This is highest leverage step. It reduces coupling before file moves.

### Tasks

- [~] For each domain, define small repository interfaces near consumer code.
- [~] Replace direct `db.DB` dependency in constructors with domain-local interfaces.
- [x] Keep `*db.SqliteDB` satisfying those interfaces implicitly.
- [ ] Avoid creating one new shared `ports` package; keep interfaces close to domain code.

### Candidate Splits

- [x] `account` repo: user/session/admin methods.
- [ ] `catalog` repo: movie read/write cache methods.
- [ ] `lists` repo: list CRUD, watchlist lookup, item ordering.
- [ ] `watchlog` repo: watched CRUD, watch history queries.
- [ ] `analytics` repo: stats/reporting queries.

### Exit Criteria

- No domain service constructor accepts full `db.DB` unless truly necessary.
- Boundaries are explicit in code.

## Phase 3: Move Account Domain

Status: [x] Completed

### Source

- `internal/services/auth.go`
- `internal/models/auth.go`

### Tasks

- [x] Move auth service logic into `internal/account`.
- [x] Move `User`, `Session`, `UserWithStats` into `internal/account`.
- [x] Keep password hashing and session generation inside account package.
- [x] Replace external imports from `internal/models` with `internal/account` types.
- [x] Preserve watchlist bootstrap behavior for new users.

### Known Coupling

- Account currently depends on list/watchlist initialization.

### Exit Criteria

- [x] No account logic remains in `internal/services/auth.go`.
- [x] Account code depends only on account-local interfaces plus explicit collaborators.

## Phase 4: Move Catalog Domain

Status: [ ] Not started

### Source

- `internal/services/movie.go`
- `internal/models/movie.go`
- `internal/models/person_details.go`
- `internal/models/search.go`
- `internal/models/genre.go`
- `internal/models/cast.go`

### Tasks

- [ ] Move movie/person/search logic into `internal/catalog`.
- [ ] Move TMDB mapping functions into catalog package.
- [ ] Keep TMDB client as edge dependency injected into catalog services.
- [ ] Move catalog types out of `internal/models`.
- [ ] Keep DB caching behavior unchanged.

### Important Note

- TMDB package currently leaks into model layer. Stop that here.

### Exit Criteria

- Catalog owns movie/person/search types and mapping.
- `internal/models/movie.go` no longer exists or is reduced to unrelated leftovers.

## Phase 5: Move Lists Domain

Status: [ ] Not started

### Source

- `internal/services/list.go`
- `internal/services/list_ordering.go`
- `internal/services/list_watchlist_display.go`
- `internal/models/list.go`

### Tasks

- [ ] Move list entities and logic into `internal/lists`.
- [ ] Keep watchlist rules inside lists domain.
- [ ] Keep ordering helpers inside lists domain.
- [ ] Move list view shaping logic if still truly list-specific.
- [ ] Replace old imports across handlers and services.

### Known Coupling

- Lists depends on catalog for ensuring movie metadata exists during some flows.

### Exit Criteria

- Lists and watchlist behavior live together.
- No list-specific code remains in generic services/models packages.

## Phase 6: Move Watchlog Domain

Status: [ ] Not started

### Source

- watched CRUD and history portions of `internal/services/watched.go`
- watched models from `internal/models/watched.go`

### Tasks

- [ ] Extract watched CRUD and history into `internal/watchlog`.
- [ ] Keep explicit dependency on lists port for watchlist removal.
- [ ] Keep explicit dependency on catalog port for ensuring movies exist when needed.
- [ ] Move watched-related types out of `internal/models`.
- [ ] Keep current error semantics like `ErrWatchedEntryConflict`.

### Important Note

- Do not move stats/import-export into watchlog package.

### Exit Criteria

- Watchlog owns watched entry lifecycle.
- Cross-domain dependencies are narrow and explicit.

## Phase 7: Split Analytics Out Of Watchlog

Status: [ ] Not started

### Source

- `internal/services/stats.go`
- stats-related methods attached to `WatchedService`

### Tasks

- [ ] Create `internal/analytics` service(s) for reporting and stats queries.
- [ ] Move analytics types from `internal/models` as needed.
- [ ] Stop attaching reporting logic to watchlog service.
- [ ] Keep read-model style dependency on watchlog/catalog/list data through interfaces.

### Exit Criteria

- Stats/reporting no longer live on watched service.
- Analytics package is read-focused and isolated.

## Phase 8: Create App-Layer Orchestrators

Status: [ ] Not started

### Source

- `internal/services/home.go`
- import/export parts spread across watched/list/api models
- `internal/models/import.go`

### Tasks

- [ ] Move home aggregation to `internal/dashboard`.
- [ ] Move import/export orchestration to `internal/libraryio`.
- [ ] Keep these packages thin; they should compose domains, not own domain rules.
- [ ] Move shared import/export DTOs to `internal/libraryio` unless better owned elsewhere.

### Exit Criteria

- Cross-domain composition no longer lives in core domain packages.

## Phase 9: Refactor HTTP By Feature

Status: [ ] Not started

### Why

Do this late. Otherwise handler churn hides domain boundary work.

### Current Source

- `internal/handlers/pages`
- `internal/handlers/htmx`
- `internal/handlers/api`

### Tasks

- [ ] Create `internal/http` package tree organized by feature/domain.
- [ ] Split handlers by feature instead of transport bucket.
- [ ] Keep transport-specific helpers where useful, but feature packages own endpoints.
- [ ] Update `internal/routes/routes.go` to wire new handlers.
- [ ] Keep static/assets and image handlers at edge.

### Possible Shape

```txt
internal/http/
  account/
  catalog/
  lists/
  watchlog/
  analytics/
  dashboard/
  assets/
  images/
```

### Exit Criteria

- Route wiring reflects feature ownership.
- Handlers no longer depend on giant mixed service bundles.

## Phase 10: Remove Old Generic Buckets

Status: [ ] Not started

### Tasks

- [ ] Delete remaining code from `internal/services`.
- [ ] Delete remaining code from `internal/models`.
- [ ] Remove compatibility aliases if any were added temporarily.
- [ ] Clean package docs and imports.
- [ ] Run formatting, linting, and tests.

### Exit Criteria

- Generic domain buckets are gone.
- New package structure is primary structure of app.

## Cross-Cutting Cleanup

Status: [ ] Not started

### Tasks

- [ ] Reduce implicit `common.GetUser(ctx)` usage where practical.
- [ ] Decide where request-context auth lookup should stop and explicit user IDs should begin.
- [ ] Keep context-based access at app edge if full change is too large.
- [ ] Add or update tests as package moves happen.
- [ ] Avoid changing runtime behavior while restructuring.

## Risks To Watch

- Hidden coupling through `common.GetUser(ctx)`.
- Watchlist invariants split across account, lists, and watchlog.
- Big compile churn from moving types out of `internal/models`.
- DB adapter may expose cross-domain assumptions in conversion code.
- Handler refactor may create noisy diffs if started too early.

## Suggested Execution Order

1. Phase 0
2. Phase 1
3. Phase 2
4. Phase 3
5. Phase 4
6. Phase 5
7. Phase 6
8. Phase 7
9. Phase 8
10. Phase 9
11. Phase 10
12. Cross-cutting cleanup as needed throughout

## Progress Log

- [x] Initial repo-specific refactor plan created.
- [x] Phase 0 started.
- [x] First domain package created.
- [x] First service converted to local interface dependency.
- [x] First domain fully moved.
- [ ] Old `internal/services` removed.

## Working Notes

- Keep edits small enough that `make test` stays useful after each slice.
- Prefer compile-first slices over perfect final layout.
- If a move creates too much churn, stop and split dependency interfaces first.
