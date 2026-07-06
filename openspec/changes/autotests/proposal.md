## Why

CLAUDE.md already flags this: "There is no test suite in this repo yet." Every scrape/parse
change so far (`fix-numbers`, `bytes-vs-packets`) has been verified with throwaway scratch
scripts run once and discarded, so there's no regression protection — a future edit to
`fetchPorts`/`parseNum`/the helper functions could silently reintroduce the split-counter bug
or break header-driven counter detection with nothing to catch it. The project already has
real captured switch HTML in `examples/*.html`, which is exactly the fixture material CLAUDE.md
recommends testing against.

## What Changes

- Add a `main_test.go` with:
  - Table-driven unit tests for the small pure helpers: `state`, `link`, `duplex`, `onoff`,
    `poeType`, `speed`, `parseNum` (including the split 32-bit-half counter case and the
    negative-number edge case).
  - Fixture-driven tests for `fetchPorts` against `examples/1.html`-`4.html`, asserting the
    exact `Port.Counters` map produced per fixture (covering both the packet-only layout and
    the mixed packet/byte layout from the `bytes-vs-packets` change).
  - Fixture-driven tests for `fetchPortStatus` and `fetchPoE` against new small synthetic
    fixtures under `testdata/` (no real captures exist yet for `port.cgi?page=status`,
    `pse_system.cgi`, or `pse_port.cgi?page=stats`), covering speed/duplex parsing and PoE
    port/system parsing.
  - All three fetch functions are tested by pointing `ProfileConfig.Address` at an
    `httptest.Server` that serves the fixture files, exercising the real `fetchDoc` HTTP path
    rather than reimplementing parsing logic in the test.
- Add `go test ./...` to the documented commands in CLAUDE.md.
- No changes to `main.go`'s runtime behavior.

## Capabilities

### New Capabilities
- `parsing-test-coverage`: Automated test coverage for the HTML-scraping/parsing functions and
  their small pure helpers, run against fixture HTML rather than a live switch.

### Modified Capabilities
(none)

## Impact

- New files: `main_test.go`, `testdata/port_status.html`, `testdata/pse_system.html`,
  `testdata/pse_port.html`.
- Affected docs: CLAUDE.md's Commands section (add `go test ./...`) and its note that no test
  suite exists yet.
- No production code changes; `go.mod`/`go.sum` unaffected (uses only the stdlib
  `net/http/httptest` and `testing` packages).
