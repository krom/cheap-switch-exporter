## Purpose

Defines how the exporter parses per-port counter values from a switch's `port.cgi?page=stats`
table: which counter kind each column represents, how a cell's numeric text is converted to a
value (including split 64-bit counters), and which metrics are emitted per switch.

## Requirements

### Requirement: Split 32-bit-half counter recombination
The exporter SHALL recombine a numeric table cell whose text matches the pattern `<high>-<low>`
(decimal digits, a single hyphen, decimal digits) into a 64-bit value computed as
`high * 4294967296 + low`, matching the value the switch's own page-embedded JavaScript would
compute.

#### Scenario: Non-zero high half
- **WHEN** a cell's text is `"1-901525430"`
- **THEN** the parsed value is `5196492726` (i.e. `1*4294967296 + 901525430`)

#### Scenario: Zero high half
- **WHEN** a cell's text is `"0-2549448529"`
- **THEN** the parsed value is `2549448529`

### Requirement: Plain numeric values unaffected
When a cell's text does not match the `<high>-<low>` split-counter pattern, the exporter SHALL
parse it the same way it did before this change: as a plain decimal number, or as `0` if empty,
`"-"`, or otherwise not parseable.

#### Scenario: Plain decimal counter
- **WHEN** a cell's text is `"71817"`
- **THEN** the parsed value is `71817`

#### Scenario: Empty or placeholder value
- **WHEN** a cell's text is `""` or `"-"`
- **THEN** the parsed value is `0`

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
