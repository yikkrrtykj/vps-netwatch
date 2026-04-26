# vps-netwatch

[![Build vps-netwatch image](https://github.com/yikkrrtykj/vps-netwatch/actions/workflows/vps-netwatch-image.yml/badge.svg)](https://github.com/yikkrrtykj/vps-netwatch/actions/workflows/vps-netwatch-image.yml)

`vps-netwatch` 是我的 VPS 网络监控面板，基于 Nezha Dashboard 二次开发。

当前重点不是重写服务器监控，而是在哪吒已有的 Dashboard、Agent、Web 终端、服务监控能力上，增加更适合 VPS 线路观察的延迟视图。

## 功能

- 使用哪吒 Agent 接入多台 VPS。
- 保留哪吒原有服务器状态、流量、CPU、内存、磁盘和 Web 终端。
- 默认启用 TSDB，用于保存历史监控数据。
- 首页视图按钮旁增加“延迟”入口。
- `/dashboard/netwatch/latency` 延迟面板按目标聚合曲线，例如只显示 `上海电信`、`上海联通` 两条线。
- 延迟图支持鼠标悬停查看具体数值，鼠标滚轮缩放时间范围，双击恢复完整时间范围。

## 镜像

```text
ghcr.io/yikkrrtykj/vps-netwatch:latest
ghcr.io/yikkrrtykj/vps-netwatch:main
```

每次推送到 `main` 后，GitHub Actions 会自动构建并推送 `linux/amd64` 和 `linux/arm64` 镜像。

## 安装 Dashboard

以下以 Ubuntu 22.04、root 用户为例。

### 1. 安装 Docker

```bash
apt update
apt install -y ca-certificates curl
install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" > /etc/apt/sources.list.d/docker.list
apt update
apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

检查：

```bash
docker version
docker compose version
```

注意：不要使用旧的 `docker-compose` v1 搭配较新的 Docker，可能会遇到 `ContainerConfig` 错误。优先使用 `docker compose`。

### 2. 创建目录

```bash
mkdir -p /opt/nezha/dashboard/data
cd /opt/nezha/dashboard
```

### 3. 写入配置

创建 `/opt/nezha/dashboard/data/config.yaml`：

```yaml
listen_port: 8008
language: zh_CN
site_name: vps-netwatch
install_host: 你的主控IP:8008
tls: false

tsdb:
  data_path: data/tsdb
  retention_days: 30
  min_free_disk_space_gb: 1
  max_memory_mb: 256
```

把 `你的主控IP` 改成 Dashboard 所在 VPS 的公网 IP 或域名。

### 4. 写入 Docker Compose

创建 `/opt/nezha/dashboard/docker-compose.yaml`：

```yaml
services:
  dashboard:
    image: ghcr.io/yikkrrtykj/vps-netwatch:latest
    container_name: nezha-dashboard
    restart: always
    volumes:
      - ./data:/dashboard/data
    ports:
      - "8008:8008"
```

启动：

```bash
docker compose pull
docker compose up -d
docker logs --tail 80 nezha-dashboard
```

日志里看到下面内容说明启动成功：

```text
TSDB initialized successfully
Dashboard::START ON :8008
```

### 5. 登录后台

浏览器打开：

```text
http://你的主控IP:8008/dashboard/
```

默认账号：

```text
用户名：admin
密码：admin
```

第一次登录后请立刻修改密码。

## 接入 VPS Agent

在 Dashboard 后台进入 `服务器` 页面，添加或复制 Agent 安装命令。

如果某台 VPS 缺依赖，先安装：

```bash
apt update
apt install -y curl unzip
```

然后在每台被监控 VPS 上执行后台生成的 Agent 安装命令。

安装完成后，回到 Dashboard 首页确认服务器在线。

## 添加延迟监控

后台进入 `服务` 页面，新增两个服务监控。

上海电信：

```text
名称：上海电信
类型：ICMP-Ping
目标：202.96.209.133
间隔：30 秒
覆盖范围：全部服务器
显示在服务页：开启
```

上海联通：

```text
名称：上海联通
类型：TCP-Ping
目标：210.22.84.3:53
间隔：30 秒
覆盖范围：全部服务器
显示在服务页：开启
```

等 1-2 分钟后，打开：

```text
http://你的主控IP:8008/dashboard/netwatch/latency
```

首页服务器列表上方的圆形按钮旁也会出现一个“延迟”入口，可以直接点进去。

## 更新

```bash
cd /opt/nezha/dashboard
docker compose pull
docker compose up -d
docker logs --tail 80 nezha-dashboard
```

## 回滚到原版哪吒

把 `/opt/nezha/dashboard/docker-compose.yaml` 里的镜像改回：

```yaml
image: ghcr.io/nezhahq/nezha:latest
```

然后执行：

```bash
cd /opt/nezha/dashboard
docker compose pull
docker compose up -d
```

## 开发说明

- 本仓库保留上游模块名 `github.com/nezhahq/nezha`，避免大规模修改 import。
- 运行文件不要直接在生产 VPS 上手改，应该改源码、推送仓库、等待镜像构建，再更新 Docker 镜像。
- 上游项目使用 Apache-2.0 license。
