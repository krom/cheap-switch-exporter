## MODIFIED Requirements

### Requirement: fetchPortStatus and fetchPoE fixture tests
The repository SHALL have automated tests for `fetchPortStatus` and `fetchPoE` that serve fixture
HTML through an HTTP test server and assert the parsed `PortStatus`/`PoEPort`/consumption values.
`fetchPortStatus` SHALL be covered against both known `port.cgi?page=status` group-header layouts,
each backed by a real captured fixture under `examples/`: the separate-group (`Duplex Mode` +
`Negotiation Speed`) layout from an SKS3200-5E1X device, and the merged-group (`Speed/Duplex`)
layout from an SKS3200-8E1X-P device.

#### Scenario: Port status fixture, separate Duplex/Speed groups
- **WHEN** `fetchPortStatus` is run against a server serving
  `examples/sks3200-5e1x_port_status.html`
- **THEN** the returned `PortStatus` entries have `SpeedMbps` and `FullDuplex` matching that
  fixture's real negotiated values for each port

#### Scenario: Port status fixture, merged Speed/Duplex group
- **WHEN** `fetchPortStatus` is run against a server serving
  `examples/sks3200-8e1x-p_port_status.html`, including a `Link Down` port and a `10GFull` port
- **THEN** the returned `PortStatus` entries have `SpeedMbps` and `FullDuplex` matching the
  fixture's actual negotiated speeds (e.g. `1000`, `2500`, `10000`, and `0` for the `Link Down`
  port), not `0` for every port

#### Scenario: PoE fixture
- **WHEN** `fetchPoE` is run against a server serving the PoE test fixtures
- **THEN** the returned system consumption value and each port's `State`/`Power`/`Type`/
  `Watts`/`Voltage`/`Current` match the fixture's values
