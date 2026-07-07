## Why

`config.yaml.example` now has a `poll_rate_seconds` field (both globally under a new `settings:`
block and, per the intended design, overridable per-profile), but `main.go` never reads or acts
on it — `RootConfig` has no `Settings` field at all. Today's scrape model is also
request-driven, not poll-driven: per CLAUDE.md, "there's no caching/poll-loop — Prometheus's
scrape interval drives the polling", meaning every `/metrics` request synchronously re-scrapes
every configured switch. That couples the exporter's response latency (and the switches' load)
directly to whatever interval Prometheus happens to scrape at, and makes `poll_rate_seconds`
meaningless since nothing uses it. Separately, `timeout_seconds` is read per-profile but has no
global default/override path and a low hardcoded fallback (5s) with no documented precedence.

## What Changes

- **BREAKING**: Add a top-level `settings:` block to the config format
  (`RootConfig.Settings`) with `poll_rate_seconds` and `timeout_seconds`. Add
  `ProfileConfig.PollRateSeconds` (`poll_rate_seconds`) so a profile can override the global
  setting.
- Resolve both settings with the same precedence at startup: profile-specific value, if set →
  else global `settings.<field>`, if set → else a hardcoded default (`60` seconds for poll rate,
  `10` seconds for timeout — **BREAKING**: raises the previous undocumented 5s timeout default
  to 10s).
- **BREAKING**: Replace request-driven scraping with background polling: each profile gets its
  own goroutine that scrapes that switch every `poll_rate_seconds` (its own ticker/interval,
  independent of other profiles), storing the latest parsed results in an in-memory,
  mutex-protected snapshot. `/metrics` requests read from these snapshots instead of triggering
  a live scrape, so response time no longer depends on switch latency.
- `timeout_seconds` (resolved per the precedence above) continues to bound each individual HTTP
  request to a switch, now used by the background poller instead of by a request-time scrape.

## Capabilities

### New Capabilities
- `config-loading` (extends the config shape introduced by the earlier `update-to-config`
  change): adds the `settings:` block and per-profile `poll_rate_seconds`, and documents the
  resolution precedence for both `poll_rate_seconds` and `timeout_seconds`.
- `background-polling`: each profile is scraped on its own interval by a dedicated background
  goroutine, with results cached in memory and served on `/metrics` without a live scrape.

### Modified Capabilities
(none — no capability has been archived yet for this repo, so there's nothing on disk to diff
against; `config-loading` above supersedes the shape from `update-to-config` in place)

## Impact

- Affected code: `RootConfig`, `ProfileConfig`, `main()` (config resolution + goroutine
  startup), `PortStatsCollector`/`Collect()` (read from cache instead of scraping).
- Behavior change: `/metrics` no longer reflects a scrape taken at request time — it reflects
  the most recent background poll, which may be up to `poll_rate_seconds` old. Switches are now
  polled at a fixed cadence regardless of how often `/metrics` is scraped.
- Config format: `config.yaml.example`/README need a `settings:` block documented (follow-up
  docs task, not blocking this change's core behavior).
- No new external dependencies (uses stdlib `context`/`time`/`sync`).
