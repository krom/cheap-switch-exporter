## ADDED Requirements

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
