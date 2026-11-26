# INSTALL.md — 小橘（DeepTalk）部署手册

> **这份文件是给 AI 读的**：把本文件内容整段交给 AI，说「按这个部署」，它会照着执行。
> 人类也能直接照着做。两条路线：**A 直接部署**（二进制 + systemd，不用 Docker）、**B Docker Compose 部署**。
> 选一条走完即可，不要混用。

---

## 0. 通用信息（两条路线都一样）

| 项 | 值 |
| --- | --- |
| 服务端口 | `9821`（容器内/进程内固定） |
| 数据目录 | 数据库 `deeptalk.db`、上传图片、日志、SSH 密钥都在这里 |
| 配置文件 | `configs/app.yaml`（唯一配置源；环境变量 `DEEPTALK_*` 只做覆盖） |
| 默认管理员 | `admin` / `123456`，**仅在空库首次启动时写入**，登录后立即改密码 |
| 健康检查 | `curl -fsS http://127.0.0.1:9821/healthz` → `{"code":0,...,"status":"ok"}` |
| 必填密钥 | `DEEPTALK_SECRET`（≥16 位，`openssl rand -hex 32`），**首次启动后永不更换** |

**部署完成后还有两件事**（两条路线都一样，见第 3 节）：

1. 用 `admin/123456` 登录 `/admin/`，改密码；
2. 在「厂商与模型」里添加厂商（Base URL + API Key）与模型，再到「系统设置」设为默认模型。
   **新库没有任何厂商和模型**，不做这步对话会提示「当前没有可用模型」。

> 厂商、API Key、输出预算、生成超时、套餐等全部在后台配置，**不要写进配置文件**。

---

## A. 直接部署（二进制 + systemd，不用 Docker）

### A1. 构建机器上准备

需要：Go **1.26+**、Node.js **22+**、pnpm **9+**。

```bash
git clone <repo> deeptalk && cd deeptalk

# 一条命令：构建前端 → 拷进 server/web/dist → 编译二进制
cd server
make dist
# 产物：server/bin/deeptalk
```

跨平台编译（在 x86 机器上出 ARM 包等）：

```bash
make ui-build && make embed       # 先内嵌前端（只做一次）
make build-linux                  # bin/deeptalk-linux-amd64
make build-linux-arm64            # bin/deeptalk-linux-arm64
```

> 前端必须内嵌进二进制，否则页面 404。`make dist` 已经包含这一步。

### A2. 部署到服务器

需要：一台 Linux（amd64/arm64），能开 9821 端口。

```bash
# 1) 建目录结构
sudo useradd -r -u 10001 -s /usr/sbin/nologin deeptalk || true
sudo mkdir -p /opt/deeptalk/{configs,data,logs}
sudo chown -R deeptalk:deeptalk /opt/deeptalk

# 2) 传二进制与配置（在构建机上执行）
scp server/bin/deeptalk root@<server>:/opt/deeptalk/
scp -r server/configs/app.yaml root@<server>:/opt/deeptalk/configs/

# 3) 写环境文件（在服务器上，含密钥，权限 600）
sudo tee /opt/deeptalk/.env >/dev/null <<EOF
DEEPTALK_ENV=prod
DEEPTALK_ADDR=0.0.0.0:9821
DEEPTALK_BASE_URL=https://your-domain.example.com
DEEPTALK_SECRET=$(openssl rand -hex 32)
DEEPTALK_DB=/opt/deeptalk/data/deeptalk.db
DEEPTALK_UPLOAD_DIR=/opt/deeptalk/data/uploads
DEEPTALK_COOKIE_SECURE=true
DEEPTALK_SIGN_MODE=enforce
DEEPTALK_CONFIG_DIR=/opt/deeptalk/configs
EOF
sudo chmod 600 /opt/deeptalk/.env
```

### A3. 注册 systemd 服务

```bash
sudo tee /etc/systemd/system/deeptalk.service >/dev/null <<'EOF'
[Unit]
Description=DeepTalk (XiaoJu) AI workspace
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=deeptalk
Group=deeptalk
WorkingDirectory=/opt/deeptalk
EnvironmentFile=/opt/deeptalk/.env
ExecStart=/opt/deeptalk/deeptalk
Restart=always
RestartSec=3
# 优雅关闭：SSE 生成需要时间收尾
KillSignal=SIGTERM
TimeoutStopSec=30
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now deeptalk
sudo systemctl status deeptalk --no-pager
```

### A4. 验证

```bash
curl -fsS http://127.0.0.1:9821/healthz
journalctl -u deeptalk -n 50 --no-pager      # 看启动日志
```

- 升级：重新 `make dist` → `scp` 覆盖二进制 → `sudo systemctl restart deeptalk`
- 卸载：`sudo systemctl disable --now deeptalk`（数据在 `/opt/deeptalk/data`，按需保留）

---

## B. Docker Compose 部署

需要：Docker 24+ 与 `docker compose` 插件。**服务器上不需要 Go / Node**。

### B1. 二选一：本地构建镜像，或拉现成镜像

**B1-a 在目标机构建**（适合内网/离线，源码在机器上）：

