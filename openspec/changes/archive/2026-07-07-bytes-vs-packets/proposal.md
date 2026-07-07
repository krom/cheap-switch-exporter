## Why

`fetchPorts` reads the `port.cgi?page=stats` table purely by column position
(`tds.Eq(3)`..`tds.Eq(6)`), always interpreting those four columns as
TxGood/TxBad/RxGood/RxBad **packet** counts. But switch firmware varies which four counter
columns it actually reports: `examples/4.html` has `TxGoodPkt, TxBadPkt, RxGoodPkt, RxBadPkt`
(matching today's assumption), while `examples/1.html`-`3.html` instead have
`TxGoodPkt, RxGoodPkt, TxGoodBytes, RxGoodBytes` — no bad-packet columns at all, and two of the
four columns are **byte** counters, not packet counters. Parsing the latter positionally with
today's code mislabels byte counts as `RxBad`/mixes up columns, producing wrong metrics instead
of just missing ones.

## What Changes

- Parse the `port.cgi?page=stats` table header row (`<th>`) instead of assuming a fixed column
  order, matching each counter column against the full combination space
  `{Tx,Rx} x {Good,Bad} x {Pkt,Bytes}` (8 possible counter kinds).
- Add Prometheus metrics for the byte-counter kinds that don't exist today:
  `port_tx_good_bytes`, `port_tx_bad_bytes`, `port_rx_good_bytes`, `port_rx_bad_bytes`, alongside
  the existing `port_tx_good_pkt`, `port_tx_bad_pkt`, `port_rx_good_pkt`, `port_rx_bad_pkt`.
- For a given switch/profile, only emit the metrics whose corresponding column header was
  actually present on that switch's page — a counter kind absent from the header SHALL NOT be
  emitted (not even as `0`), rather than guessed from column position.
- No breaking change for already-supported switches: a page with `TxGoodPkt, TxBadPkt,
  RxGoodPkt, RxBadPkt` headers continues to emit exactly the same 4 metrics as before.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `counter-parsing`: extends counter parsing (previously just the split 64-bit-halves value
  format from `fix-numbers`) to also cover *which* counter a column represents, derived from its
  header text rather than a fixed position, and to support byte-counter kinds in addition to
  packet-counter kinds.

## Impact

- Affected code: `fetchPorts`, `Port` struct, `PortStatsCollector` (new metric descriptors),
  `Collect()` (conditional emission per profile based on which counters that switch reports).
- New metrics: `port_tx_good_bytes`, `port_tx_bad_bytes`, `port_rx_good_bytes`,
  `port_rx_bad_bytes`. Existing metric names/semantics for already-supported switches are
  unchanged.
- No config changes.
