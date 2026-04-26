# vps-netwatch

我的 VPS 网络监控面板，基于哪吒监控二次开发。

现在主要做一件事：在原来的 VPS 状态页里，加一个更方便看的延迟视图。点首页圆形按钮里的“延迟”，就在当前页面展开；再点一次收起。

## 现在有的

- 多台 VPS 在线状态、CPU、内存、磁盘、流量。
- 哪吒原来的 Web 终端和服务监控。
- 默认开启 TSDB，方便保存历史数据。
- 首页内嵌延迟图。
- 可以先选一台 VPS，再看这台 VPS 到上海电信、上海联通、其它 VPS 的 ping。
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
mkdir -p /opt/nezha/dashboard/data
cd /opt/nezha/dashboard
```

### 3. 写配置

作用：设置监听端口、站点名和历史数据目录。

`/opt/nezha/dashboard/data/config.yaml`：

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

把 `你的主控IP` 换成主控 VPS 的公网 IP 或域名。

### 4. 写 Compose

作用：指定用我的镜像启动 Dashboard。

`/opt/nezha/dashboard/docker-compose.yaml`：

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

在后台 `服务器` 页面复制 Agent 安装命令，到每台 VPS 上执行。

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
```

再新增：

```text
名称：上海联通
类型：TCP-Ping
目标：210.22.84.3:53
间隔：30 秒
覆盖范围：全部服务器
```

### 8. 添加 VPS 互 ping

作用：看每台 VPS 到其它 VPS 的延迟。

比如你有一台 `stable-wasp`，公网 IP 是 `1.2.3.4`，就在后台 `服务` 里新增：

```text
名称：VPS stable-wasp
类型：ICMP-Ping
目标：1.2.3.4
间隔：30 秒
覆盖范围：全部服务器，或者只选需要测试它的服务器
```

每台需要作为目标的 VPS 都加一个这样的监控。等一两分钟，回到首页，点圆形按钮里的“延迟”，先选择源 VPS，再看它到各个目标的曲线。

## 更新

作用：拉最新镜像，不动数据。

```bash
cd /opt/nezha/dashboard
docker compose pull
docker compose up -d
```

## 回滚

作用：如果新版本有问题，先切回原版哪吒。

把 compose 里的镜像改成：

```yaml
image: ghcr.io/nezhahq/nezha:latest
```

然后：

```bash
cd /opt/nezha/dashboard
docker compose pull
docker compose up -d
```

## 开发

不要直接改 VPS 上的运行文件。改源码，推到 GitHub，等镜像构建好，再更新服务器上的 Docker 镜像。
