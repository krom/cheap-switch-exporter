## Why

The v2 dialect named ports after the bare `Port_Id` JSON field (e.g. `"1"`), while the default
HTML dialect names ports `"Port 1"`. Per-port `comments` config is keyed by port name and
matched positionally, so a `comments` map written the way the README and default dialect
document it (`Port 1: Uplink`) silently failed to match any v2 port, leaving the `comment`
label empty for every v2 profile.

## What Changes

- v2 port names are now formatted as `"Port " + Port_Id` (e.g. `"Port 1"`), matching the
  default dialect's naming convention, so `comments` keys are consistent across both dialects.
- Updated `TestV2ClientFetchPorts` expectations in `main_test.go` to the new `"Port N"` names.
- Corrected the v2 example in `README.md`, which had documented the old plain-number naming
  as if it were the intended behavior.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `v2-switch-client`: port names emitted for the v2 dialect are now `"Port <Port_Id>"` instead
  of the bare `Port_Id` value, so they align with the default dialect's port-naming convention
  and with `comments` config lookups.

## Impact

- `main.go`: `v2Client.FetchPorts` port-name construction.
- `main_test.go`: v2 fetch-port test expectations.
- `README.md`: v2 profile config example.
- No breaking change to metric names/labels themselves, but the `port`/`comment` label
  *values* emitted for v2 profiles change from e.g. `"1"` to `"Port 1"` — existing `comments`
  config for v2 profiles keyed on bare numbers must be updated to `"Port N"` keys.
