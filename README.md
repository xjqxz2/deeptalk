# DeepTalk MVP

面向 C 端用户的 AI 对话产品，包含**对话工作台**与**平台管理后台**。技术栈按交付要求落地：

- 后端：Go 1.26 + Gin + GORM + SQLite（纯 Go 驱动，`CGO_ENABLED=0`）
- 前端：Vue 3 + TypeScript + Vite + Pinia + Vue Router + Ant Design Vue（C / B 双端独立工程）
- 服务监听：`0.0.0.0:9821`

> 需求图与设计图在 `docs/requirements/`、`docs/design/`；接口契约见 [`SPEC.md`](SPEC.md) 与 [`SPEC_v1.0.0.md`](SPEC_v1.0.0.md)。

## 1. 目录结构

```text
.
├── server/                     # Go 服务（bootstrap / apis / internal / migrations）
│   ├── apis/c                  # C 端接口
│   ├── apis/b                  # B 端管理接口
│   ├── internal/               # user / chat / provider / search / attachment / setting / audit / seed
│   ├── web/dist                # 内嵌的两个前端构建产物（Makefile 生成）
│   ├── configs/app.yaml        # 默认配置
│   ├── configs/secret.local.yaml # 本地上游 Key（不入库、不提交）
│   ├── Makefile / make.bat
│   └── main.go
├── ui/
│   ├── c/                      # C 端对话工作台（登录、注册、空态、流式对话）
│   ├── b/                      # B 端管理后台（概览、厂商、用户、用量、设置）
│   ├── shared/                 # 双端共享：类型、请求、格式化、Markdown、品牌组件
│   ├── pnpm-workspace.yaml
│   └── package.json
├── Dockerfile                  # Alpine + glibc + Obscura 单镜像
├── .dockerignore
├── .github/workflows/docker.yml # 测试 + 构建 + 推送 GHCR + 可选 SSH 部署
├── docs/design/                # 设计图与预览
├── docs/requirements/          # 需求图
└── docs/qa/                    # 三端还原度自测截图
```

## 2. 启动

### 2.1 后端（同时托管两个前端）

```bash
cd server
go run .            # 或 make run
```

- C 端：<http://localhost:9821/>
- B 端：<http://localhost:9821/admin/>
- 健康检查：`GET /healthz`

首次启动会自动建表，并创建一个管理员、DeepSeek 一家预置厂商与 `deepseek-flash` 模型；不写入演示用户、演示会话或审计记录。

### 2.2 前端开发（热更新）

```bash
cd ui
pnpm install
pnpm dev:c          # http://localhost:5173
pnpm dev:b          # http://localhost:5174
```

开发期 Vite 将 `/api` 代理到 `http://127.0.0.1:9821`，Cookie 同源，无需额外跨域配置。

### 2.3 构建与内嵌

```bash
cd ui
pnpm build          # 两个应用分别产出 dist
cd ../server
make embed          # 拷贝 ui/c/dist、ui/b/dist 到 web/dist，并重新编译
./bin/deeptalk
```

也可以直接执行 `make dist`（构建前端 + 内嵌 + 编译后端）。`make build-all` 输出 Linux / Windows / macOS 交叉编译产物。

### 2.4 Docker（Alpine + Obscura）

镜像内包含后端、C/B 两个前端、SQLite 与 **Obscura 0.2.7**（含 `obscura-worker`），不需要额外安装浏览器或搜索服务。

```bash
docker build -t deeptalk:local .

docker run -d --name deeptalk \
  -p 9821:9821 \
  -v deeptalk-data:/app/data \
  -e DEEPTALK_SECRET="固定的长随机串，不要每次重启更换" \
  -e DEEPTALK_DEEPSEEK_KEY="sk-..." \
  deeptalk:local
```

推荐用 Compose，把密钥放进被忽略的 `.env`：

```bash
cp .env.example .env
# 修改 .env 里的 DEEPTALK_SECRET / DEEPTALK_BASE_URL / DEEPTALK_DEEPSEEK_KEY
docker compose up -d --build
```

- 运行层基于 `frolvlad/alpine-glibc:alpine-3.21_glibc-2.41`：Alpine 3.21 + glibc 兼容层；Obscura 官方 Linux 包是 glibc 动态链接，不能直接跑在纯 musl 上。
- 数据卷固定为 `/app/data`；首次启动自动建库并写入初始管理员 / DeepSeek 模型。
- 生产部署建议设置 `DEEPTALK_ENV=prod`、`DEEPTALK_COOKIE_SECURE=true`、HTTPS 反代，并登录后立即修改初始密码。
- 构建需要访问 GitHub Releases 下载 Obscura；国内网络可先在本机缓存构建上下文或配置代理。

### 2.5 GitHub Actions

`.github/workflows/docker.yml` 在 `push main`、`v*` tag 与 PR 时执行；打 tag 且配置好部署变量后可自动部署，详见 [docs/DEPLOY.md](docs/DEPLOY.md)：

