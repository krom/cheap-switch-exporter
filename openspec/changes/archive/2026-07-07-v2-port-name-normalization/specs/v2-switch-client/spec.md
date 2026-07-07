## MODIFIED Requirements

### Requirement: v2 port and counter data from JSON
For a profile resolved to the `v2` dialect, the exporter SHALL fetch `/port_statistics.json`
and parse it as JSON (a map of dynamically-named `Port_N` keys, not an array) to populate
per-port state and Tx/Rx good/bad packet counters, without any HTML parsing. The port name used
for metric labels and `comments` config lookup SHALL be formatted as `"Port " + Port_Id` (e.g.
`Port_Id: "1"` yields port name `"Port 1"`), matching the naming convention used by the default
HTML dialect, rather than the bare `Port_Id` value.

#### Scenario: Port fields map onto existing metrics
- **WHEN** `/port_statistics.json` returns a port entry with `Port_Status: "Enabled"`,
  `TxGoodPkt: "123"`, `TxBadPkt: "0"`, `RxGoodPkt: "456"`, `RxBadPkt: "0"`
- **THEN** the exporter emits that port's state as enabled and its Tx/Rx good/bad packet
  counters as `123`/`0`/`456`/`0`, using the same metric names the default dialect uses for
  these counter kinds

#### Scenario: Port count driven by actual keys present, not PortNum
- **WHEN** `/port_statistics.json`'s `PortNum` field disagrees with the number of `Port_`
  prefixed keys actually present in the response
- **THEN** the exporter emits metrics for the `Port_` keys actually present, not for a count
  derived from `PortNum`

#### Scenario: Port name normalized to match default dialect
- **WHEN** `/port_statistics.json` returns a port entry with `Port_Id: "1"`
- **THEN** the exporter emits that port's `port` label, and looks up its `comments` config
  entry, using the name `"Port 1"` (not the bare value `"1"`)
