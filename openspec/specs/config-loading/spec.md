## Purpose

Defines how `config.yaml` specifies global and per-profile poll rate and timeout settings,
and how those values are resolved to an effective value for each profile.

## Requirements

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

### Requirement: Profile field selects switch dialect
Each profile entry's `profile` field (`ProfileConfig.Profile`) SHALL select which switch
dialect the exporter uses to scrape that profile: an empty value or `"default"` selects the
existing cookie+HTML-scrape dialect, and `"v2"` selects the JSON-based v2 dialect. Any other
value SHALL be rejected as a fatal configuration error at startup.

#### Scenario: Empty or default profile field preserves existing behavior
- **WHEN** a profile entry has no `profile` field, or `profile: default`
- **THEN** that profile is scraped using the existing cookie+HTML-scrape dialect, unchanged
  from before this capability existed

#### Scenario: v2 profile field selects the new dialect
- **WHEN** a profile entry has `profile: v2`
- **THEN** that profile is scraped using the v2 JSON dialect

#### Scenario: Unknown profile value rejected at startup
- **WHEN** a profile entry has `profile: something-else` (not empty, `"default"`, or `"v2"`)
- **THEN** the exporter fails to start and logs an error identifying the invalid value and
  the profile

### Requirement: poe:1 rejected on v2 profiles
A profile entry with `profile: v2` and `poe: 1` SHALL be rejected as a fatal configuration
error at startup, since the v2 dialect does not support PoE.

#### Scenario: v2 profile with PoE enabled fails to start
- **WHEN** a profile entry has `profile: v2` and `poe: 1`
- **THEN** the exporter fails to start and logs an error identifying the profile and the
  unsupported `poe: 1` + `profile: v2` combination

#### Scenario: v2 profile without PoE starts normally
- **WHEN** a profile entry has `profile: v2` and `poe: 0` (or `poe` omitted)
- **THEN** the exporter starts normally and scrapes that profile using the v2 dialect
