## 1. Extract config handling

- [x] 1.1 Create `config.go` with `RootConfig`, `GlobalSettings`, `ProfileConfig`,
      `NamedProfile`, `resolveProfileConfig`, `validateProfileConfig` moved from `main.go`
- [x] 1.2 Add `loadConfig(path string) ([]NamedProfile, error)` to `config.go`, extracting the
      file-read + YAML-unmarshal + per-profile resolve/validate loop currently inline in
      `main()`
- [x] 1.3 Update `main()` to call `loadConfig("config.yaml")` and `log.Fatal` on error
- [x] 1.4 `go build ./... && go vet ./...`

## 2. Extract models and switch client abstraction

- [x] 2.1 Create `model.go` with `CounterKind`, `counterMetricNames`, `counterMetricHelp`,
      `Port`, `PortStatus`, `PoEPort`
- [x] 2.2 Create `switchclient.go` with the `switchClient` interface and `newSwitchClient`
- [x] 2.3 `go build ./... && go vet ./...`

## 3. Extract collector and background polling

- [x] 3.1 Create `collector.go` with `profileState`, `snapshot`, `PortStatsCollector`,
      `NewCollector`, `Describe`, `Collect`
- [x] 3.2 Create `poll.go` with `scrapeOnce`, `pollProfile`, `StartPolling`
- [x] 3.3 `go build ./... && go vet ./...`

## 4. Extract switch dialect implementations

- [x] 4.1 Create `defaultclient.go` with `defaultClient`, `fetchDoc`, `fetchPorts`,
      `fetchPortStatus`, `fetchPoE`
- [x] 4.2 Create `v2client.go` with `v2Client`, `v2Login`, `v2PortJSON`, `linkStatusRe`,
      `parseLinkStatus`
- [x] 4.3 Create `parse.go` with `state`, `link`, `duplex`, `onoff`, `poeType`, `speed`,
      `parseNum`, `md5hex`
- [x] 4.4 Confirm `main.go` now contains only `main()` (and package-level imports it needs)
- [x] 4.5 `go build ./... && go vet ./...`

## 5. Add doc comments and wrap errors

- [x] 5.1 Add doc comments to exported identifiers introduced/moved above that lack one
      (`Port`, `PortStatus`, `PoEPort`, `PortStatsCollector`, `NewCollector`, `RootConfig`,
      `GlobalSettings`, `ProfileConfig`, `NamedProfile`, etc.)
- [x] 5.2 Wrap errors returned from `fetchDoc`, `fetchPorts`, `fetchPortStatus`, `fetchPoE`,
      `v2Login`, `v2Client.FetchPorts`'s JSON decode, and `loadConfig` with a short static
      prefix identifying the failing operation (`fmt.Errorf("...: %w", err)`), without changing
      control flow or which errors are returned
- [x] 5.3 `go build ./... && go vet ./...`

## 6. Split tests to match

- [x] 6.1 Split `main_test.go` into `config_test.go`, `collector_test.go`,
      `defaultclient_test.go`, `v2client_test.go`, `parse_test.go` (and any others needed),
      moving each test function next to the source file it exercises, keeping shared test
      helpers (e.g. `portCounters`) in whichever file their callers are concentrated in, or a
      shared `helpers_test.go` if used across multiple files
- [x] 6.2 `go test ./...` passes with the same test names/count as before the split

## 7. Docs and final verification

- [x] 7.1 Update `CLAUDE.md`'s "Architecture" section (currently says "the entire exporter is a
      single file: `main.go`") to describe the new multi-file layout
- [x] 7.2 Full verification: `go build ./... && go vet ./... && go test ./...`
- [x] 7.3 Review `git diff --stat` to confirm no unintended file left behind (e.g. `main.go`
      shrunk to just `main()`, no orphaned helper duplicated across files)
