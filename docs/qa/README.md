# DeepTalk QA 证据

当前版本已移除空态场景卡，并接入 Obscura 联网搜索。截图基于 **1440×900 / Asia/Shanghai**。

## C 端

| 页面 | 截图 |
| --- | --- |
| 登录 | [final-c-login.png](final-c-login.png) |
| 空态（无场景卡） | [final-c-chat-empty.png](final-c-chat-empty.png) |
| 联网搜索中 | [final-c-searching.png](final-c-searching.png) |
| 联网搜索来源（思考链下方） | [final-c-search.png](final-c-search.png) |
| 图片消息与视觉回答 | [final-c-image.png](final-c-image.png) |
| 纯图片消息 | [final-c-image-only.png](final-c-image-only.png) |
| 纯问候不联网、不提示 | [final-c-greeting.png](final-c-greeting.png) |
| 无限版用量卡 | [final-c-unlimited.png](final-c-unlimited.png) |
| 粘贴图片预览 | [final-c-paste.png](final-c-paste.png) |
| 移动端空态（414×896） | [final-c-mobile.png](final-c-mobile.png) |
| 移动端抽屉 | [final-c-mobile-sidebar.png](final-c-mobile-sidebar.png) |

## B 端

| 页面 | 截图 |
| --- | --- |
| 管理员登录 | [final-b-login.png](final-b-login.png) |
| 概览 | [final-b-overview.png](final-b-overview.png) |
| 厂商与模型 | [final-b-providers.png](final-b-providers.png) |
| 用户管理 | [final-b-users.png](final-b-users.png) |
| 无限版用户详情 | [final-b-unlimited.png](final-b-unlimited.png) |
| 非管理员拦截 | [final-b-guard.png](final-b-guard.png) |
| 系统设置 | [final-b-settings.png](final-b-settings.png) |

## 验证说明

- `ui/tests/smoke.mjs`：18 项断言通过（登录、空态无场景卡、联网搜索入口、SSE 回答、删除会话、B 端登录/厂商/用户/设置）。
- 图片链路已实测：上传 → 预览 → 发送 → 视觉回答（模型识别蓝色）→ 刷新后缩略图仍在；剪贴板粘贴图片也能生成预览。
- 联网搜索开关常驻，不再发送后自动关闭；服务端对纯问候/短句跳过检索。
- 思维链流式期间自动展开；大块上游 delta 会按 28 字符切块，保证前端持续看到逐字输出。
- 并发上限已实测：厂商与模型均可设到 10000；厂商并发修改后模型表同步。
- 无限版已实测：套餐切换后 quota=0，C 端用量卡显示“不限”，B 端显示无限版且不可单独调额度。
- 非管理员访问 `/admin/overview` 会被前端守卫重定向到登录页并显示“不是平台管理员”；`/b/**` 接口返回 403。
- C 端顶栏已去掉保存状态、蓝色新建图标与模型选择；模型选择移动到输入区右下角，原“上下文 32K”文案移除；移动端思考/联网按钮只保留图标以避免挤压。
- 联网搜索：通过 Obscura MCP 浏览器抓取搜索页，SSE 依次返回 `search`、`sources`，来源写入 `messages.sources_json`；回答里的 `[1]`、`[2]` 上标可点击打开原网页，不再展示单独来源面板。
- 图片附件：选择文件或粘贴图片后先上传预览，发送时通过 `attachment_ids` 关联；消息内显示缩略图，视觉模型直接读取图片内容。单条最多 4 张、单张 8MB。
- 当前种子：1 个管理员（Wind）、1 家 DeepSeek 厂商、1 个 `deepseek-flash` 模型；不再写入演示用户/会话/审计。
- Docker 镜像与 GitHub Action 见仓库根目录 `Dockerfile`、`.github/workflows/docker.yml`；本机未安装 Docker，镜像构建需在 CI 或 Docker 环境验证。
