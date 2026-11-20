# DeepTalk 小程序 / APP 客户端（uni-app）

面向**微信小程序**，并预留 **APP（App-Plus）** 能力。技术栈：uni-app（Vue 3 + Vite）+ Vant，
页面与组件**优先使用 `.nvue`**，以便后续直接复用到 APP 的原生渲染层。

## 为什么用 WebSocket

小程序没有 `EventSource`，也无法消费分块响应，所以流式回答统一走 WebSocket：

```text
客户端 → 服务端：{"type":"stream"|"stop"|"ping", ...}
服务端 → 客户端：{"type":"<事件名>","payload":<与 SSE 一致的数据>}
```

事件名与 Web 端 SSE 完全一致：`meta` / `reasoning` / `delta` / `activity` / `sources` /
`notice` / `usage` / `error` / `done`，因此两端可以共用同一套后端逻辑。

鉴权：小程序不方便保存 Cookie，登录接口会额外返回 `token`，之后所有请求带
`Authorization: Bearer <token>`；WebSocket 用 `?token=` 连接（小程序建连无法自定义头时也能用）。

## 目录结构

```text
src/
├── api/            # 接口层（uni.request 封装、附件上传）
├── components/     # nvue 组件：消息、思维链、输入框
├── config/         # 接口与 WebSocket 地址
├── pages/
│   ├── chat/       # 聊天主页面（消息流、历史抽屉、分享）
│   └── login/      # 登录 / 注册
├── store/          # 登录态与聊天状态（reactive，无需 pinia）
├── utils/          # WebSocket 客户端、Markdown 解析、跨端 UI
└── static/         # 图标
```

## 本地开发（H5 调试）

小程序真机调试成本高，日常用 H5 编译来调页面：

```bash
cd ui/uniclient
npm install                 # 首次
npm run dev:h5              # http://localhost:5273
```

接口默认代理到 `https://deeptalk.peon.top`，可用环境变量改成本地后端：

```bash
VITE_PROXY_TARGET=http://127.0.0.1:9821 npm run dev:h5
```

## 构建

```bash
npm run build:h5          # H5 产物，用于冒烟测试
npm run build:mp-weixin   # 微信小程序，产物在 dist/build/mp-weixin
npm run build:app         # App 资源，交给 HBuilderX 云打包
```

微信小程序需要在 `src/manifest.json` 填入 `mp-weixin.appid`，并在小程序后台把
`https://deeptalk.peon.top` 加入 request / socket 合法域名。App 端同理配置 `appid`。

## 冒烟测试

```bash
npm run dev:h5
cd ui && node uniclient/tests/uniclient-smoke.mjs
```

用例覆盖：登录页渲染 → 登录跳转 → 历史消息（Markdown 列表/代码/思维链）渲染 →
发送消息（WebSocket 帧）→ 历史会话抽屉。

## 设计取舍

- **nvue 优先**：`.nvue` 在 H5 / 小程序端会被编译为普通 vue 页面，因此一套代码可以三端复用；
  nvue 只支持 flex 布局与类选择器，所以组件内没有使用后代选择器。
- **正文不用 v-html**：nvue 没有 DOM，`utils/markdown.js` 把 Markdown 解析成块与行内 span，
  再用 `<text>` 嵌套渲染，支持标题、列表、引用、代码块、加粗、行内代码与 `[1]` 引用标号。
- **Vant 只在 H5 生效**：Vant 组件依赖 DOM，无法在 nvue / 小程序运行。`utils/ui.js` 做了跨端封装：
  H5 用 Vant 的 Toast / Dialog，模型选择用 `van-popup + van-picker`；小程序与 APP 退回 uni 原生
  `showToast` / `showModal` / `showActionSheet`，行为一致但不需要额外组件库。
- **A2UI 暂不渲染**：小程序端没有 HTML 容器，收到 `a2ui` 事件时先忽略，正文照常展示。

## 已知限制

- 分享只复制 PWA 分享链接，小程序内不支持直接打开；
- 历史消息一次性加载，暂未做分页；
- 图片附件支持上传与展示，暂未做放大预览与多图排序。
