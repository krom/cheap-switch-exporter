## 1. Fixtures

- [x] 1.1 Add `examples/sks3200-8e1x-p_port_status.html`, a real capture of
      `port.cgi?page=status` from an SKS3200-8E1X-P (10.110.0.17), including the two config-form
      tables and the 9-row data table (with a `Link Down` port and a `10GFull` port).
- [x] 1.2 Add `examples/sks3200-8e1x-p_pse_port_stats.html`, a real capture of
      `pse_port.cgi?page=stats` from the same device (no PoE example existed yet).
- [x] 1.3 (found during implementation) Replace `testdata/port_status.html` — a synthetic,
      never-real single-header-row fixture — with a real capture,
      `examples/sks3200-5e1x_port_status.html`, from the SKS3200-5E1X (10.110.0.18), whose actual
      two-row colspan group-header layout the synthetic fixture didn't match. Updated
      `TestFetchPortStatus`'s expected values (now `TestFetchPortStatus_SeparateDuplexSpeedGroups`)
      to match the real capture's data.

## 2. Parsing helper

- [x] 2.1 Add `parseSpeedDuplex(s string) (speedMbps float64, fullDuplex float64)` to `parse.go`,
      handling `"<digits>[G]Full|Half"` and the literal `"Link Down"`, falling back to `0, 0` for
      unrecognized input.
- [x] 2.2 Add table-driven unit tests for `parseSpeedDuplex` covering `"1000Full"`, `"2500Full"`,
      `"10GFull"`, `"10Half"`, `"Link Down"`, and an unrecognized string.

## 3. fetchPortStatus header-driven column detection

- [x] 3.1 In `defaultclient.go`, change `fetchPortStatus` to operate on `doc.Find("table").Last()`
      (the real per-port data table) instead of scanning `doc.Find("table tr")` across the whole
      document.
- [x] 3.2 Walk that table's first (group-header) `<tr>`'s `<th>` cells, tracking a running column
      offset and each header's `colspan` (default 1), to locate: a `"Duplex Mode"` group's Actual
      column, a `"Negotiation Speed"` group's Actual column, or a `"Speed/Duplex"` group's Actual
      column. Implemented as: the Actual column is always the last column of a colspan-2 group
      (both real layouts observed put Config first, Actual second), so the offset+colspan
      arithmetic alone locates it without needing to also inspect the second header row's text —
      simpler than matching sub-header wording and not dependent on "Actual" vs "Actual Status"
      phrasing at all.
- [x] 3.3 Update the data-row loop to use whichever column(s) were found: separate group ->
      existing `duplex()`/`speed()` on their respective Actual columns; merged group ->
      `parseSpeedDuplex` on its Actual column. If neither group is found, leave
      `SpeedMbps`/`FullDuplex` at their zero values rather than reading an unrelated column.
- [x] 3.4 Add a `fetchPortStatus` test case serving `examples/sks3200-8e1x-p_port_status.html` via
      `httptest.Server`, asserting `SpeedMbps`/`FullDuplex` for every port, including the
      `Link Down` and `10GFull` ports.
- [x] 3.5 Confirm the separate-group `fetchPortStatus` test still passes (now against the real
      `examples/sks3200-5e1x_port_status.html` capture — see 1.3).

## 4. Verification

- [x] 4.1 Run `go vet ./...` and `go test ./...` — pass.
- [x] 4.2 Rebuilt and pointed the exporter at `config.yaml`'s `poe_switch` profile (10.110.0.17).
      `/metrics` now reports `port_link_speed` 1000/1000/1000/2500/2500/10000 for ports 1-4,6,9 and
      `0` for down ports 5,7,8, matching the live Port Setting page (`port_link_full_duplex`
      likewise correct). Also found and fixed, with the user's confirmation, an unrelated
      pre-existing typo in the user's local (gitignored) `config.yaml`: `poe_switch` had
      `profile: poe`, an invalid value that made the exporter refuse to start; changed to
      `profile: default`.
