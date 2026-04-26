package topology

import "github.com/yikkrrtykj/vps-netwatch/internal/model"

func Default() model.Topology {
	return model.Topology{
		Nodes: []model.TopologyNode{
			{ID: "terminal", Label: "终端/游戏电脑", Type: "client", Status: "unknown"},
			{ID: "local-mihomo", Label: "本地 mihomo", Type: "proxy", Status: "unknown"},
			{ID: "proxy-vm", Label: "Windows 代理 VM", Type: "proxy", Status: "unknown"},
			{ID: "cisco-c3850", Label: "Cisco C3850 核心", Type: "switch", Status: "external"},
			{ID: "watchguard-m670", Label: "WatchGuard M670", Type: "firewall", Status: "external"},
			{ID: "mikrotik", Label: "MikroTik", Type: "router", Status: "external"},
			{ID: "isp", Label: "ISP", Type: "wan", Status: "external"},
			{ID: "vps", Label: "VPS/Dashboard", Type: "server", Status: "unknown"},
			{ID: "game-server", Label: "目标游戏服务器", Type: "destination", Status: "unknown"},
		},
		Edges: []model.TopologyEdge{
			{From: "terminal", To: "local-mihomo", Label: "本机代理可选"},
			{From: "terminal", To: "proxy-vm", Label: "内网代理"},
			{From: "proxy-vm", To: "cisco-c3850"},
			{From: "cisco-c3850", To: "watchguard-m670"},
			{From: "watchguard-m670", To: "mikrotik"},
			{From: "mikrotik", To: "isp"},
			{From: "isp", To: "vps"},
			{From: "vps", To: "game-server", Label: "探测/出口"},
		},
	}
}

