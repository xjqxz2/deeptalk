# 小橘（DeepTalk）配置与运行教程

从零把项目跑起来，包含：环境准备 → 配置 → 本地开发 → 首次进入 → 配置模型 → 生产部署 → 常见问题。

---

## 0. 先理解「配置在哪」

项目**只有一个配置源**：`server/configs/app.yaml`。环境变量只是覆盖层，不是第二份配置。

```text
app.yaml（默认值，提交到仓库）
  └─ app-{env}.yaml（可选，按环境覆盖，如 app-prod.yaml）
      └─ secret.local.yaml（可选，本地密钥，已在 .gitignore）
          └─ DEEPTALK_* 环境变量（最高优先级，容器/CI 用）
```

**哪些东西不在配置文件里**（这是刻意的）：

| 内容 | 存在哪 | 在哪改 |
| --- | --- | --- |
| 厂商、模型、API Key | 数据库 | 管理后台「模型接入」 |
| 输出预算、生成超时、注册开关、检索来源 | 数据库（settings 表） | 管理后台「系统设置」 |
| 套餐、限额 | 数据库 | 管理后台「套餐」 |
| 管理员账号 | 数据库（迁移写入） | 空库首次启动自动创建 |

> 判断标准：**后台能改的，就不要写进配置文件**。否则同一件事有两个真相，改了一个另一个还在生效。

---

## 1. 环境准备

| 依赖 | 版本 | 说明 |
| --- | --- | --- |
| Go | 1.26+ | `go version` 确认 |
| Node.js | 22+ | `node -v` |
| pnpm | 9+ | `corepack enable && corepack prepare pnpm@latest --activate` |
| Obscura | 可选 | 联网检索的浏览器引擎；没有也能跑，自动回退普通 HTTP |

```bash
git clone <repo> deeptalk && cd deeptalk
```

Go 依赖与缓存统一放在仓库上一级（`Makefile` 已配好），首次 `make dev` 会自动下载。

---

## 2. 配置

### 2.1 最小可运行配置

复制环境变量样例，填一个密钥即可：

```bash
cp .env.example .env
# 生成密钥并填入 .env 的 DEEPTALK_SECRET
openssl rand -hex 32
```

`DEEPTALK_SECRET` 用来加密厂商 API Key 与会话签名。**首次启动后不要再改**，改了数据库里已加密的 Key 全部作废。

其它项默认值都能直接跑；要改端口、时区、上传目录等，优先改 `server/configs/app.yaml`。

### 2.2 常用配置项（app.yaml）

| 键 | 默认 | 说明 |
| --- | --- | --- |
| `app.addr` | `0.0.0.0:9821` | 监听地址；局域网访问必须是 `0.0.0.0` |
| `app.base_url` | `http://localhost:9821` | 对外地址，用于 Cookie 域与日志 |
| `app.timezone` | `Asia/Shanghai` | 注入给模型的时间上下文 |
| `database.dsn` | `data/deeptalk.db` | SQLite 路径（相对 server 目录） |
| `auth.session_ttl_hours` | `168` | 会话有效期 |
| `auth.default_quota_tokens` | `4000000` | 新用户默认额度 |
| `security.sign_mode` | `log` | 写请求签名：`off` / `log` / `enforce` |
| `upstream.reasoning_effort` | `medium` | 深度思考强度 |
| `context.compact_ratio` | `0.85` | 上下文压缩触发比例 |
| `web_search.enabled` | `true` | 是否启用联网检索 |
| `web_search.obscura_bin` | `obscura` | Obscura 可执行文件 |
| `mcp.ssh.enabled` | `true` | 是否给模型 SSH 能力 |
| `mcp.ssh.key_bits` | `4096` | 每用户 SSH 密钥位数 |
| `uploads.dir` / `max_mb` | `data/uploads` / `8` | 图片附件 |
| `wechat.app_id` / `secret` | 空 | 小程序凭据，不需要就留空 |

### 2.3 环境变量覆盖（只在需要时用）

`DEEPTALK_*` 覆盖同名配置，全部清单见 `.env.example`。容器部署时用它注入密钥，不要把密钥写进 `app.yaml`。

---

## 3. 本地跑起来

### 3.1 后端

```bash
cd server
make dev          # 等价于 go run .，监听 0.0.0.0:9821
```

启动后：

- 服务：<http://localhost:9821>
- 健康检查：<http://localhost:9821/healthz>

