## 1. Fix config.yaml.example

- [x] 1.1 Rewrite `config.yaml.example` as valid YAML: `profiles:` list, each entry a single-key map (profile name -> settings) with `address`, `username`, `password`, `timeout_seconds`, `poe`, `comments`.
- [x] 1.2 Remove the unused `profile` and `poll_rate_seconds` fields from the example.
- [x] 1.3 Include at least two example profile entries so multi-profile usage is obvious at a glance.

## 2. Update README

- [x] 2.1 Rewrite the "Configuration" section's YAML snippet to show the `profiles:` list structure with `comments` and `poe`.
- [x] 2.2 Update surrounding prose (if any) that references single-device/flat-field config to describe multi-profile config instead.

## 3. Verify

- [x] 3.1 Copy the updated `config.yaml.example` to a scratch `config.yaml`, fill in placeholder values, and confirm `go run main.go` starts without a YAML/parse error and `/metrics` returns data for the configured profile(s).
