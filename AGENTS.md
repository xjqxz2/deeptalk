# AGENTS.md — 小橘（DeepTalk）

> **给 AI 代理和人类贡献者：动手前先读完本文件。**
> 本文件是项目的唯一事实来源（single source of truth）。任何改变**架构、流程、命令、约定、已知坑**的提交，**必须在同一提交内更新本文件**。
> 不要新建第二份说明文档来替代它；补充细节请链接到 `docs/`，并把入口写进第 11 节。

---

## 1. 项目是什么

「小橘」是一个带联网检索与思考链的个人 AI 对话工作台（内部代号 DeepTalk）。一个 Go 二进制同时提供 API、PWA、管理后台与小程序下载。

| 端 | 说明 |
| --- | --- |
| C 端 PWA | `ui/c`，移动优先，可安装，深色跟随系统 |
| B 端管理后台 | `ui/b`，挂在 `/admin/`，Ant Design Vue |
| 小程序 / APP | `ui/uniclient`（uni-app + `.nvue`），微信小程序与 H5 预览共用一套逻辑 |
| 后端 | `server`，Go 1.26 + Gin + GORM + SQLite |

核心能力：流式对话、思考链（始终开启）、联网检索（三种来源）、生成中插话、A2UI 卡片、图片附件、套餐与计费、会话分享、灵感录（memo）。

---

## 2. 技术栈

- **后端**：Go 1.26 / Gin / GORM / SQLite（WAL + busy_timeout）/ `log/slog` / `go:embed` 内嵌前端产物
- **前端**：Vue 3 + TypeScript + Vite + Pinia；C 端 PWA（Workbox），B 端 Ant Design Vue
- **共享包**：`@deeptalk/shared`（类型、请求封装、签名、主题令牌）
- **小程序**：uni-app（`.nvue` 编译为 WXSS/WXML）+ Vant（仅 H5 层）
- **镜像**：多阶段 Docker（Node 构建 UI → Go 编译 → Alpine + Obscura 运行时）

---

## 3. 目录结构

```text
server/
  apis/
    c/          C 端控制器（含 SSE 流与 WS）
    b/          管理后台控制器
    middleware/ 鉴权、签名、审计、限流
    requests/   入参 DTO（按业务域分文件）
    responses/  出参 DTO + 统一信封（按业务域分文件）
  bootstrap/    依赖装配：config、database、http、service 生命周期
  internal/
    chat/       会话、消息、流式生成、工具调用、上下文压缩
    provider/   厂商与模型、三种上游协议适配、加解密
    search/     自建检索、DeepSeek 检索、抓页、MCP/Obscura
    billing/    用量与三段计价
    sshmcp/     只做 SSH 的 MCP server（`deeptalk ssh-mcp` 子命令）
    user/ setting/ audit/ attachment/ memo/ wechat/ security/ logging/ seed/ mcp/ compliance/
  web/          内嵌前端产物（dist/c、dist/b；仓库内是占位文件）
  configs/      app.yaml（默认配置）、secret.local.yaml（本地密钥，勿提交）
ui/
  c/            PWA
  b/            管理后台
  shared/       共享类型与请求层
  uniclient/    小程序 / APP / H5
docs/           DEPLOY.md、design/、requirements/、qa/（含 ux 截图与 regress 回归证据）
scripts/        deploy.sh 等
Dockerfile      ui 构建 → Go 编译 → 运行层
```

---

## 4. 本地开发

```bash
# 后端（默认 0.0.0.0:9821）
cd server && make dev

# C 端 PWA（:5173，代理到 9821）
cd ui && pnpm dev:c

# 管理后台（:5174，路径 /admin/）
cd ui && pnpm dev:b

# 小程序（微信开发者工具导入 dist/dev/mp-weixin）
cd ui/uniclient && pnpm dev:mp-weixin

# 小程序 H5 预览
cd ui/uniclient && pnpm build:h5 && pnpm preview:h5
```

- 配置：`server/configs/app.yaml` + `server/configs/secret.local.yaml`；生产用 `DEEPTALK_*` 环境变量覆盖。
- Go 缓存目录统一在仓库上一级（`.gocache` / `.gomodcache` / `.gopath`），Makefile 已设置，不要改用全局缓存。
- 本地数据库与日志在 `server/data/`（SQL 日志按天滚动到 `data/logs/`，保留 7 天）。

