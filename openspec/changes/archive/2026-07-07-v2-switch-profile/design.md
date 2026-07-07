## Context

The exporter currently supports exactly one switch dialect, hardcoded into free functions
(`fetchDoc`/`fetchPorts`/`fetchPortStatus`/`fetchPoE` in `main.go`): a stateless cookie
(`md5(username+password)`, sent on every request, no login round-trip) and HTML tables
parsed positionally with goquery.

A second device family — confirmed live against a real `SKS3200-8E2X` at 10.110.0.19 — uses
a fundamentally different interface (captured fixtures: `examples/sks3200-8e2x_auth.md`,
`examples/sks3200-8e2x_port_statistics.json`):

- **Auth**: `GET /authorize?loginusr=<md5(username)>&loginpwd=<md5(password)>` (username and
  password hashed *separately*, not concatenated) sets two real session cookies (`user`,
  `session`) returned via `Set-Cookie`. This is a genuine login step, not a precomputed
  stateless cookie.
- **Data**: `GET /port_statistics.json`, real `Content-Type: application/json`, shaped as
  `{"PortNum": "10", "Port_1": {"Port_Id","Port_Status","Link_Status","TxGoodPkt","TxBadPkt","RxGoodPkt","RxBadPkt"}, ..., "Port_N": {...}}`
  — a map with dynamic `Port_N` keys (not a JSON array), one combined call replacing the
  current model's two HTML scrapes (`page=stats` + `page=status`).

Conveniently, `TxGoodPkt`/`TxBadPkt`/`RxGoodPkt`/`RxBadPkt` are the *exact* string values
already used as `CounterKind` constants in `main.go` (`main.go:85-92`), so v2 port counters
drop straight into the existing `Port.Counters map[CounterKind]float64` with no translation
table. `Port_Status` (`"Enabled"`) also matches the existing `state()` helper's
`strings.Contains(s, "Enable")` check.

The one field with no existing equivalent is `Link_Status`, which encodes link state, speed,
and duplex together in one string (`"Link Down"` or `"<number><Mbps|Gbps><Full|Half>"`),
where the current model gets these as three separate already-split values from its
`page=status` HTML table.

The recently-shipped background poller (`scrapeOnce`/`pollProfile`/`StartPolling`,
archived change `poll-rate-seconds`) already isolates "fetch this profile's data" behind a
single per-tick call, which is the natural seam to plug a second dialect into.

## Goals / Non-Goals

**Goals:**
- Support v2-family switches (JSON status endpoint + real login) for port state, link,
  speed, duplex, and Tx/Rx good/bad packet counters — functional parity with what the
  default dialect already reports for non-PoE profiles.
- Select the dialect per profile via the existing `ProfileConfig.Profile` field
  (`yaml:"profile"`), defaulting to today's behavior so existing configs are unaffected.
- Keep the background poller's shape (one `scrapeOnce` call per tick per profile) unchanged;
  only what happens *inside* that call for a v2 profile is new.

**Non-Goals:**
- PoE support for v2 devices. No v2 PoE device has been observed or confirmed to exist;
  designing that fetch path now would be pure speculation. `profile: v2` + `poe: 1` is
  rejected as a config error instead (see Decisions).
- Persisting/reusing v2 session cookies across poll ticks. Confirmed with the user: one
  login per `scrapeOnce` call, session discarded immediately after. No retry-on-expiry
  logic is needed as a result.
- Auto-detecting the dialect (e.g., by probing for a redirect to `login.html`). Confirmed
  with the user: explicit `profile: v2` in config only, for this change.
- Byte counters, VLAN, STP, or any other v2 page beyond port stats (loop protection, IGMP
  snooping, etc. exist in the v2 UI but are out of scope — same "just port/PoE metrics"
  scope the exporter already has for the default dialect).

## Decisions

### 1. `switchClient` interface, chosen once per profile

```go
type switchClient interface {
    FetchPorts() ([]Port, []PortStatus, error)
    FetchPoE() (float64, []PoEPort, error)
}
```

`scrapeOnce` resolves a profile's client once (based on `profile.Config.Profile`) and calls
it instead of the free `fetchPorts`/`fetchPortStatus`/`fetchPoE` functions directly. The
existing dialect becomes `defaultClient{cfg}` wrapping today's logic unchanged; the new
dialect is `v2Client{cfg}`.

**Alternative considered**: keep free functions and `if cfg.Profile == "v2" { ... }` inside
each of them. Rejected — the two dialects don't share fetch mechanics (HTML scrape vs. JSON,
stateless cookie vs. login), so branching inside shared functions would mostly be an
if/else fork with no shared body, while gaining nothing over two small types satisfying one
interface. The interface also makes "which dialect does this profile use" a single
resolution point instead of N scattered checks.

### 2. `Profile` field semantics: `""`/`"default"` vs `"v2"`, rejected otherwise

`ProfileConfig.Profile` (already declared, currently unread) becomes the dialect selector.
Resolved once alongside `Timeout`/`PollRateSeconds` in `main()`'s existing per-profile
resolution loop. Any value other than `""`, `"default"`, or `"v2"` is a fatal config error
at startup (fail fast, matching the repo's existing style of `log.Fatal` on bad config
rather than silently ignoring it).

