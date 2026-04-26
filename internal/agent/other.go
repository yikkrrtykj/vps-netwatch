//go:build !linux

package agent

import (
	"context"
	"os"
	"time"

	"github.com/yikkrrtykj/vps-netwatch/internal/config"
	"github.com/yikkrrtykj/vps-netwatch/internal/model"
)

func LocalStatus(ctx context.Context, cfg config.Agent) model.VPSNodeStatus {
	hostname, _ := os.Hostname()
	name := cfg.Name
	if name == "" {
		name = hostname
	}
	return model.VPSNodeStatus{
		ID:        cfg.ID,
		Name:      name,
		PublicIP:  cfg.PublicIP,
		Labels:    cfg.Labels,
		UpdatedAt: time.Now().UTC(),
		Error:     "host metrics are only implemented for linux agents in this release",
	}
}

