package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsAndAgentIntervals(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := []byte(`
dashboard:
  public_url: "https://watch.example.com"
auth:
  token: "secret"
agents:
  - id: "hk"
    name: "HK"
collectors:
  - id: "lan"
    interval: "15s"
`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Dashboard.Listen != "127.0.0.1:8787" {
		t.Fatalf("unexpected listen: %s", cfg.Dashboard.Listen)
	}
	if cfg.Agents[0].IntervalRaw != "10s" || cfg.Agents[0].Interval.Seconds() != 10 {
		t.Fatalf("agent interval was not defaulted: %#v", cfg.Agents[0])
	}
	if cfg.Collectors[0].Interval.Seconds() != 15 {
		t.Fatalf("collector interval was not parsed: %#v", cfg.Collectors[0])
	}
}