首次启动会自动建库（`server/data/deeptalk.db`），并在**空库**情况下写入管理员账号。

### 3.2 前端

```bash
cd ui
pnpm install
pnpm dev:c        # PWA，http://localhost:5173（代理到 9821）
pnpm dev:b        # 管理后台，http://localhost:5174/admin/
```

小程序：

```bash
cd ui/uniclient
pnpm dev:mp-weixin   # 用微信开发者工具导入 dist/dev/mp-weixin
pnpm dev:h5          # 浏览器预览，http://localhost:5273
```

---

## 4. 首次进入

**默认管理员：`admin` / `123456`**（空库首次启动时由迁移写入）。

请在登录后**立刻修改密码**：右上角账号菜单 →「账号与密码」。

> 库里只要有任意用户，迁移就不会再插入 admin，也不会改任何现有账号。

接着配置模型（见下一节），否则对话会返回「当前没有可用模型」。

---

## 5. 配置厂商与模型

全部在后台完成，不需要改配置文件。

1. 用管理员登录 <http://localhost:5174/admin/>
2. 左侧「模型接入」→「新增厂商」
   - 名称：如 `DeepSeek`
   - Base URL：如 `https://api.deepseek.com`
   - API Key：填入后在数据库里以 AES-GCM 加密存储
   - 协议：`chat`（默认）/ `responses` / `anthropic`，取决于上游
3. 在同一页添加模型：填写 `model_id`（上游的真实标识，如 `deepseek-chat`）、显示名、上下文窗口、三段价格
4. 「系统设置」里把**默认模型**设为刚添加的模型（可选设置兜底模型）
5. 回到 C 端发一条消息验证

> 也可以点击厂商卡片上的「测试」验证连通性，成功后卡片会显示延迟。

---

## 6. 生产部署

### 6.1 一键部署（推荐）

```bash
# 服务器目录
mkdir -p /srv/deeptalk && cd /srv/deeptalk
cp .env.production.example .env    # 填 DEEPTALK_SECRET / BASE_URL / 端口
./deploy.sh latest                  # 拉镜像并重启
curl http://127.0.0.1:9821/healthz
```

发布流水线：push `main` → GitHub Actions 构建镜像 → 拉取 → 重启 → 健康检查。

### 6.2 生产必改项

| 项 | 值 | 原因 |
| --- | --- | --- |
| `DEEPTALK_ENV` | `prod` | 启用生产校验 |
| `DEEPTALK_BASE_URL` | 真实域名 | Cookie 与会话 |
| `DEEPTALK_COOKIE_SECURE` | `true` | 仅 HTTPS 传输 |
| `DEEPTALK_SIGN_MODE` | `enforce` | 拒绝未签名写请求 |
| `DEEPTALK_SECRET` | 随机长串 | 加密密钥，**永不更换** |

数据目录 `<data>/ssh` 必须属于容器用户（否则 SSH MCP 报权限错误）：

```bash
chown -R 10001:10001 /srv/deeptalk/data/ssh
```

---

## 7. 常见问题

**Q：登录提示「请填写邮箱与密码」**
账号是 `admin` 还是邮箱都可以，但密码不能为空；确认 `.env` 没有被覆盖。

**Q：对话返回「当前没有可用模型」**
去后台添加厂商与模型，并在系统设置里设为默认模型。

**Q：联网检索很慢或没有结果**
默认顺序是 Obscura CLI → 普通 HTTP → MCP 浏览器。没有安装 Obscura 时只走 HTTP；可在「系统设置」把检索来源改为 `deepseek` 或 `builtin`。

**Q：改了 `DEEPTALK_SECRET` 后厂商都报鉴权失败**
密钥更换后已加密的 Key 无法解密，需要到后台重新填写 API Key。

**Q：如何新增管理员**
后台「用户管理」→ 目标用户右侧「···」→「设为管理员」。同一菜单可取消。
系统保证：不能取消自己，也不能取消最后一个管理员。

**Q：数据库能重置吗**
可以删掉 `data/deeptalk.db` 重来（会丢失全部数据）；**生产环境禁止**。

---

## 8. 常用命令

```bash
cd server && make test    # 后端单测
cd server && make vet     # 静态检查
cd ui && pnpm typecheck   # 前端类型检查
cd ui && pnpm build       # 构建 C/B 端产物
```

更多架构细节见仓库根目录 `AGENTS.md`。
