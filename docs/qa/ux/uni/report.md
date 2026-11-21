# 小橘树洞 uni-app H5 客户端 · 逐交互点测试报告

- 测试时间：2026-11-21 13:20–14:05 UTC
- 被测地址：`http://127.0.0.1:5323`（H5 构建产物 + `/api` 代理到 `http://127.0.0.1:9861`）
- 视口：390×844，`isMobile`、`hasTouch`、DPR=2，locale=zh-CN
- 账号：`ux-user@deeptalk.local / Qa-pass-2026`；另注册新账号 `qa-reg-*@deeptalk.local` 验证注册/资料页
- 方式：Playwright（chromium headless shell）+ 每个交互点「操作前截图 → 真实点击/输入 → 操作后截图」，记录 DOM 状态、`pageerror`、`console.error/warning`、`/api` 响应状态

## 结论

| 指标 | 数值 |
| --- | --- |
| 交互点 | 54 |
| 通过 | 41 |
| 失败 | 13 |
| 截图 | 117 张（`NN-名称-before/after.png`） |
| pageerror | 0 |
| 非预期 console.error | 1 类（`TypeError: e.onRate is not a function`，赞/踩触发） |
| 非预期接口错误 | 0（2 个 400 为用例预期的校验失败：错误密码、非法邮箱） |

| 阶段 | 交互点 | 通过 |
| --- | --- | --- |
| A 登录页 | 6 | 5 |
| B 注册 | 3 | 3 |
| C 聊天页/抽屉 | 16 | 13 |
| D 输入区 | 9 | 4 |
| E 消息区 | 4 | 2 |
| F 资料页 | 8 | 8 |
| G 小程序预览页 | 4 | 4 |
| H 深色模式 | 4 | 2 |

## 失败清单

| # | 区域 | 元素/动作 | 现象 | 期望 | 截图 |
| --- | --- | --- | --- | --- | --- |
| 5 | 登录页 | 「忘记密码？」 | 点击后无任何变化：无跳转、无弹窗、无 toast；该元素是没有点击处理的 `span` | 有可见反馈（跳转/弹窗/提示重置方式） | `10-login-forgot-after.png` |
| 13 | 抽屉 | 搜索框输入「冒泡」 | 输入值正确，但会话列表不按关键词过滤（11 → 12 条，随会话总量变化）；H5 侧 `@search` 没有把关键词回写到 `keyword`，`onSearch` 始终按空串查询 | 列表按关键词过滤 | `26-chat-drawer-search-after.png` |
| 14 | 抽屉 | 搜索「zzzz」 | 仍显示全部会话，不出现「没有匹配的对话」 | 显示空状态 | `28-chat-drawer-search-nomatch-after.png` |
| 20 | 抽屉 | 更多 → 删除 | 二次确认框只剩「取消 / 确认」两个按钮，标题与「确定删除「标题」？」正文全空；`confirm({title,content})` 与 `confirm(title, message)` 签名不匹配，对象被当成 title 传给 Vant Dialog | 确认框显示「删除对话」及被删会话标题 | `41-chat-delete-confirm-after.png` |
| 27 | 输入区 | 多行输入（Shift+Enter） | 1 行高度 52px，3 行后仍是 52px，`scrollHeight=78 / clientHeight=52`，内容在框内滚动 | 输入框随行数自动增高（上限 192px） | `55-composer-multiline-after.png` |
| 30 | 输入区 | 生成中回车插话 | 输入框被清空，但既无「插话」标签、无「排队中」提示、也无 `interject` WS 帧；`store/chat.js` 用 `chat.conversation?.id` 判断，而 store 里只有 `chat.current`，条件恒假直接 return，插话文本被静默丢弃；另外生成中发送按钮被「停止生成」替换，触屏没有插话入口（仅键盘 Enter 可触发） | 出现「插话」标记 + 「排队中，本轮结束后处理」 | `63-composer-interject-after.png` |
| 31 | 输入区 | 插话结果 | 本轮结束后栈内检索不到插话文本（0 次），没有第二轮回答，`interject` 帧 0 条 | 插话被处理并产生第二轮回答 | `66-composer-interject-result-after.png` |
| 32 | 输入区 | 停止生成（发送后 0.6s） | `stop` 帧已发出、发送按钮恢复，但消息区没有「已停止生成」；过早停止时流式占位消息还没有内容，而状态文案放在 `.message-actions` 里，无内容时整块不渲染，用户看不到任何停止反馈 | 停止后在消息上显示「已停止生成」 | `69-composer-stop-after.png` |
| 34 | 输入区 | 点击模型名 | 无任何弹层（`van-popup` 元素可见高度 0、`van-picker` 数量 0），模型名不变；`main.js` 只 `import 'vant/lib/index.css'`，没有 `app.use(Vant)` 或注册 `van-popup/van-picker`，聊天页模板里的 Vant 组件退化成未注册自定义标签 | 底部弹出模型 Picker 并列出模型，确认后输入区模型名变化 | `73-composer-model-picker-open.png` / `74-composer-model-picker-after.png` |
| 37 | 消息区 | 「赞」 | 点击后按钮只出现一层聚焦底色，未进入已点赞高亮状态；控制台抛 `TypeError: e.onRate is not a function`（编译产物 `pages-chat-chat.*.js`），`onRate` 回调未正确绑定 | 按钮高亮并调用反馈接口 | `81-message-like-after.png` |
| 38 | 消息区 | 「踩」 | 与「赞」相同：只出现聚焦底色、不进入 active，抛同样的 `TypeError` | 按钮高亮并调用反馈接口 | `83-message-dislike-after.png` |
| 51 | 深色模式 | 登录页 | 面板是硬编码浅色渐变 `#f7f9ff → #fff`（`@media (max-width:1024px) .auth-panel`，在文件里晚于深色规则，覆盖了 `var(--dt-bg)`），文字却是深色模式的浅色 `#e6e9ef`，对比度 ≈ 1.16:1，标题几乎不可见 | 深色背景 + 可读文字（≥4.5:1） | `111-dark-login-after.png` |
| 52 | 深色模式 | 聊天页 | 顶栏、用户气泡、输入区变深色，但消息列表背景仍为纯白，助手正文/助手名用浅色 `#e6e9ef`，对比度 1.22:1，回答不可读 | 消息区深色背景、文字可读 | `113-dark-chat-after.png` |

