## 1. Fix parseNum

- [x] 1.1 Replace the `"0-"`-prefix special case in `parseNum` (main.go:361) with a general
      `^\d+-\d+$` check: split on `-`, parse both halves as integers, and return
      `high*4294967296 + low` as `float64`.
- [x] 1.2 Keep the existing fallback (`strconv.ParseFloat`, with `""`/`"-"` → `0`) for strings
      that don't match the split-counter pattern.

## 2. Verify

- [x] 2.1 Confirm `1-901525430` parses to `5196492726` and `1-907301261` parses to `5202268557`
      (matches the recombination the switch's own inline JS in `examples/2.html` performs).
- [x] 2.2 Confirm plain values are unaffected: `"71817"` → `71817`, `""` → `0`, `"-"` → `0`.
- [x] 2.3 Run `go vet ./...` and `go build -o cheap-switch-exporter .` to confirm no regressions.
