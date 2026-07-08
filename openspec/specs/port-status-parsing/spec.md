## Purpose

Defines how the exporter determines negotiated Duplex/Speed values from `port.cgi?page=status`,
across the different group-header column layouts used by supported switch models.

## Requirements

### Requirement: Header-driven Duplex/Speed column detection
The exporter SHALL determine which data column(s) in `port.cgi?page=status`'s real per-port table
(the last `<table>` in the document) hold the negotiated Duplex and Speed values by reading that
table's group-header row (`<th>` cells, using `colspan` to determine each group's width) rather
than assuming a fixed column position or a fixed total column count. A group header whose text is
`"Duplex Mode"` SHALL be treated as a separate Duplex group; a group header whose text is
`"Negotiation Speed"` SHALL be treated as a separate Speed group; a group header whose text is
`"Speed/Duplex"` SHALL be treated as one merged Speed+Duplex group. Within any colspan-2 group,
the second column (whose second-row sub-header text contains `"Actual"`) SHALL be used as that
group's live/negotiated value; the first column (`"Config"`/configured value) SHALL NOT be used
for Duplex/Speed reporting.

#### Scenario: Separate Duplex and Speed groups
- **WHEN** the data table's group-header row is `Port, State, Duplex Mode (colspan 2),
  Negotiation Speed (colspan 2), Flow Control (colspan 2)`, with sub-headers `Config Attribute,
  Actual Status` under each colspan-2 group, and a data row has `"Half Duplex"` under Duplex
  Mode's Actual column and `"10M"` under Negotiation Speed's Actual column
- **THEN** the resulting `PortStatus` has `SpeedMbps: 10` and `FullDuplex: 0`

#### Scenario: Merged Speed/Duplex group
- **WHEN** the data table's group-header row is `Port, State, Speed/Duplex (colspan 2), Flow
  Control (colspan 2)`, with sub-headers `Config, Actual` under each colspan-2 group, and a data
  row has `"1000Full"` under Speed/Duplex's Actual column
- **THEN** the resulting `PortStatus` has `SpeedMbps: 1000` and `FullDuplex: 1`, and the Flow
  Control group's Actual column (e.g. `"On"`/`"Off"`) is NOT used as the speed value

### Requirement: Merged Speed/Duplex cell parsing
For the merged Speed/Duplex group, the exporter SHALL parse the combined Actual cell text as: one
or more digits, an optional `G` suffix meaning the digits are Gbps (otherwise the digits are
already Mbps), followed by `Full` or `Half`. The literal value `"Link Down"` SHALL parse as speed
`0` and duplex `0` (not full duplex). Any other unrecognized text SHALL parse as speed `0` and
duplex `0`, rather than erroring.

#### Scenario: Plain Mbps value
- **WHEN** the combined cell text is `"2500Full"`
- **THEN** the parsed speed is `2500` Mbps and duplex is full

#### Scenario: Gbps value
- **WHEN** the combined cell text is `"10GFull"`
- **THEN** the parsed speed is `10000` Mbps and duplex is full

#### Scenario: Link down
- **WHEN** the combined cell text is `"Link Down"`
- **THEN** the parsed speed is `0` and duplex is not full

#### Scenario: Half duplex at low speed
- **WHEN** the combined cell text is `"10Half"`
- **THEN** the parsed speed is `10` Mbps and duplex is not full

### Requirement: Unrecognized layout degrades to no speed/duplex data
The exporter SHALL leave SpeedMbps and FullDuplex at zero for a profile's ports, rather than
misreading an unrelated column, when neither a separate Duplex+Speed group pair nor a merged
Speed/Duplex group is found in the data table's group-header row. This matches how unrecognized
counter headers are already handled on the `port.cgi?page=stats` page.

#### Scenario: Header text not recognized
- **WHEN** the data table's group-header row contains none of `"Duplex Mode"`, `"Negotiation
  Speed"`, or `"Speed/Duplex"`
- **THEN** `SpeedMbps` and `FullDuplex` are `0` for every port on that profile, and no unrelated
  column (e.g. Flow Control) is used as a substitute
