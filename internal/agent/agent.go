package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/yikkrrtykj/vps-netwatch/internal/config"
	"github.com/yikkrrtykj/vps-netwatch/internal/model"
	"github.com/yikkrrtykj/vps-netwatch/internal/probe"
)

type Runner struct {
	cfg        config.Config
	httpClient *http.Client
}

func New(cfg config.Config) *Runner {
	return &Runner{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (r *Runner) Collect(ctx context.Context, agentID string) model.CollectorPush {
	agentCfg := Find(r.cfg, agentID)
	status := LocalStatus(ctx, agentCfg)
	egress := probe.Egress(ctx)
	egress.CollectorID = agentID
	if status.PublicIP == "" && egress.IP != "" {
		status.PublicIP = egress.IP
	}

	return model.CollectorPush{
		CollectorID: agentID,
		Timestamp:   time.Now().UTC(),
		Egress:      &egress,
		Latency:     tagProbeResults(probe.RunAll(ctx, r.cfg.Probes), agentID),
		VPSNodes:    []model.VPSNodeStatus{status},
	}
}

func (r *Runner) Push(ctx context.Context, push model.CollectorPush, dashboardURL string) error {
	if dashboardURL == "" {
		return nil
	}
	data, err := json.Marshal(push)
	if err != nil {
		return err
	}
	endpoint := strings.TrimRight(dashboardURL, "/") + "/api/collector/v1/push"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if r.cfg.Auth.Token != "" {
		req.Header.Set("Authorization", "Bearer "+r.cfg.Auth.Token)
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("dashboard push returned %s", resp.Status)
	}
	return nil
}

func Find(cfg config.Config, agentID string) config.Agent {
	for _, item := range cfg.Agents {
		if item.ID == agentID {
			return item
		}
	}
	hostname, _ := os.Hostname()
	return config.Agent{
		ID:           agentID,
		Name:         hostname,
		DashboardURL: cfg.Dashboard.PublicURL,
		IntervalRaw:  "10s",
		Interval:     10 * time.Second,
	}
}

func DashboardURL(cfg config.Config, agentID string) string {
	item := Find(cfg, agentID)
	if item.DashboardURL != "" {
		return item.DashboardURL
	}
	return cfg.Dashboard.PublicURL
}

func Interval(cfg config.Config, agentID string) time.Duration {
	item := Find(cfg, agentID)
	if item.Interval > 0 {
		return item.Interval
	}
	return 10 * time.Second
}

func tagProbeResults(results []model.ProbeResult, collectorID string) []model.ProbeResult {
	for i := range results {
		results[i].CollectorID = collectorID
	}
	return results
}
