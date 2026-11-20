# 前端端到端冒烟测试

`smoke.mjs` 用 Playwright 覆盖 C 端与 B 端的核心链路：

1. C 端登录、空态、新建对话、真实 SSE 回答、删除对话；
2. B 端登录、DeepSeek 厂商卡片与配置弹窗、管理员详情与额度弹窗、注册开关的关闭与恢复。

## 运行

```bash
# 先启动后端（同时托管前端）
cd server && go run .

# 再运行测试
cd ui && node tests/smoke.mjs
```

可通过 `BASE_URL` 指定地址，通过 `PW_CHROMIUM` 指定浏览器可执行文件：

```bash
BASE_URL=http://127.0.0.1:9821 PW_CHROMIUM=/path/to/chrome-headless-shell node tests/smoke.mjs
```

默认使用 Playwright 缓存的 `chromium_headless_shell-1248`。脚本会自动把仓库内
`.browser-deps` 的 `LD_LIBRARY_PATH` 与 `FONTCONFIG_FILE` 注入子进程，以便在缺少
系统字体 / 依赖的 Linux 环境里运行；正常的开发机无需这部分配置。

## 其余检查脚本

这些脚本用 Playwright 拦截接口，不需要后端与真实模型：

| 脚本 | 覆盖内容 |
| --- | --- |
| `pwa-check.mjs` | manifest、图标、Service Worker、离线页、iOS 安装说明 |
| `citations-check.mjs` | 正文 `[1]` 引用上标、来源卡片已移除、思考文案 |
| `parallel-check.mjs` | 生成中可新建/切换会话、侧栏生成徽标、检索活动提示 |
| `a2ui-check.mjs` | A2UI 消息经官方 Lit Renderer 渲染成组件 |

`pwa-check` / `a2ui-check` / `citations-check` 默认访问 `pnpm --filter @deeptalk/c preview` 的
`http://127.0.0.1:4173`；`parallel-check` 还需要先启动慢速 SSE 服务：

```bash
cd ui
pnpm --filter @deeptalk/c preview --port 4173 &
node tests/sse-mock-server.mjs &
node tests/parallel-check.mjs
```

## 约定

- 测试会创建并删除一段对话，不会修改种子账号的密码或额度。
- 注册开关测试结束后会恢复为“开放注册”。
- 若需要在 CI 中跑，建议另起独立数据库（`DEEPTALK_DB=/tmp/deeptalk-ci.db`）并 seed。