## 次要问题（步骤判定为通过，但建议修复）

| 位置 | 现象 | 截图 |
| --- | --- | --- |
| 重命名弹窗 | 长标题的 `placeholder` 溢出输入框边界，文字压在边框外 | `34-chat-rename-modal-after.png` |
| 分享链接 | 链接复制成功，但页面自定义 toast「分享链接已复制」没有出现，只显示 `uni.setClipboardData` 自带的「内容已复制」 | `38-chat-share-toast-after.png` |
| 第 3 张轮次截图 | 流式生成时列表不自动跟随，最新回答/操作行常被留在可视区下方 | `63-composer-interject-after.png` |
| 小程序预览页 | 「微信一键登录」卡片本身正常，但页面下半部分 `.messages`（`flex:1;height:0`）在 H5 下没有收缩高度，输入区浮在消息内容之上：代码块底部 749px > 输入区顶部 684px，代码块与消息操作行被遮挡 | `100-mp-wechat-card-before.png` |
| 深色模式抽屉 | 用量卡片仍是白底（文字可读，但与深色主题不一致） | `115-dark-drawer-after.png` |
| 深色模式登录页 | 「登录即表示同意…」上方出现一条深色分隔线，与浅色面板冲突 | `111-dark-login-after.png` |

## 通过的交互点（摘要）

- 登录：邮箱/密码输入与聚焦、密码掩码（无眼睛图标）、错误密码提示「邮箱或密码不正确」、改正后登录进入聊天页
- 注册：立即注册切换、填邮箱密码、注册后自动登录并显示空会话
- 聊天页：顶栏菜单/标题/头像、抽屉开合、新建对话、会话点击切换、会话「⋯」菜单、重命名（含输入与确认）、分享（剪贴板写入成功）、删除确认与取消、用量卡片（免费版 / 本月 ¥0.00 / ¥1.00 / 5 小时 34/34/34（随用量变化））、账号卡片与头像 ActionSheet 的「退出登录 → 取消」
- 输入区：输入、发送、流式回答（「这是一段模拟回答：…」）、复制、重新生成、附件按钮取消不报错
- 消息区：复制（按钮变「已复制」、剪贴板内容正确）、重新生成、思维过程展开/收起、代码块复制（内容正确）
- 资料页：头像选择取消、昵称输入、稍后再说、保存并进入、密码过短前端提示、非法邮箱服务端提示「邮箱格式不正确」、合法邮箱提交成功并提示「已可在电脑端登录」
- 小程序预览页：微信一键登录卡片（logo/标题/副标题/绿色按钮 `rgb(7,193,96)`/协议文案）渲染正常
- 深色模式：聊天页用户气泡与输入区、抽屉、资料页

## 控制台与接口

- `pageerror`：0
- `console.error`：仅 1 类非预期 —— `TypeError: e.onRate is not a function`（赞/踩各触发一次）；另有 2 条 `Failed to load resource: 400`，对应上面两个预期校验失败用例
- `console.warning`：`文件选择器对话框只能在由用户激活时显示`（附件按钮调用 `uni.chooseImage`，无功能影响）
- `/api` ≥400：`POST /api/v1/c/auth/login` 400（错误密码，预期）、`POST /api/v1/c/auth/profile/complete` 400（非法邮箱，预期）
- 其余接口（`auth/me`、`models`、`conversations`、`usage`、`profile`、`feedback` 等）均 2xx

## 备注

1. 测试过程中 `5323` 的 H5 静态预览服务自行退出，我用仓库自带脚本重新拉起（未改代码）：`cd ui/uniclient && PORT=5323 HOST=0.0.0.0 PROXY_TARGET=http://127.0.0.1:9861 node scripts/serve-h5.mjs`。
2. 账号 `ux-user@deeptalk.local` 为共享测试账号，测试期间有其他会话写入同一后端（会话数量与部分消息列表包含测试期外的内容）；本文结论均以当次操作前后的差值和 DOM/网络证据为准。
3. 产物：`manifest.json`（54 条：step/action/target/expected/observed/ok/screenshot + phase/screenshot_before）、`errors.json`（原始控制台/接口记录）、117 张截图。
