# StellAtlas Service 部署说明

本文档说明如何使用 GitHub Release 生成的 zip 制品部署 `stellatlas-service`，并将服务加入 `systemd` 管理。

## 制品说明

Tag 发布后会生成两个 Linux zip 包：

- `stellatlas-service-linux-amd64.zip`
- `stellatlas-service-linux-arm64.zip`

每个 zip 包内只包含两个文件：

- `stellatlas-service`：服务端可执行文件
- `application.yaml`：启动配置文件，来源于仓库 `cmd/application.yml`

根据服务器架构选择对应制品：

```bash
uname -m
```

- 输出 `x86_64` 时使用 `stellatlas-service-linux-amd64.zip`
- 输出 `aarch64` 或 `arm64` 时使用 `stellatlas-service-linux-arm64.zip`

## 部署目录

建议使用以下目录结构：

```text
/opt/stellatlas-service/
  stellatlas-service
  application.yaml
```

创建运行用户和部署目录：

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin stellatlas || true
sudo mkdir -p /opt/stellatlas-service
```

解压制品并安装文件：

```bash
unzip stellatlas-service-linux-amd64.zip -d /tmp/stellatlas-service

sudo install -m 0755 /tmp/stellatlas-service/stellatlas-service /opt/stellatlas-service/stellatlas-service
sudo install -m 0640 /tmp/stellatlas-service/application.yaml /opt/stellatlas-service/application.yaml
sudo chown -R stellatlas:stellatlas /opt/stellatlas-service
```

如果是 arm64 服务器，将 zip 文件名替换为 `stellatlas-service-linux-arm64.zip`。

## 配置调整

编辑 `/opt/stellatlas-service/application.yaml`：

```bash
sudo vi /opt/stellatlas-service/application.yaml
```

生产环境至少需要确认以下配置：

```yaml
app:
  env: prod

http:
  server:
    port: 8010

redis:
  enabled: true
  addr: 127.0.0.1:6379

postgresql:
  enabled: true
  dsn: postgres://stellhub:admin@127.0.0.1:5432/stellatlas-service?sslmode=disable
```

注意事项：

- `postgresql.dsn` 需要改成真实 PostgreSQL 地址、用户和密码。
- `redis.addr` 需要改成真实 Redis 地址。
- 首次启动前应先使用数据库 owner 或迁移账号执行仓库根目录的 `schema.sql`。
- 如果 PostgreSQL 尚未准备好，可以保持 `postgresql.ping_on_startup: false`，服务会先完成启动，但业务写入仍依赖数据库可用。

## 手动启动验证

在加入 `systemd` 前，可以先手动启动一次：

```bash
cd /opt/stellatlas-service
sudo -u stellatlas ./stellatlas-service
```

另开一个终端验证 HTTP 状态：

```bash
curl http://127.0.0.1:8010/health
curl http://127.0.0.1:8010/api/stellatlas/v1/status
```

确认无误后停止手动进程，再继续配置 `systemd`。

## systemd 配置

创建 unit 文件：

```bash
sudo vi /etc/systemd/system/stellatlas-service.service
```

写入以下内容：

```ini
[Unit]
Description=StellAtlas CMDB Service
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=stellatlas
Group=stellatlas
WorkingDirectory=/opt/stellatlas-service
ExecStart=/opt/stellatlas-service/stellatlas-service
Restart=on-failure
RestartSec=5s
KillSignal=SIGINT
TimeoutStopSec=30s
Environment=TZ=Asia/Shanghai
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

加载并启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable stellatlas-service
sudo systemctl start stellatlas-service
```

查看状态：

```bash
sudo systemctl status stellatlas-service
```

查看日志：

```bash
sudo journalctl -u stellatlas-service -f
```

重启服务：

```bash
sudo systemctl restart stellatlas-service
```

停止服务：

```bash
sudo systemctl stop stellatlas-service
```

## 升级流程

下载新 tag 对应的 zip 包后执行：

```bash
unzip stellatlas-service-linux-amd64.zip -d /tmp/stellatlas-service-new

sudo systemctl stop stellatlas-service
sudo install -m 0755 /tmp/stellatlas-service-new/stellatlas-service /opt/stellatlas-service/stellatlas-service
sudo chown stellatlas:stellatlas /opt/stellatlas-service/stellatlas-service
sudo systemctl start stellatlas-service
```

升级时通常不要直接覆盖生产环境的 `application.yaml`。如新版本新增配置项，应先对比新制品中的 `application.yaml`，再手动合并到生产配置。

## 常见问题

### 服务启动后访问不到

先确认监听端口：

```bash
sudo systemctl status stellatlas-service
sudo journalctl -u stellatlas-service -n 100
ss -lntp | grep 8010
```

再确认 `application.yaml` 中的 `http.server.port` 是否与访问端口一致。

### PostgreSQL 连接失败

检查 `postgresql.dsn`、数据库账号权限和网络连通性。首次部署时，需要先执行 `schema.sql` 初始化表结构。

### Redis 连接失败

检查 `redis.addr` 是否正确，以及 Redis 服务是否允许当前机器访问。Redis 只承担高频读缓存，数据库仍然是 CMDB 主数据源。