---

## 5. 构建、测试与检查

```bash
cd server && make test      # go test ./...
cd server && make vet       # go vet ./...
cd server && make fmt       # gofmt -w .

cd ui && pnpm typecheck     # vue-tsc，C/B 两端
cd ui && pnpm build         # 两端产物，内嵌前需要
cd server && make ui        # 构建前端并拷进 server/web/dist
```

- 后端单测与控制器测试在 `server/internal/**/**_test.go`、`server/apis/router_test.go`。改动路由或鉴权务必跑 `go test ./apis/...`（重复注册路由会 panic）。
- 前端回归用 Playwright（`ui/` 下运行，依赖 `.browser-deps` 与 `~/.cache/ms-playwright/chromium_headless_shell-1248`）：

```js
const deps = path.resolve('../.browser-deps')
process.env.LD_LIBRARY_PATH = [path.join(deps,'root/usr/lib/x86_64-linux-gnu'), path.join(deps,'root/usr/lib'),
  path.join(deps,'root/lib/x86_64-linux-gnu'), process.env.LD_LIBRARY_PATH || ''].join(':')
process.env.FONTCONFIG_FILE = path.join(deps,'fonts.conf'); process.env.FONTCONFIG_PATH = deps
```

- 回归证据统一放 `docs/qa/regress/<范围>/`，文件名用 `BUG代码-结论.png`。

---

## 6. 发布流程

```text
push main → GitHub Actions（.github/workflows/docker.yml）
          → ghcr.nju.edu.cn/xjqxz2/deeptalk:latest
          → ssh root@peon.top 'cd /srv/deeptalk && ./deploy.sh latest'
          → curl http://127.0.0.1:9821/healthz
```

- 生产目录 `/srv/deeptalk`；数据库 `/srv/deeptalk/data/deeptalk.db`；环境变量在 `/srv/deeptalk/.env`。
- 生产保留最近 3 个镜像（`KEEP_IMAGES=3`），旧镜像自动清理。
- 生产是**只读发布目标**：先在本地/CI 验证，再部署；**永远不要 reset 生产数据库**，`DEEPTALK_SECRET` 不得更换（换了等于作废所有厂商密钥与签名）。
- 拉镜像偶发 500，重试即可；部署后必须确认 `healthz` 返回 `{"code":0,...,"status":"ok"}`。
- 小程序包：`ui/uniclient` 构建后打包上传 `/srv/deeptalk/artifacts/deeptalk-mp-weixin.zip`。

---

## 7. 架构要点（改代码前必读）

### 7.1 请求链路

`Gin 路由（/api/v1/c|b）→ 中间件（鉴权 → 签名 → 审计）→ 控制器 → internal 服务 → GORM`。
响应统一信封 `{code,msg,data}`；错误走 `responses.Fail/FromError`，不要直接写 HTTP 状态码。

### 7.2 流式回答（SSE，带断线续传）

- 入口 `POST /api/v1/c/conversations/:id/stream`，事件：`meta / delta / reasoning / reasoning_reset / tool_call / tool_result / sources / notice / usage / done / error / ping`。
- **生成与 HTTP 连接解耦**：`apis/c/stream_hub.go` 为每个「用户+会话」维护一个 `streamRun`，事件带 `id: <seq>` 写入缓冲，后台 goroutine 继续生成。
- 断线后客户端带 `{resume:true, last_seq:N}` 重新 POST，服务端补发 N 之后的事件再实时推送；回答结束后缓冲保留 2 分钟。
- **没有监听者也不取消**：跑完照常落库，用户下次进入会话能看到完整回答。生成中每 ~2 秒把已产出内容写回数据库。
- 停止必须显式调用 `POST /c/conversations/:id/stop`（断连接已经停不掉生成）；`GET /c/conversations/:id/stream-state` 返回是否有后台生成。
- 进程启动时会把遗留的 `streaming` 消息收尾为 `stopped`（重启后没有对应后台任务）。

### 7.3 小程序 WebSocket

