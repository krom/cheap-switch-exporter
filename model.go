package main

// CounterKind identifies one of the 8 Tx/Rx x Good/Bad x Pkt/Bytes counter columns a
// switch's port.cgi?page=stats table may report. Its string value is also the exact
// header text used to recognize that column, e.g. <th>TxGoodPkt</th>.
type CounterKind string

const (
	TxGoodPkt   CounterKind = "TxGoodPkt"
	TxBadPkt    CounterKind = "TxBadPkt"
	TxGoodBytes CounterKind = "TxGoodBytes"
	TxBadBytes  CounterKind = "TxBadBytes"
	RxGoodPkt   CounterKind = "RxGoodPkt"
	RxBadPkt    CounterKind = "RxBadPkt"
	RxGoodBytes CounterKind = "RxGoodBytes"
	RxBadBytes  CounterKind = "RxBadBytes"
)

// counterMetricNames maps each CounterKind to the Prometheus metric name emitted for it.
var counterMetricNames = map[CounterKind]string{
	TxGoodPkt:   "port_tx_good_pkt",
	TxBadPkt:    "port_tx_bad_pkt",
	TxGoodBytes: "port_tx_good_bytes",
	TxBadBytes:  "port_tx_bad_bytes",
	RxGoodPkt:   "port_rx_good_pkt",
	RxBadPkt:    "port_rx_bad_pkt",
	RxGoodBytes: "port_rx_good_bytes",
	RxBadBytes:  "port_rx_bad_bytes",
}

// counterMetricHelp maps each CounterKind to its Prometheus metric HELP text.
var counterMetricHelp = map[CounterKind]string{
	TxGoodPkt:   "TX good packets",
	TxBadPkt:    "TX bad packets",
	TxGoodBytes: "TX good bytes",
	TxBadBytes:  "TX bad bytes",
	RxGoodPkt:   "RX good packets",
	RxBadPkt:    "RX bad packets",
	RxGoodBytes: "RX good bytes",
	RxBadBytes:  "RX bad bytes",
}

// Port holds one switch port's admin state, link status, and whichever Tx/Rx counters
// that switch's stats page reports.
type Port struct {
	Name       string
	State      string
	LinkStatus string
	// Counters holds only the counter kinds this switch's page actually reported;
	// a kind absent here was not present in the table header, and must not be
	// emitted (not even as 0).
	Counters map[CounterKind]float64
}

// PortStatus holds one switch port's negotiated link speed and duplex, as reported by
// the switch's separate port-status page/endpoint.
type PortStatus struct {
	Name       string
	SpeedMbps  float64
	FullDuplex float64
}

// PoEPort holds one switch port's PoE state, power, class, and electrical readings.
type PoEPort struct {
	Name    string
	State   float64
	Power   float64
	Type    float64
	Watts   float64
	Voltage float64
	Current float64
}
