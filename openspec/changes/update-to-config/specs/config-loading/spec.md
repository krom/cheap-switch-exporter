## ADDED Requirements

### Requirement: Multi-profile config structure
The exporter SHALL load `config.yaml` as a `profiles:` list, where each list entry is a single-key map from a profile name to that profile's settings (`address`, `username`, `password`, `timeout_seconds`, `poe`, `comments`). The exporter SHALL poll every profile in the list on each `/metrics` scrape.

#### Scenario: Multiple profiles configured
- **WHEN** `config.yaml` contains two entries under `profiles:`, each keyed by a distinct profile name
- **THEN** the exporter scrapes both switches on each `/metrics` request and labels their metrics with the corresponding profile name as the `switch` label

### Requirement: Per-port comment labels
Each profile SHALL support an optional `comments` map from port name (e.g. `Port 1`) to a human-readable label, which is attached to that port's metrics as the `comment` label.

#### Scenario: Port has a configured comment
- **WHEN** a profile's `comments` map has an entry `Port 6: Core Switch`
- **THEN** metrics for `Port 6` on that profile are emitted with `comment="Core Switch"`

### Requirement: Default timeout
When a profile omits `timeout_seconds` (or sets it to `0`), the exporter SHALL default the scrape timeout for that profile to 5 seconds.

#### Scenario: timeout_seconds omitted
- **WHEN** a profile entry in `config.yaml` has no `timeout_seconds` field
- **THEN** the exporter uses a 5 second timeout when scraping that switch

### Requirement: Documented example matches parsed fields
`config.yaml.example` SHALL be valid YAML and SHALL only contain fields that `ProfileConfig` parses (`address`, `username`, `password`, `timeout_seconds`, `poe`, `comments`), structured as a `profiles:` list of named entries.

#### Scenario: Copying the example produces a working config
- **WHEN** a user runs `cp config.yaml.example config.yaml` and fills in real switch details
- **THEN** the resulting file parses without error and the exporter starts and scrapes the configured switch(es)
