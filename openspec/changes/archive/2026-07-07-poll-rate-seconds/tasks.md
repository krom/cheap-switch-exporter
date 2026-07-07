## 1. Config: settings block and precedence resolution

- [x] 1.1 Add `GlobalSettings` struct (`PollRateSeconds`, `TimeoutSeconds`) and a `Settings
      GlobalSettings` field on `RootConfig` (`yaml:"settings"`).
- [x] 1.2 Add `PollRateSeconds int` (`yaml:"poll_rate_seconds"`) to `ProfileConfig`.
- [x] 1.3 In `main()`, extend the existing per-profile resolution loop to resolve both
      `Timeout` (profile → `settings.timeout_seconds` → `10`) and `PollRateSeconds`
      (profile → `settings.poll_rate_seconds` → `60`), replacing the old hardcoded `5`.

## 2. In-memory snapshot cache

- [x] 2.1 Add a `profileState` struct holding the last scraped `[]Port`, `[]PortStatus`, PoE
      consumption, `[]PoEPort`, and last error, guarded by `sync.RWMutex`.
- [x] 2.2 Add `states map[string]*profileState` to `PortStatsCollector`, keyed by profile name,
      initialized in `NewCollector`.

## 3. Background poller

- [x] 3.1 Add a `scrapeOnce(profile NamedProfile, state *profileState, errors
      prometheus.Counter)` function that calls `fetchPorts`/`fetchPortStatus`/(`fetchPoE` if
      `PoE == 1`) and swaps the results into `state` under its write lock, incrementing
      `errors` on `fetchPorts` failure.
- [x] 3.2 Add a `pollProfile(ctx context.Context, profile NamedProfile, state *profileState,
      interval time.Duration, errors prometheus.Counter)` function: calls `scrapeOnce`
      immediately, then on every `time.Ticker` tick at `interval`, until `ctx.Done()`.
- [x] 3.3 Add `(c *PortStatsCollector) StartPolling(ctx context.Context)` that launches one
      `pollProfile` goroutine per profile using its resolved `PollRateSeconds`.

## 4. Collect() reads from cache

- [x] 4.1 Replace `Collect()`'s calls to `fetchPorts`/`fetchPortStatus`/`fetchPoE` with reads
      from `c.states[p.Name]` under its read lock, preserving the existing metric emission
      logic (including the per-counter-kind conditional emission from `bytes-vs-packets`).
- [x] 4.2 Keep `exporter_scrape_errors_total`/`exporter_last_scrape_duration_seconds` on the
      collector, incrementing/setting them from the poller and `Collect()` respectively (per the
      design's documented shift in what each now measures).

## 5. Wire up shutdown and verify

- [x] 5.1 In `main()`, create a `context.Context` cancelled on the existing SIGINT/SIGTERM
      signal, call `collector.StartPolling(ctx)` after registering the collector, and cancel the
      context before exiting.
- [x] 5.2 Add/extend tests: resolution precedence for `poll_rate_seconds`/`timeout_seconds`
      (profile / settings / hardcoded default), and `Collect()` serving cached data without
      triggering a new HTTP request (e.g. an httptest server that fails the test if hit during
      `Collect()`, only expecting hits from an explicit `scrapeOnce`/poll call).
- [x] 5.3 Update `config.yaml.example` (already has a draft `settings:` block — confirm it
      matches the final field names) and README's Configuration section to document `settings:`
      and per-profile `poll_rate_seconds`.
- [x] 5.4 Run `go vet ./...`, `go test ./...`, and `go build -o cheap-switch-exporter .` to
      confirm no regressions.
