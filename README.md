# vps-netwatch

[![Build vps-netwatch image](https://github.com/yikkrrtykj/vps-netwatch/actions/workflows/vps-netwatch-image.yml/badge.svg)](https://github.com/yikkrrtykj/vps-netwatch/actions/workflows/vps-netwatch-image.yml)

`vps-netwatch` 是基于 [Nezha Monitoring](https://github.com/nezhahq/nezha) 的自定义 Dashboard 仓库。

当前策略不是重写一套 VPS 监控系统，而是复用哪吒成熟的 Dashboard、Agent、服务监控、延迟图和 Web 终端能力，在这个 fork 上逐步增加代理链路、mihomo/Clash、出口 IP、ASN、地区和游戏网络诊断相关功能。

## 当前状态

- 代码基线：Nezha Dashboard fork
- Go module：保持上游 `github.com/nezhahq/nezha`
- 默认分支：`main`
- 镜像仓库：`ghcr.io/yikkrrtykj/vps-netwatch`
- 构建架构：`linux/amd64`、`linux/arm64`
- 自定义页面：`/netwatch/latency`，用于一页查看所有 ICMP/TCP 服务监控延迟曲线

推送到 `main` 后，GitHub Actions 会构建并推送：

```text
ghcr.io/yikkrrtykj/vps-netwatch:main
ghcr.io/yikkrrtykj/vps-netwatch:latest
```

推送 tag，例如 `v1.0.0`，会额外生成：

```text
ghcr.io/yikkrrtykj/vps-netwatch:v1.0.0
```

## 推荐使用方式

先安装原版哪吒，确认基础监控功能稳定，再切换到本仓库镜像。

这样做的好处是：VPS 接入、Agent 通信、延迟图、服务监控、Web 终端这些基础能力先用上游稳定实现兜住；后续改造只通过源码提交和镜像发布，不直接修改服务器上已经安装的运行文件。

详细步骤见 [VPS-NETWATCH.md](VPS-NETWATCH.md)。

## 延迟总览

部署本仓库镜像后，可以直接访问：

```text
http://你的主控地址:8008/netwatch/latency
```

这个页面会聚合哪吒后台“服务”页里配置的 `ICMP-Ping` 和 `TCP-Ping` 任务，把不同 VPS 节点到不同目标的延迟曲线放到同一张图里。适合快速查看游戏服、运营商线路、区域节点之间的延迟变化。

如果页面没有数据，先在后台 `服务` 页添加至少一个 Ping/TCP 监控任务，并等待一次采集周期。

## 快速切换到自定义镜像

原版哪吒 Docker 安装后，主控配置通常在：

```text
/opt/nezha/dashboard/docker-compose.yaml
/opt/nezha/dashboard/data/config.yaml
```

把 `/opt/nezha/dashboard/docker-compose.yaml` 里的 Dashboard 镜像改成：

```yaml
image: ghcr.io/yikkrrtykj/vps-netwatch:latest
```

然后重启 Dashboard：

```bash
cd /opt/nezha/dashboard
sudo docker compose pull
sudo docker compose up -d
```

如果 GHCR 拉取失败，先检查仓库的 Packages 是否已经发布，并确认 package visibility 是 public。

## 回滚

把镜像改回原版哪吒：

```yaml
image: ghcr.io/nezhahq/nezha:latest
```

然后执行：

```bash
cd /opt/nezha/dashboard
sudo docker compose pull
sudo docker compose up -d
```

## 部署参考

本仓库提供了最小 Docker Compose 参考：

- [deploy/vps-netwatch/docker-compose.yml](deploy/vps-netwatch/docker-compose.yml)
- [deploy/vps-netwatch/config.example.yaml](deploy/vps-netwatch/config.example.yaml)

如果你是从原版哪吒切换过来，优先沿用 `/opt/nezha/dashboard/data` 里的现有数据和配置，不要直接覆盖生产数据目录。

## 后续改造方向

第一阶段保持哪吒原功能稳定，只补充网络诊断能力：

- 每台 VPS 出口 IP、ASN、地区检测
- mihomo/Clash external-controller 只读数据源
- 游戏服务器目标 IP 观察和延迟诊断
- 代理链路、规则命中、连接目标排障视图

## 上游同步

本仓库仍按 Nezha 源码树维护。同步上游时建议先确认自己的改动已经提交，再合并上游：

```bash
git fetch upstream master
git merge upstream/master
```

遇到冲突时，优先保留上游监控主逻辑，再重新套用本仓库的自定义功能。

## 上游项目

- Dashboard：[nezhahq/nezha](https://github.com/nezhahq/nezha)
- Agent：[nezhahq/agent](https://github.com/nezhahq/agent)
- 官方文档：[中文文档](https://nezhahq.github.io/index.html) / [English](https://nezhahq.github.io/en_US/index.html)

本仓库遵循上游 Apache-2.0 license。
