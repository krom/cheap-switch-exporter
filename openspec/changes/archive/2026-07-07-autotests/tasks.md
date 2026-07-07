## 1. Helper unit tests

- [x] 1.1 Create `main_test.go` with table-driven tests for `state`, `link`, `duplex`, `onoff`,
      `poeType`, and `speed` covering each recognized input string and an unrecognized fallback.
- [x] 1.2 Add table-driven tests for `parseNum`: split-counter recombination (`"1-901525430"` →
      `5196492726`, `"0-2549448529"` → `2549448529`), plain numbers (`"71817"` → `71817`),
      empty/placeholder (`""`, `"-"` → `0`), and the negative-number case (`"-123"` → `-123`).

## 2. fetchPorts fixture tests

- [x] 2.1 Add a test helper that starts an `httptest.Server` serving a given fixture file for
      `/port.cgi` requests and returns a `ProfileConfig` pointed at it (`Address` set to the
      server's host, stripped of the `http://` scheme).
- [x] 2.2 Test `fetchPorts` against `examples/4.html`: assert each port's `Counters` map is
      exactly `{TxGoodPkt, TxBadPkt, RxGoodPkt, RxBadPkt}` with the fixture's values.
- [x] 2.3 Test `fetchPorts` against `examples/1.html`, `examples/2.html`, and `examples/3.html`:
      assert each port's `Counters` map is exactly `{TxGoodPkt, RxGoodPkt, TxGoodBytes,
      RxGoodBytes}` with the fixture's values (verifying split-counter recombination where
      applicable, e.g. `examples/2.html` port 1).

## 3. fetchPortStatus and fetchPoE fixture tests

- [x] 3.1 Create `testdata/port_status.html`, a small synthetic fixture matching the column
      layout `fetchPortStatus` expects (name at column 0, duplex at column 3, speed at column
      5), covering at least one of each recognized speed/duplex value.
- [x] 3.2 Create `testdata/pse_system.html` and `testdata/pse_port.html`, small synthetic
      fixtures matching `fetchPoE`'s expectations (`input[name="pse_con_pwr"]` value attribute;
      a 7-column PoE port table), covering each recognized `poeType`/`onoff` value.
- [x] 3.3 Test `fetchPortStatus` and `fetchPoE` against these fixtures via the same
      `httptest.Server` helper, asserting the parsed values match the fixtures.

## 4. Docs and verification

- [x] 4.1 Add `go test ./...` to CLAUDE.md's Commands section and remove/update the note that no
      test suite exists.
- [x] 4.2 Run `go test ./...` and `go vet ./...` to confirm the full suite passes.
