## Purpose

Defines the automated test coverage the repository maintains for the exporter's parsing logic:
the pure string/number parsing helpers and the fixture-driven HTML-scraping functions
(`fetchPorts`, `fetchPortStatus`, `fetchPoE`).

## Requirements

### Requirement: Helper function unit tests
The repository SHALL have automated unit tests, runnable via `go test ./...`, covering each pure
parsing helper (`state`, `link`, `duplex`, `onoff`, `poeType`, `speed`, `parseNum`) with
table-driven cases for their documented behaviors, including `parseNum`'s split 32-bit-half
counter recombination and its plain-number/empty/negative-number fallback cases.

#### Scenario: parseNum regression coverage
- **WHEN** `go test ./...` is run
- **THEN** it exercises `parseNum("1-901525430")`, `parseNum("71817")`, `parseNum("")`,
  `parseNum("-")`, and `parseNum("-123")`, asserting `5196492726`, `71817`, `0`, `0`, and `-123`
  respectively

### Requirement: fetchPorts fixture tests
The repository SHALL have automated tests for `fetchPorts` that serve each of
`examples/1.html`-`4.html` through an HTTP test server and assert the exact `Port.Counters` map
produced for every port, covering both the packet-only column layout and the mixed packet/byte
column layout.

#### Scenario: Packet-only fixture
- **WHEN** `fetchPorts` is run against a server serving `examples/4.html`
- **THEN** each port's `Counters` map contains exactly `TxGoodPkt`, `TxBadPkt`, `RxGoodPkt`,
  `RxBadPkt` with the values present in that fixture, and no `*Bytes` keys

#### Scenario: Mixed packet/byte fixture
- **WHEN** `fetchPorts` is run against a server serving `examples/2.html`
- **THEN** each port's `Counters` map contains exactly `TxGoodPkt`, `RxGoodPkt`, `TxGoodBytes`,
  `RxGoodBytes` with the values present in that fixture (including split 32-bit-half
  recombination for non-zero high halves), and no `TxBadPkt`/`RxBadPkt` keys

### Requirement: fetchPortStatus and fetchPoE fixture tests
The repository SHALL have automated tests for `fetchPortStatus` and `fetchPoE` that serve
synthetic fixture HTML (under `testdata/`) matching the column layout those functions currently
assume, through an HTTP test server, and assert the parsed `PortStatus`/`PoEPort`/consumption
values.

#### Scenario: Port status fixture
- **WHEN** `fetchPortStatus` is run against a server serving the status test fixture
- **THEN** the returned `PortStatus` entries have the expected `SpeedMbps` and `FullDuplex`
  values for each port in the fixture

#### Scenario: PoE fixture
- **WHEN** `fetchPoE` is run against a server serving the PoE test fixtures
- **THEN** the returned system consumption value and each port's `State`/`Power`/`Type`/
  `Watts`/`Voltage`/`Current` match the fixture's values
