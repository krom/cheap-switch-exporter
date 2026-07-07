## Context

`main.go` is a single file mixing pure parsing helpers (`state`, `link`, `duplex`, `onoff`,
`poeType`, `speed`, `parseNum`) with HTTP-fetching functions that parse the response with
goquery (`fetchPorts`, `fetchPortStatus`, `fetchPoE`, via the shared `fetchDoc`). `fetchDoc`
builds its request URL from `cfg.Address` (`"http://"+cfg.Address+path`), so it's already
testable without modification: pointing `cfg.Address` at an `httptest.Server`'s host (stripping
the `http://` scheme httptest includes in `.URL`) exercises the real HTTP + goquery parsing path
against fixture bytes, with no need to extract a separate "parse this goquery.Document" function
or refactor `fetchDoc`'s signature.

Real captured fixtures already exist for the `port.cgi?page=stats` page
(`examples/1.html`-`4.html`, added while investigating the `bytes-vs-packets` change) covering
both the packet-only and mixed packet/byte column layouts. No equivalent captures exist yet for
`port.cgi?page=status`, `pse_system.cgi`, or `pse_port.cgi?page=stats`.

## Goals / Non-Goals

**Goals:**
- Cover every function CLAUDE.md calls out (`fetchPorts`, `fetchPortStatus`, `fetchPoE`, and the
  small helpers) with automated tests runnable via `go test ./...`, no live switch required.
- Reuse the existing `examples/*.html` captures as-is for `fetchPorts` tests rather than
  duplicating or re-deriving fixture data.
- Keep `main.go` production code untouched — this is test-only, matching the proposal's
  "no runtime behavior changes" scope.

**Non-Goals:**
- Not achieving 100% coverage of `Collect()`/`NewCollector` (Prometheus collector wiring) — low
  value to test given it's mostly plumbing over already-tested `fetch*` functions, and doing so
  well would require a `prometheus/client_golang/prometheus/testutil` dependency addition, which
  is out of scope here.
- Not adding new switch-model support or changing `fetchPortStatus`/`fetchPoE`'s column
  assumptions — the synthetic fixtures for those two just need to match what the current code
  already expects, not model a specific real switch.
- Not adding CI wiring (e.g. GitHub Actions) — just the tests themselves, runnable locally via
  `go test ./...`.

## Decisions

- **Point `fetchDoc` at `httptest.Server` instead of extracting a parse-only function.** Each
  test starts an `httptest.NewServer` whose handler serves the right fixture file based on the
  request path (`/port.cgi`, `/pse_system.cgi`, `/pse_port.cgi`) and `page` query param, then
  sets `cfg.Address = strings.TrimPrefix(server.URL, "http://")` before calling
  `fetchPorts(cfg)`/etc. This tests the real code path (URL construction, cookie header,
  goquery parsing) instead of a parallel test-only parsing function that could drift from
  the real one.
  - Alternative considered: refactor `fetchPorts` etc. to accept an `io.Reader` or
    `*goquery.Document` directly, bypassing HTTP — rejected because it would require touching
    production code purely for testability, and the `httptest` approach already gets full
    coverage without that.
- **Reuse `examples/*.html` directly as `fetchPorts` test fixtures** (via `os.ReadFile` or
  serving the file straight from the `examples/` directory in the test's `httptest` handler)
  rather than copying them into `testdata/`. They already encode the real column-layout
  variance this change most wants regression protection for.
- **Add synthetic `testdata/` fixtures for `fetchPortStatus`/`fetchPoE`** since no real captures
  exist for those endpoints yet. Build them to match the exact column counts/positions the
  current code reads (`tds.Eq(3)` duplex, `tds.Eq(5)` speed, 7-column PoE port table,
  `input[name="pse_con_pwr"]` for system consumption) so the tests verify today's documented
  behavior; if a future change captures real HTML for these pages (like `bytes-vs-packets` did
  for stats), these synthetic fixtures should be replaced/supplemented then.
- **Table-driven tests for the pure helpers**, including the `parseNum` split-counter and
  negative-number cases already manually verified during `fix-numbers` (`"1-901525430"` →
  `5196492726`, `"-123"` → `-123`) — this change is what turns that throwaway verification into
  a permanent regression test.

## Risks / Trade-offs

- [Risk] Synthetic status/PoE fixtures could encode a wrong assumption about real switch HTML
  if the current code's column assumptions are themselves wrong (untested against a real
  switch) → Mitigation: explicitly comment in the fixture files that they're synthetic
  (mirroring current code's expectations, not a real capture), so future readers don't mistake
  them for a captured reference the way `examples/*.html` are.
- [Risk] `examples/*.html` were added for switch-model-support investigation, not originally
  intended as committed test fixtures — reusing them ties their contents to test stability →
  Mitigation: acceptable trade-off; they're already committed and real, and CLAUDE.md explicitly
  recommends fixture-based testing against exactly this kind of sample.