```bash
git clone <repo> deeptalk && cd deeptalk
cp .env.example .env
sed -i "s|^DEEPTALK_SECRET=.*|DEEPTALK_SECRET=$(openssl rand -hex 32)|" .env

docker compose up -d --build          # 首次构建几分钟
docker compose logs -f
```

- 数据在 docker 卷 `deeptalk-data`（不是宿主机目录）
- 备份：`docker run --rm -v deeptalk-data:/data -v "$PWD":/backup alpine tar -czf /backup/deeptalk-$(date +%F).tar.gz -C /data .`

**B1-b 拉现成镜像**（生产推荐，配合 `scripts/deploy.sh`）：

```bash
sudo mkdir -p /srv/deeptalk/data
sudo chown -R 10001:10001 /srv/deeptalk/data     # 容器以 uid=10001 运行，必须授权

cd /srv/deeptalk
# 放入 docker-compose.prod.yml、scripts/deploy.sh
cp .env.production.example .env

# 编辑 .env，至少改这几项
vi .env
#   DEEPTALK_IMAGE=ghcr.nju.edu.cn/xjqxz2/deeptalk
#   DEEPTALK_IMAGE_TAG=latest
#   DEEPTALK_SECRET=<openssl rand -hex 32>
#   DEEPTALK_BASE_URL=https://your-domain.example.com
#   DEEPTALK_DATA_DIR=/srv/deeptalk/data
#   DEEPTALK_ENV=prod
#   DEEPTALK_COOKIE_SECURE=true
#   DEEPTALK_SIGN_MODE=enforce

chmod +x deploy.sh
./deploy.sh latest
```

`deploy.sh` 的行为：停容器 → 备份数据（db/wal/shm）→ 拉镜像 → 启动 → 轮询 `/healthz` 60 秒 →
**失败自动回滚到上一个 tag** → 成功才写入新 tag → 只清理本仓库旧镜像（保留 3 个）。

指定版本 / 回滚：`./deploy.sh v1.2.0`。

### B2. 验证

```bash
docker compose ps
docker compose logs --tail=50 deeptalk
curl -fsS http://127.0.0.1:9821/healthz
```

### B3. 反向代理（生产必须）

容器只监听 HTTP，HTTPS 交给 Nginx：

```nginx
location / {
    proxy_pass http://127.0.0.1:9821;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;

    # SSE 必须关闭缓冲，否则回答会攒着一次性输出
    proxy_buffering off;
    proxy_cache off;
    proxy_read_timeout 3600s;
}
```

---

## 3. 首次进入（两条路线都要做）

1. 打开 `https://<你的域名>/admin/`，用 **`admin` / `123456`** 登录
2. 右上角账号菜单 →「账号与密码」→ **立即修改密码**
3. 左侧「厂商与模型」→「新增厂商」：
   - 名称：`DeepSeek`
   - Base URL：`https://api.deepseek.com`
   - API Key：填入（数据库内 AES-GCM 加密存储）
   - 协议：`chat`（默认）/ `responses` / `anthropic`
4. 同页添加模型：`model_id` 填上游真实标识（如 `deepseek-chat`）、显示名、上下文窗口、三段价格
5. 「系统设置」→ 把**默认模型**设为刚添加的模型
6. 打开首页发一条消息，确认能正常流式回答

---

## 4. 故障排查

| 现象 | 原因 | 处理 |
| --- | --- | --- |
| 页面 404 / 白屏 | 二进制没内嵌前端 | 用 `make dist` 重新构建，别只 `make build` |
| `config load failed` | 找不到 `configs/app.yaml` | 设置 `DEEPTALK_CONFIG_DIR` 指向配置目录 |
| 容器起不来，日志报权限 | 数据目录属主不对 | `chown -R 10001:10001 <data dir>` |
| 对话提示「当前没有可用模型」 | 还没配厂商/模型 | 见第 3 节第 3~5 步 |
| 登录报「请填写邮箱与密码」 | 密码为空或请求被截断 | 账号 `admin`，密码 `123456` |
| 厂商鉴权失败 | 换过 `DEEPTALK_SECRET` | 已加密的 Key 作废，后台重新填 API Key |
| 回答一次性全出来 | 反代开了缓冲 | Nginx 加 `proxy_buffering off` |
| 上传图片失败 | 数据目录不可写 | 检查 `DEEPTALK_UPLOAD_DIR` 与目录权限 |
| SSH 工具报权限错误 | `<data>/ssh` 属主不对 | `chown -R 10001:10001 <data>/ssh` |

---

## 5. 部署检查清单

```text
[ ] 端口 9821 已放行 / 已挂反代
[ ] DEEPTALK_SECRET 已生成并妥善保存（后续不再更换）
[ ] 健康检查返回 status=ok
[ ] admin/123456 能登录，且已改密码
[ ] 已添加厂商 + 模型，并设为默认模型
[ ] 发一条消息能流式返回
[ ] 数据目录已纳入备份
```

更多细节：配置项与本地开发见 `docs/GETTING-STARTED.md`，发布流水线与回滚见 `docs/DEPLOY.md`。
