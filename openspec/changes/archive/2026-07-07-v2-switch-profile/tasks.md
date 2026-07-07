## 1. Config: profile field validation

- [x] 1.1 In `main()`'s existing per-profile resolution loop, validate each profile's
      `Profile` field is one of `""`, `"default"`, or `"v2"`; `log.Fatal` with the profile
      name and invalid value otherwise.
- [x] 1.2 In the same loop, `log.Fatal` if a profile has `Profile == "v2"` and `PoE == 1`,
      naming the profile in the error.

## 2. switchClient abstraction

- [x] 2.1 Define `type switchClient interface { FetchPorts() ([]Port, []PortStatus, error); FetchPoE() (float64, []PoEPort, error) }`.
- [x] 2.2 Add `type defaultClient struct { cfg ProfileConfig }` implementing `switchClient`
      by calling the existing `fetchPorts`/`fetchPortStatus`/`fetchPoE` functions unchanged.
- [x] 2.3 Add a `newSwitchClient(cfg ProfileConfig) switchClient` resolver that returns
      `defaultClient` or `v2Client` based on the profile's resolved `Profile` field.
- [x] 2.4 Update `scrapeOnce` to resolve a `switchClient` for the profile once per call and
      use it instead of calling `fetchPorts`/`fetchPortStatus`/`fetchPoE` directly (skip the
      `FetchPoE` call entirely for v2 profiles, per the "no PoE fetch for v2" requirement).

## 3. v2 login

- [x] 3.1 Add a helper to perform `GET /authorize?loginusr=<md5(username)>&loginpwd=<md5(password)>`
      against the profile's address (respecting the profile's resolved `Timeout`), returning
      the response's cookies (or an error).
- [x] 3.2 Add `type v2Client struct { cfg ProfileConfig }` and have its `FetchPorts` call the
      login helper immediately before fetching data, using only that call's cookies (no
      caching/reuse across calls).

## 4. v2 port/counter data via JSON

- [x] 4.1 Add a `v2PortJSON` struct matching `/port_statistics.json`'s per-port shape
      (`Port_Id`, `Port_Status`, `Link_Status`, `TxGoodPkt`, `TxBadPkt`, `RxGoodPkt`,
      `RxBadPkt`), and unmarshal the top-level response as `map[string]json.RawMessage` so
      `Port_N` keys can be iterated regardless of `PortNum`.
- [x] 4.2 In `v2Client.FetchPorts`, build `[]Port` from the `Port_` prefixed keys actually
      present (ignore `PortNum` as a loop bound; use it only as a sanity check/log if
      desired), mapping `Port_Status`/`TxGoodPkt`/`TxBadPkt`/`RxGoodPkt`/`RxBadPkt` directly
      onto `Port.State`/`Port.Counters[TxGoodPkt/TxBadPkt/RxGoodPkt/RxBadPkt]`.
- [x] 4.3 Add `parseLinkStatus(s string) (linkUp bool, speedMbps float64, fullDuplex bool, ok bool)`
      parsing `"Link Down"` and `"<number><Mbps|Gbps><Full|Half>"`; `ok == false` for
      anything else.
- [x] 4.4 In `v2Client.FetchPorts`, use `parseLinkStatus` to build each port's
      `[]PortStatus` entry; on `ok == false`, log a warning naming the profile/port and omit
      that port's `PortStatus` entry (its `Port`/counters entry is still included).
- [x] 4.5 Add `v2Client.FetchPoE` returning `(0, nil, nil)` (never called per task 2.4, but
      satisfies the interface).

## 5. Tests and docs

- [x] 5.1 Add table-driven tests for `parseLinkStatus` covering: known link-up formats
      (`"1000MbpsFull"`, `"2500MbpsFull"`, `"100MbpsFull"`, `"10GbpsFull"`), `"Link Down"`,
      and at least one unrecognized-format case.
- [x] 5.2 Add a test for the profile-field/PoE validation (task 1.1/1.2) covering: valid
      values (`""`, `"default"`, `"v2"`), an invalid value, and `v2`+`poe:1`.
- [x] 5.3 Add a `v2Client.FetchPorts` test using `examples/sks3200-8e2x_port_statistics.json`
      as fixture data served by an `httptest.Server` that also serves `/authorize`, verifying
      login happens before the data fetch and that ports/counters/link status parse as
      expected.
- [x] 5.4 Update `config.yaml.example` and README's Configuration section to document
      `profile: v2`, its constraints (no `poe: 1`), and what it changes about how that
      profile is scraped.
- [x] 5.5 Run `go vet ./...`, `go test ./...`, and `go build -o cheap-switch-exporter .` to
      confirm no regressions.
