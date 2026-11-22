# 小橘 小程序 / APP 客户端（uni-app）

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

## 预览构建产物（给手机 / 同事看效果）

```bash
VITE_API_BASE='' npm run build:h5   # 走同源代理，不写死域名
npm run preview:h5                  # 0.0.0.0:5274，静态文件 + /api + WebSocket 代理
```

`scripts/serve-h5.mjs` 会把 `/api` 与 WebSocket 升级一起代理到 `PROXY_TARGET`
（默认生产环境），所以预览页可以直接用线上账号登录、真正流式出字：

```bash
PORT=5275 PROXY_TARGET=http://127.0.0.1:9821 npm run preview:h5   # 指向本地后端
```

## 微信小程序

```bash
npm run build:mp-weixin     # 产物：dist/build/mp-weixin
```

产物约 300KB（zip 80KB），已确认不含 antd / markdown-it / highlight.js / DOMPurify / Vant 这些 H5 专用依赖。

导入微信开发者工具的步骤：

1. 打开「微信开发者工具 → 导入项目」，目录选 `ui/uniclient/dist/build/mp-weixin`
2. `project.config.json` 里默认是 `touristappid`（游客模式，能看界面不能真机调试）；正式调试要填自己的 AppID
   （`src/manifest.json` 的 `mp-weixin.appid`，重新构建即可）
3. 在公众平台「开发管理 → 开发设置 → 服务器域名」里把下面两个域名加进去：
   - request 合法域名：`https://deeptalk.peon.top`
   - socket 合法域名：`wss://deeptalk.peon.top`
4. 开发阶段可以先在开发者工具「详情 → 本地设置」勾选「不校验合法域名」直接跑

小程序端与 H5 共用同一套业务代码，差异只在：

- 页面/组件走 `.nvue` 分支（uni 组件 + rpx 样式），不加载 PWA 的 DOM 样式与图标库
- 流式回答走 `uni.connectSocket`（H5 走浏览器原生 WebSocket），协议与事件名完全一致
- 登录态存本地存储，请求带 `Authorization: Bearer`，WebSocket 用 `?token=` 连接

## A2UI

A2UI 是 agent 产出声明式 UI 的协议。官方渲染器 `@a2ui/lit` 依赖 Web Components 与 DOM，
**微信小程序跑不了**；但 A2UI 消息本身是纯 JSON（`createSurface` / `updateComponents` /
`updateDataModel`），所以在小程序端自己实现了渲染层：

- `src/utils/a2ui.js`：把消息折叠成「区块 → 条目」两层结构，解析 `{path}` / `{literal}` 取值
- `src/components/a2ui/renderer.nvue`：用 `view / text / button / image` 渲染，三端通用
- 支持的基础 catalog 组件：`Text`（含 h1–h4 / caption 变体）、`Column`、`Row`、`List`、
  `Card`、`Divider`、`Image`、`Button`；其它类型降级成一行提示，不会白屏
- 线上真实用到的只有 `Column / Card / Text / Row / List / Divider / Button` 七种，已全部覆盖

已知限制：A2UI 里的 `Button` 点击不会回传动作（PWA 端由 Lit 渲染器派发事件），
小程序端点击只给出提示；如果需要，可以接成「把动作名作为用户消息发回模型」。

## 检查小程序端的排版

小程序端的样式（`.nvue` 分支）可以用同一个 H5 构建来预览：

```text
http://<host>:5274/#/pages/mp-preview/mp-preview
```

这个页面直接引用 `components/message-item.nvue` 与 `components/composer.nvue`，用固定的示例消息渲染，
不依赖登录与后端。改 `.nvue` 样式后刷新即可看到效果，比反复走微信开发者工具快。
手机上布局的关键点：`.page` 必须是 `flex-direction: column` + `height: 100%`，
`scroll-view` 必须是 `flex: 1; height: 0`（微信里只写 `flex: 1` 不会收缩，会把输入框顶出屏幕），
并打开 `enable-flex`。

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

## 设计与还原对照

- 设计图（四屏线框 + 落地要点）：[`docs/design/07-uniclient-mobile.svg`](../../docs/design/07-uniclient-mobile.svg)
- 逐屏截图对照（上排 PWA、下排小程序端，390×844）：[`docs/design/uniclient-mobile-compare.svg`](../../docs/design/uniclient-mobile-compare.svg)
- 截图脚本：`ui/tests/mobile-shots.mjs`（PWA）、`ui/uniclient/tests/shots.mjs`（小程序端）

新增页面时先用这两个脚本各截一次，对比尺寸、字号、间距后再改样式，避免凭感觉调整。

## 与 PWA 的视觉一致性

H5 端直接用 PWA 的设计体系，不另起一套：

- `src/styles/pwa.css` 由 `ui/c` 的组件样式脚本抽取生成（消息、思维链、输入框、顶栏、侧栏、用量卡、登录页）
- `src/main.js` 在 H5 端引入 `ui/shared/src/theme.css`，字体、颜色、圆角、阴影全部同一套令牌
- 正文用 `src/utils/pwa-markdown.js`（与 `ui/shared/src/markdown.ts` 同步）：markdown-it + highlight.js + DOMPurify，
  代码块头、表格、引用、`[n]` 上标与 PWA 完全一致
- 图标用 `@ant-design/icons-vue`，与 PWA 是同一套
- `src/components/h5/*` 是 H5 专用组件，结构与 class 名对齐 PWA 的 MessageItem / ThinkingPanel / Composer /
  ChatTopbar / ChatSidebar；`.nvue` 组件继续服务小程序与 APP

注意：uni-app 会把模板里的 `<button>` / `<input>` / `<textarea>` 编译成 `uni-button` / `uni-input` 等自定义元素
并注入自己的样式，也会把 CSS 里的 `button.primary` 改写成 `uni-button.primary`。所以 H5 组件用渲染函数创建
原生控件（`h5/btn.vue`、`h5/native-input.vue`、`h5/textarea.vue`），样式一律只用类选择器。

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
