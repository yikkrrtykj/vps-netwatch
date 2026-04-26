# vps-netwatch

`vps-netwatch` 是一个独立的 VPS/代理/游戏网络观测项目。默认形态类似哪吒监控：一个 dashboard 看总面板，多台 VPS 跑轻量 agent 接入；同时额外补齐 mihomo/Clash 连接、游戏服务器目标 IP、出口检测和延迟诊断。

## 架构

- `dashboard`：运行在一台主 VPS 上，提供 Web 面板、API、SQLite 历史记录和所有节点总览。
- `agent`：运行在每台接入 VPS 上，采集 CPU、内存、磁盘、网卡流量、uptime、sing-box 状态，并主动推送到 dashboard。
- `collector`：可选，运行在内网 Windows 代理 VM 或 Linux VM，只读读取 mihomo API，帮助分析游戏连接和出口。
- `终端/游戏电脑`：默认不安装 agent、不抓包、不装驱动。

## 第一版能力

- 实时连接：读取 mihomo `/connections`，展示目标 IP/端口、协议、规则、代理链路、进程名和实时上下行。
- 出口检测：collector 检测当前公网出口 IP，dashboard 显示最近上报结果。
- 延迟诊断：对配置里的目标做 TCP connect RTT；ICMP/UDP 后续按权限和设备能力扩展。
- VPS 监控：接收多台 VPS agent 上报的 CPU、内存、磁盘、流量、uptime、sing-box 状态。
- 接入拓扑：默认展示 dashboard、VPS agent、可选 collector、mihomo API 和探测目标之间的关系；不写入任何私有内网设备型号。

## 快速开始

```bash
cp config.example.yaml config.yaml
go mod tidy
go run ./cmd/dashboard -config config.yaml
```

在内网代理 VM 上运行 collector：

```bash
go run ./cmd/collector -config config.yaml
```

在其他 VPS 上运行 agent 接入总面板：

```bash
go run ./cmd/agent -config config.yaml -id hk-vps-01
```

前端开发：

```bash
cd web
npm install
npm run dev
```

生产构建后，`dashboard` 会优先服务 `web/dist` 静态文件：

```bash
cd web && npm run build
cd ..
go build -o bin/dashboard ./cmd/dashboard
go build -o bin/agent ./cmd/agent
go build -o bin/collector ./cmd/collector
```

Docker Compose 运行主控：

```bash
cp config.example.yaml config.yaml
docker compose -f deploy/docker-compose.yml up -d --build
```

## 配置说明

核心配置在 `config.yaml`：

- `dashboard.listen`：dashboard 监听地址；Docker 部署可用 `0.0.0.0:8787`，只走本机反代时可改成 `127.0.0.1:8787`。
- `dashboard.public_url`：collector 推送目标。
- `auth.token`：dashboard API 和 collector push 使用的 Bearer token。
- `agents`：多台 VPS agent 的接入配置。
- `mihomo.controllers`：一个或多个 mihomo external-controller，只读访问，可选。
- `vps_nodes`：VPS 节点定义。
- `probes`：要主动检测延迟的目标。

## API

- `GET /api/connections`
- `GET /api/egress`
- `GET /api/latency`
- `GET /api/vps/nodes`
- `GET /api/topology`
- `GET /api/errors`
- `POST /api/collector/v1/push`

带 token 时使用：

```bash
curl -H "Authorization: Bearer change-me" http://127.0.0.1:8787/api/connections
```

## 安全边界

- mihomo controller 不建议暴露到公网。
- agent/collector 都是主动推送到 dashboard，不需要 dashboard 反连内网或其他 VPS。
- collector 只读读取 mihomo，并由内网主动推送到 VPS。
- 默认不保存包内容，只保存连接元数据、VPS 状态和探测结果。
- 如果终端本地 mihomo 不开放 controller，第一版只能看到代理 VM/VPS 能看到的连接。

## 和哪吒的关系

- 哪吒是成熟方案，本项目采用相同的主控 + agent 接入思路，但额外面向代理出口、mihomo 连接和游戏延迟排障。
