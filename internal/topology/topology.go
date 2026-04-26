package topology

import "github.com/yikkrrtykj/vps-netwatch/internal/model"

func Default() model.Topology {
	return model.Topology{
		Nodes: []model.TopologyNode{
			{ID: "browser", Label: "浏览器", Type: "client", Status: "external"},
			{ID: "dashboard", Label: "Dashboard 主控", Type: "server", Status: "unknown"},
			{ID: "vps-agent", Label: "VPS Agent", Type: "agent", Status: "unknown"},
			{ID: "collector", Label: "可选 Collector", Type: "collector", Status: "optional"},
			{ID: "mihomo-api", Label: "mihomo API", Type: "proxy", Status: "optional"},
			{ID: "probe-targets", Label: "探测目标", Type: "destination", Status: "unknown"},
		},
		Edges: []model.TopologyEdge{
			{From: "browser", To: "dashboard", Label: "查看面板"},
			{From: "vps-agent", To: "dashboard", Label: "主动上报"},
			{From: "collector", To: "dashboard", Label: "连接摘要"},
			{From: "collector", To: "mihomo-api", Label: "只读读取"},
			{From: "vps-agent", To: "probe-targets", Label: "延迟/出口探测"},
		},
	}
}
