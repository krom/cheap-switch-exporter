## Why

The entire exporter lives in one 778-line `main.go` (config, models, collector, both switch
dialects, background polling, and helpers all interleaved), which the project's own CLAUDE.md
calls out as "the entire exporter is a single file." That made sense while the exporter was
small, but it now covers two switch dialects, a background poller, and a Prometheus collector,
and a single flat file makes it harder to navigate, review, and extend (e.g. adding a third
dialect). This change splits `main.go` into cohesive files within the same package and tidies
up a few code-quality rough edges, without changing any externally observable behavior.

## What Changes

- Split `main.go` into multiple files, same `package main`, grouped by the existing
  `// ================= SECTION =================` boundaries already present in the file:
  - `config.go` — `RootConfig`, `GlobalSettings`, `ProfileConfig`, `NamedProfile`,
    `resolveProfileConfig`, `validateProfileConfig`, plus a new extracted `loadConfig` /
    `buildProfiles` helper (see below).
  - `model.go` — `CounterKind`, counter name/help maps, `Port`, `PortStatus`, `PoEPort`.
  - `collector.go` — `profileState`, `PortStatsCollector`, `NewCollector`, `Describe`, `Collect`.
  - `switchclient.go` — the `switchClient` interface and `newSwitchClient` dialect factory.
  - `poll.go` — `scrapeOnce`, `pollProfile`, `StartPolling`.
  - `defaultclient.go` — the default HTML dialect: `fetchDoc`, `fetchPorts`, `fetchPortStatus`,
    `fetchPoE`, `defaultClient`.
  - `v2client.go` — the v2 JSON dialect: `v2Client`, `v2Login`, `v2PortJSON`, `linkStatusRe`,
    `parseLinkStatus`.
  - `parse.go` — small stateless helpers: `state`, `link`, `duplex`, `onoff`, `poeType`, `speed`,
    `parseNum`, `md5hex`.
  - `main.go` — retains only `main()`.
  - `main_test.go` split into matching `*_test.go` files alongside each new source file.
- Extract config file reading + profile resolution out of `main()` into a `loadConfig(path
  string) ([]NamedProfile, error)` function in `config.go`, so it's unit-testable without
  `log.Fatal` calls, and `main()` becomes a thin wrapper that calls it and handles the error.
- Add package-level and exported-symbol doc comments (Go convention: exported identifiers get a
  doc comment starting with their name) where currently missing.
- Wrap propagated errors with `fmt.Errorf("...: %w", err)` at existing return points that
  currently return bare errors, so failures are traceable to which switch/profile/request
  caused them (no new error paths, no changed error *behavior*, just clearer messages).
- Update `CLAUDE.md` to describe the new multi-file layout instead of "the entire exporter is
  a single file."

**Non-behavioral**: no metric names/labels, config schema, HTTP endpoints, or scrape semantics
change. This is a structural and readability refactor only.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
(none — no spec-level requirement changes; existing `config-loading`, `background-polling`,
and `v2-switch-client` specs continue to hold, just implemented across more files)

## Impact

- `main.go` → split into `config.go`, `model.go`, `collector.go`, `switchclient.go`, `poll.go`,
  `defaultclient.go`, `v2client.go`, `parse.go`, `main.go`.
- `main_test.go` → split into matching `*_test.go` files.
- `CLAUDE.md` — architecture description updated to match the new file layout.
- No changes to `config.yaml` schema, Docker/compose files, or public behavior.
