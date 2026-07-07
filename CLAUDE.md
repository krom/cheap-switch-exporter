# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Prometheus exporter for budget network switches that don't support SNMP. It scrapes port/PoE
statistics by logging into the switch's web management interface (HTML pages) and parsing the
returned tables with goquery, rather than querying SNMP OIDs.

The exporter is `package main`, split into several files by concern (see Architecture below)
rather than one flat file.

## Commands

```bash
go mod download                 # install dependencies
go run .                        # run directly (reads ./config.yaml)
go build -o cheap-switch-exporter .
go vet ./...                    # static checks
go test ./...                   # unit + fixture-driven tests (*_test.go)

docker build -t cheap-switch-exporter .
docker compose up                # uses compose.yaml, mounts ./config.yaml
```

Tests are split to mirror the source files (`config_test.go`, `defaultclient_test.go`,
`v2client_test.go`, `collector_test.go`, `parse_test.go`, plus shared fixture helpers in
`helpers_test.go`). Pure helpers (`state`/`link`/`speed`/`parseNum`/etc.) are covered with
table-driven tests, and `fetchPorts`/`fetchPortStatus`/`fetchPoE` by pointing `ProfileConfig` at
an `httptest.Server` that serves fixture HTML — real captures for `fetchPorts` under
`examples/*.html`, synthetic fixtures for the other two under `testdata/` (no real captures for
those pages exist yet). When adding coverage for a new switch model, prefer adding/using real
captured HTML the way `examples/*.html` already do.

## Architecture

- **`config.go`** (`RootConfig` / `ProfileConfig` / `loadConfig`): `config.yaml` defines a list
  of named *profiles* under `profiles:`, each with its own switch `address`, credentials,
  `timeout_seconds`, `poe` flag, and a `comments` map (port name -> human label used as a
  Prometheus label). Multiple profiles let one exporter instance poll several physical switches
  at once. `loadConfig` reads and validates `config.yaml` into a `[]NamedProfile`; `main.go`
  just calls it and wires up the collector/HTTP server.

- **`model.go`**: the shared `Port`/`PortStatus`/`PoEPort`/`CounterKind` data types both switch
  dialects populate.

- **`switchclient.go`**: the `switchClient` interface and `newSwitchClient` factory that picks a
  profile's dialect implementation (`defaultclient.go` or `v2client.go`) based on its `profile`
  config value.

- **`defaultclient.go`** — the original dialect. Auth is a cookie whose value is
  `md5(username + password)` (see `md5hex` in `parse.go`) — there's no login POST/session flow,
  just this cookie set on every request (`fetchDoc`). `fetchDoc` does a raw `GET` against a
  `.cgi` endpoint with a `page` query param and the auth cookie, then parses the response as
  HTML via goquery. Three scrapes exist:
  - `fetchPorts` — `/port.cgi?page=stats` — per-port admin state, link status, tx/rx good/bad
    packet counts.
  - `fetchPortStatus` — `/port.cgi?page=status` — per-port negotiated speed and duplex, joined
    into `fetchPorts` results by port name.
  - `fetchPoE` — `/pse_system.cgi` (total consumption) and `/pse_port.cgi?page=stats` (per-port
    PoE state/power/class/watts/voltage/current), only scraped when a profile has `poe: 1`.

  All three parse an HTML `<table>` by iterating `<tr>` rows and indexing `<td>` cells
  positionally — the column order in these functions must match the specific switch firmware's
  page layout. This is the part most likely to need adjustment when supporting a new switch
  model (see the supported-devices table in README.md).

- **`v2client.go`** — the `v2` dialect (e.g. SKS3200-8E2X): a real `/authorize` login instead of
  a static cookie, and JSON port statistics instead of HTML tables. No PoE support.

- **`collector.go`** (`PortStatsCollector`): implements the `prometheus.Collector` interface.
  `Collect()` reads each profile's cached snapshot (`profileState`, kept fresh by the background
  pollers in `poll.go`) rather than scraping live, and emits metrics labeled with `port`,
  `comment`, `switch` (profile name), and `address`. Scrape failures increment
  `exporter_scrape_errors_total` and skip that profile rather than failing the whole `/metrics`
  response. `exporter_last_scrape_duration_seconds` tracks total collect time across all
  profiles.

- **`poll.go`**: one background goroutine per profile (`StartPolling`/`pollProfile`), scraping on
  that profile's resolved `poll_rate_seconds` interval and writing into its `profileState`.

- **`parse.go`**: small stateless string/number parsing helpers shared by both dialects.

- **HTTP server** (`main.go`): `/metrics` via `promhttp.Handler()` on port `8080`, no TLS (see
  README Limitations).

## Adding support for a new switch model

Since parsing is positional (`tds.Eq(n)`), supporting a new switch generally means:
1. Capture the HTML the target switch's `.cgi` pages return for `port.cgi?page=stats`,
   `port.cgi?page=status`, and (if PoE) `pse_system.cgi` / `pse_port.cgi?page=stats`.
2. Compare column order/count against what `fetchPorts`/`fetchPortStatus`/`fetchPoE` expect,
   and adjust the `tds.Eq(...)` indices or `Length()` checks accordingly.
3. Update the supported-devices table in `README.md`.

## `examples/` — sample switch HTML pages

`examples/*.html` holds raw `port.cgi?page=stats` responses captured from a switch model not
yet supported by `fetchPorts`, kept as fixtures for adjusting the parser (and for the
fixture-based test suite proposed in "Commands" above). Notable quirks visible in these
samples, relative to what `fetchPorts` currently assumes:

- **Different column layout**: the table columns are `Port, State, Link Status, TxGoodPkt,
  RxGoodPkt, TxGoodBytes, RxGoodBytes` — 7 columns like `fetchPorts` expects, but there are no
  separate Tx/Rx *bad*-packet columns; the last two columns are byte counters instead. Mapping
  this model onto `Port.TxBad`/`Port.RxBad` isn't a direct fit and needs a decision (e.g. leave
  bad-packet gauges at 0, or add byte-counter metrics).
- **Split 64-bit counters**: `examples/2.html` and `examples/3.html` encode each counter as two
  32-bit halves in a `"<high>-<low>"` string (e.g. `<td id=port0-txgood>1-901525430</td>`), which
  the page's inline `<script>` recombines client-side as `high*4294967296 + low`. Since goquery
  doesn't execute JS, a parser for this model must replicate that recombination itself rather
  than passing the cell text through `parseNum` as-is. `examples/1.html` instead has plain
  already-summed values with no script and no `id` attributes — likely a different
  firmware/page state (all counters reset to 0), useful as the "simple" fixture.
