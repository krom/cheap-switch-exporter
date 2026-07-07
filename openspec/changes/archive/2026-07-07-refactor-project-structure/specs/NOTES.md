No capability spec changes.

This change is a structural refactor of `main.go` into multiple files within the same
package, plus non-behavioral code-quality cleanup (doc comments, error-message wrapping,
extracting a testable `loadConfig` function). It does not add, modify, remove, or rename any
requirement in `config-loading`, `background-polling`, or `v2-switch-client`.

Archive this change with `openspec archive refactor-project-structure --skip-specs` since there
are no delta specs to apply.
