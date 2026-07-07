## Purpose

Defines how the exporter polls each configured switch profile in the background on its own
resolved interval, caching the latest snapshot so `/metrics` requests are served without
performing a live scrape.

## Requirements

### Requirement: Per-profile background polling goroutine
The exporter SHALL start one background goroutine per configured profile, each independently
scraping that profile's switch (ports, port status, and PoE data if enabled) on a repeating
interval equal to that profile's resolved `poll_rate_seconds`, rather than scraping in response
to an incoming `/metrics` request.

#### Scenario: Independent intervals per profile
- **WHEN** profile A resolves to a 15 second poll rate and profile B resolves to a 60 second
  poll rate
- **THEN** profile A's switch is scraped roughly every 15 seconds and profile B's roughly every
  60 seconds, independently of each other and of any `/metrics` request

### Requirement: In-memory snapshot cache
The exporter SHALL store each profile's most recently successfully scraped data in memory,
safe for concurrent access, and SHALL NOT perform a live scrape while handling a `/metrics`
request.

#### Scenario: /metrics served from cache
- **WHEN** a `/metrics` request arrives between two poll ticks for a profile
- **THEN** the response reflects that profile's last successfully scraped snapshot, and no new
  HTTP request is made to that profile's switch as part of handling the `/metrics` request

### Requirement: Resolved timeout bounds each poll request
Each profile's background poll SHALL use that profile's resolved `timeout_seconds` value as the
HTTP client timeout for requests to its switch.

#### Scenario: Slow switch times out without blocking other profiles
- **WHEN** a profile's switch does not respond within its resolved `timeout_seconds`
- **THEN** that profile's poll attempt fails (incrementing `exporter_scrape_errors_total`) without
  blocking or delaying any other profile's poll goroutine

### Requirement: Clean shutdown of poller goroutines
On receiving SIGINT/SIGTERM, the exporter SHALL signal all per-profile poller goroutines to stop
before the process exits.

#### Scenario: Graceful shutdown
- **WHEN** the exporter process receives SIGINT
- **THEN** all background poller goroutines observe a cancellation signal and stop before the
  process exits
