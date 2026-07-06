## ADDED Requirements

### Requirement: Split 32-bit-half counter recombination
When a numeric table cell's text matches the pattern `<high>-<low>` (decimal digits, a single
hyphen, decimal digits), the exporter SHALL treat it as a 64-bit counter split into two 32-bit
halves and compute its value as `high * 4294967296 + low`, matching the value the switch's own
page-embedded JavaScript would compute.

#### Scenario: Non-zero high half
- **WHEN** a cell's text is `"1-901525430"`
- **THEN** the parsed value is `5202268557` (i.e. `1*4294967296 + 901525430`)

#### Scenario: Zero high half
- **WHEN** a cell's text is `"0-2549448529"`
- **THEN** the parsed value is `2549448529`

### Requirement: Plain numeric values unaffected
When a cell's text does not match the `<high>-<low>` split-counter pattern, the exporter SHALL
parse it the same way it did before this change: as a plain decimal number, or as `0` if empty,
`"-"`, or otherwise not parseable.

#### Scenario: Plain decimal counter
- **WHEN** a cell's text is `"71817"`
- **THEN** the parsed value is `71817`

#### Scenario: Empty or placeholder value
- **WHEN** a cell's text is `""` or `"-"`
- **THEN** the parsed value is `0`
