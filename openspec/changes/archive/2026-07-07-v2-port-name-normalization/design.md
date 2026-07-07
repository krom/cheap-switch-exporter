## Context

The default (HTML) dialect names ports by the exact text of the first `<td>` cell in the
switch's stats table, which for every supported model reads `"Port N"`. The v2 (JSON) dialect
instead used the bare `Port_Id` field from `/port_statistics.json` (e.g. `"1"`) directly as the
port name. Since `comments` config and the `port`/`comment` metric labels are keyed by this
name, and users write `comments` the same way regardless of dialect (following the README's
`Port 1: <label>` convention), v2 profiles never matched their configured comments.

## Goals / Non-Goals

**Goals:**
- Make v2 port names consistent with the default dialect's `"Port N"` format so `comments`
  config behaves identically across dialects.

**Non-Goals:**
- Changing how the default (HTML) dialect names ports.
- Introducing a general port-name-normalization/aliasing layer (e.g. accepting either `"1"` or
  `"Port 1"` as a comments key) — a single canonical format is simpler and matches existing
  documented behavior.

## Decisions

- Format the v2 port name as `"Port " + p.PortId` at the point `Port{}` is constructed in
  `v2Client.FetchPorts`, rather than normalizing later (e.g. in the collector). This keeps the
  name canonical everywhere it's used (counters, statuses, comments lookup) with a one-line
  change.
  - Alternative considered: normalize/strip prefixes when looking up `comments` in the
    collector, accepting both `"1"` and `"Port 1"`. Rejected as unnecessary complexity — no
    switch model prints bare numbers in a user-facing state, so there's no reason to support
    two config formats.

## Risks / Trade-offs

- **Existing v2 deployments with `comments` keyed on bare numbers will stop matching** →
  Mitigated by documenting the new expected format in the README; this is a config-only fix
  users must apply (change `1: Uplink` to `Port 1: Uplink`).
