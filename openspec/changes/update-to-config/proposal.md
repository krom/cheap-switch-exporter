## Why

`main.go` was updated to support multiple switch profiles under a `profiles:` list (with per-profile `comments`, `poe`, `timeout_seconds`), but the documentation was never brought in sync. `config.yaml.example` has invalid YAML (missing colon after `profiles`) and references fields (`profile`, `poll_rate_seconds`) that `ProfileConfig` doesn't parse, so copying it as a starting config produces a broken or misleading setup. README's "Configuration" section still documents the old single-device, flat-field format, which no longer matches how the exporter is actually configured.

## What Changes

- Rewrite `config.yaml.example` as valid YAML matching `ProfileConfig` exactly: `profiles:` list of named entries, each with `address`, `username`, `password`, `timeout_seconds`, `poe`, and a `comments` map. Remove the unused `profile` and `poll_rate_seconds` fields from the example.
- Rewrite the README "Configuration" section to document the multi-profile structure (naming a profile, per-profile fields, `comments` map, `poe` flag) instead of the old flat single-device format.
- No changes to `main.go` or parsing logic — this is a documentation/example-only change.

## Capabilities

### New Capabilities
- `config-loading`: Formalizes the existing (but previously unspecified) multi-profile config format that `main.go` already implements, so `config.yaml.example` and the README can be verified against a documented contract.

### Modified Capabilities
(none — the underlying parsing behavior in `main.go` is unchanged; only documentation/examples are corrected to match it)

## Impact

- Affected files: `config.yaml.example`, `README.md`.
- No code, API, or dependency changes.
