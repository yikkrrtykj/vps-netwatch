package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/yikkrrtykj/vps-netwatch/internal/config"
	"github.com/yikkrrtykj/vps-netwatch/internal/model"
)

func TCP(ctx context.Context, target config.ProbeTarget) model.ProbeResult {
	start := time.Now()
	address := net.JoinHostPort(target.Host, fmt.Sprint(target.Port))
	dialer := net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	rtt := time.Since(start)
	result := model.ProbeResult{
		Name:      target.Name,
		Host:      target.Host,
		Port:      target.Port,
		Protocol:  "tcp",
		RTT:       rtt,
		RTTMillis: float64(rtt.Microseconds()) / 1000,
		OK:        err == nil,
		CheckedAt: time.Now().UTC(),
	}
	if err != nil {
		result.Error = err.Error()
		return result
	}
	_ = conn.Close()
	return result
}

func RunAll(ctx context.Context, targets []config.ProbeTarget) []model.ProbeResult {
	results := make([]model.ProbeResult, 0, len(targets))
	for _, target := range targets {
		switch target.Protocol {
		case "", "tcp":
			results = append(results, TCP(ctx, target))
		default:
			results = append(results, model.ProbeResult{
				Name:      target.Name,
				Host:      target.Host,
				Port:      target.Port,
				Protocol:  target.Protocol,
				OK:        false,
				Error:     "unsupported probe protocol in first release",
				CheckedAt: time.Now().UTC(),
			})
		}
	}
	return results
}

func Egress(ctx context.Context) model.EgressResult {
	result := model.EgressResult{
		Source:    "https://api.ipify.org",
		CheckedAt: time.Now().UTC(),
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org?format=json", nil)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	resp, err := (&http.Client{Timeout: 6 * time.Second}).Do(req)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()
	var payload struct {
		IP string `json:"ip"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		result.Error = err.Error()
		return result
	}
	result.IP = payload.IP
	return result
}

