## Purpose

Defines how the exporter scrapes switches using the "v2" dialect: a JSON-based API reached via
an `/authorize` login request followed by a `/port_statistics.json` fetch, as an alternative to
the existing cookie+HTML-scrape dialect.

## Requirements

### Requirement: v2 login before each scrape
For a profile resolved to the `v2` dialect, the exporter SHALL perform a login request
(`GET /authorize` with `loginusr`/`loginpwd` set to the MD5 hex digest of the username and
password respectively, hashed separately) before fetching port data on every scrape, and
SHALL use the session cookies returned by that login for the subsequent data request. The
session SHALL NOT be cached or reused across scrapes.

#### Scenario: Successful login precedes data fetch
- **WHEN** a `v2` profile is scraped
- **THEN** the exporter first requests `/authorize?loginusr=<md5(username)>&loginpwd=<md5(password)>`,
  then requests `/port_statistics.json` using the cookies from that login's response

#### Scenario: Each scrape logs in independently
- **WHEN** a `v2` profile is scraped twice in a row (two separate poll ticks)
- **THEN** a separate `/authorize` login request is made for each scrape; no session state
  from the first scrape is reused by the second

### Requirement: v2 port and counter data from JSON
For a profile resolved to the `v2` dialect, the exporter SHALL fetch `/port_statistics.json`
and parse it as JSON (a map of dynamically-named `Port_N` keys, not an array) to populate
per-port state and Tx/Rx good/bad packet counters, without any HTML parsing.

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

### Requirement: v2 Link_Status parsing
For a profile resolved to the `v2` dialect, the exporter SHALL parse each port's
`Link_Status` string into link-up/down, negotiated speed (Mbps), and duplex, recognizing
`"Link Down"` as link-down and `"<number><Mbps|Gbps><Full|Half>"` (e.g. `"1000MbpsFull"`,
`"10GbpsFull"`) as link-up with the given speed and duplex.

#### Scenario: Known link-up format parsed
- **WHEN** a port's `Link_Status` is `"2500MbpsFull"`
- **THEN** the exporter emits that port as link-up, full-duplex, at 2500 Mbps

#### Scenario: Gbps unit converted to Mbps
- **WHEN** a port's `Link_Status` is `"10GbpsFull"`
- **THEN** the exporter emits that port's speed as 10000 Mbps

#### Scenario: Link down
- **WHEN** a port's `Link_Status` is `"Link Down"`
- **THEN** the exporter emits that port as link-down, with no speed/duplex value implied

#### Scenario: Unrecognized Link_Status format
- **WHEN** a port's `Link_Status` does not match any recognized format (known or
  documented-but-unverified, e.g. an unexpected unit or duplex value)
- **THEN** the exporter logs a warning identifying the profile and port, omits that port's
  link/speed/duplex metrics for this scrape, and still emits that port's other metrics
  (state, Tx/Rx counters) normally

### Requirement: No PoE fetch for v2 profiles
The exporter SHALL NOT perform any PoE-related HTTP requests for a profile resolved to the
`v2` dialect, regardless of configuration, since no v2 PoE endpoint is defined by this
capability.

#### Scenario: v2 profile never triggers PoE requests
- **WHEN** a profile is resolved to the `v2` dialect
- **THEN** the exporter makes no requests to any PoE-related endpoint for that profile