`/api/v1/c/ws`，帧格式 `{type, ...}`：`send / stop / interject`。**`ws.send(frame)` 内部已经 `JSON.stringify`，调用方必须传对象，不能传字符串**（否则服务端解析失败直接断连）。

### 7.4 上游协议适配

每家厂商在 `provider.api_style` 三选一：

| api_style | 请求路径 | 鉴权 |
| --- | --- | --- |
| `chat` | `{base}/chat/completions` | `Authorization: Bearer` |
| `responses` | `{base}/responses` | `Authorization: Bearer` |
| `anthropic` | `{base}/v1/messages` | `x-api-key` + `anthropic-version: 2023-06-01` |

适配器在 `internal/provider/{upstream,responses,anthropic}.go`，统一产出 `StreamEvent{content|reasoning}`、`[]ToolCall`、`CompletionUsage{PromptTokens, CompletionTokens, CachedTokens}`。新增协议请复用这套中间表示，不要在业务层分支。

### 7.5 联网检索

后台「联网检索来源」三选一（`setting.search_provider`）：

- `self`：自建引擎（Bing/Sogou/Baidu…），取页顺序 **Obscura CLI → 普通 HTTP → MCP 浏览器**；MCP 很慢（导航要等 5s），连续失败 3 次熔断 10 分钟。
- `deepseek`：官方 `/v1/tools/search` 与 `/v1/tools/fetch`，按次计费。
- `builtin`：模型内置 `deepseek_web_search`（按 token 计费），此时不注册本地 `web_search` 工具，二者不能混用。

预算：引擎单次 6s、文章页 15s（上限 30s）、拿到结果后最多再等 1.5s、单次抓页 8s、每轮最多 4 次 `web_search`。
抓页失败**不提示用户**，直接用已收集的标题/摘要作答。

### 7.6 超时语义

后台「生成超时」（10–300s，生产 180）是**空闲超时**：只要流里有数据（含 `: keep-alive` 注释行）就重置计时，连续 N 秒没有任何字节才中止。不要把它改回「首字节超时」。

### 7.7 计费

`billing.CostMilliCentsCached(input, cached, output, priceInput, priceCached, priceOutput)`，单价为**元 / 百万 token**，内部单位 1 元 = 100000 milli-cents。缓存命中的 token 单独计价，三家上游字段不同：`prompt_cache_hit_tokens`、`prompt_tokens_details.cached_tokens`、`input_tokens_details.cached_tokens`、`cache_read_input_tokens`。

### 7.8 安全与签名

写请求（POST/PUT/PATCH/DELETE）需签名：`X-DT-Ts / Nonce / Key / Sign`，规范串 `METHOD\nPATH\nSORTED_QUERY\nSHA256(BODY)\nTS\nNONCE`。生产 `DEEPTALK_SIGN_MODE=enforce`。客户端密钥来自 `GET /c/auth/sign-key`，**签名失败会自动刷新密钥并重试一次**。GET/HEAD/OPTIONS 与 multipart 上传跳过签名。

### 7.9 SSH MCP（远端运维能力）

`internal/sshmcp` 是一个**独立 MCP server**，由主程序以 `deeptalk ssh-mcp` 子命令启动（stdio 传输），与主服务解耦。

- **只做 SSH**：不执行本机命令、不读写本机文件；命令全部在远端执行。
- **每用户一把密钥**：`<data>/ssh/<owner>/id_rsa`（RSA 4096，0600），首次使用自动生成，`ssh_public_key` 返回 authorized_keys 行与 SHA256 指纹。
- **主机指纹 TOFU**：首次连接写入 `<owner>/known_hosts`，之后不一致直接拒绝（可能中间人）。
- **会话归属校验**：会话绑定 owner，A 用户的 session_id 在 B 用户下直接报错。
- 工具：`ssh_public_key` / `ssh_connect`（host、port、username、可选 workdir）/ `ssh_exec`（默认 60s、上限 300s，输出 200KB 截断）/ `ssh_disconnect` / `ssh_sessions`。
- `workdir` 只是「每条命令前先 cd」的便利，**不是安全边界**；要硬隔离请给目标机上的专用低权限用户。
- `owner` 由宿主注入，模型不要自己设置；接入聊天时必须在服务端覆盖该字段。

