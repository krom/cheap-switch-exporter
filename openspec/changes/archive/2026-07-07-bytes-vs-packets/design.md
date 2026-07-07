## Context

`fetchPorts` (main.go:220) currently reads the `port.cgi?page=stats` table purely positionally:

```go
res = append(res, Port{
    Name:       tds.Eq(0).Text(),
    State:      tds.Eq(1).Text(),
    LinkStatus: tds.Eq(2).Text(),
    TxGood:     parseNum(tds.Eq(3).Text()),
    TxBad:      parseNum(tds.Eq(4).Text()),
    RxGood:     parseNum(tds.Eq(5).Text()),
    RxBad:      parseNum(tds.Eq(6).Text()),
})
```

This assumes every switch's page has exactly the columns `Port, State, Link Status, TxGoodPkt,
TxBadPkt, RxGoodPkt, RxBadPkt` in that order (`examples/4.html`). But `examples/1.html`-`3.html`
show a different, equally real layout: `Port, State, Link Status, TxGoodPkt, RxGoodPkt,
TxGoodBytes, RxGoodBytes` — no bad-packet columns, and two columns are byte counters rather than
packet counters. Reading these positionally would put a byte counter into `RxBad`, which is
wrong both in value and in semantics (packets vs. bytes are different units and shouldn't share
a metric).

The full space of counter columns these switches use is the combination of three independent
choices: direction (`Tx`/`Rx`), quality (`Good`/`Bad`), and unit (`Pkt`/`Bytes`) — 8 possible
kinds, of which each observed switch model reports exactly 4.

## Goals / Non-Goals

**Goals:**
- Identify each stats-table counter column by its header text instead of its position, so any
  subset/order of the 8 `{Tx,Rx}x{Good,Bad}x{Pkt,Bytes}` kinds parses correctly.
- Add metrics for the byte-counter kinds so they're not silently dropped or mismapped.
- Emit only the metrics a given switch's page actually reports — no fabricated zeros for absent
  counters, so dashboards/alerts don't misread "not reported" as "reported as zero".
- Keep exact backward compatibility for the already-supported packet-only layout (same 4 metric
  names, same values).

**Non-Goals:**
- Not changing `fetchPortStatus` (speed/duplex) or `fetchPoE` parsing — those are unaffected by
  this header-detection change.
- Not adding support for counter kinds beyond the 8-combination space (e.g. no evidence of an
  "Errors" or "Discards" column in the current examples).
- Not changing the `parseNum` split-counter recombination logic itself (already handled by the
  `fix-numbers` change) — this change only decides *which* field a parsed value is stored into.

## Decisions

- **Parse the header row instead of skipping it.** Today `doc.Find("table tr").Each` skips row
  `i==0` unconditionally. Change this to read that row's `<th>` cells once per `fetchPorts` call,
  build a `map[int]counterKind` (column index -> recognized kind) using a regex/matcher over the
  8 combinations, and use that map when walking the data rows instead of fixed `tds.Eq(n)`
  indices for counters. `Port`/`State`/`Link Status` columns remain identified by their fixed
  leading position (0, 1, 2), since every observed example keeps them there and unlike the
  counters they aren't part of the combinatorial header.
  - Alternative considered: match by header text position-independently for *all* seven columns,
    including Port/State/Link Status — rejected as unnecessary complexity since no example shows
    those reordered, and it would require inventing header-text matching rules for free-text
    columns that don't follow a combination grammar.
- **Represent parsed counters as a small map/struct with presence tracking**, e.g.
  `Port.Counters map[CounterKind]float64` (or 8 `*float64`/`(value, ok)` pairs), rather than the
  current fixed `TxGood/TxBad/RxGood/RxBad float64` fields — a plain `float64` can't distinguish
  "0 packets" from "not reported by this switch", and the requirement is to omit unreported
  counters entirely, not report them as zero.
- **Add 4 new `*prometheus.Desc` fields** to `PortStatsCollector` for the byte-counter metrics
  (`port_tx_good_bytes`, `port_tx_bad_bytes`, `port_rx_good_bytes`, `port_rx_bad_bytes`), and loop
  over all 8 kinds in `Collect()`, emitting a `prometheus.MustNewConstMetric` only for kinds
  present in that port's parsed counter map.
- **Header matching should be case/whitespace-tolerant but strict on the combination grammar**:
  build the recognized set from the 8 concatenations directly (`TxGoodPkt`, `TxGoodBytes`,
  `TxBadPkt`, `TxBadBytes`, `RxGoodPkt`, `RxGoodBytes`, `RxBadPkt`, `RxBadBytes`) and compare
  against trimmed header text; an unrecognized header column is skipped (not fatal), so a switch
  with some yet-unseen extra column doesn't break parsing of the columns that are understood.

## Risks / Trade-offs

- [Risk] A switch reports a counter kind under different header text than the 8 exact strings
  observed so far (e.g. extra whitespace, different capitalization) → Mitigation: normalize
  header text (trim, case-fold) before matching against the 8 known combinations.
- [Risk] Changing `Port`'s counter representation from fixed fields to a map touches every call
  site that reads `pt.TxGood` etc. in `Collect()` → Mitigation: scope this change to update all
  those call sites in the same PR/commit; there's exactly one consumer (`Collect()`), so this is
  a contained, mechanical update.
- [Risk] Dashboards/alerts built against the 4 existing packet metrics keep working unchanged
  (verified via the packet-only backward-compatibility scenario in the spec); new byte metrics
  are purely additive so existing consumers are unaffected either way.
