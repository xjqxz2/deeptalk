# GitHub Actions → GHCR → 服务器部署

目标：打 tag 后由 GitHub Actions 构建镜像、推送到 GHCR，并通过 SSH 让服务器拉取新镜像。
服务器不安装 Node / Go / pnpm，只安装 Docker 与 Compose。

## 1. 服务器准备

要求：

- Linux 服务器（当前 workflow 构建 `linux/amd64`；ARM 服务器需要改 workflow，见文末）
- Docker Engine 24+
- Docker Compose v2（`docker compose` 命令）
- 一个专用部署用户，例如 `deploy`，不要直接用 root
- 服务器能访问 `ghcr.io`

目录结构：

```text
/srv/deeptalk/
├── docker-compose.prod.yml
├── .env
├── deploy.sh
├── backups/
└── data/
```

把项目里的 `docker-compose.prod.yml`、`scripts/deploy.sh` 放到 `/srv/deeptalk/`，并创建 `.env`：

```bash
sudo mkdir -p /srv/deeptalk/{data,backups}
sudo chown -R deploy:deploy /srv/deeptalk
cd /srv/deeptalk
cp .env.production.example .env
# 编辑 .env：DEEPTALK_IMAGE、DEEPTALK_SECRET、DEEPTALK_BASE_URL、DEEPTALK_DEEPSEEK_KEY
```

`.env` 至少：

```dotenv
DEEPTALK_IMAGE=ghcr.io/xjqxz2/deeptalk
DEEPTALK_IMAGE_TAG=v1.0.0
DEEPTALK_SECRET=固定的长随机串
DEEPTALK_BASE_URL=https://your-domain.example.com
DEEPTALK_PORT=9821
DEEPTALK_DATA_DIR=/srv/deeptalk/data
DEEPTALK_DEEPSEEK_KEY=sk-...
DEEPTALK_ENV=prod
DEEPTALK_COOKIE_SECURE=true
TZ=Asia/Shanghai
```

首次手动启动：

```bash
cd /srv/deeptalk
DEEPTALK_IMAGE_TAG=v1.0.0 docker compose -f docker-compose.prod.yml up -d
curl -fsS http://127.0.0.1:9821/healthz
```

## 2. GHCR 私有镜像认证

如果 GHCR 包是私有的，服务器需要能拉取：

1. 创建一个 GitHub PAT（classic），只勾选 `read:packages`；
2. 在服务器上执行：

```bash
echo "$GHCR_READ_TOKEN" | docker login ghcr.io -u YOUR_GITHUB_USER --password-stdin
```

这会把凭据保存到部署用户的 `~/.docker/config.json`，`docker compose pull` 会复用它。
如果包设为 Public，可跳过这一步。

## 3. SSH 部署密钥

**不要把私钥发给我，也不要提交到仓库。**

在本地生成一对专用密钥：

```bash
ssh-keygen -t ed25519 -C "github-actions-deeptalk" -f ~/.ssh/deeptalk_deploy -N ""
```

把**公钥**安装到服务器部署用户：

```bash
ssh-copy-id -i ~/.ssh/deeptalk_deploy.pub deploy@YOUR_SERVER
```

验证：

```bash
ssh -i ~/.ssh/deeptalk_deploy deploy@YOUR_SERVER 'cd /srv/deeptalk && ./deploy.sh --help || true'
```

然后去 GitHub 仓库：

`Settings → Secrets and variables → Actions`

新增 Secrets：

| Name | 值 |
| --- | --- |
| `DEPLOY_HOST` | 服务器 IP / 域名 |
| `DEPLOY_USER` | 部署用户，例如 `deploy` |
| `DEPLOY_PORT` | SSH 端口，默认 `22`，可省略 |
| `DEPLOY_SSH_KEY` | **私钥文件内容**（`cat ~/.ssh/deeptalk_deploy` 的完整内容） |
| `DEPLOY_KNOWN_HOSTS` | `ssh-keyscan -H YOUR_SERVER` 的输出 |

新增 Variables：

| Name | 值 |
| --- | --- |
| `DEPLOY_ENABLED` | `true`，不设置则只构建不部署 |
| `DEPLOY_PATH` | `/srv/deeptalk`，可选 |

> Action 使用私钥登录；公钥只放在服务器 `~/.ssh/authorized_keys`。
> `DEPLOY_KNOWN_HOSTS` 用于防止 SSH 中间人攻击。

`deploy.sh` 必须是可执行：

