# vps-netwatch 使用说明

`vps-netwatch` 是一个 VPS 网络监控面板，用来集中查看多台 VPS 的在线状态、资源占用、流量、延迟和后续代理/游戏网络诊断。

## 推荐部署方式

1. 主控 VPS 运行 Dashboard。
2. 其他 VPS 使用后台生成的 Agent 安装命令接入。
3. 所有功能修改都走源码提交和镜像构建，不直接改服务器里的运行文件。

## 主控 Dashboard

在主控 VPS 准备目录：

```bash
mkdir -p /opt/vps-netwatch/dashboard/data
cd /opt/vps-netwatch/dashboard
```

创建 `/opt/vps-netwatch/dashboard/data/config.yaml`：

```yaml
listen_port: 8008
language: zh_CN
site_name: vps-netwatch
install_host: 主控VPS公网IP:8008
tls: false

tsdb:
  data_path: data/tsdb
  retention_days: 30
  min_free_disk_space_gb: 1
  max_memory_mb: 256
```

创建 `/opt/vps-netwatch/dashboard/docker-compose.yaml`：

```yaml
services:
  dashboard:
    image: ghcr.io/yikkrrtykj/vps-netwatch:latest
    container_name: vps-netwatch-dashboard
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
docker logs --tail 80 vps-netwatch-dashboard
```

访问：

```text
http://主控VPS公网IP:8008
http://主控VPS公网IP:8008/dashboard
```

第一次登录后先修改默认密码。

## 接入其他 VPS

进入后台的服务器页面，添加服务器并复制 Agent 安装命令，在每台 VPS 上执行。新的 VPS 不需要提前写进本仓库；Agent 连上后会自动出现在面板里。

如果你有域名，建议把 Agent 连接地址设置成不走 CDN 的域名或 `公网IP:8008`。如果暂时没有域名，就先用：

```text
主控VPS公网IP:8008
```

## 镜像构建

推送到 `main` 后，GitHub Actions 会构建并推送：

```text
ghcr.io/yikkrrtykj/vps-netwatch:main
ghcr.io/yikkrrtykj/vps-netwatch:latest
```

打 tag，例如 `v1.0.0`，会额外推送：

```text
ghcr.io/yikkrrtykj/vps-netwatch:v1.0.0
```

## 自定义延迟面板

首页圆形按钮里的延迟图标会展开内嵌 Ping 面板。

当前能力：

- VPS 快览：每台机器下方直接显示最新延迟；表格视图显示剩余时间、最大带宽、实时网络速率和 IPv4/IPv6。
- 目标向导：输入 `IP/域名` 自动创建 ICMP，输入 `IP:端口` 自动创建 TCP。
- 异常标记：自动提示峰值、持续抖动和丢包区间。
- mihomo/Clash 连接发现：读取 `/connections` 后可以把连接目标一键加入监控。
- 选择日期查看当天延迟历史。
- 在 ICMP 和 TCP Ping 监控之间切换。
- 勾选显示极值标签，默认关闭。
- 勾选显示平均线，默认关闭。
- 临时选择一台其它 VPS 做互 ping 目标。

带宽标签可以写在服务器名称里：

```text
香港 NHK-Lite@1Gbps
```

也可以写在服务器公开备注里：

```text
bandwidth=1Gbps
```

如果想在带宽旁边显示剩余时间，可以在服务器公开备注里写：

```text
expire=2026-05-20
remaining=22d
```

目标向导示例：

```text
1.1.1.1          -> ICMP-Ping
example.com      -> ICMP-Ping
1.1.1.1:443      -> TCP-Ping
example.com:443  -> TCP-Ping
```

mihomo/Clash 连接发现需要填写主控 Dashboard 能访问到的 external-controller 地址，例如：

```text
http://192.168.50.10:9090
```

如果控制器设置了 `secret`，也要在面板里填密钥。

## 后续方向

- 游戏服务器目标 IP 观察和延迟诊断。
- 每台 VPS 的出口 IP/ASN/地区检测。
- 代理链路、规则命中、连接目标排障视图。

## 更新

```bash
cd /opt/vps-netwatch/dashboard
docker compose pull
docker compose up -d
```
