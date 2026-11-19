# DeepTalk MVP 接口契约 · SPEC v1.0.0

> 依据 `docs/requirements/` 五张需求图与 `docs/design/` 六张设计图编写。
> 技术栈：Go 1.26 + Gin + GORM + SQLite（纯 Go 驱动）；前端 Vue 3 + TypeScript + Vite + Ant Design Vue。
> 服务监听：`0.0.0.0:9821`；路由统一前缀 `/api/v1`。

## 0. 全局约定

### 0.1 统一响应信封

所有 JSON 接口（SSE 除外）返回：

```json
{ "code": 0, "msg": "ok", "data": {} }
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| code | int | 0 成功；>0 失败，见 0.2 |
| msg | string | 成功为 `ok`，失败为可直接展示给用户的中文消息 |
| data | object / array / null | 业务数据；失败时为 `null` |

### 0.2 错误码

| code | 含义 | HTTP |
| --- | --- | --- |
| 0 | 成功 | 200 |
| 1001 | 参数不合法 | 400 |
| 1002 | 未登录 / 会话失效 | 401 |
| 1003 | 无权限（非管理员） | 403 |
| 1004 | 资源不存在 | 404 |
| 1005 | 冲突（邮箱已注册等） | 409 |
| 1006 | 账号被锁定 / 登录受限 | 423 |
| 1007 | 额度不足 | 402 |
| 1008 | 注册已关闭 | 403 |
| 2001 | 服务端内部错误 | 500 |
| 3001 | 上游模型调用失败 | 502 |

### 0.3 鉴权

- 登录 / 注册成功后由服务端下发 **HttpOnly Cookie**：`dt_session`，`Path=/`，`SameSite=Lax`，有效期 7 天。
- 前端所有请求携带 `credentials: 'include'`；开发期通过 Vite 代理同源访问，不产生跨域 Cookie 问题。
- 管理后台接口 `/api/v1/b/**` 额外校验 `role == admin`。
- 会话在服务端存储（表 `sessions`，仅存 token 的 SHA-256），用户被限制登录、改密、强制退出时立即失效。
- 公开接口：`POST /c/auth/register`、`POST /c/auth/login`、`GET /c/auth/config`、`GET /healthz`。

### 0.4 分页

列表类接口统一：`page`（默认 1）、`page_size`（默认 8，最大 50）；响应含 `total / page / page_size`。

---

## 1. C 端 · 账号与安全

### 1.1 注册配置（公开）

`GET /api/v1/c/auth/config`

Response `data`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| allow_registration | bool | 后台“开放注册”开关，关闭后注册页隐藏表单 |
| has_users | bool | 是否已有账号；首个注册账号自动成为管理员 |
| site_name | string | 站点名，固定 `DeepTalk` |
| web_search_enabled | bool | 后台是否配置了可用的联网搜索；false 时 C 端入口置灰 |

### 1.2 注册

`POST /api/v1/c/auth/register`

Request Body：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| email | string | 是 | 邮箱，统一转小写后唯一；不发送验证邮件 |
| password | string | 是 | ≥8 位且同时包含字母与数字 |

Response `data.user`：

```json
{
  "id": 1,
  "email": "xjqxz_2@163.com",
  "nickname": "Wind",
  "role": "admin",
  "plan": "free",
  "quota_tokens": 4000000,
  "used_tokens": 0,
  "used_percent": 0,
  "disabled": false,
  "created_at": "2026-10-02T10:00:00+08:00",
  "last_login_at": null,
  "last_login_city": "",
  "avatar_text": "W"
}
```

异常：`1005` 邮箱已注册；`1008` 注册已关闭；`1001` 密码规则不满足。
行为：注册成功即写会话 Cookie（注册即登录）；首个账号 `role=admin`，其余 `role=user`；昵称取邮箱 `@` 前缀。

### 1.3 登录

`POST /api/v1/c/auth/login`

Request Body：`email`、`password`。

行为与异常：

- 成功：下发 Cookie，刷新 `last_login_at`，清零失败计数。
- 邮箱不存在 / 密码错误统一返回 `1001` + `邮箱或密码不正确`，不泄露账号是否存在。
- 连续 5 次失败锁定 10 分钟：`1006` + `登录失败次数过多，请 10 分钟后再试`。
- 被限制登录：`1006` + `该账号已被限制登录，请联系管理员`。
- 响应 `data.user` 结构同 1.2。

### 1.4 退出

`POST /api/v1/c/auth/logout` → 撤销当前会话并清除 Cookie，`data=null`。

### 1.5 当前用户

`GET /api/v1/c/auth/me` → `data.user`；未登录返回 `1002`。

### 1.6 修改密码

`PUT /api/v1/c/auth/password`

Request Body：`old_password`、`new_password`（≥8 位且含字母与数字）。

行为：校验旧密码；更新 Argon2id 哈希；撤销该用户全部会话（含当前），前端跳转登录页。
异常：`1001` 旧密码错误 / 新密码不合规。

### 1.7 用量

`GET /api/v1/c/usage` → `data`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| used_tokens | int64 | 本月已用 tokens |
| quota_tokens | int64 | 套餐额度 |
| used_percent | int | 0–100，向上取整 |
| reset_days | int | 距额度重置剩余天数 |
| plan | string | free / pro / team / unlimited |

### 1.8 可用模型

`GET /api/v1/c/models` → `data.items[]`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | int | 模型主键，用于发起对话 |
| model_id | string | 上游模型 ID，如 `deepseek-flash` |
| display_name | string | 展示名，如 `DeepSeek-V4.1-Flash` |
| description | string | 用途说明 |
| provider_name | string | 厂商展示名 |
| context_window | int | 上下文窗口（tokens） |
| is_default | bool | 是否平台默认 |

仅返回 `enabled=true` 的模型；普通用户不接触任何上游 Key。

### 1.9 图片附件

`POST /api/v1/c/attachments`，`Content-Type: multipart/form-data`，字段 `file`：

- 仅支持 PNG / JPEG / GIF / WebP；服务端用 `http.DetectContentType` + 图片解码二次校验；
- 单张 ≤ `uploads.max_mb`（默认 8MB），单条消息最多 4 张；
- 返回 `data.attachment`：`{id,name,mime,size,width,height,url}`；
- 文件随机命名落盘，原始文件名仅用于展示；记录归属用户。

`GET /api/v1/c/attachments/:id`：返回图片二进制，要求登录且只能读取本人附件；响应带 `Cache-Control: private` 与 `X-Content-Type-Options: nosniff`。

发送消息时通过 `attachment_ids` 关联；图片以 data URL 形式作为 OpenAI 兼容的多模态 `image_url` 内容发送给上游。历史消息不重复上传图片，重新生成会复用当前用户消息的附件快照。

---

## 2. C 端 · 会话与消息

会话/消息全部按 `user_id` 隔离，越权访问统一 `1004`。

### 2.1 会话列表

`GET /api/v1/c/conversations?keyword=SSE`

Response `data.items[]`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | int | 会话 ID |
| title | string | 标题；新建为“新对话”，首轮回答完成后自动生成 ≤16 字标题 |
| model_id | int | 最近使用的模型主键 |
| updated_at | string | 最近更新时间（RFC3339） |
| message_count | int | 消息条数 |

按 `updated_at` 倒序；前端按今天 / 7 天内 / 30 天内 / 更早分组。

### 2.2 新建会话

`POST /api/v1/c/conversations`，Body 可空或 `{"title":"..."}` → `data.conversation`（同上单项）。
生成中禁止切换会话的约束由前端保证，服务端允许并发会话。

### 2.3 会话详情与消息

`GET /api/v1/c/conversations/:id` → `data`：

```json
{
  "conversation": { "id": 6, "title": "SSE 超时与重连策略", "model_id": 1, "updated_at": "..." },
  "messages": [
    {
      "id": 11,
      "role": "user",
      "content": "帮我把这段 SSE 代理的超时与重连逻辑改成可配置的，并补上单元测试。",
      "reasoning": "",
      "status": "complete",
      "model_name": "",
      "input_tokens": 26,
      "output_tokens": 0,
      "duration_ms": 0,
      "feedback": "",
      "sources": [],
      "attachments": [],
      "created_at": "2026-11-19T14:32:00+08:00"
    }
  ]
}
```

消息字段：`role` ∈ `user|assistant`；`status` ∈ `complete|streaming|stopped|error`；`feedback` ∈ `""|up|down`；`sources` 是联网来源数组（没有检索时为空数组）；`attachments` 是图片附件元数据数组。

### 2.4 重命名 / 删除会话

- `PATCH /api/v1/c/conversations/:id`，Body `{"title":"..."}`（1–60 字）→ `data.conversation`
- `DELETE /api/v1/c/conversations/:id` → `data=null`（级联删除消息）

### 2.5 流式发送（SSE）

`POST /api/v1/c/conversations/:id/stream`

Request Body：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| content | string | 是 | 用户输入，1–8000 字 |
| model_id | int | 否 | 模型主键；缺省用平台默认模型 |
| deep_thinking | bool | 否 | 是否请求深度思考；前端默认关闭，开启后按服务端 `reasoning_effort` 请求 |
| web_search | bool | 否 | 是否允许联网检索；开关常驻，纯问候/短句由服务端跳过 |
| attachment_ids | int[] | 否 | 本次消息的图片附件 ID，最多 4 个；重新生成时忽略并复用上一条用户消息 |
| regenerate | bool | 否 | 重新生成：忽略 content，重跑最后一条用户消息 |

Response：`text/event-stream`，`Cache-Control: no-cache`，`X-Accel-Buffering: no`。

事件序列：

```
event: meta
data: {"conversation_id":6,"assistant_message_id":12,"model_name":"DeepSeek-V4.1-Flash","title":"SSE 超时与重连策略"}

event: search
data: {"status":"searching"}

event: sources
data: {"items":[{"title":"来源标题","url":"https://example.com","snippet":"摘要"}]}

event: reasoning
data: {"delta":"先确认 SSE 的取消语义……"}

event: delta
data: {"delta":"把超时、重试……"}

event: usage
data: {"input_tokens":26,"output_tokens":312,"total_tokens":338,"duration_ms":1200,"tokens_per_sec":18}

event: done
data: {"assistant_message_id":12,"status":"complete","title":"..."}
```

- 失败：`event: error` + `data: {"code":3001,"message":"可展示的中文原因"}`，随后关闭流；已插入的用户消息保留。
- 联网检索：`web_search=true` 时先做轻量意图判断；纯问候/短句直接跳过，实质问题才发 `search` / `sources`。模型先选择来源（cn.bing / 知乎 / 搜狗 / 百度）；不确定时中文问题回退知乎/搜狗/百度，英文问题回退 cn.bing/知乎；多引擎用加权 RRF 融合并做问题词相关性过滤，百度权重最低。来源面板位于思考链下方；检索结果作为系统上下文注入并写入 `messages.sources_json`。
- 图片：`attachment_ids` 在 `meta.user_message.attachments` 中回显；当前用户消息以多模态 `content` 数组发给上游；历史消息只保留文字与附件元数据，避免重复传输大图。
- 深度思考：`deep_thinking=true` 时服务端先关闭思维链，fork 完整上下文让模型选择 `none/low/medium/high`（Auto）；短问候直接 `none`。前端只保留一个开关，不展示强度。
- 心跳：空闲每 15s 发送 `event: ping`。
- 客户端中断（停止生成 / Esc / 关闭页面）：服务端 `context` 取消，保留已输出内容并把消息标记为 `stopped`。
- 额度不足：返回 `1007`（SSE 前置 JSON 响应），提示“本月额度已用完，请联系管理员调整额度”。
- 演示模式：仅当厂商未配置 API Key 时走本地模拟上游；已配置 Key 但上游失败时直接返回错误，不再用演示文案掩盖问题。
- 短问候（如 `HI` / `你好`）即使 `deep_thinking=true` 也跳过 reasoning，直接走普通回答。

### 2.6 消息反馈

`POST /api/v1/c/messages/:id/feedback`，Body `{"rating":"up"|"down"|""}` → `data=null`；仅允许操作本人助手消息。

---

## 3. B 端 · 管理后台

### 3.1 概览

`GET /api/v1/b/overview` → `data`：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| total_users | int | 用户总数 |
| new_this_month | int | 本月新增 |
| active_7d | int | 7 日活跃（按最近登录） |
| active_rate | int | 周活跃率百分比 |
| monthly_tokens | int64 | 本月 tokens 总量 |
| restricted | int | 已限制登录账号数 |
| quota_alert | int | 额度将尽账号数（≥90%） |
| providers_enabled | int | 已启用厂商数 |
| models_enabled | int | 已启用模型数 |
| conversation_count | int | 平台会话总数 |
| monthly_messages | int | 本月消息数 |

### 3.2 厂商列表

`GET /api/v1/b/providers?filter=all|builtin|custom`

`data`：`{ items: Provider[], counts: {all, builtin, custom, enabled}, health: Health }`

Provider：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | int | 主键 |
| name | string | 显示名称 |
| code | string | 唯一标识 |
| kind | string | `deepseek|openai|aliyun|moonshot|custom` |
| base_url | string | 上游 Base URL |
| api_key_masked | string | 掩码（`sk-••••1234`），Key 只写不读 |
| has_api_key | bool | 是否已配置密钥 |
| timeout_sec | int | 超时秒数 |
| max_concurrency | int | 最大并发 |
| enabled | bool | 启用开关 |
| verified | bool | 连接测试是否通过 |
| status | string | `active|pending|failed|disabled`，用于卡片状态胶囊 |
| is_builtin | bool | 预置厂商 |
| is_default | bool | 平台默认厂商 |
| model_count | int | 模型数量 |
| models | Model[] | 该厂商模型（列表接口可选返回） |

### 3.3 新建 / 编辑厂商

- `POST /api/v1/b/providers`
- `PUT /api/v1/b/providers/:id`

Request Body：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| name | string | 是 | 显示名称 |
| base_url | string | 是 | 必须 http(s)；生产禁止私网地址（本地开发放行 127.0.0.1） |
| api_key | string | 新建必填 | 为空表示编辑时保留原 Key |
| timeout_sec | int | 否 | 默认 60，范围 5–300 |
| max_concurrency | int | 否 | 默认 4，范围 1–10000；厂商并发变化时同步到该厂商全部模型 |
| enabled | bool | 否 | 保存后是否启用 |
| is_default | bool | 否 | 是否设为平台默认厂商 |
| models | ModelInput[] | 否 | 全量覆盖该厂商模型列表 |

ModelInput：`model_id`（必填）、`display_name`、`description`、`context_window`、`max_concurrency`、`enabled`。

行为：Key 使用 AES-GCM 加密入库；保存前不强制测试（失败不覆盖现有配置）；写审计日志。

`PATCH /api/v1/b/providers/:id`，Body `{"enabled":bool}`：关闭厂商需二次确认；关闭后其模型不再下发给 C 端。

`DELETE /api/v1/b/providers/:id`：自定义厂商可删，预置厂商禁止删除（`1003`）；有关联模型时级联删除。

### 3.4 连接测试

- `POST /api/v1/b/providers/:id/test`：测试已保存厂商。
- `POST /api/v1/b/providers/test`：测试未保存配置（Body 同 3.3，`id` 可选用于复用已存 Key）。

Response `data`：`{ ok, latency_ms, models_found, models: string[], message }`

规则：10s 超时；成功返回延迟与模型数；失败返回可读原因（`401 密钥无效`、`无法连接上游` 等）；演示厂商返回本地模拟结果。

### 3.5 模型路由

- `GET /api/v1/b/models` → `data.items[]`：Model + `provider_name`、`is_default`、`is_fallback`、`sort_order`
- `POST /api/v1/b/models` / `PUT /api/v1/b/models/:id`
- `PATCH /api/v1/b/models/:id/enabled`，Body `{"enabled":bool}`
- `GET /api/v1/b/routing` → `{ default_model_id, fallback_model_id }`
- `PUT /api/v1/b/routing`，Body 同上；停用模型不可设为默认 / 兜底。

Model 字段：`id`、`provider_id`、`provider_name`、`model_id`、`display_name`、`description`、`context_window`、`max_concurrency`（1–10000）、`enabled`、`is_default`、`is_fallback`。

### 3.6 接入健康

`GET /api/v1/b/health` → `{ upstreams_total, upstreams_available, errors_1h, first_byte_p95_ms, updated_at, items: [{provider_id,name,ok,latency_ms}] }`。
演示环境由最近测试结果 + 启用状态推导；30s 自动刷新由前端负责。

### 3.7 最近变更

`GET /api/v1/b/audit-logs?limit=6` → `{ items: [{id,actor_name,action,detail,created_at}] }`。

### 3.8 用户统计

`GET /api/v1/b/users/stats` → `{ total,new_this_month,active_7d,active_rate,monthly_tokens,monthly_growth,restricted,quota_alert }`。

### 3.9 用户列表

`GET /api/v1/b/users?q=&status=all|normal|restricted|quota&sort=recent|created|usage&page=1&page_size=8`

`status`：`normal` 正常；`restricted` 已限制；`quota` 额度将尽（≥90% 且未受限）。
`data`：`{ items: User[], total, page, page_size, stats }`，User 结构见 1.2，另含 `conversation_count`、`last_login_ip`。

### 3.10 用户详情

`GET /api/v1/b/users/:id` → `data.user`，含 `conversation_count`、`message_count`、`quota_reset_days`、`last_login_ip`。不返回任何对话正文。

### 3.11 调整额度

`PATCH /api/v1/b/users/:id/quota`，Body `{"quota_tokens": 4000000}`（≥已用量，1M–100M）。写审计日志。

### 3.11.1 调整套餐

`PATCH /api/v1/b/users/:id/plan`，Body `{"plan":"free|pro|team|unlimited"}`。

- 套餐决定默认额度：free 4M、pro 20M、team 50M；unlimited 不设月度上限（quota=0），不可单独调整额度。
- 若已用量超过新套餐额度，额度会被抬到「已用量 + 1M」，避免出现负剩余。
- 写审计日志。

### 3.12 限制 / 恢复登录

`PATCH /api/v1/b/users/:id/disabled`，Body `{"disabled": true}`。

- 限制：立即撤销该用户全部会话，后续登录返回 `1006`；管理员账号不可被限制（`1003`）。
- 恢复：用户需重新登录。

### 3.13 重置密码

`POST /api/v1/b/users/:id/reset-password` → `data: { "password": "Dt-8f3k2m", "expires_hint": "仅显示一次" }`。
生成一次性强密码，撤销全部会话；不发送邮件。写审计日志。

### 3.14 强制退出 / 新建账号 / 删除

- `POST /api/v1/b/users/:id/logout-all` → 撤销全部会话。
- `POST /api/v1/b/users`，Body `{email,password,nickname,plan,quota_tokens}`，用于关闭注册后管理员建号。
- `DELETE /api/v1/b/users/:id`：MVP 不物理删除，返回 `1003` 并提示“删除账号为 P1”。

### 3.15 导出 CSV

`GET /api/v1/b/users/export.csv` → `text/csv`，字段仅账号字段：`id,email,nickname,role,plan,quota_tokens,used_tokens,disabled,created_at,last_login_at`。不含任何对话正文。

### 3.16 系统设置

- `GET /api/v1/b/settings` → `{ allow_registration, default_model_id, fallback_model_id, max_output_tokens, site_name, version, storage_used_bytes, storage_total_bytes }`
- `PUT /api/v1/b/settings`，Body `{allow_registration?: bool, default_model_id?: int, fallback_model_id?: int, max_output_tokens?: int}`，写审计日志；输出预算范围 1024–1,000,000，默认 384000，下一条消息生效。

### 3.17 用量统计

`GET /api/v1/b/usage?days=14` → `{ days: [{date,tokens,messages}], totals: {tokens,messages,active_users} }`，按天聚合最近消息；不足的天补 0。

---

## 4. 数据模型（SQLite）

| 表 | 关键字段 | 说明 |
| --- | --- | --- |
| users | id,email(uniq),nickname,password_hash,role,plan,quota_tokens,used_tokens,disabled,failed_attempts,locked_until,created_at,last_login_at,last_login_ip,last_login_city | 账号；邮箱小写唯一 |
| sessions | id,user_id,token_hash,expires_at,revoked_at,ip,user_agent,created_at | 服务端会话 |
| conversations | id,user_id,title,model_id,created_at,updated_at,deleted_at,summary,summary_until_id,summary_tokens,summary_updated_at | 会话按用户隔离；summary 为前缀压缩记忆 |
| messages | id,conversation_id,user_id,role,content,reasoning,status,model_name,input_tokens,output_tokens,duration_ms,feedback,sources_json,attachments_json,created_at | 消息、用量、联网来源与附件快照 |
| attachments | id,user_id,message_id,name,mime,size,width,height,storage_name,created_at | 图片附件；storage_name 随机且唯一 |
| providers | id,name,code(uniq),kind,base_url,api_key_enc,api_key_masked,timeout_sec,max_concurrency,enabled,verified,is_builtin,is_default,sort_order,created_at,updated_at | 上游厂商 |
| models | id,provider_id,model_id,display_name,description,context_window,max_concurrency,enabled,is_default,is_fallback,sort_order | 模型路由 |
| settings | key(uniq),value,updated_at | allow_registration / default_model_id / fallback_model_id |
| logs | id,actor_id,actor_name,actor_email,action,target_type,target_id,detail,ip,created_at | 后台操作审计 |

迁移脚本放 `server/migrations/0001_init.sql`；启动时使用 GORM `AutoMigrate` 以便 MVP 首次运行，生产以迁移脚本为准。

## 5. 设计还原约定（前端）

- 设计令牌：主色 `#4D6BFE`（hover `#3B5BDB`）、浅底 `#EEF2FF`、正文 `#101828`、次级 `#475467`、弱化 `#667085/#98A2B3`、分割线 `#EAECF0/#EFF1F5`、页面底 `#F7F8FA`、侧栏 `#FAFBFC`、成功 `#12B76A`、警告 `#F79009`、危险 `#F04438`、代码块 `#171A21`。
- C 端：侧栏 280（<1024 收为 64 图标条）、顶栏 64、消息列 704 居中、输入卡最大 752；空态仅保留品牌问候与输入卡，不展示场景卡。
- B 端：侧栏 240、内容留白 32、卡片间距 16/20、右栏 312、表格行高 56（用户）/50（模型）。
- 认证页：左品牌区 620 + 右表单区，表单宽 380、字段高 46；<1024 隐藏品牌区。
- 桌面端保证 1280–1920 正常；手机端提供抽屉式导航与单列布局。

## 6. 启动、托管与验收

### 6.1 后端托管两个前端

两个前端构建完成后由后端统一输出 `index.html`：

- C 端构建产物：`ui/c/dist` → 内嵌到 `server/web/dist/c`，对外路径 `/`，静态资源 `/assets/**`。
- B 端构建产物：`ui/b/dist` → 内嵌到 `server/web/dist/b`，对外路径 `/admin/`，静态资源 `/admin/assets/**`。
- `server/apis/router.go` 的 `mountFrontend` 负责静态文件与 history 路由回退；`/api/**` 未命中仍返回 JSON 错误。
- 构建命令：`cd ui && pnpm build`，再执行 `cd server && make embed`（等价于 `make dist`）。
- 开发期仍可使用 Vite：C 端 `127.0.0.1:5173`，B 端 `127.0.0.1:5174`，`/api` 代理到 `9821`。

```bash
# 后端（生产形态，内嵌两个前端）
cd server && go run .            # http://localhost:9821 与 /admin/

# C 端（开发）
cd ui && pnpm dev:c              # http://localhost:5173

# B 端（开发）
cd ui && pnpm dev:b              # http://localhost:5174
```

初始管理员：`xjqxz_2@163.com` / `123456` / 昵称 Wind（首次登录后请修改密码）。
种子数据：仅创建 1 个管理员、DeepSeek 一家厂商与 `deepseek-flash` 模型；不写演示用户、会话与审计记录。
上游 Key 通过 `server/configs/secret.local.yaml` 或 `DEEPTALK_DEEPSEEK_KEY` 注入，AES-GCM 加密入库。

### 6.2 Docker 与搜索运行时

- `Dockerfile` 采用多阶段构建：Node 构建 C/B → Go 内嵌编译 → Alpine + glibc 运行层。
- 运行层内置 Obscura 0.2.7 与 `obscura-worker`，联网搜索无需外部浏览器；`web_search.obscura_bin` 默认 `obscura`。
- Obscura 官方 Linux 包依赖 glibc，因此使用 `frolvlad/alpine-glibc:alpine-3.21_glibc-2.41`（Alpine 3.21 + glibc 2.41），不是纯 musl。
- 数据目录 `/app/data` 声明为 volume；图片落在 `/app/data/uploads`，随数据卷持久化；镜像暴露 `9821`，健康检查 `GET /healthz`。
- `.github/workflows/docker.yml` 在 push main / `v*` tag 与 PR 时运行后端测试、前端 typecheck/lint/build，并构建推送 GHCR 镜像（PR 不推送）。
