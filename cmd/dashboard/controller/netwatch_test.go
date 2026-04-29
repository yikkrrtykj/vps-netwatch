package controller

import (
	"testing"
	"time"

	"github.com/nezhahq/nezha/model"
)

func TestNetwatchNormalizeMonitorTarget(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantTarget string
		wantType   uint8
	}{
		{
			name:       "host becomes icmp",
			input:      "1.1.1.1",
			wantTarget: "1.1.1.1",
			wantType:   model.TaskTypeICMPPing,
		},
		{
			name:       "host port becomes tcp",
			input:      "example.com:443",
			wantTarget: "example.com:443",
			wantType:   model.TaskTypeTCPPing,
		},
		{
			name:       "url keeps host port",
			input:      "https://example.com:8443/path",
			wantTarget: "example.com:8443",
			wantType:   model.TaskTypeTCPPing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTarget, gotType, err := netwatchNormalizeMonitorTarget(tt.input)
			if err != nil {
				t.Fatalf("netwatchNormalizeMonitorTarget() error = %v", err)
			}
			if gotTarget != tt.wantTarget || gotType != tt.wantType {
				t.Fatalf("netwatchNormalizeMonitorTarget() = %q/%d, want %q/%d", gotTarget, gotType, tt.wantTarget, tt.wantType)
			}
		})
	}
}

func TestNetwatchServerMetadataLabels(t *testing.T) {
	if got := netwatchServerBandwidthFromText("香港 NHK-Lite@1Gbps"); got != "1Gbps" {
		t.Fatalf("bandwidth from name = %q, want 1Gbps", got)
	}
	if got := netwatchServerBandwidthFromText("bandwidth=500Mbps, expire=2026-05-02"); got != "500Mbps" {
		t.Fatalf("bandwidth from note = %q, want 500Mbps", got)
	}

	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.Local)
	if got := netwatchServerRemainingFromText("expire=2026-05-01 bandwidth=1Gbps", now); got != "余 3 天" {
		t.Fatalf("remaining from expire = %q, want 余 3 天", got)
	}
	if got := netwatchServerRemainingFromText("剩余=22天, 带宽=500Mbps", now); got != "余 22 天" {
		t.Fatalf("remaining from days = %q, want 余 22 天", got)
	}
}