---

## 8. 代码约定

- **注释用中文**，导出的类型、函数、关键分支都要有说明；前端同理。不要写「永真」注释。
- 后端分层：控制器只做参数校验与编排，业务放 `internal/<模块>`；`apis/requests` 与 `apis/responses` **按业务域分文件**（auth/conversation/provider/model/user/plan/settings/overview/usage），不要堆回单文件。
- 所有管理操作写审计：`recordAudit(c, h.audits, "user.delete", "user", id, "…")`。
- 前端请求一律走 `@deeptalk/shared/request.ts`（含签名与自愈），不要自己 `fetch`。
- 颜色一律用主题令牌 `--dt-*`，不允许在组件里写死十六进制；深色靠 `prefers-color-scheme` + 令牌，写死会导致深色下白底白字。
- 列表数据更新要用 `splice` 就地替换，**不要给 `run.messages` 赋新数组**（`messages.value` 持有原数组引用，赋值会让界面渲染旧数据）。
- ant-design-vue 的 `Modal.confirm` 返回控制器**不是 Promise**，必须用 `onOk/onCancel` 包一层 Promise 再 `await`，否则确认框形同虚设（曾导致「点删除直接删库」）。

---

## 9. 已知坑（都踩过，别再踩）

| 坑 | 后果 | 正确做法 |
| --- | --- | --- |
| `await Modal.confirm(...)` | 不等用户确认就执行 | 包 Promise + `onOk/onCancel` |
| 给列表 ref 赋新数组 | 界面不刷新 | `splice(0, len, ...next)` |
| `chatSocket.send(JSON.stringify(x))` | 双重编码，服务端断连 | 传对象 |
| `.nvue` 默认 `flex-direction: column` | 横向工具条竖排 | 显式 `row` |
| `.nvue` 用 `open-type="chooseAvatar"` / `type="nickname"` / `@click.stop` / `env()` | 整页白屏，无报错 | 用 `uni.chooseImage`、普通 input、固定 padding |
| 深色下写死浅色背景 | 白底白字（表头、行内代码） | 深色覆盖放样式表**末尾**并用令牌；`code:not(.hljs)` 保持 specificity |
| 首字节超时 | 慢模型边想边搜被误杀 | 用空闲超时 |
| MCP 浏览器优先 | 每次检索白等 5s | CLI → HTTP → MCP |
| 抓页失败弹提示 | 噪音，且当时已有可用结果 | 只写日志，用摘要作答 |
| 模型被删后压缩上下文 | `model not found`，压缩长期失败 | 回退默认模型 |
| 路由重复注册 | 测试 panic | 新增路由前先 grep |
| 在同一个 `bash -c` 里 `pkill -f <模式>` | 杀掉自己 | 用脚本文件或按 PID 杀 |

---

## 10. 维护规则（必须遵守）

**触发条件**（满足任意一条，就更新本文件）：

1. 新增/删除模块、路由分组、服务或前端页面；
2. 改变数据流（SSE、WS、工具调用、检索链路、计费口径）；
3. 改变构建、测试、发布命令或端口；
4. 新踩到并修好的坑（写进第 9 节，一行说清「现象 → 正确做法」）；
5. 新增环境变量、配置项、后台设置项。

写法要求：**事实、可执行、简短**。能一行说清就别写一段；过时内容立即删除，不做考古。

提交信息用 Conventional Commits：`feat(chat): …` / `fix(admin): …` / `docs(agents): …`，正文说明「为什么」，不要只写「改了什么」。

---

## 11. 文档索引

- `docs/DEPLOY.md` — 部署脚本、回滚、环境变量
- `docs/requirements/` — 需求地图
- `docs/design/` — 设计稿与小程序包
- `docs/qa/ux/` — 交互测试截图（pwa / admin / uni）
- `docs/qa/regress/` — 回归证据（b / c / c2 / c3）
- `SPEC.md`、`SPEC_v1.0.0.md` — 接口与数据模型规范
- `ChatDemo.md` — 流式交错（思考 ↔ 工具）协议说明
