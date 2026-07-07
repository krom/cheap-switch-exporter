Captured from a live `SKS3200-8E2X` (10.110.0.19), credentials `admin`/`admin`.

Login (GET, not POST — both fields are `md5(value)` of the plain username/password
separately, not `md5(username+password)` like the current cookie-auth model):

```
GET /authorize?loginusr=21232f297a57a5a743894a0e4a801fc3&loginpwd=21232f297a57a5a743894a0e4a801fc3
```

Response sets two cookies (real stateful session, unlike the current model's stateless
precomputed cookie):

```
Set-Cookie: user=21232f297a57a5a743894a0e4a801fc3
Set-Cookie: session=19fb83480197667b938e863582832882   # random per-login token
```

Port/counter data (GET, with the two cookies above attached), real
`Content-Type: application/json; charset=utf-8` — see `sks3200-8e2x_port_statistics.json`:

```
GET /port_statistics.json
```

Notes for a future parser:
- `Link_Status` combines link state + speed + duplex in one string: `"Link Down"` or
  `"<number><Mbps|Gbps><Full|Half>"` (e.g. `"1000MbpsFull"`, `"2500MbpsFull"`,
  `"10GbpsFull"`, `"100MbpsFull"`). Only Full-duplex/up values have been observed live;
  Half-duplex and 10M/5000M speeds are unconfirmed guesses based on the pattern.
- `Port_Status`, `TxGoodPkt`, `TxBadPkt`, `RxGoodPkt`, `RxBadPkt` map directly onto the
  existing `Port` struct fields.
- No PoE menu/pages exist on this device — this model doesn't support PoE at all.
- No `examples/` HTML fixtures exist for this device family since none of the
  `port.cgi`-style pages exist here; everything is served as JSON from `.json` endpoints.
