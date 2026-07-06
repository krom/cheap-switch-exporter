## Context

`parseNum` (main.go:361) is the single conversion point used everywhere a table cell's text
becomes a metric value: `fetchPorts`'s Tx/Rx counters, `fetchPoE`'s watts/voltage/current, and
the PoE system consumption reading. Some switch firmware (`examples/2.html`, `examples/3.html`)
renders 64-bit counters as two decimal 32-bit halves joined by a hyphen, e.g.
`<td id=port0-txgood>1-901525430</td>`, and relies on inline JS to recombine them:

```js
statArr = document.getElementById('port0-txgood').innerHTML.split('-');
document.getElementById('port0-txgood').innerHTML =
    parseInt(statArr[0]*4294967296) + parseInt(statArr[1]);
```

i.e. `value = high*4294967296 + low`, where `4294967296 == 2^32`. Since goquery only reads the
raw HTML (no JS execution), `parseNum` currently receives the unrecombined `"1-901525430"`
string. Today it special-cases stripping a literal `"0-"` prefix (covering only the case where
`high == 0`), then falls through to `strconv.ParseFloat` for anything else — which fails to
parse `"1-901525430"` as a float, the error is discarded, and the function returns `0`. So any
counter whose high half is non-zero silently reports as `0`.

`examples/1.html` (no `id` attributes, plain resolved values) and `examples/4.html` (a
different, already-supported switch model with plain `TxGoodPkt/TxBadPkt/RxGoodPkt/RxBadPkt`
columns) show that most cells are and will remain simple decimal strings — the split-counter
format is specific to certain firmware/pages, not universal.

## Goals / Non-Goals

**Goals:**
- Make `parseNum` correctly recombine `"<high>-<low>"` split counters into their true decimal
  value, for any field that goes through it (not just Tx/RxGood).
- Preserve existing behavior for plain numeric strings (e.g. `"71817"`, `"0"`, empty string).

**Non-Goals:**
- No changes to table column indices/layout, `fetchPorts`/`fetchPoE` structure, or support for
  the different column set seen in `examples/1.html`-`3.html` (missing Tx/RxBad columns,
  `TxGoodBytes`/`RxGoodBytes` instead) — that's a separate new-switch-model change.
- No test suite scaffolding beyond what's needed to verify this fix (full fixture-based test
  suite is tracked separately per CLAUDE.md).

## Decisions

- **Detect the split format structurally, not by prefix.** Replace the `"0-"`-prefix special case
  with a general check: if the (trimmed) string matches `^\d+-\d+$`, split on `-`, parse both
  halves as `uint64` (or `float64`, see below), and compute `high*4294967296 + low`. This
  subsumes the old `"0-"` case (`0*4294967296 + low == low`) as one instance of the general rule,
  rather than keeping it as a separate special path.
  - Alternative considered: keep parsing the whole string as one number after replacing `-` with
    nothing — rejected, since that would silently mis-parse `"1-901525430"` as a huge but wrong
    concatenated integer instead of the correct recombined value.
- **Parse halves as integers, not floats**, before combining (`strconv.ParseUint` or
  `ParseInt`), then convert the combined result to `float64` for the existing `float64` return
  type. Doing the multiply/add in float64 directly risks precision loss for the `high` term
  (`high * 4294967296`) once `high` is large enough that intermediate float math loses integer
  precision — using integer arithmetic for the combination matches the switch's own `parseInt`
  based JS and avoids that.
- **Only reinterpret strings matching the exact `digits-digits` pattern** as split counters;
  anything else (plain digits, empty, non-numeric) falls through to the existing
  `strconv.ParseFloat` path unchanged. This avoids accidentally misinterpreting unrelated
  hyphenated text as a counter.

## Risks / Trade-offs

- [Risk] A legitimately negative number formatted as `"-123"` could look superficially similar to
  the split format → Mitigation: the `^\d+-\d+$` pattern requires digits on *both* sides of the
  hyphen, so a leading `-` (no digits before it) won't match and falls through to
  `ParseFloat` as before.
- [Risk] No live switch available to verify against real traffic → Mitigation: verify using the
  captured fixtures in `examples/2.html`/`examples/3.html`, checking the recombined value matches
  what the page's own inline JS would compute (e.g. `1-901525430` → `5202268557`).
