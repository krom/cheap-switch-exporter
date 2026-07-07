# 🔌 Cheap Switch Exporter

Prometheus Exporter for low-cost network switches without SNMP support

## 📖 Overview

This Prometheus exporter retrieves port statistics from switches that lack SNMP functionality, enabling monitoring through a web-based interface.

## 🎯 Purpose

Many budget-friendly network switches do not support standard SNMP monitoring. This exporter provides a workaround by scraping port statistics directly from the switch's web interface.

## 🖥️ Supported Devices

| Manufacturer | Model | Status | Contributor |
|--------------|-------|--------|-------------|
| Ampcom | WAMJHJ-8125MNG | ✅ Verified | @askainet |
| Horaco | ZX-SWTGW215AS | ✅ Verified | @askainet |
| Horaco | ZX-SWTGW218AS | ✅ Verified | @pvelati |
| Horaco | HC-SWTGW218AS |  ✅ Verified | @arthurbarton |
| Horaco | HC-SWTGW124AS |  ✅ Verified | @arthurbarton |
| KeepLink | KP-9000-9XHPML-X | ✅ Verified | @jfallot and @adamchabin |
| Sodola | SL-SWTG124AS | ✅ Verified | @dennyreiter |
| Anhui Seeker | SKS3200-8E2X | ✅ Verified (`profile: v2`) | - |

## 🚀 Installation

### Prerequisites

- Go 1.23+
- Docker (optional)

### Direct Installation

1. Clone the repository
2. Download dependencies
```bash
go mod download
```

3. Copy configuration template
```bash
cp config.yaml.example config.yaml
```

4. Edit `config.yaml` with your switch details and parameters
5. Run the exporter
```bash
go run main.go
```

### Docker Deployment

```bash
# Build Docker image
docker build -t cheap-switch-exporter .

# Run container
docker run -v "./config.yaml:/config.yaml" -p 8080:8080 cheap-switch-exporter
```

## 📝 Configuration

`config.yaml` defines a `profiles:` list. Each entry is a single-key map from a profile
name (used as the `switch` label in metrics) to that profile's settings, so one exporter
instance can poll multiple physical switches at once:

```yaml
profiles:
  - desk_switch:                   # profile name -> used as the "switch" label
      address: "192.168.1.1"       # IP or hostname of the switch
      username: "admin"            # Web interface username
      password: "password"         # Web interface password
      timeout_seconds: 5           # Per-request timeout, overrides settings.timeout_seconds
      poll_rate_seconds: 30        # Poll interval, overrides settings.poll_rate_seconds
      poe: 0                       # Enable PoE page scrape (1 = enabled)
      comments:                    # Optional per-port labels, keyed by port name
        Port 1: Port 1
        Port 2: Port 2
        Port 6: Core Switch

  - core_switch:                   # add more profiles to monitor more switches
      address: "192.168.1.2"
      username: "admin"
      password: "password"
      poe: 1
      comments:
        Port 1: Uplink

  - v2_switch:                     # a "v2"-dialect switch (e.g. SKS3200-8E2X): JSON status
      profile: v2                  # endpoint + real login, instead of HTML+static cookie
      address: "192.168.1.3"
      username: "admin"
      password: "password"
      comments:                    # v2 port names are plain numbers (JSON "Port_Id"),
        1: Uplink                  # not "Port 1"

settings:                          # optional; defaults applied to any profile that
  poll_rate_seconds: 60            # doesn't set its own value (hardcoded defaults:
  timeout_seconds: 10              # 60s poll rate, 10s timeout, if omitted entirely)
```

Each switch is polled in the background on its own `poll_rate_seconds` interval (not on every
Prometheus scrape), and `/metrics` is served from the most recently polled data in memory.
`timeout_seconds` bounds each individual request to that switch. Both values are resolved with
the same precedence: the profile's own value, if set, else the global `settings` value, if set,
else the hardcoded default.

### `profile` (switch dialect)

`profile` selects which switch dialect the exporter uses to talk to that entry's device:

- Omitted, `""`, or `"default"` — the original dialect (stateless cookie auth, HTML tables).
- `"v2"` — a second device family (e.g. `SKS3200-8E2X`) that logs in via a real session and
  exposes port/counter data as JSON instead of HTML. PoE is not supported on this dialect;
  `poe: 1` combined with `profile: v2` fails at startup.

Any other value fails at startup. This is purely a per-profile scrape mechanism — metric
names and labels are unaffected by which dialect a profile uses.

## 📊 Exposed Metrics

- `port_state`: Port enabled/disabled status
- `port_link_status`: Port link up/down status
- `port_tx_good_pkt`: Transmitted good packets
- `port_tx_bad_pkt`: Transmitted bad packets
- `port_rx_good_pkt`: Received good packets
- `port_rx_bad_pkt`: Received bad packets
- `port_tx_good_bytes`: Transmitted good bytes
- `port_tx_bad_bytes`: Transmitted bad bytes
- `port_rx_good_bytes`: Received good bytes
- `port_rx_bad_bytes`: Received bad bytes

Not every switch model reports all 8 counters above — only the ones present on that
switch's page are exposed (e.g. some models report packet counts only, others report a mix
of packet and byte counts).

### PoE metrics (when enabled in config)

- `poe_port_power_on`: PoE port power on/off 
- `poe_port_state`: State of the PoE port (1=Enable, 0=Disable)
- `poe_port_type`: PoE port type class
- `poe_port_voltage`: PoE port voltage in volts
- `poe_port_watts`: PoE port power consumption in watts
- `poe_port_current_ma`: PoE port current in mA
- `poe_system_consumption_watts`: Total PoE consumption in watts

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a new Pull Request

## 🚨 Limitations

- Requires web interface access to the switch
- Polling-based metrics collection
- Authentication via web interface credentials
- No TLS

## 📄 License

MIT License, see [LICENSE](LICENSE) file.

## 🐛 Issues

Report issues on the GitHub repository's issue tracker.
