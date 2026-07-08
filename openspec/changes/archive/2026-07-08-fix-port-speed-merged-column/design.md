## Context

`fetchPortStatus` scrapes `port.cgi?page=status` and assumes one fixed column layout. Two real
switch models on the "default" dialect turn out to have different layouts for this specific page:

- SKS3200-5E1X (`desk_switch`, 10.110.0.18): 8 `<td>`s per row — `Port, State, Duplex-Config,
  Duplex-Actual, Speed-Config, Speed-Actual, Flow-Config, Flow-Actual`. Matches the code today.
- SKS3200-8E1X-P (`poe_switch`, 10.110.0.17): 6 `<td>`s per row — `Port, State,
  SpeedDuplex-Config, SpeedDuplex-Actual, Flow-Config, Flow-Actual`. Speed and Duplex share one
  merged "Actual" cell (e.g. `"1000Full"`, `"10GFull"`, `"Link Down"`).

Both are `profile: default` — there is no config-level signal distinguishing them, and per the
user's constraint we're not introducing one. Column count is the only reliable signal available
from the parsed table itself.

## Goals / Non-Goals

**Goals:**
- Correctly report `SpeedMbps` and `FullDuplex` for both layouts without regressing the other.
- Keep the fix inside `fetchPortStatus`/`parse.go` — no new `profile:` value, no changes to
  `switchclient.go` dispatch.

**Non-Goals:**
- Handling hypothetical third layouts (e.g. different colspan arrangements) — add branches then,
  following this same pattern, if/when a real device needs it.
- Header-driven column detection (as `fetchPorts` does for counters). The status page's headers
  use `colspan`/`rowspan` across two `<tr>`s, which goquery flattens in a way that doesn't cleanly
  map to per-column labels the way `fetchPorts`' single-header-row table does. Column count is
  simpler and sufficient for the two known layouts.

## Decisions

**Header-driven column detection, not raw column count.** Both real devices were re-inspected
specifically for their header structure (see `examples/sks3200-8e1x-p_port_status.html` for the
6-column device; the 8-column device's page — not yet saved as an example — has the same shape
with an extra group). Both put the real per-port data in the *last* `<table>` in the document
(the earlier tables are per-port config-select forms), whose first `<tr>` is a group-header row
with `colspan` attributes:

- 8-column device: `Port`(rowspan2), `State`(rowspan2), `Duplex Mode`(colspan2), `Negotiation
  Speed`(colspan2), `Flow Control`(colspan2) — Duplex and Speed are separate groups.
- 6-column device: `Port`(rowspan2), `State`(rowspan2), `Speed/Duplex`(colspan2), `Flow
  Control`(colspan2) — Duplex and Speed are one merged group.

Each colspan-2 group's second column (in the following header row) is always `"Actual
Status"`/`"Actual"` — i.e. the live negotiated value, as opposed to the first column,
`"Config Attribute"`/`"Config"` — the configured target. So the exporter can walk the group-header
`<th>` row once, track a running column offset, and record which data-column index holds
`Duplex-Actual`, `Speed-Actual`, or the merged `SpeedDuplex-Actual`, based purely on group label
text (`"Duplex Mode"`, `"Negotiation Speed"`, `"Speed/Duplex"`) rather than a hardcoded index or a
raw `<td>` count. This is the same style `fetchPorts` already uses for counter columns
(`columnKinds` keyed by header text) — extending that pattern here instead of introducing a
different, count-based mechanism keeps the two scrapers consistent and makes the code self-
documenting: the column mapping is derived from what the switch itself labels each column as.
Superseded alternative: dispatching on `tds.Length()` (8 vs 6) — works for the two known devices,
but is an incidental signal (a future layout could coincidentally also have 6 or 8 columns with
different meaning) and doesn't explain itself the way header text does.

**New `parseSpeedDuplex` helper in `parse.go`**, used only for the merged-group case; the existing
`speed()`/`duplex()` helpers are reused unchanged for the separate-group case. Combined-cell
formats seen live: `"1000Full"`, `"2500Full"`, `"10GFull"`, and by extrapolation from the switch's
own config `<select>` options (`10M/Half`, `10M/Full`, `100M/Half`, `100M/Full`, `1000M/Full`,
`2500M/Full`, `10G/Full`) `"10Half"`, `"100Half"`, `"100Full"` etc., plus the literal `"Link
Down"`. Rule: digits, optional `G` suffix (meaning the number is Gbps, multiply by 1000 for Mbps;
no suffix means the number is already Mbps), then `Full`/`Half`. Unrecognized strings (including
`"Link Down"`) yield speed 0; `"Link Down"` also yields duplex 0 (Half), matching how `duplex()`
already treats non-"Full" strings.

**Fixtures: real captures, kept under `examples/`.** Per CLAUDE.md, `examples/*.html` holds real
device captures and is preferred over synthetic fixtures when available ("When adding coverage
for a new switch model, prefer adding/using real captured HTML the way `examples/*.html` already
do."). Added `examples/sks3200-8e1x-p_port_status.html` (the merged-group `port.cgi?page=status`
capture, 9 ports including a `Link Down` row and a `10GFull` row) and
`examples/sks3200-8e1x-p_pse_port_stats.html` (a `pse_port.cgi?page=stats` capture, since no PoE
example existed yet either). `defaultclient_test.go` reads both directly from `examples/`, the
same way the existing `fetchPorts` tests already read `examples/1.html`-`4.html`. The existing
synthetic `testdata/port_status.html` (8-column, separate-group layout) is left in place as-is —
it isn't being replaced, just supplemented.

## Risks / Trade-offs

- **Header labels are literal string matches** (`"Duplex Mode"`, `"Negotiation Speed"`,
  `"Speed/Duplex"`, and `"Actual Status"`/`"Actual"` for the sub-header) → a firmware variant that
  phrases these differently (e.g. different capitalization or wording) would fail to match and
  fall back to "column not found" (see below), not a wrong value. Mitigation: match sub-header
  text by prefix/`Contains("Actual")` rather than exact string, so `"Actual Status"` and `"Actual"`
  both match without listing every wording variant.
- **No group header matched at all** (neither `"Duplex Mode"`+`"Negotiation Speed"` nor
  `"Speed/Duplex"` found) → previously this silently produced `SpeedMbps: 0`; with header-driven
  detection it does the same (column index not found → value omitted/zero), so behavior for a
  wholly new, unrecognized layout doesn't regress, it just doesn't get speed data until someone
  adds a mapping for it — consistent with how `fetchPorts` already treats unrecognized counter
  headers.
- **`parseSpeedDuplex` format list is inferred from the `<select>` options, not exhaustively
  observed live** (only `1000Full`, `2500Full`, `10GFull`, `Link Down` were actually captured).
  Mitigation: regex handles the general pattern rather than an enum of exact strings, and falls
  back to speed 0 for anything unrecognized rather than panicking or misparsing.
