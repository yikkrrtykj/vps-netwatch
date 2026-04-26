package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/yikkrrtykj/vps-netwatch/internal/config"
	"github.com/yikkrrtykj/vps-netwatch/internal/mihomo"
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

func (r *Runner) Collect(ctx context.Context, collectorID string) model.CollectorPush {
	push := model.CollectorPush{
		CollectorID: collectorID,
		Timestamp:   time.Now().UTC(),
	}

	for _, ctrl := range r.cfg.Mihomo.Controllers {
		client := mihomo.New(ctrl)
		connections, err := client.Connections(ctx)
		if err == nil {
			push.ConnectionControllers = append(push.ConnectionControllers, ctrl.Name)
			push.Connections = append(push.Connections, connections...)
		} else {
			push.Errors = append(push.Errors, model.CollectorError{
				Source:    "mihomo:" + ctrl.Name,
				Message:   err.Error(),
				Timestamp: time.Now().UTC(),
			})
		}

		traffic, err := client.Traffic(ctx)
		if err == nil {
			push.Traffic = append(push.Traffic, traffic)
		}
	}

	egress := probe.Egress(ctx)
	egress.CollectorID = collectorID
	push.Egress = &egress
	push.Latency = probe.RunAll(ctx, r.cfg.Probes)
	tagProbeResults(push.Latency, collectorID)
	return push
}

func (r *Runner) Push(ctx context.Context, push model.CollectorPush, dashboardURL string) error {
	if dashboardURL == "" {
		return nil
	}
	endpoint := strings.TrimRight(dashboardURL, "/") + "/api/collector/v1/push"
	data, err := json.Marshal(push)
	if err != nil {
		return err
	}
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

func DashboardURL(cfg config.Config, collectorID string) string {
	for _, item := range cfg.Collectors {
		if item.ID == collectorID && item.DashboardURL != "" {
			return item.DashboardURL
		}
	}
	if len(cfg.Collectors) > 0 && cfg.Collectors[0].DashboardURL != "" {
		return cfg.Collectors[0].DashboardURL
	}
	return cfg.Dashboard.PublicURL
}

func Interval(cfg config.Config, collectorID string) time.Duration {
	for _, item := range cfg.Collectors {
		if item.ID == collectorID && item.Interval > 0 {
			return item.Interval
		}
	}
	if len(cfg.Collectors) > 0 && cfg.Collectors[0].Interval > 0 {
		return cfg.Collectors[0].Interval
	}
	return 10 * time.Second
}

func tagProbeResults(results []model.ProbeResult, collectorID string) {
	for i := range results {
		results[i].CollectorID = collectorID
	}
}