```bash
chmod +x /srv/deeptalk/deploy.sh
```

## 4. 触发部署

```bash
git tag v1.0.0
git push origin v1.0.0
```

GitHub Actions 会：

1. 跑 Go 测试、前端 typecheck / lint / build；
2. 构建并推送：
   - `ghcr.io/xjqxz2/deeptalk:v1.0.0`
   - `ghcr.io/xjqxz2/deeptalk:1.0.0`
   - `ghcr.io/xjqxz2/deeptalk:1.0`
   - 以及 `sha-<commit>` 标签；
3. 通过 SSH 执行：

```bash
cd /srv/deeptalk
./deploy.sh v1.0.0
```

`deploy.sh` 会：

- 停容器，备份 SQLite + uploads 到 `backups/`；
- 拉取新 tag；
- `docker compose up -d`；
- 轮询 `/healthz` 20 次 × 2 秒；
- 健康失败时回滚到上一个 tag；
- 成功后把新 tag 写回 `.env`。

普通 `git push main` 只构建镜像，不会部署；只有 `v*` tag 且 `DEPLOY_ENABLED=true` 才部署。

## 4.1 反向代理（Nginx）

容器只监听 `9821`，由 Nginx 终止 TLS 并转发。除了常规的 SSE 透传，还要注意两点：

```nginx
server {
    # 静态资源体积较大，必须启用 gzip，否则每次首屏都要传近 1MB
    gzip on;
    gzip_comp_level 6;
    gzip_min_length 1024;
    gzip_vary on;
    gzip_proxied any;
    # 关键：Go 静态文件返回的是 text/javascript，不写这一项 JS 不会被压缩
    gzip_types text/plain text/css text/javascript application/javascript
               application/json image/svg+xml application/manifest+json;

    location / {
        proxy_pass http://deeptalk_backend;
        proxy_http_version 1.1;

        # SSE 必须逐字下发，不能被缓冲或压缩
        proxy_buffering off;
        proxy_cache off;
        proxy_request_buffering off;
        proxy_set_header Connection "";
        chunked_transfer_encoding on;
        proxy_read_timeout 3600s;
    }
}
```

`gzip_types` 里不要包含 `text/event-stream`，否则流式回答会被整段压缩后才下发。

小程序 / APP 客户端的流式回答走 WebSocket，需要单独放行 `Upgrade`：

```nginx
location = /api/v1/c/ws {
    proxy_pass http://deeptalk_backend;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_read_timeout 3600s;
    proxy_send_timeout 3600s;
    proxy_buffering off;
}
```

没有这四行（尤其是 `Upgrade` 与 `Connection`）时，握手会以 400 失败。

## 4.2 旧镜像清理

`deploy.sh` 每次部署成功后会自动清理本项目的历史镜像，只保留最新 3 个
（当前运行版本 + 两个可回滚版本），并跳过所有正在被容器使用的镜像：

```bash
# 默认保留 3 个，可临时调整
KEEP_IMAGES=5 ./deploy.sh latest
```

清理只匹配 `.env` 里 `DEEPTALK_IMAGE` 指向的仓库，不会影响服务器上的其它服务。

## 5. 回滚

手动指定上一个 tag：

```bash
cd /srv/deeptalk
./deploy.sh v0.9.0
```

备份在：

```text
/srv/deeptalk/backups/deeptalk-YYYYmmdd-HHMMSS.tar.gz
```

## 6. 数据库与密钥

- `/srv/deeptalk/data` 必须持久化，不能放在容器可写层；
- `DEEPTALK_SECRET` 首次启动后不要更换，否则厂商 API Key 无法解密；
- 换服务器时，把 `data/` 和同一个 `DEEPTALK_SECRET` 一起迁移；
- SQLite 备份必须在容器停止后做，脚本已处理。

## 7. ARM 服务器

当前 workflow 只构建 `linux/amd64`。如果服务器是 ARM64：

- 在 workflow 加 `docker/setup-qemu-action@v3`；
- 把 `platforms` 改为 `linux/amd64,linux/arm64`；
- 或单独构建 `linux/arm64`。

## 8. 可选：不使用 SSH

如果不想让 GitHub Actions 持有 SSH 私钥，可以：

- 在服务器运行 Watchtower，定时拉取固定 tag；
- 或使用自托管 Runner；
- 或手动在服务器执行 `docker compose pull && docker compose up -d`。

当前推荐仍是 **tag + SSH + 健康检查/回滚**，因为发布可控、可回滚。
