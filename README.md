# vps-netwatch

`vps-netwatch` 是一个独立的 VPS/代理/游戏网络观测项目。默认形态类似哪吒监控：一台 VPS 跑 `dashboard` 做总面板，其他 VPS 跑轻量 `agent` 接入；内网代理 VM 可选跑 `collector`，用来读取 mihomo/Clash 连接。

## 架构

- `dashboard`：主控面板，运行在一台主 VPS 上，负责 Web UI、API、SQLite 数据库和所有 VPS 总览。
- `agent`：探针，运行在每台被监控 VPS 上，主动上报 CPU、内存、磁盘、网卡流量、uptime、sing-box 状态和出口 IP。
- `collector`：可选组件，运行在内网代理 VM 上，只读读取 mihomo API，用来分析连接目标 IP、端口、规则和代理链路。
- `终端/游戏电脑`：默认不安装任何东西，不抓包，不装驱动。

## 第一版能力

- VPS 监控：多台 VPS agent 自动接入总面板。
- 出口检测：agent/collector 上报当前公网出口 IP。
- 延迟诊断：对配置里的目标做 TCP connect RTT。
- mihomo 连接：可选读取 `/connections` 和 `/traffic`，显示目标 IP/端口、协议、规则、代理链路、进程名和实时上下行。

## 主控 VPS

主控就是你打开网页看的那台 VPS。没有域名也可以，先用公网 IP。

```bash
git clone https://github.com/yikkrrtykj/vps-netwatch.git
cd vps-netwatch
nano config.yaml
```

最小 `config.yaml`：

```yaml
dashboard:
  listen: "0.0.0.0:8787"
  public_url: "http://主控VPS公网IP:8787"
  data_path: "./data/vps-netwatch.db"

auth:
  token: "改成一个很长的随机密码"

mihomo:
  controllers: []

vps_nodes: []
probes: []
```

然后运行主控：

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

浏览器访问：

```text
http://主控VPS公网IP:8787
```

`config.example.yaml` 只是参考模板，不一定要先 `cp config.example.yaml config.yaml`。你可以直接新建 `config.yaml` 并粘贴上面的最小配置。

## 新增 VPS 探针

新买一台 VPS 后，不需要先在总面板手动新增 agent。只要这台 VPS 跑 agent，并且使用同一个 `token`，它就会自动出现在主控面板。

在新 VPS 上：

```bash
git clone https://github.com/yikkrrtykj/vps-netwatch.git
cd vps-netwatch
nano config.yaml
```

最小 agent 配置：

```yaml
dashboard:
  public_url: "http://主控VPS公网IP:8787"

auth:
  token: "和主控一样的token"
```

编译并运行：

```bash
go build -o bin/agent ./cmd/agent
./bin/agent -config config.yaml -id jp-vps-01
```

`jp-vps-01` 换成这台 VPS 的唯一名字，比如 `hk-vps-01`、`sg-vps-02`、`us-vps-01`。

长期运行可以用 systemd：

```bash
go build -o bin/agent ./cmd/agent
sudo mkdir -p /opt/vps-netwatch
sudo cp -r bin config.yaml /opt/vps-netwatch/
sudo cp deploy/vps-netwatch-agent.service /etc/systemd/system/
sudo sed -i 's/-id main-vps/-id jp-vps-01/' /etc/systemd/system/vps-netwatch-agent.service
sudo systemctl daemon-reload
sudo systemctl enable --now vps-netwatch-agent
```

## 内网代理 VM 可选

如果你只想做多 VPS 监控，可以先不部署 collector。

如果你想看 mihomo/Clash 连接、游戏服务器 IP、代理链路，就在内网代理 VM 上跑 collector。你的代理 VM 现在是 Windows 也能用，只要它能访问 mihomo external-controller。

如果你习惯 Ubuntu，我更推荐把代理 VM 换成 Ubuntu：

- 更适合长期运行服务，systemd 管理省心。
- mihomo、collector、日志、自动重启都更好维护。
- 不需要 Windows GUI，资源占用更低。
- 以后做网络工具和排障命令也更顺手。

但如果现有 Windows 代理已经很稳定，不需要为了第一版强行迁移。`collector` 的原则很简单：它跑在哪里都可以，只要能访问 mihomo API，并能访问主控 `dashboard.public_url`。

Ubuntu 代理 VM 上的 collector 最小配置：

```yaml
dashboard:
  public_url: "http://主控VPS公网IP:8787"

auth:
  token: "和主控一样的token"

collectors:
  - id: "lan-proxy-vm"
    dashboard_url: "http://主控VPS公网IP:8787"
    interval: "10s"

mihomo:
  controllers:
    - name: "proxy-vm"
      base_url: "http://127.0.0.1:9090"
      secret: ""
      role: "proxy"
```

运行：

```bash
go build -o bin/collector ./cmd/collector
./bin/collector -config config.yaml -id lan-proxy-vm
```

## 哪些 IP 和 URL 要改

- `主控VPS公网IP`：必须改成主控 VPS 的公网 IP。
- `auth.token`：必须改，主控、agent、collector 三边保持一致。
- `dashboard.public_url`：没有域名就用 `http://主控VPS公网IP:8787`。
- 有域名和反代后，再改成 `https://你的域名`。
- `agents` 和 `vps_nodes` 都不是必填；新 VPS agent 能自动上报。
- `probes` 可先留空，后续再加游戏服务器 IP 或关键节点。

## API

- `GET /api/connections`
- `GET /api/egress`
- `GET /api/latency`
- `GET /api/vps/nodes`
- `GET /api/errors`
- `POST /api/collector/v1/push`

带 token 时使用：

```bash
curl -H "Authorization: Bearer 你的token" http://主控VPS公网IP:8787/api/vps/nodes
```

## 安全边界

- 不要把 mihomo controller 暴露到公网。
- agent/collector 都是主动推送到 dashboard，不需要 dashboard 反连内网。
- 默认不保存包内容，只保存连接元数据、VPS 状态和探测结果。
- 如果终端本地 mihomo 不开放 controller，第一版只能看到代理 VM/VPS 能看到的连接。
