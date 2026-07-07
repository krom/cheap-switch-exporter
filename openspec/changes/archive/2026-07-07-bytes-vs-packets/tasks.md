## 1. Model counter kinds

- [x] 1.1 Define a `CounterKind` type enumerating the 8 combinations (`TxGoodPkt`, `TxBadPkt`,
      `TxGoodBytes`, `TxBadBytes`, `RxGoodPkt`, `RxBadPkt`, `RxGoodBytes`, `RxBadBytes`) and a
      lookup from normalized header text (trimmed) to `CounterKind`.
- [x] 1.2 Replace `Port`'s fixed `TxGood/TxBad/RxGood/RxBad float64` fields with a
      `Counters map[CounterKind]float64` (present-only entries).

## 2. Header-driven parsing in fetchPorts

- [x] 2.1 In `fetchPorts`, read the header row's `<th>` cells (currently skipped via `i==0`) and
      build a `map[int]CounterKind` from column index to recognized counter kind, skipping
      unrecognized header columns.
- [x] 2.2 For each data row, populate `Port.Counters` using that column-index map (via
      `parseNum`) instead of the fixed `tds.Eq(3..6)` indices; keep `Name`/`State`/`LinkStatus`
      read from columns 0/1/2 as today.

## 3. Collector: new metrics + conditional emission

- [x] 3.1 Add `*prometheus.Desc` fields for `port_tx_good_bytes`, `port_tx_bad_bytes`,
      `port_rx_good_bytes`, `port_rx_bad_bytes` in `PortStatsCollector`/`NewCollector`, alongside
      the existing 4 packet-counter descs.
- [x] 3.2 In `Collect()`, replace the 4 fixed `MustNewConstMetric` calls for Tx/RxGood/Bad with a
      loop over the 8 `CounterKind`s that emits a metric only when `pt.Counters` has an entry for
      that kind.

## 4. Verify

- [x] 4.1 Using `examples/4.html` (packet-only header), confirm exactly `port_tx_good_pkt`,
      `port_tx_bad_pkt`, `port_rx_good_pkt`, `port_rx_bad_pkt` are produced per port, with the
      same values as before this change.
- [x] 4.2 Using `examples/1.html` (or `2.html`/`3.html`) header (`TxGoodPkt, RxGoodPkt,
      TxGoodBytes, RxGoodBytes`), confirm `port_tx_bad_pkt`/`port_rx_bad_pkt` are absent and
      `port_tx_good_pkt`, `port_rx_good_pkt`, `port_tx_good_bytes`, `port_rx_good_bytes` are
      present with correct values.
- [x] 4.3 Run `go vet ./...` and `go build -o cheap-switch-exporter .` to confirm no regressions.
- [x] 4.4 Update the metrics list in `README.md` to include the new byte-counter metrics.
