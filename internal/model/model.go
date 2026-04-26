package model

import "time"

type Connection struct {
	ID          string    `json:"id"`
	Controller  string    `json:"controller"`
	Network     string    `json:"network"`
	SourceIP    string    `json:"source_ip"`
	SourcePort  int       `json:"source_port"`
	DestIP      string    `json:"dest_ip"`
	DestPort    int       `json:"dest_port"`
	Host        string    `json:"host"`
	Rule        string    `json:"rule"`
	RulePayload string    `json:"rule_payload"`
	Chains      []string  `json:"chains"`
	Process     string    `json:"process"`
	ProcessPath string    `json:"process_path"`
	Upload      int64     `json:"upload"`
	Download    int64     `json:"download"`
	StartedAt   time.Time `json:"started_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TrafficSample struct {
	Controller string    `json:"controller"`
	UploadKBps int64     `json:"upload_kbps"`
	DownKBps   int64     `json:"down_kbps"`
	Timestamp  time.Time `json:"timestamp"`
}

type EgressResult struct {
	CollectorID string    `json:"collector_id"`
	IP          string    `json:"ip"`
	IPv6        string    `json:"ipv6,omitempty"`
	ASN         string    `json:"asn,omitempty"`
	Country     string    `json:"country,omitempty"`
	Source      string    `json:"source"`
	CheckedAt   time.Time `json:"checked_at"`
	Error       string    `json:"error,omitempty"`
}

type ProbeResult struct {
	Name      string        `json:"name"`
	Host      string        `json:"host"`
	Port      int           `json:"port"`
	Protocol  string        `json:"protocol"`
	RTT        time.Duration `json:"rtt"`
	RTTMillis  float64       `json:"rtt_ms"`
	OK        bool          `json:"ok"`
	Error     string        `json:"error,omitempty"`
	CheckedAt time.Time     `json:"checked_at"`
}

type VPSNodeStatus struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	PublicIP  string            `json:"public_ip"`
	Labels    map[string]string `json:"labels"`
	CPU       float64           `json:"cpu"`
	Memory    ResourceUsage     `json:"memory"`
	Disk      ResourceUsage     `json:"disk"`
	Net       NetworkUsage      `json:"net"`
	UptimeSec int64             `json:"uptime_sec"`
	Services  []ServiceStatus   `json:"services"`
	UpdatedAt time.Time         `json:"updated_at"`
	Error     string            `json:"error,omitempty"`
}

type ResourceUsage struct {
	Used  uint64  `json:"used"`
	Total uint64  `json:"total"`
	Ratio float64 `json:"ratio"`
}

type NetworkUsage struct {
	RXBytes uint64 `json:"rx_bytes"`
	TXBytes uint64 `json:"tx_bytes"`
}

type ServiceStatus struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
	Error  string `json:"error,omitempty"`
}

type TopologyNode struct {
	ID     string            `json:"id"`
	Label  string            `json:"label"`
	Type   string            `json:"type"`
	Status string            `json:"status"`
	Meta   map[string]string `json:"meta,omitempty"`
}

type TopologyEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label,omitempty"`
}

type Topology struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

type CollectorPush struct {
	CollectorID string             `json:"collector_id"`
	Timestamp   time.Time          `json:"timestamp"`
	Connections []Connection       `json:"connections"`
	Traffic     []TrafficSample    `json:"traffic"`
	Egress      *EgressResult      `json:"egress,omitempty"`
	Latency     []ProbeResult      `json:"latency"`
	VPSNodes    []VPSNodeStatus    `json:"vps_nodes"`
	Topology    *Topology          `json:"topology,omitempty"`
	Errors      []CollectorError   `json:"errors,omitempty"`
}

type CollectorError struct {
	Source    string    `json:"source"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
