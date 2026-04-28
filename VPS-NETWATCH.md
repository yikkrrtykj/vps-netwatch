# vps-netwatch 使用说明

`vps-netwatch` 不再从零实现监控系统。这个仓库现在直接以哪吒监控为底座：先安装原版哪吒，确认多 VPS 接入、延迟图、服务监控、Web 终端这些基础能力都正常；后续再在本仓库改源码并构建自己的镜像。

## 推荐路线

1. 主控 VPS 先安装原版哪吒 Dashboard。
2. 其他 VPS 用哪吒后台生成的 agent 安装命令接入。
3. 确认面板稳定后，再把 Dashboard 镜像切换成 `vps-netwatch` 自定义镜像。
4. 之后所有功能修改都走源码提交和镜像构建，不直接改服务器里的运行文件。

## 第一步：安装原版哪吒

在主控 VPS 上运行官方安装脚本：

```bash
curl -L https://raw.githubusercontent.com/nezhahq/scripts/refs/heads/main/install.sh -o nezha.sh
chmod +x nezha.sh
sudo ./nezha.sh
```

安装时建议选择 Docker 方式。端口可以先用默认 `8008`，没有域名也可以先用公网 IP。

安装完成后访问：

```text
http://主控VPS公网IP:8008
```

后台入口：

```text
http://主控VPS公网IP:8008/dashboard
```

第一次登录默认账号和密码通常都是 `admin`。登录后第一件事是改强密码。

## 第二步：接入其他 VPS

进入哪吒后台的服务器页面，复制面板生成的安装命令，在每台 VPS 上运行。新的 VPS 不需要提前写进本仓库；agent 连上后会自动出现在面板里。

如果你有域名，建议在后台把 Agent 连接地址设置成不走 CDN 的域名或 `公网IP:8008`。如果暂时没有域名，就先用：

```text
主控VPS公网IP:8008
```

## 第三步：构建 vps-netwatch 镜像

这个仓库已经是哪吒源码树。后续要改功能时，直接改这里的 Go、前端模板或配置，然后推送到 GitHub。

推送到 `main` 后，GitHub Actions 会构建并推送：

```text
ghcr.io/yikkrrtykj/vps-netwatch:main
ghcr.io/yikkrrtykj/vps-netwatch:latest
```

打 tag，例如 `v1.0.0`，会额外推送：

```text
ghcr.io/yikkrrtykj/vps-netwatch:v1.0.0
```

## 第四步：把主控切到自定义镜像

原版哪吒 Docker 安装后，配置一般在：

```text
/opt/nezha/dashboard/docker-compose.yaml
/opt/nezha/dashboard/data/config.yaml
```

把 `/opt/nezha/dashboard/docker-compose.yaml` 里的镜像改成：

```yaml
image: ghcr.io/yikkrrtykj/vps-netwatch:latest
```

然后重启：

```bash
cd /opt/nezha/dashboard
sudo docker compose pull
sudo docker compose up -d
```

如果要回滚原版，把镜像改回：

```yaml
image: ghcr.io/nezhahq/nezha:latest
```

再执行同样的 `pull` 和 `up -d`。

## 本仓库的自定义方向

第一阶段先保持哪吒原功能稳定，不急着大改。

后续再逐步加：

- mihomo/Clash external-controller 只读数据源。
- 游戏服务器目标 IP 观察和延迟诊断。
- 每台 VPS 的出口 IP/ASN/地区检测。
- 面板里增加代理链路、规则命中、连接目标的排障视图。

原则是：VPS 监控、Web 终端、服务监控、延迟图这些成熟能力继续用哪吒；`vps-netwatch` 只补你需要的代理和游戏网络诊断。

## 自定义延迟面板

本仓库把自定义 Ping 延迟能力合并到首页概览里的内嵌延迟面板。进入首页后点击圆形的延迟图标，就能展开面板。

当前能力：

- 选择日期查看当天延迟历史。
- 在 ICMP 和 TCP Ping 监控之间切换。
- 勾选显示极值标签，默认关闭。
- 勾选显示平均线，默认关闭。
- 点击截图按钮导出当前图表 PNG。

## 同步上游

本地仓库保留了 `upstream` remote：

```bash
git fetch upstream master
git merge upstream/master
```

同步前先确认自己的改动已经提交。遇到冲突时，优先保留哪吒上游的监控主逻辑，再重新套我们的自定义功能。
