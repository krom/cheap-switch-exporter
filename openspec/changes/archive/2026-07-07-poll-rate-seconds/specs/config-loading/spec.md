## ADDED Requirements

### Requirement: Global settings block
`config.yaml` SHALL support a top-level `settings:` block with `poll_rate_seconds` and
`timeout_seconds` fields, providing defaults that apply to every profile unless overridden.

#### Scenario: Global settings applied to a profile without overrides
- **WHEN** `config.yaml` has `settings: {poll_rate_seconds: 30, timeout_seconds: 8}` and a
  profile with no `poll_rate_seconds`/`timeout_seconds` of its own
- **THEN** that profile is polled every 30 seconds using an 8 second request timeout

### Requirement: Per-profile override of poll_rate_seconds
Each profile entry SHALL support an optional `poll_rate_seconds` field that overrides the global
setting for that profile only.

#### Scenario: Profile overrides the global poll rate
- **WHEN** `settings.poll_rate_seconds` is `60` and a specific profile sets its own
  `poll_rate_seconds: 15`
- **THEN** that profile is polled every 15 seconds while other profiles use the 60 second global
  default

### Requirement: Resolution precedence and defaults
For both `poll_rate_seconds` and `timeout_seconds`, the effective value for a profile SHALL be
resolved as: the profile's own value, if set and non-zero; otherwise the global `settings` value,
if set and non-zero; otherwise a hardcoded default of `60` seconds for `poll_rate_seconds` and
`10` seconds for `timeout_seconds`.

#### Scenario: Nothing configured, hardcoded defaults apply
- **WHEN** a profile has no `poll_rate_seconds`/`timeout_seconds` and `config.yaml` has no
  `settings:` block at all
- **THEN** that profile is polled every 60 seconds using a 10 second request timeout

#### Scenario: Profile value takes precedence over global setting
- **WHEN** `settings.timeout_seconds` is `5` and a profile sets `timeout_seconds: 20`
- **THEN** that profile's requests use a 20 second timeout, not 5
