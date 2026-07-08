# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.1] - 2026-07-08

### Fixed

- `fetchPortStatus` no longer reports `port_link_speed` (and, incidentally, `port_link_full_duplex`)
  as `0` for switch models that merge Speed and Duplex into a single `port.cgi?page=status`
  column (e.g. SKS3200-8E1X-P). Previously, the parser assumed a fixed 8-column table layout and
  read an unrelated Flow Control column as if it were Speed; it now detects the table's actual
  group-header layout (`Duplex Mode` + `Negotiation Speed` vs. a merged `Speed/Duplex` group) and
  parses whichever column really holds the negotiated value, the same way `fetchPorts` already
  detects counter columns by header text instead of a hardcoded position.

### Added

- New parsing helper `parseSpeedDuplex` for combined Speed/Duplex cell values (e.g. `"1000Full"`,
  `"10GFull"`, `"Link Down"`).
- Real captured HTML fixtures for `fetchPortStatus`/PoE parsing: `examples/sks3200-5e1x_port_status.html`,
  `examples/sks3200-8e1x-p_port_status.html`, `examples/sks3200-8e1x-p_pse_port_stats.html`,
  replacing a synthetic, never-verified-against-a-real-device fixture.

## [1.0.0] - 2026-07-07

### Added

- Multi-profile configuration: `config.yaml` now defines a `profiles:` list, each with its own
  switch address, credentials, `poe` flag, `comments` labels, and per-profile `poll_rate_seconds`/
  `timeout_seconds` overrides, plus a top-level `settings:` block for global defaults.
- Background polling: each profile is now scraped on its own repeating interval by a dedicated
  goroutine, with `/metrics` served from an in-memory cached snapshot instead of scraping
  synchronously on every request. **BREAKING**: replaces the previous request-driven scrape model.
- Support for a second switch dialect (`profile: v2`, e.g. SKS3200-8E2X): real session-based login
  and JSON port statistics, instead of the cookie-based HTML scraping used by the default dialect.
  PoE is not supported on this dialect.
- Header-driven counter column detection for `fetchPorts`: counter columns on
  `port.cgi?page=stats` are now identified by their `<th>` text instead of a fixed position,
  correctly supporting switch firmware that reports byte counters (`TxGoodBytes`/`RxGoodBytes`)
  in addition to, or instead of, packet counters.
- Split 32-bit-half counter recombination in `parseNum`, so counters reported by some firmware as
  `"<high>-<low>"` pairs are combined into their real 64-bit value instead of silently parsing as `0`
  once the high half is non-zero.
- GitHub Actions workflow to build and publish a Docker image to GHCR
  (`ghcr.io/krom/cheap-switch-exporter`) on pushed version tags, with semantic-version tagging.
- First automated test suite: table-driven unit tests for the parsing helpers, plus
  fixture-driven tests for `fetchPorts`/`fetchPortStatus`/`fetchPoE` against real captured and
  synthetic switch HTML.
- Documentation for additional supported switch models and dialect screenshots in `README.md`.

### Changed

- v2 dialect port names normalized to `"Port N"` to match the default dialect's naming, so
  `comments` config keys apply consistently across both dialects.
- Split the single `main.go` into files grouped by concern (`config.go`, `model.go`,
  `switchclient.go`, `defaultclient.go`, `v2client.go`, `collector.go`, `poll.go`, `parse.go`),
  with no change in externally observable behavior.
- `config.yaml.example` and the README's Configuration section rewritten to match the
  multi-profile format actually implemented by `ProfileConfig`.
