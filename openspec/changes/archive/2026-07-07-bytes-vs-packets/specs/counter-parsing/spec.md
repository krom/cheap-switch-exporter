## ADDED Requirements

### Requirement: Header-driven counter column detection
The exporter SHALL determine which counter each data column in the `port.cgi?page=stats` table
represents by reading that column's header (`<th>`) text, rather than assuming a fixed column
position. A header SHALL be recognized as a counter column when its text matches one of the 8
combinations of `{Tx, Rx} x {Good, Bad} x {Pkt, Bytes}` (e.g. `TxGoodPkt`, `RxBadBytes`).
Non-counter header columns (`Port`, `State`, `Link Status`) SHALL continue to be identified by
position/name as today.

#### Scenario: Packet-only counters (currently supported layout)
- **WHEN** the table header is `Port, State, Link Status, TxGoodPkt, TxBadPkt, RxGoodPkt, RxBadPkt`
- **THEN** the exporter maps column 3 to TxGoodPkt, column 4 to TxBadPkt, column 5 to RxGoodPkt,
  and column 6 to RxBadPkt

#### Scenario: Mixed packet and byte counters
- **WHEN** the table header is `Port, State, Link Status, TxGoodPkt, RxGoodPkt, TxGoodBytes,
  RxGoodBytes`
- **THEN** the exporter maps column 3 to TxGoodPkt, column 4 to RxGoodPkt, column 5 to
  TxGoodBytes, and column 6 to RxGoodBytes (not TxBad/RxBad, regardless of position)

### Requirement: Per-switch conditional metric emission
For each profile, the exporter SHALL emit a Prometheus metric for a given counter kind only if
that counter's column header was present on that switch's `port.cgi?page=stats` page. Counter
kinds whose header is absent SHALL NOT be emitted for that profile (not even as a `0` value).

#### Scenario: Switch without bad-packet counters
- **WHEN** a profile's page header has no `TxBadPkt`/`RxBadPkt` columns (only
  `TxGoodPkt`/`RxGoodPkt`/`TxGoodBytes`/`RxGoodBytes`)
- **THEN** `port_tx_bad_pkt` and `port_rx_bad_pkt` are not emitted for that profile's ports, while
  `port_tx_good_pkt`, `port_rx_good_pkt`, `port_tx_good_bytes`, and `port_rx_good_bytes` are

#### Scenario: Switch with only packet counters (backward compatibility)
- **WHEN** a profile's page header is `TxGoodPkt, TxBadPkt, RxGoodPkt, RxBadPkt` (as in
  `examples/4.html`, an already-supported model)
- **THEN** the exporter emits exactly `port_tx_good_pkt`, `port_tx_bad_pkt`, `port_rx_good_pkt`,
  and `port_rx_bad_pkt` for that profile, with the same values as before this change

### Requirement: Byte-counter metrics
The exporter SHALL expose Prometheus metrics for byte-counter kinds, in addition to the existing
packet-counter metrics: `port_tx_good_bytes`, `port_tx_bad_bytes`, `port_rx_good_bytes`,
`port_rx_bad_bytes`, each using the same `port`/`comment`/`switch`/`address` labels as the
existing packet-counter metrics.

#### Scenario: Byte counter present in header
- **WHEN** a profile's page header includes `TxGoodBytes`
- **THEN** the exporter emits `port_tx_good_bytes` for that profile's ports, using the
  recombined counter value (see split 32-bit-half counter recombination) from that column
