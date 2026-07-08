## Why

`fetchPortStatus` always reports `SpeedMbps: 0` for the `poe_switch` profile (SKS3200-8E1X-P,
10.110.0.17), even though the switch's own web UI (System -> Port Setting) shows real negotiated
speeds (1000Full, 2500Full, 10GFull, Link Down). The parser assumes an 8-column
`port.cgi?page=status` table (`Port, State, Duplex-Config, Duplex-Actual, Speed-Config,
Speed-Actual, Flow-Config, Flow-Actual`) and reads column 5 as Speed. On this device the page
only has 6 columns because Speed and Duplex are merged into one "Speed/Duplex" column pair
(`Port, State, SpeedDuplex-Config, SpeedDuplex-Actual, Flow-Config, Flow-Actual`); column 5 there
is actually Flow Control Actual (`"On"`/`"Off"`), which never parses as a number. The real speed
value (e.g. `"1000"` in `"1000Full"`) is trapped inside column 3, which today is only read for
its `Full`/`Half` substring for the (accidentally correct) duplex value.

Confirmed live by curling both real devices with their auth cookie: SKS3200-5E1X (`desk_switch`,
10.110.0.18) genuinely has the assumed 8-column layout and works correctly today; SKS3200-8E1X-P
(10.110.0.17) has the 6-column merged layout and is broken. Both use `profile: default`, so the
fix must live inside `fetchPortStatus` itself rather than as a new dialect/profile.

## What Changes

- `fetchPortStatus` (`defaultclient.go`) reads the group-header row of the real data table (the
  last `<table>` in the document) to find which data column holds Duplex-Actual, Speed-Actual, or
  a merged Speed/Duplex-Actual value — by header label text (`"Duplex Mode"`, `"Negotiation
  Speed"`, `"Speed/Duplex"`), the same way `fetchPorts` already maps counter columns by header
  text, instead of assuming a fixed column index or branching on raw column count.
  - Separate-group layout (e.g. SKS3200-5E1X): unchanged result, reading Duplex-Actual and
    Speed-Actual from wherever the header row says they are.
  - Merged-group layout (SKS3200-8E1X-P): a new helper parses the combined Speed/Duplex-Actual
    cell into both a speed (Mbps) and a duplex flag; Flow Control's own Actual column is never
    misread as Speed.
- New parsing helper (`parse.go`) recognizes the merged-cell formats seen live: `"1000Full"`,
  `"2500Full"`, `"10GFull"`, `"10Half"`, `"100Half"`, etc. (digits, optional `G` suffix meaning
  Gbps, then `Full`/`Half`), plus the literal `"Link Down"` (speed 0, duplex 0/Half).
- Add real captured HTML fixtures under `examples/`: `sks3200-8e1x-p_port_status.html` (9 ports,
  including a `Link Down` row and a `10GFull` row) and `sks3200-8e1x-p_pse_port_stats.html` (no
  PoE example existed yet). A new `fetchPortStatus` test case exercises the merged-group path
  against the first, alongside the existing separate-group fixture/test.

## Capabilities

### New Capabilities
- `port-status-parsing`: defines how the exporter parses per-port link speed and duplex from
  `port.cgi?page=status`, covering both the 8-column (separate Duplex/Speed columns) and
  6-column (merged Speed/Duplex column) table layouts observed on real switches.

### Modified Capabilities
- `parsing-test-coverage`: the `fetchPortStatus` fixture test requirement is updated to require
  coverage of both the 8-column and 6-column layouts, with the 6-column fixture being a real
  captured page rather than a synthetic one.

## Impact

- `defaultclient.go`: `fetchPortStatus` gains header-driven column detection (group-header text
  and colspan), replacing its fixed column indices.
- `parse.go`: new `parseSpeedDuplex` (or equivalently named) helper function.
- `defaultclient_test.go`: new test case for the merged-group layout.
- `examples/`: two new real-captured fixture files, `sks3200-8e1x-p_port_status.html` and
  `sks3200-8e1x-p_pse_port_stats.html`.
- No config schema changes; no new `profile:` value; `poe_switch` in `config.yaml` requires no
  changes to pick up the fix.
