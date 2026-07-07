## Context

`main.go`'s `RootConfig`/`ProfileConfig` types and `main()` config-loading loop already fully implement the multi-profile format (see `openspec/changes/update-to-config/specs/config-loading/spec.md`). `config.yaml.example` and README predate this and still reflect an older single-device, flat-field format, so they no longer match reality:

- `config.yaml.example` is missing a colon after `profiles` (invalid YAML) and lists `profile` and `poll_rate_seconds`, neither of which `ProfileConfig` parses (`Profile` is an unused struct field; `poll_rate_seconds` doesn't exist on the struct at all).
- README's "Configuration" section shows top-level `address`/`username`/`password`/`poll_rate_seconds` keys with no `profiles:` wrapper.

## Goals / Non-Goals

**Goals:**
- Make `config.yaml.example` valid YAML that, when copied, produces a config `main.go` can load and use as-is (aside from filling in real credentials).
- Make README's Configuration section describe the actual `profiles:` list structure, including `comments` and `poe`.

**Non-Goals:**
- No changes to `main.go`, `ProfileConfig`, or scrape/collector logic.
- Not deciding the fate of the unused `Profile` field or a `poll_rate_seconds` feature — out of scope per the scoping decision to keep this docs/example-only.

## Decisions

- Base the example on the real-world shape already in the (gitignored) working `config.yaml`, since that's proven to parse and run correctly, rather than inventing a new example from scratch.
- Drop `profile` and `poll_rate_seconds` from the example entirely rather than keeping them as unused placeholders, since dead fields in an example file mislead new users into thinking they do something.
- Keep the README example minimal (one profile) but explicitly show a second profile stub in prose/comment so multi-profile usage is obvious without bloating the snippet.

## Risks / Trade-offs

- [Risk] Someone relies on the currently-undocumented `profile` field expecting it to do something → Mitigation: it's already unused by `main.go`'s logic (only `Name` from the map key and the rest of `ProfileConfig` are used), so removing it from the example has no behavioral effect.
- [Risk] Docs drift again next time `ProfileConfig` changes → out of scope to fix generally here, but the new `config-loading` spec gives future changes something concrete to update.