### 3. One login per poll tick for v2, no session caching

`v2Client.FetchPorts()` performs the full `/authorize` → cookies → `/port_statistics.json`
sequence every time it's called (i.e., every poll tick, per the user's explicit choice).
This is simpler than the alternative (cache `user`/`session` cookies on the client, retry
login on a redirect-to-login-page) and acceptable because v2 sessions are short-lived
anyway — caching would mostly just move where the login call happens, not avoid it.

**Trade-off accepted**: every poll tick against a v2 profile is two HTTP round-trips
(login + fetch) instead of one. At the poll intervals this exporter targets (tens of
seconds, per the `poll-rate-seconds` change), this is negligible.

### 4. `Link_Status` parsing

New helper, e.g. `parseLinkStatus(s string) (linkUp bool, speedMbps float64, fullDuplex bool)`:
- `"Link Down"` → `(false, 0, false)`.
- Otherwise, regex `^(\d+)(Mbps|Gbps)(Full|Half)$` → `linkUp = true`, `duplex = (dup == "Full")`,
  `speedMbps = number * (1 if Mbps else 1000)`.

Confirmed live: `1000MbpsFull`, `2500MbpsFull`, `100MbpsFull`, `10GbpsFull`, `Link Down`.
**Unconfirmed** (pattern-inferred, not observed): `Half` duplex, and `10Mbps`/`5000Mbps`
speeds. See Open Questions.

### 5. `poe: 1` + `profile: v2` is a config error

Since no v2 PoE endpoint has been captured or confirmed to exist, silently ignoring
`poe: 1` on a v2 profile would produce a config that looks like it's collecting PoE metrics
but isn't — a silent gap. Failing fast at startup surfaces the mismatch immediately instead.

**Alternative considered**: silently skip PoE fetch for v2 profiles regardless of the `poe`
flag. Rejected as more surprising than an explicit error for the reason above.

### 6. JSON parsing via `encoding/json` into a typed struct, not `map[string]any`

The `Port_N` dynamic-key shape (`Port_1`, `Port_2`, ... `Port_N` as sibling object keys
rather than array elements) doesn't map onto a single struct with `encoding/json` without
either a custom `UnmarshalJSON` or a generic `map[string]json.RawMessage` pass, since the
key set depends on `PortNum`. Plan: unmarshal into
`map[string]json.RawMessage` (or `map[string]struct{...}`), read `PortNum` for the count
(or just iterate all keys matching a `Port_` prefix, sorted numerically by the trailing
`Port_Id` field or by natural key sort), and unmarshal each matched value into a
`v2PortJSON` struct with the 5 needed fields (`Port_Id`, `Port_Status`, `Link_Status`,
`TxGoodPkt`, `TxBadPkt`, `RxGoodPkt`, `RxBadPkt`).

## Risks / Trade-offs

- **[Risk]** `Link_Status` regex is derived from a single live device's currently-observed
  link states, missing Half-duplex and 10M/5000M speed samples → **Mitigation**: implement
  the regex to accept the full documented pattern space now (best effort), but flag in the
  PR description that Half-duplex/10M/5000M are unverified; adjust from real fixtures if a
  differently-negotiated port is ever captured (same fixture-driven-adjustment pattern the
  repo already uses for other switch models).
- **[Risk]** Doubling HTTP calls per v2 poll tick (login + fetch) increases load on the
  switch's management CPU compared to the default dialect's single-request-per-scrape model
  → **Mitigation**: acceptable at typical poll intervals (tens of seconds+); revisit if a
  future user reports switch-side login-rate issues.
- **[Risk]** `PortNum` might not always equal the number of `Port_N` keys present (e.g. a
  device with gaps) → **Mitigation**: iterate the actual `Port_` prefixed keys present in
  the response rather than trusting `PortNum` as a loop bound.

## Migration Plan

Additive only — no existing config or metric changes. Deploy as a normal release; profiles
without a `profile` field (or with `profile: default`) are byte-for-byte unaffected. No
rollback concerns beyond the usual "revert the release."

## Open Questions

- **`poe: 1` + `profile: v2` — hard config error, or silent no-op?** This design proposes a
  hard error (Decision 5). Flagging for explicit confirmation before implementation, since
  it's the one behavior choice not already discussed with the user.
- **Unverified `Link_Status` formats** (Half duplex, 10Mbps/5000Mbps): no live fixture exists
  for these. Should the parser reject unrecognized formats loudly (metric omitted + logged
  warning) or fail closed some other way? Proposal: log a warning and skip emitting
  link/speed/duplex for that port (but still emit its counters), rather than failing the
  whole scrape over one unparseable port.
- Are there other v2-family models, and do any of them expose PoE? Unknown — the user has
  only ever seen this one device on the v2 interface. Left entirely out of scope per
  Non-Goals; a future change would need its own fixture capture if a PoE-capable v2 device
  turns up.
