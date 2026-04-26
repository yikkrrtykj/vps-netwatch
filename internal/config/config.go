package config

import (
	"errors"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Dashboard  DashboardConfig  `yaml:"dashboard" json:"dashboard"`
	Auth       AuthConfig       `yaml:"auth" json:"auth"`
	Collectors []Collector      `yaml:"collectors" json:"collectors"`
	Agents     []Agent          `yaml:"agents" json:"agents"`
	Mihomo     MihomoConfig     `yaml:"mihomo" json:"mihomo"`
	VPSNodes   []VPSNode        `yaml:"vps_nodes" json:"vps_nodes"`
	Probes     []ProbeTarget    `yaml:"probes" json:"probes"`
}

type DashboardConfig struct {
	Listen    string `yaml:"listen" json:"listen"`
	PublicURL string `yaml:"public_url" json:"public_url"`
	DataPath  string `yaml:"data_path" json:"data_path"`
}

type AuthConfig struct {
	Token string `yaml:"token" json:"-"`
}

type Collector struct {
	ID           string        `yaml:"id" json:"id"`
	DashboardURL string        `yaml:"dashboard_url" json:"dashboard_url"`
	Interval     time.Duration `yaml:"-" json:"interval"`
	IntervalRaw  string        `yaml:"interval" json:"-"`
}

type Agent struct {
	ID           string            `yaml:"id" json:"id"`
	Name         string            `yaml:"name" json:"name"`
	PublicIP     string            `yaml:"public_ip" json:"public_ip"`
	DashboardURL string            `yaml:"dashboard_url" json:"dashboard_url"`
	Interval     time.Duration     `yaml:"-" json:"interval"`
	IntervalRaw  string            `yaml:"interval" json:"-"`
	Labels       map[string]string `yaml:"labels" json:"labels"`
}

type MihomoConfig struct {
	Controllers []MihomoController `yaml:"controllers" json:"controllers"`
}

type MihomoController struct {
	Name    string `yaml:"name" json:"name"`
	BaseURL string `yaml:"base_url" json:"base_url"`
	Secret  string `yaml:"secret" json:"-"`
	Role    string `yaml:"role" json:"role"`
}

type VPSNode struct {
	ID       string            `yaml:"id" json:"id"`
	Name     string            `yaml:"name" json:"name"`
	PublicIP string            `yaml:"public_ip" json:"public_ip"`
	Labels   map[string]string `yaml:"labels" json:"labels"`
}

type ProbeTarget struct {
	Name     string `yaml:"name" json:"name"`
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Protocol string `yaml:"protocol" json:"protocol"`
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	applyDefaults(&cfg)
	if err := normalize(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Default() Config {
	cfg := Config{
		Dashboard: DashboardConfig{
			Listen:   "127.0.0.1:8787",
			DataPath: "./data/vps-netwatch.db",
		},
	}
	applyDefaults(&cfg)
	return cfg
}

func applyDefaults(cfg *Config) {
	if cfg.Dashboard.Listen == "" {
		cfg.Dashboard.Listen = "127.0.0.1:8787"
	}
	if cfg.Dashboard.DataPath == "" {
		cfg.Dashboard.DataPath = "./data/vps-netwatch.db"
	}
	for i := range cfg.Collectors {
		if cfg.Collectors[i].IntervalRaw == "" {
			cfg.Collectors[i].IntervalRaw = "10s"
		}
	}
	for i := range cfg.Agents {
		if cfg.Agents[i].IntervalRaw == "" {
			cfg.Agents[i].IntervalRaw = "10s"
		}
	}
	for i := range cfg.Probes {
		if cfg.Probes[i].Protocol == "" {
			cfg.Probes[i].Protocol = "tcp"
		}
	}
}

func normalize(cfg *Config) error {
	for i := range cfg.Collectors {
		d, err := time.ParseDuration(cfg.Collectors[i].IntervalRaw)
		if err != nil {
			return err
		}
		if d <= 0 {
			return errors.New("collector interval must be positive")
		}
		cfg.Collectors[i].Interval = d
	}
	for i := range cfg.Agents {
		d, err := time.ParseDuration(cfg.Agents[i].IntervalRaw)
		if err != nil {
			return err
		}
		if d <= 0 {
			return errors.New("agent interval must be positive")
		}
		cfg.Agents[i].Interval = d
	}
	return nil
}
