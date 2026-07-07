## 1. Implementation

- [x] 1.1 Format v2 port names as `"Port " + p.PortId` in `v2Client.FetchPorts` (`main.go`)

## 2. Tests

- [x] 2.1 Update `TestV2ClientFetchPorts` expectations in `main_test.go` to the new `"Port N"`
      names (counters lookup, status-by-name lookup, link/state checks)
- [x] 2.2 Run `go build ./...` and `go test ./...` to confirm the change and existing coverage
      pass

## 3. Docs

- [x] 3.1 Correct the v2 profile `comments` example in `README.md` to use `Port 1` keys instead
      of bare numbers
