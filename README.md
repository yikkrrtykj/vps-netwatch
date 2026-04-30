# vps-netwatch

我的 VPS 网络监控面板，用来集中看多台 VPS 的状态、流量、延迟和后续代理链路诊断。

现在主要做一件事：在原来的 VPS 状态页里，加一个更方便看的延迟视图。点首页圆形按钮里的“延迟”，就在当前页面展开；再点一次收起。

## 现在有的

- 多台 VPS 在线状态、CPU、内存、磁盘、流量。
- Web 终端和服务监控。
- 默认开启 TSDB，方便保存历史数据。
- 首页内嵌延迟图。
- 内嵌面板里有 VPS 快览，每台机器下方直接显示最新延迟，也可以显示带宽和剩余时间，不显示价格。
- 目标向导：输入 `IP/域名` 自动创建 ICMP，输入 `IP:端口` 自动创建 TCP。
- 异常标记：自动提示峰值、持续抖动和丢包区间，点击标记可以缩放到对应时间段。
- mihomo/Clash 连接发现：读取 `/connections` 后可以把游戏服或连接目标一键加入监控。
- 可以先选一台 VPS，默认看它到上海电信、上海联通的 ping。
- 需要排障时，可以临时选择一台其它 VPS 做互 ping 目标。
- 鼠标悬停看具体延迟，滚轮缩放时间，双击恢复。

## 镜像

```text
ghcr.io/yikkrrtykj/vps-netwatch:latest
```

## 安装步骤

### 1. 准备 Docker

作用：让 Dashboard 用容器跑，后面升级只需要拉新镜像。

Ubuntu 22.04 可以直接装 Docker 官方版：

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

### 2. 创建 Dashboard 目录

作用：保存配置、数据库和历史数据。

```bash
mkdir -p /opt/vps-netwatch/dashboard/data
cd /opt/vps-netwatch/dashboard
```

### 3. 写配置

作用：设置监听端口、站点名和历史数据目录。

`/opt/vps-netwatch/dashboard/data/config.yaml`：

```yaml
listen_port: 8008
language: zh_CN
site_name: vps-netwatch
install_host: IP:8008
tls: false

tsdb:
  data_path: data/tsdb
  retention_days: 30
  min_free_disk_space_gb: 1
  max_memory_mb: 256
```

把 `你的主控IP` 换成主控 VPS 的公网 IP 或域名。

### 4. 写 Compose

作用：指定用我的镜像启动 Dashboard。

`/opt/vps-netwatch/dashboard/docker-compose.yaml`：

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

看到 `Dashboard::START ON :8008` 就说明起来了。

### 5. 登录后台

作用：改密码、接入 VPS、添加延迟监控。

```text
http://你的主控IP:8008/dashboard/
```

默认账号：

```text
admin / admin
```

第一次进去先改密码。

### 6. 接入 VPS

作用：让每台 VPS 上报状态，并执行延迟探测。

在后台 `服务器` 页面点添加服务器，复制 Agent 安装命令，到那台 VPS 上执行。

新 VPS 上线后，只要后台显示在线，就会自动加入首页的 VPS 选择里。默认的上海电信、上海联通延迟监控不需要为每台 VPS 单独再配一次。

如果提示缺依赖：

```bash
apt update
apt install -y curl unzip
```

### 7. 添加运营商延迟监控

作用：让每台 VPS 去测上海电信、上海联通。

后台进入 `服务`，新增：

```text
名称：上海电信
类型：ICMP-Ping
目标：202.96.209.133
间隔：30 秒
覆盖范围：全部服务器
指定服务器：留空
```

再新增：

```text
名称：上海联通
类型：TCP-Ping
目标：210.22.84.3:53
间隔：30 秒
覆盖范围：全部服务器
指定服务器：留空
```

这里不要只勾某几台机器。保持“全部服务器”，以后新接入的 VPS 才会自动开始测延迟。

### 8. VPS 互 ping

作用：需要排障时，临时看当前 VPS 到另一台 VPS 的延迟。

这个不用手动建一堆服务监控。回到首页，点圆形按钮里的“延迟”，先选要查看的源 VPS，再在 `互 ping 目标` 里选择另一台 VPS。

默认是关闭的，所以平时只看上海电信、上海联通。一次只会启用一个 VPS 互 ping 目标；换目标时，旧目标会自动停掉。选完后等一次采集，页面会自己刷新出曲线。

### 9. 带宽和剩余时间标签

首页内嵌面板的 VPS 快览会自动显示实时上下行和总传输。如果想像 `1Gbps`、`500Mbps` 这样显示套餐带宽，可以把服务器名称写成：

```text
香港 NHK-Lite@1Gbps
```

也可以在服务器公开备注里写：

```text
bandwidth=1Gbps
```

如果想在带宽旁边显示剩余时间，可以在服务器公开备注里写：

```text
expire=2026-05-20
remaining=22d
```

### 10. 目标向导和连接发现

在首页内嵌延迟面板里，`目标向导` 支持：

```text
1.1.1.1          -> ICMP-Ping
example.com      -> ICMP-Ping
1.1.1.1:443      -> TCP-Ping
example.com:443  -> TCP-Ping
```

`mihomo / Clash 连接发现` 读取的是 external-controller 的 `/connections`。如果 Dashboard 和 mihomo 不在同一台机器或同一内网，主控 VPS 可能访问不到 `127.0.0.1:9090`，这时要填主控能访问到的控制器地址。

## 常见问题

新加的 VPS 没有延迟数据时，先看后台 `服务` 里的 `上海电信`、`上海联通`：

```text
覆盖范围：全部服务器
指定服务器：留空
```

如果这两项没问题，只要新 VPS 的 Agent 在线，等一两分钟就会开始有数据，不需要再做别的配置。

## 更新

作用：拉最新镜像，不动数据。

```bash
cd /opt/vps-netwatch/dashboard
docker compose pull
docker compose up -d
```

## 开发

不要直接改 VPS 上的运行文件。改源码，推到 GitHub，等镜像构建好，再更新服务器上的 Docker 镜像。
