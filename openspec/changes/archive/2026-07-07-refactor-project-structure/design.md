## Context

`main.go` is 778 lines, already internally organized into ten
`// ================= SECTION =================` comment blocks (CONFIG, MODELS, POLLING
STATE, COLLECTOR, SWITCH CLIENT, BACKGROUND POLLING, FETCH/PARSE, V2 CLIENT, HELPERS, MAIN).
Tests live in a single parallel `main_test.go`. The user asked for (a) splitting into files
along sensible boundaries and (b) general code-quality maturity improvements, but explicitly
chose the lower-risk option of staying in one `package main` rather than introducing a
`cmd/`+`internal/` layout.

## Goals / Non-Goals

**Goals:**
- One file per cohesive concern, matching the existing section comments, so file boundaries
  need no behavior change to introduce.
- `main()` reduced to wiring only: load config, build collector, start polling, start HTTP
  server, wait for shutdown signal.
- Config loading (`os.ReadFile` + `yaml.Unmarshal` + per-profile resolve/validate) becomes a
  plain function returning `([]NamedProfile, error)` instead of living inline in `main()` with
  `log.Fatal` calls, so it can be unit tested.
- Exported identifiers (`Port`, `PortStatus`, `PoEPort`, `PortStatsCollector`, `NewCollector`,
  etc.) get doc comments per Go convention.
- Errors returned from parsing/fetch functions are wrapped with context (`fmt.Errorf("fetch
  ports: %w", err)`-style) so a failure log line names the operation that failed.
- Tests split to mirror the new source files.

**Non-Goals:**
- No `cmd/`+`internal/` package split (rejected by the user for this pass — higher migration
  cost, cross-package type exports, import rewrites).
- No behavior changes: no new config fields, no new metrics, no changed error *conditions*,
  no changed HTTP server shutdown semantics (see Open Questions — flagged as a separate
  follow-up, not part of this refactor).
- No dependency updates or `go.mod` changes.

## Decisions

- **File boundaries follow the existing section comments** rather than inventing a new
  grouping, since the code is already organized that way and it's the path of least risk /
  easiest review (a near-mechanical cut-and-paste, not a redesign).
  - `defaultclient.go` and `v2client.go` names (not `htmlclient.go` / `jsonclient.go`) are
    chosen to match the existing `defaultClient`/`v2Client` type names used in the code and
    docs (README's "profile: v2", CLAUDE.md's dialect terminology).
  - `switchclient.go` holds only the interface + factory (`newSwitchClient`), not either
    implementation, so it's the natural place to add a third dialect later without touching
    either existing client file.
- **Extract `loadConfig` out of `main()`**: the current `main()` inlines file reading, YAML
  unmarshal, and per-profile resolve/validate with `log.Fatal` sprinkled through it, which is
  both untested (no test touches `main()`) and not reusable. Moving it to a `loadConfig(path
  string) ([]NamedProfile, error)` function in `config.go` lets `main()` do
  `profiles, err := loadConfig("config.yaml"); if err != nil { log.Fatal(err) }` and lets a new
  test exercise the resolve+validate wiring end-to-end against a real YAML fixture, closing a
  gap the existing table-driven tests (which call `resolveProfileConfig`/`validateProfileConfig`
  directly) don't cover.
- **Error wrapping is additive, not a rewrite**: only wrap errors at points that currently
  return a bare `err` from a lower-level call (e.g. `fetchDoc`, `v2Login`, JSON decode) with a
  short static prefix identifying the operation. No new validation, no new failure modes, no
  change to what `Collect()` does with the returned error (still increments
  `exporter_scrape_errors_total` and skips that profile).
- **Test split mirrors source split 1:1** (`config_test.go`, `model_test.go` if any pure model
  tests exist, `collector_test.go`, `defaultclient_test.go`, `v2client_test.go`, `parse_test.go`)
  rather than keeping one giant `main_test.go`, for the same navigability reason as the source
  split. Fixture files (`examples/`, `testdata/`) are unaffected — only which `_test.go` file
  contains the test function moves.

## Risks / Trade-offs

- **Mechanical move errors (unused imports, misplaced unexported helpers) breaking the build**
  → Mitigated by moving code in small groups and running `go build ./... && go vet ./... && go
  test ./...` after each file split, not once at the end.
- **Git history/blame continuity for moved code** → `git mv` is not usable here since content is
  being split out of one file into several (not a straight rename); Go tooling and reviewers
  generally accept this for a one-time structural refactor. Mitigation: keep this as a single,
  clearly-labeled commit/PR so history readers see one refactor commit, not scattered churn.
- **Scope creep into unrelated code-quality changes** → Mitigated by the Non-Goals list above;
  anything not explicitly listed (e.g. graceful HTTP shutdown, config hot-reload, structured
  logging) is out of scope for this change.

## Migration Plan

1. Create new files with the moved code, section by section, verifying `go build`/`go vet`/
   `go test` after each move.
2. Reduce `main.go` down to just `main()` once all other sections are moved.
3. Split `main_test.go` into matching `*_test.go` files, verifying `go test ./...` still passes
   with identical test names/coverage.
4. Update `CLAUDE.md`'s "single file" description to describe the new layout.
5. No deployment/runtime migration needed — this is a compile-time-only, same-binary change;
   rollback is a normal `git revert` if needed.

## Open Questions

- Whether to also add graceful HTTP server shutdown (currently `go http.ListenAndServe(...)`
  runs undrained and the process just exits on signal) is a real gap toward "a more mature
  product," but it's a behavior change against the `background-polling` spec's shutdown
  requirement, not a pure refactor — proposed as a separate follow-up change rather than
  bundled here.