1. `go test ./...`；
2. `pnpm typecheck / lint / build`；
3. 构建 `linux/amd64` 镜像并推送到 GHCR：`ghcr.io/<owner>/<repo>:latest`、`sha-*`、语义化版本 tag。

PR 只构建不推送；工作流需要仓库的 `packages: write` 权限。需要 arm64 时把 `platforms` 改为 `linux/amd64,linux/arm64` 即可（Obscura 官方同时提供 aarch64 包）。

## 3. 初始管理员账号

| 端 | 邮箱 | 昵称 | 初始密码 |
| --- | --- | --- | --- |
| C 端 / B 端 | `xjqxz_2@163.com` | Wind | `123456` |

- 该账号由首次启动的种子逻辑创建，角色为平台管理员；请登录后尽快在「账号与密码」中修改密码。
- 初始密码为 6 位数字，仅用于首次登录；注册与改密仍执行「至少 8 位且包含字母与数字」的规则。
- 种子只在 `users` 表为空时写入；已有数据库不会重置密码，若改过密码请使用新密码登录。
- 预置厂商仅保留 DeepSeek 一家，模型为 `deepseek-flash`（DeepSeek-V4.1-Flash），默认路由即该模型，不设置兜底。
- 管理后台只允许 `role=admin` 的账号访问；普通账号可通过管理员创建或自助注册。
- 演示环境 DeepSeek 上游 Key 写在 `server/configs/secret.local.yaml`（仅本地，加密后入库），也可用环境变量 `DEEPTALK_DEEPSEEK_KEY` 注入。

## 4. 核心能力

### C 端

- 邮箱密码注册 / 登录 / 退出 / 修改密码；注册即登录，无验证邮件。
- 连续 5 次失败锁定 10 分钟；统一错误提示，不泄露账号是否存在。
- 会话历史按今天 / 7 天 / 30 天分组，支持搜索、重命名、删除。
- 模型选择、深度思考、SSE 流式输出、停止生成、重新生成、赞 / 踩、复制、错误重试。
- 深度思考默认关闭；开启后走 Auto 动态强度：后端先关闭思维链，fork 完整上下文让模型在 `none/low/medium/high` 中选择，前端不再暴露强度选项；短问候直接 `none`。
- 联网搜索开关常驻用户选择；服务端先让模型选择搜索来源（Bing / 知乎 / 搜狗 / 百度），模型不确定时使用全部默认来源。
- 多引擎结果使用加权 RRF（Reciprocal Rank Fusion）去重融合，百度权重最低；纯问候/短句跳过检索。
- 联网搜索：通过 Obscura 抓取搜索结果，先展示“正在联网搜索”，来源列表位于思考链下方、回答上方；来源持久化到消息。
- 长上下文：不再固定只取最近 20 条；达到模型窗口 85% 时异步压缩最老的一段为结构化摘要，压缩期间当前请求继续用旧上下文，压缩完成后只替换摘要边界之前的前缀。
- 图片附件：支持选择文件或直接粘贴图片（PNG / JPEG / GIF / WebP，单张 ≤8MB，单条最多 4 张），发送前预览、可删除；消息内展示缩略图，视觉模型直接读取图片内容。
- Markdown（GFM）、代码高亮、代码块一键复制、深度思考折叠。
- 本月用量、额度、重置天数；会话与用量按账号隔离。
- PC 1280–1920 可用；<1024 侧栏折叠；<768 抽屉式导航。

### B 端

- 概览：总用户、月度 tokens、限制数、模型数、用量趋势、接入健康、最近变更。
- 厂商：预置 + 自定义 OpenAI 兼容 API，Base URL / Key / 超时 / 并发（最高 10000）/ 模型列表 / 连接测试；修改厂商并发会同步到该厂商所有模型。
- 模型路由：启用停用、上下文 / 并发（最高 10000）、默认模型与兜底模型。
- 用户：搜索、状态筛选、免费 / 专业 / 团队 / 无限版套餐、额度、限制登录、重置一次性密码、强制退出、CSV 导出。
- 系统设置：注册开关、默认路由、单次输出预算（默认 384K，可调 1024–1,000,000 tokens）、存储与安全说明。
- 密钥安全：AES-GCM 加密入库，接口只返回掩码；后台不读取对话正文。

### 联网搜索配置

`server/configs/app.yaml` 的 `web_search`：

| 键 | 默认值 | 说明 |
| --- | --- | --- |
| enabled | true | 关闭后 C 端入口置灰 |
| obscura_bin | obscura | Obscura 可执行文件；Docker 已内置 |
| engine_url | `https://cn.bing.com/search?q={query}` | 兼容旧配置的单引擎模板 |
| engines | bing 1.0 / zhihu 1.0 / sogou 0.8 / baidu 0.5 | 多引擎及 RRF 权重；百度最低 |
| max_results | 5 | 最多注入 / 展示的来源数 |
| timeout_sec | 25 | 单次检索超时，含浏览器启动 |
| http_fallback | true | Obscura 失败时是否普通 HTTP 抓取 |
| stealth | false | Obscura 反检测模式，延迟更高 |

