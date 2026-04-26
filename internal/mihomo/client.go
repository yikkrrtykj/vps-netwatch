package mihomo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yikkrrtykj/vps-netwatch/internal/config"
	"github.com/yikkrrtykj/vps-netwatch/internal/model"
)

type Client struct {
	controller config.MihomoController
	httpClient *http.Client
}

func New(controller config.MihomoController) *Client {
	return &Client{
		controller: controller,
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (c *Client) Connections(ctx context.Context) ([]model.Connection, error) {
	var raw connectionsResponse
	if err := c.getJSON(ctx, "/connections", &raw); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]model.Connection, 0, len(raw.Connections))
	for _, item := range raw.Connections {
		out = append(out, normalizeConnection(c.controller.Name, item, now))
	}
	return out, nil
}

func (c *Client) Traffic(ctx context.Context) (model.TrafficSample, error) {
	var raw trafficResponse
	if err := c.getJSON(ctx, "/traffic", &raw); err != nil {
		return model.TrafficSample{}, err
	}
	return model.TrafficSample{
		Controller: c.controller.Name,
		UploadKBps: raw.Up,
		DownKBps:   raw.Down,
		Timestamp:  time.Now().UTC(),
	}, nil
}

func (c *Client) Version(ctx context.Context) (map[string]any, error) {
	var raw map[string]any
	if err := c.getJSON(ctx, "/version", &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) getJSON(ctx context.Context, path string, dest any) error {
	base := strings.TrimRight(c.controller.BaseURL, "/")
	if _, err := url.ParseRequestURI(base); err != nil {
		return fmt.Errorf("invalid mihomo base_url %q: %w", c.controller.BaseURL, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return err
	}
	if c.controller.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.controller.Secret)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("mihomo %s returned %s", path, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

type connectionsResponse struct {
	Connections []rawConnection `json:"connections"`
}

type rawConnection struct {
	ID          string      `json:"id"`
	Metadata    rawMetadata `json:"metadata"`
	Upload      int64       `json:"upload"`
	Download    int64       `json:"download"`
	Start       string      `json:"start"`
	Chains      []string    `json:"chains"`
	Rule        string      `json:"rule"`
	RulePayload string      `json:"rulePayload"`
}

type rawMetadata struct {
	Network         string `json:"network"`
	Type            string `json:"type"`
	SourceIP        string `json:"sourceIP"`
	DestinationIP   string `json:"destinationIP"`
	SourcePort      any    `json:"sourcePort"`
	DestinationPort any    `json:"destinationPort"`
	Host            string `json:"host"`
	Process         string `json:"process"`
	ProcessPath     string `json:"processPath"`
}

type trafficResponse struct {
	Up   int64 `json:"up"`
	Down int64 `json:"down"`
}

func normalizeConnection(controller string, raw rawConnection, now time.Time) model.Connection {
	startedAt := now
	if raw.Start != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, raw.Start); err == nil {
			startedAt = parsed.UTC()
		}
	}
	network := raw.Metadata.Network
	if network == "" {
		network = raw.Metadata.Type
	}
	return model.Connection{
		ID:          raw.ID,
		Controller:  controller,
		Network:     strings.ToLower(network),
		SourceIP:    raw.Metadata.SourceIP,
		SourcePort:  toInt(raw.Metadata.SourcePort),
		DestIP:      raw.Metadata.DestinationIP,
		DestPort:    toInt(raw.Metadata.DestinationPort),
		Host:        raw.Metadata.Host,
		Rule:        raw.Rule,
		RulePayload: raw.RulePayload,
		Chains:      raw.Chains,
		Process:     raw.Metadata.Process,
		ProcessPath: raw.Metadata.ProcessPath,
		Upload:      raw.Upload,
		Download:    raw.Download,
		StartedAt:   startedAt,
		UpdatedAt:   now,
	}
}

func toInt(v any) int {
	switch value := v.(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		n, _ := value.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(value)
		return n
	default:
		return 0
	}
}

