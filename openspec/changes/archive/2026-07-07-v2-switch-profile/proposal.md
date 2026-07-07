## Why

All switch support so far assumes one specific vendor's web UI: a stateless precomputed
auth cookie and HTML tables scraped positionally with goquery. A second device family
(`SKS3200-8E2X`, confirmed live at 10.110.0.19, likely representative of other switches
using the same vendor firmware) exposes a completely different interface — a real login
call that establishes a session, and a JSON status endpoint instead of HTML tables. The
exporter currently has no way to talk to this family at all; profiles pointed at one of
these switches fail to scrape.

## What Changes

- Add a per-profile switch dialect selector, reusing the existing but currently-unread
  `ProfileConfig.Profile` (`yaml:"profile"`) field: unset or `"default"` keeps today's
  behavior; `"v2"` selects a new client implementation for this device family.
- Introduce a `switchClient` abstraction with two implementations — the existing
  cookie+HTML-scrape logic (renamed/wrapped as the default client) and a new v2 JSON
  client — selected once per profile and used by the existing background poller
  (`scrapeOnce`/`pollProfile`).
- New v2 client behavior:
  - Logs in via `GET /authorize?loginusr=<md5(username)>&loginpwd=<md5(password)>` once
    per poll tick (not cached/reused across ticks — v2 sessions are short-lived and a
    fresh login per tick is simpler than tracking expiry).
  - Fetches `GET /port_statistics.json` using the session cookies from that login and
    parses the JSON directly (no goquery/HTML parsing needed for this family).
  - Parses the combined `Link_Status` field (e.g. `"1000MbpsFull"`, `"10GbpsFull"`,
    `"Link Down"`) into the existing link/speed/duplex metrics.
- `poe: 1` on a `profile: v2` entry is rejected as a config error at startup — no v2
  device with PoE has been observed, and this change does not attempt to design a v2 PoE
  fetch path speculatively (see design.md open questions).
- **BREAKING**: none for existing configs — `profile` defaulted to unread before, so
  omitting it (or leaving `profile: default`) preserves current behavior exactly.

## Capabilities

### New Capabilities
- `v2-switch-client`: Auth (login + session cookies) and port/counter data retrieval for
  the v2 JSON-based device family, including `Link_Status` parsing.

### Modified Capabilities
- `config-loading`: `ProfileConfig.Profile` gains defined semantics (`""`/`"default"` vs
  `"v2"`) as the switch-dialect selector, with validation of unknown values and of
  `poe: 1` combined with `profile: v2`.

## Impact

- `main.go`: new `switchClient` interface, a `v2Client` implementation (login + JSON
  fetch + `Link_Status` parsing), config validation for `Profile`/`PoE` combination, and
  wiring the poller (`scrapeOnce`) to pick a client per profile instead of calling the
  free `fetchPorts`/`fetchPortStatus`/`fetchPoE` functions directly.
- `config.yaml.example`, `README.md`: document `profile: v2` and its constraints.
- `examples/`: real fixtures already captured (`sks3200-8e2x_port_statistics.json`,
  `sks3200-8e2x_auth.md`) for use in tests.
- No changes to metric names or existing default-profile behavior.