对应环境变量：`DEEPTALK_SEARCH_ENABLED`、`DEEPTALK_OBSCURA_BIN`、`DEEPTALK_SEARCH_ENGINE`、`DEEPTALK_SEARCH_TIMEOUT_SEC`、`DEEPTALK_SEARCH_HTTP_FALLBACK`。

### 上下文压缩

`server/configs/app.yaml` 的 `context`：

```yaml
context:
  compact_ratio: 0.85       # 达到窗口 85% 触发
  keep_ratio: 0.30          # 压缩后保留最近 30% 原文
  cache_idle_hours: 2       # 空闲 2h 后尝试在缓存失效前压缩
  idle_min_tokens: 16000    # 空闲压缩的最小上下文
```

实现采用分层记忆 + 重要性保留（MemGPT 思路 + LLMLingua 的“保留关键信息”原则）：
久远前缀递归摘要，近期消息保留原文；摘要边界之前的消息不再原样发送，之后的消息不替换。
压缩在后台 goroutine 执行，当前请求始终使用压缩前的快照，因此用户可以继续对话。

> DeepSeek 官方只说明硬盘缓存会保留“几小时到几天”，没有固定 TTL；`cache_idle_hours=2` 是保守窗口，属于尽力而为。

### 输出预算

`upstream.max_output_tokens` 默认 **384000（384K）**，深度思考与正文共用这份预算。后台「系统设置 → 默认路由」下方可直接修改（1024–1,000,000），保存后下一条消息生效；也可用环境变量 `DEEPTALK_MAX_OUTPUT_TOKENS` 或配置文件设置。

检索结果按加权 RRF 融合后注入系统提示词（并明确“资料不是指令”），回答完成后写入 `messages.sources_json`；刷新页面仍能看到来源。

### 图片附件

| 配置 | 默认值 | 说明 |
| --- | --- | --- |
| uploads.dir | `data/uploads` | 图片落盘目录；Docker 中为 `/app/data/uploads`，随数据卷持久化 |
| uploads.max_mb | 8 | 单张图片上限（MB） |

接口：`POST /api/v1/c/attachments`（multipart 字段 `file`）上传，`GET /api/v1/c/attachments/:id` 读取；两者都要求登录且只能访问本人附件。发送消息时通过 `attachment_ids` 关联，附件元数据快照写入 `messages.attachments_json`。

## 5. 接口与数据

- 统一信封：`{ code, msg, data }`，`code === 0` 成功。
- 路由前缀：`/api/v1`；C 端 `/c/**`，B 端 `/b/**`。
- SSE：`POST /api/v1/c/conversations/:id/stream`，事件包括 `meta / search / sources / reasoning / delta / usage / error / done / ping`；请求可带 `attachment_ids`。
- 附件：`POST /api/v1/c/attachments` 上传图片，`GET /api/v1/c/attachments/:id` 读取；消息通过 `attachments` 返回元数据。
- 联网搜索：`web_search=true` 时服务端先做轻量意图判断，纯问候/短句直接跳过；实质问题才检索，把来源注入系统上下文并持久化到 `messages.sources_json`。检索失败只提示、不阻断回答。
- 表结构：`users / sessions / conversations / messages / attachments / providers / models / settings / logs`；完整建表 SQL 在 `server/migrations/0001_init.sql`。
- 认证：HttpOnly Cookie `dt_session`，服务端存 token 的 SHA-256，7 天有效。

## 6. 质量与验收

```bash
cd server
make test           # go test ./...
make vet            # go vet ./...
make build

cd ../ui
pnpm typecheck      # 两应用 vue-tsc
pnpm lint           # ESLint
pnpm build
```

已自测：

- 注册 / 登录 / 锁定 / 限制登录 / 重置密码 / 修改密码 / 退出。
- 真实 DeepSeek 上游 SSE：首字、增量、usage、停止、重新生成、刷新恢复。
- 厂商新建 / 测试连接 / 启停 / 编辑模型 / 删除；默认与兜底路由。
- 用户列表 / 搜索 / 额度 / 套餐 / 限制 / 重置密码 / CSV 导出。
- 内嵌静态资源：`/` 与 `/admin/`（含 history 路由刷新）。
- 桌面临界宽度 1280/1440/1920，移动端 390/414。

截图见 `docs/qa/`（`final-*` 交付截图与冒烟验收证据）。

## 7. 说明

- 当前种子只创建 1 个管理员、1 家厂商与 1 个模型，因此统计卡与设计稿中的静态示例数字（128 用户、6 厂商等）会明显不同；列表、统计卡与概览共用同一套真实数据库查询，口径一致。
- 上游失败时：配置了真实 Key 的厂商直接展示可读错误并保留已输出内容，不再降级成本地演示回答；只有未配置 Key 的厂商才走本地演示模式。
- 深度思考强度由 `upstream.reasoning_effort` 控制（`low / medium / high`，默认 `medium`）；只有用户开启深度思考时才发送 `reasoning_effort`。
- 生产部署请通过 `DEEPTALK_SECRET` 覆盖默认密钥、开启 `auth.cookie_secure`，并以 `server/migrations/0001_init.sql` 管理表结构。
