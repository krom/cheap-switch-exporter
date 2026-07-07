## Context

Today, `PortStatsCollector.Collect()` (main.go:184) calls `fetchPorts`/`fetchPortStatus`/
`fetchPoE` synchronously, once per profile, on every single `/metrics` HTTP request — there is
no caching or background polling. `main()`'s config-resolution loop only resolves one setting
(`pc.Timeout` defaulting to `5` if zero); there is no `RootConfig.Settings`, no
`ProfileConfig.PollRateSeconds`, and nothing reads `config.yaml.example`'s already-present
`settings.poll_rate_seconds`/`settings.timeout_seconds` block.

The user wants two related things: (1) a documented, precedence-based way to configure both the
poll interval and the request timeout, globally or per-profile, and (2) an actual background
polling loop — one goroutine per profile, scraping on its own `poll_rate_seconds` interval,
caching results in memory, with `/metrics` served from that cache rather than triggering a live
scrape.

## Goals / Non-Goals

**Goals:**
- Add `settings.poll_rate_seconds`/`settings.timeout_seconds` and per-profile
  `poll_rate_seconds`, resolved with a clear, single precedence rule at startup: profile value →
  global setting → hardcoded default (60s poll, 10s timeout).
- Move scraping off the HTTP request path: one long-lived goroutine per profile polls that
  switch every (resolved) `poll_rate_seconds`, using (resolved) `timeout_seconds` as its HTTP
  client timeout, and stores the latest snapshot in memory.
- `Collect()` reads the latest cached snapshot per profile (no network I/O), so `/metrics`
  response time is decoupled from switch responsiveness/count.
- Clean shutdown: poller goroutines stop when the process receives SIGINT/SIGTERM, same as the
  existing shutdown path.

**Non-Goals:**
- Not adding a new metric for "seconds since last successful poll" or per-profile poll
  duration/status — out of scope for this change; `exporter_scrape_errors_total` and
  `exporter_last_scrape_duration_seconds` are kept but their meaning shifts (see Risks).
- Not changing `fetchPorts`/`fetchPortStatus`/`fetchPoE`'s parsing logic at all — this change is
  purely about *when* and *how often* they're called, and where their results are stored.
- Not adding config validation/hot-reload — `config.yaml` is still read once at startup.

## Decisions

- **Resolve `poll_rate_seconds`/`timeout_seconds` once at startup in `main()`**, the same place
  and style as the existing `if pc.Timeout == 0 { pc.Timeout = 5 }` resolution, extended to:
  ```go
  if pc.Timeout == 0 {
      if cfg.Settings.TimeoutSeconds != 0 {
          pc.Timeout = cfg.Settings.TimeoutSeconds
      } else {
          pc.Timeout = 10
      }
  }
  if pc.PollRateSeconds == 0 {
      if cfg.Settings.PollRateSeconds != 0 {
          pc.PollRateSeconds = cfg.Settings.PollRateSeconds
      } else {
          pc.PollRateSeconds = 60
      }
  }
  ```
  This writes the fully-resolved values back onto each profile's `ProfileConfig`, so downstream
  code (the poller, `fetchDoc`) just reads `cfg.Timeout`/`cfg.PollRateSeconds` directly with no
  precedence logic scattered elsewhere.
  - Alternative considered: resolve precedence lazily inside the poller on every tick — rejected
    as needless repeated work for a value that never changes after startup.
- **One goroutine per profile, each owning a `time.Ticker` at its own resolved interval**, rather
  than a single shared ticker at the fastest interval that scrapes everything on every tick.
  Independent intervals match the requirement directly (a profile with a slower switch or a
  looser SLA can poll less often) and avoid coupling unrelated profiles' scrape cadence.
  - Alternative considered: a single goroutine iterating all profiles on one shared tick —
    rejected because it can't honor different `poll_rate_seconds` per profile and would let one
    slow switch's timeout delay every other profile's scrape on that tick.
- **In-memory snapshot per profile, guarded by `sync.RWMutex`**: a `profileState` struct holding
  the last successfully parsed `[]Port`, `[]PortStatus`, PoE consumption/`[]PoEPort`, and last
  error, keyed by profile name in a `map[string]*profileState` on `PortStatsCollector`. The
  poller goroutine takes the write lock only while swapping in a freshly scraped snapshot;
  `Collect()` takes a read lock per profile to copy out what it needs. This keeps `Collect()`
  non-blocking with respect to an in-flight scrape (it just reads the previous snapshot until
  the new one is swapped in).
- **Poll immediately on startup, then on each tick** (`time.Ticker` doesn't fire until the first
  interval elapses), so `/metrics` has real data shortly after the process starts rather than
  waiting a full `poll_rate_seconds` for the first result.
- **Goroutines are stopped via `context.Context` cancellation**, wired to the existing
  SIGINT/SIGTERM shutdown path in `main()` (which currently just blocks on a signal channel and
  exits) — cancel the context before process exit so pollers stop cleanly rather than relying on
  the OS to reap them.

## Risks / Trade-offs

- [Risk] `/metrics` now serves data that can be up to `poll_rate_seconds` stale, instead of
  always-fresh-at-request-time → Mitigation: this is the explicitly requested behavior change
  (decoupling scrape cadence from Prometheus's request cadence); document it clearly in
  README/CLAUDE.md so operators size `poll_rate_seconds` appropriately.
- [Risk] `exporter_last_scrape_duration_seconds` (previously "how long did scraping all
  profiles take, synchronously, for this request") now measures something almost instantaneous
  (`Collect()` just copies cached data) — its historical meaning changes → Mitigation: acceptable
  given the explicit ask; noted here rather than silently changed, and no new metric is invented
  to compensate since that's out of scope.
- [Risk] Raising the default timeout from 5s to 10s changes behavior for anyone relying on the
  old implicit default without setting `timeout_seconds` → Mitigation: this is an explicit,
  intentional part of the request; call it out as **BREAKING** in the proposal/changelog.
- [Risk] A profile whose switch is unreachable never populates its snapshot, so `Collect()` has
  nothing to emit for it → Mitigation: same as today's behavior for a failed scrape (skip that
  profile's metrics for this cycle, increment `exporter_scrape_errors_total`), just triggered
  from the poller instead of from `Collect()`.
