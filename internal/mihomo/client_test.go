package mihomo

import (
	"testing"
	"time"
)

func TestNormalizeConnection(t *testing.T) {
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	conn := normalizeConnection("proxy-vm", rawConnection{
		ID: "abc",
		Metadata: rawMetadata{
			Network:         "TCP",
			SourceIP:        "192.168.10.20",
			SourcePort:      "53000",
			DestinationIP:   "203.0.113.7",
			DestinationPort: float64(443),
			Host:            "game.example.com",
			Process:         "game.exe",
		},
		Rule:   "MATCH",
		Chains: []string{"Proxy", "HK"},
		Start:  "2026-04-26T11:59:00Z",
	}, now)

	if conn.Controller != "proxy-vm" || conn.Network != "tcp" {
		t.Fatalf("unexpected connection metadata: %#v", conn)
	}
	if conn.SourcePort != 53000 || conn.DestPort != 443 {
		t.Fatalf("unexpected ports: %#v", conn)
	}
	if conn.Process != "game.exe" || conn.Chains[1] != "HK" {
		t.Fatalf("lost process or chain data: %#v", conn)
	}
}

