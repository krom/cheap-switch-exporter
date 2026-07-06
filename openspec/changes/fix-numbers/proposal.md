## Why

Some switch firmware (see `examples/2.html`, `examples/3.html`) reports 64-bit packet/byte
counters as two 32-bit halves in a single table cell, formatted as `"<high>-<low>"` (e.g.
`1-907301261`), which the page's own inline JavaScript recombines client-side as
`high*4294967296 + low` (e.g. `1-907301261` → `5202268557`). The exporter's `parseNum` doesn't
replicate that recombination: it only strips a literal `"0-"` prefix and otherwise hands the
raw `"<high>-<low>"` string to `strconv.ParseFloat`, which fails to parse and silently returns
`0`. As a result, any counter whose high half is non-zero is reported as `0` instead of its
real (potentially large) value — undercounting traffic once a counter grows past 2^32.

## What Changes

- Replace the `"0-"`-prefix special case in `parseNum` with general `"<high>-<low>"` split-counter
  recombination: detect the `digits-digits` pattern, parse both halves, and compute
  `high*4294967296 + low`, matching the switch's own JS logic. Plain numeric strings (no `-`)
  continue to parse as before.
- No changes to table column indices/layout — this only fixes how an individual cell's text is
  converted to a number, for the models where cells use the split-counter format.

## Capabilities

### New Capabilities
- `counter-parsing`: Correctly parsing per-port counter values from switch HTML, including the
  split 32-bit-halves encoding used by some firmware.

### Modified Capabilities
(none — no existing spec covers this yet)

## Impact

- Affected code: `parseNum` in `main.go`, used by `fetchPorts`, `fetchPoE`, and PoE system
  consumption parsing — anywhere a table cell's text is converted to a metric value.
- No config, API, or dependency changes.
