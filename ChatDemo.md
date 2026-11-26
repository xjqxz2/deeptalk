# 边想边做、边想边查：一份可直接落地的实现说明

> 来源：对 `deepseek-ai/chat-demo` v0.1.3 源码的逐文件分析，只抽取与「思考↔工具交替」相关的部分。
> 目标：让模型在**一次回答过程中**交替输出「思考 → 调用工具 → 看到结果 → 继续思考 → 再调用/写正文」，前端按顺序实时呈现，做到边想边做、边想边查。
> 前提：你已有自己的联网/检索工具，希望模型**优先**用它。

---

## 0. 一句话结论

这个效果**不是前端做出来的，是协议做出来的**。前端只做两件事：

1. 事件流里**按到达顺序追加块**（thinking / 工具调用 / 工具结果 / 正文），不许重排、不许合并、不许等齐了再渲染；
2. 对「工具块」按**原子块**处理（一次给全，不流式拼），对「正文/思考」按增量 delta 拼接。

剩下的一切——什么时候搜、搜几次、边想边搜还是先想完再搜——都是编排层（服务端）决定的。所以你要做的核心决策只有一个：

> **把「思考↔工具」的编排放在服务端（一次请求、一条流），还是放在客户端（多次请求、前端缝合）。**

**推荐服务端编排。** 理由见 §1.2，一句话：客户端编排会让「一条正在生成中的回答」变成「若干条回答的拼接」，续写、引用、停止、错误恢复全部要重做一遍。

---

## 1. 效果的本质

### 1.1 你要复现的观感

用户在界面上看到的是一条**纵向时间线**，内容块严格按模型产出顺序排列：

```
[思考] 用户问的是「XX 最新政策」，需要实时信息，先搜一下……
   ↓  （思考块结束，高度收起或保持展开）
[搜索] queries: ["XX 最新政策", "XX 政策 2026"]     ← 工具调用块，原子出现
   ↓  （并发展开为 2 个来源卡片）
[结果] 3 条来源：gov.cn / 某新闻 / 某论坛          ← 工具结果块，原子出现
   ↓
[思考] 第一条是官方原文，但没写生效时间；第二条提到时间，需要打开核对……
   ↓
[读页] open_page: https://gov.cn/...                ← 又一次工具调用
   ↓
[结果] 页面正文（加密签名形式回传）
   ↓
[思考] 时间对上了，可以组织答案。
   ↓
[正文] 根据……（含行内引用角标①②）
```

关键观感细节（chat-demo 的做法）：

- **思考块和工具块是"交错"的，不是"集中在前"**。第二个思考块出现在搜索结果**之后**——这就是"边想边查"与"先想完再一次性搜"的区分点。
- 思考块有**耗时**（`startedAt` / `endedAt`，展示时 `Math.max(1, round((ended-started)/1000))` 秒），工具块有**状态**（进行中/成功/失败），正文块有**引用角标**。
- 工具块**原子出现**：不是在输入参数时逐字符"打字机"，而是"决定要搜"的瞬间整块出现，然后原地进入 pending。这样才有"它想清楚了，动手了"的观感；参数逐字流动反而显得笨拙。

### 1.2 编排位置：两种做法，选一个

| | 服务端编排（chat-demo 的做法） | 客户端编排（你 自己 function calling 的常见做法） |
|---|---|---|
| 一次用户提问 | **1 个 HTTP 请求 / 1 条 SSE 流** | N 个请求（每次工具往返 +1） |
| 前端状态 | 一个 `streaming` 回复，块数组只增不改 | 要么把 N 条拼接成 1 条，要么暴露成 N 条 |
| 思考↔工具交替 | 天然连续，服务端想交替几次都行 | 每次往返都要重新注入历史，容易断节奏 |
| 思考签名 | 不用管，服务端持有 | **必须原样回传 thinking + signature**，否则报错（见 §7.3） |
| 停止生成 | 一次 abort | 要 abort 当前请求 + 阻止调度下一个 |
| 工具密钥 | 留在服务端，不下发浏览器 | 要么前端持有（不安全），要么前端调你的代理（多一跳） |
| 复用你的既有工具 | 服务端直接调，延迟最低 | 前端调，多一跳，但**工具实现可复用** |

**结论**：如果你的工具跑在自己的后端（Go/Python 服务），就走服务端编排——它和你现在的架构天然契合，且能保证"一条流、一个回复、可随时停止"。客户端编排只在"工具必须跑在浏览器里"时才值得。

下文 §2–§6 对两种做法都适用（协议与观感是共通的），§7 专门讲客户端编排。

---

## 2. 块模型：照这张表实现就够了

一条 assistant 回复 = **有序块数组**，每个块有 `type`。五种块，其中三种是你必须自己造的（思考、工具调用、工具结果）。

### 2.1 thinking（思考块）

```ts
{ type: 'thinking', thinking: string, signature?: string }
```

- 内容增量下发（`thinking_delta`），展示时可直接渲染成普通 Markdown（chat-demo 甚至给它单独一套 `thinking.markdown.css`，字号更小、颜色更淡）。
- `signature` 在块结束前由 `signature_delta` 下发，值等于本次响应的消息 id。**服务端编排时你完全可以忽略它的内容，只做"原样保存、必要时原样回传"**。
- 建议额外记 `startedAt`（`content_block_start` 时刻）和 `endedAt`（`content_block_stop` 时刻），用于"思考了 N 秒"。

### 2.2 工具调用块（server_tool_use / tool_use）

```ts
{
  type: 'server_tool_use',
  id: 'srvtoolu_s_3_<32位hex>',        // 前缀区分能力：s|ds|rl|is|dis
  name: 'deepseek_web_search' | 'deepseek_read_url' | '<你的工具名>',
  input: { action: 'search', queries: string[] }
        | { action: 'open_page', url: string }
        | { action: 'find_in_page', url: string, pattern: string }
        | { action: 'read_link', url: string },
  caller: { type: 'direct' },
  kind: 'search' | 'deep_search' | 'read_link' | 'image_search' | 'deep_image_search'
}
```

设计要点，逐条抄：

1. **`id` 里编码能力类型**。`s`=普通搜索、`ds`=深度/边想边搜、`rl`=读链接、`is`=图搜、`dis`=深度图搜。好处是：日志里一眼看出这次走了哪条路径，前端也能据此换图标/文案，不用额外字段。
2. **`kind` 与 `name` 分开**。`name` 是"工具名"（决定调用哪个后端能力），`kind` 是"这次调用属于哪种语义"（决定怎么展示）。同一个 `deepseek_web_search` 可以因 `kind` 不同而展示成"搜索"或"边想边搜"。
3. **`action` 是判别式**，不是一个带一堆可空字段的扁平对象。四种动作语义清晰：`search`（发起检索，支持多 query）、`open_page`（打开页面）、`find_in_page`（页内定位）、`read_link`（读用户给的那个链接）。**这四种覆盖了 95% 的检索类 Agent 动作，建议你直接照抄这个枚举。**
4. **`input` 一次给全**（原子）。协议层面它在 `content_block_start` 就携带完整 input，之后没有 `input_json_delta`——这是刻意的：**工具调用是"决策产物"，不是"流式文本"**。
5. `queries` 是**数组**，不是单个 `query`。这是"边想边查"能高效的关键：模型一次决定 2–4 个互补检索式并发查，而不是串行问三遍。前端的观感也更好——一次出现 3 个 chip，然后同时变成 3 张来源卡。

### 2.3 工具结果块（web_search_tool_result / web_fetch_tool_result）

搜索结果：

```ts
{
  type: 'web_search_tool_result',
  tool_use_id: '<对应调用块 id>',
  kind: 'search' | 'deep_search' | 'image_search' | 'deep_image_search',
  content: (WebSearchResult | { type: 'web_search_tool_result_error', error_code: 'unavailable' })[]
}

type WebSearchResult = {
  type: 'web_search_result',
  title: string,
  url: string,
  image_url?: string,        // 图搜召回的原图，可缺省
  snippet: string,           // 可直接展示的摘要
  site_icon: string | null,  // 取不到注册域名时为 null
  site_name: string | null,
  encrypted_content: string, // AES-256-GCM 加密的全文，客户端解不开、也不该解
  page_age: string | null    // RFC3339，无发布时间为 null
}
```

设计要点：

1. **`content` 是"结果数组 或 单元素错误数组"**，不是 `results[] + error?` 两个字段。这样"成功但零结果"（`[]`）和"失败"（`[{error_code:'unavailable'}]`）在类型上就分得开，前端一个 `map` 就能同时处理两种情况。
2. **`snippet` 与 `encrypted_content` 分离**：前端只拿得到可展示的摘要，全文以密文形式原样回传（服务端下一轮解密用）。**这条很值得抄**——它解决了"既要把检索结果喂回模型，又不能把全文泄露到浏览器"的矛盾。
3. **`site_icon` / `site_name` 允许 null**，前端必须能优雅退化成"显示域名首字母"（chat-demo 有 `SiteLogo` + `FallbackImage` 就是干这个的）。
4. **`page_age` 允许 null**，且是 RFC3339 UTC。展示时不要假设有值。

读页结果：

```ts
{
  type: 'web_fetch_tool_result',
  tool_use_id: string,
  kind: 'read_link' | 'deep_search',
  content: {
    type: 'web_fetch_result',
    url: string,
    content: { type: 'document', source: { type: 'text', media_type: 'text/plain', data: string }, title: string | null },
    published_at: string | null      // read_link 是页面解析出的原始日期串；deep_search 是 RFC3339
  } | {
    type: 'web_fetch_tool_result_error',
    error_code: 'url_not_accessible' | 'url_not_allowed' | 'unavailable'
  },
  signature?: string                 // 仅 read_link 成功时返回，加密页面，必须回传
}
```

**`signature` 是整个协议里最容易踩坑的地方**：它是被加密的页面正文，客户端无法解构，必须在后续请求里**原样带回**，服务端解密后才能把网页拼回那一轮提问的末尾。chat-demo 的规则很硬：**签名缺失/为空的 read_link，调用块和结果块一起丢弃，一个都不回传**（否则服务端会拿到"有调用无结果"的残缺历史）。

### 2.4 text（正文块）+ 引用

```ts
{ type: 'text', text: string, citations?: TextCitation[] }

type TextCitation = {
  type: 'web_search_result_location',
  url: string,
  title: string | null,
  cited_text: string,        // 该文本块截至此引用处的全部正文（去首尾空白）
  encrypted_index: string    // 恒为空串，占位用
}
```

两条硬规则（照抄）：

1. **"挂过引用的文本块之后，后续正文另起一个新块。"** 这样引用永远锚定在块尾，前端不用做"字符偏移 → 角标位置"的映射，直接在块尾渲染角标即可。**这是省掉一大坨前端代码的关键设计。**
2. **同一块内引用按 url 去重**。否则模型一句话里连引同一个来源 3 次，角标就会变成 ①②③ 挂在同一处。

`cited_text` 是"截至引用处的全部正文"，前端可用它做悬浮卡片的高亮定位（chat-demo 的 `HoverCard` 就吃这个）。如果你不想做这么细，只保留 `url` + `title` 也能跑。

### 2.5 未知块：一律丢弃，不报错

`parseEvent.ts` 的写法值得抄：**只校验消费所需字段，字段不对就返回 `null` 丢弃该事件，不抛异常**。好处是服务端加新块类型时，老客户端不会崩，只是看不见。

例外只有一处：`content_block_start` 收到客户端 `tool_use` 块时，除了丢弃，还要把该回复标记成 `prefix: false`（不可续写），因为"未完成的工具往返不能作为续写前缀"（见 §8）。

---

## 3. 流式事件契约

一条 SSE 流，`event:` 名与 JSON 的 `type` 相同，**以 `message_stop` 结束，不发 `[DONE]`**。

| 事件 | 携带 | 用途 |
|---|---|---|
| `content_block_start` | `index` + **完整初始块** | 块出现。**工具块在这一刻就带全部 input** |
| `content_block_delta` | `index` + delta | 增量：`text_delta` / `thinking_delta` / `signature_delta` / `citations_delta` / `input_json_delta` |
| `content_block_stop` | `index` | 块结束。思考块在这里记 `endedAt` |
| `message_start` | `message.id`、`model`、`usage` | 开流。`usage` 里 output 为 0 |
| `message_delta` | `delta.stop_reason`、`stop_details?`、`title?`、`usage` | 收尾：结束原因、拒绝详情、标题、累计用量 |
| `message_stop` | — | 流正常结束 |
| `ping` | — | 保活，丢弃 |

### 3.1 两个"原子 vs 增量"的边界

- **原子块**：`server_tool_use`、`web_search_tool_result`、`web_fetch_tool_result`。在 `content_block_start` 一次给全，之后只有 `content_block_stop`。
- **增量块**：`text`、`thinking`。靠 delta 拼接。

这个边界的意义：**"做"发生在瞬间，"说"发生在过程**。工具调用是决策，决策不该有打字机效果；正文是表达，表达才需要逐字浮现。你的 UI 别把两者混成一种渲染策略。

### 3.2 索引映射：千万别直接用 `index` 当数组下标

服务端会跳过某些块（比如客户端 `tool_use`、未知类型），跳过的不占位。chat-demo 用一个 `Map<服务端 index, 本地数组下标>`：

```ts
case 'content_block_start':
  blockIndexes.delete(event.index)
  const block = toReplyBlock(event.content_block, now)
  if (!block) return maybeMarkPrefixFalse(next, event)
  blockIndexes.set(event.index, next.blocks.length)
  return { ...next, blocks: [...next.blocks, block] }

case 'content_block_delta':
  const local = blockIndexes.get(event.index)
  if (local === undefined) return next      // 被丢弃的块，增量直接忽略
  ...
```

**不这么写，一旦你丢弃一个块，后续所有 delta 都会写到错位的地方。**

### 3.3 渲染节流：按帧合批，不要按 token 重渲染

chat-demo 的做法（直接抄）：

```ts
reply = applyEvent(reply, event, Date.now(), blockIndexes)   // 纯函数，产出新对象
if (document.visibilityState === 'hidden') flush()           // 后台标签页立即刷
else frame ||= requestAnimationFrame(flush)                  // 前台每帧最多一次
```

`applyEvent` 必须是**纯函数**（输入旧回复 + 事件，输出新回复），这样 rAF 合批才不会丢事件。写盘/持久化只在 `done / error / stopped` 时做一次，**不要每帧写库**。

---

## 4. 把"想"做成可看见的过程

### 4.1 开关是请求级的，不是 UI 级的

`thinking: { type: 'enabled' | 'disabled' }` 随请求下发。开启后才会有 `thinking` 块。

**建议**：不要在 UI 上给用户"思考强度"档位（chat-demo 明确是"系统自动决定是否思考以及思考深度，不需要手动选择"）。"边想边做、边想边查"本来就该由模型按问题复杂度自行决定深浅，给用户一个档位反而破坏了"它自己在想"的感受。留一个"深度思考"总开关就够。

### 4.2 计时：`startedAt` / `endedAt` + 秒级取整

```ts
const getThinkingSeconds = (startedAt, endedAt) => Math.max(1, Math.round((endedAt - startedAt) / 1000))
```

- `startedAt` 在 `content_block_start` 记，`endedAt` 在 `content_block_stop` 记。
- **`Math.max(1, …)` 不是玩笑**：快速思考会算出 0 秒，UI 显示"思考了 0 秒"很出戏。
- **回复结束时兜底**：`settleReply` 里把"还没 endedAt 的思考块"用当前时间收尾。**流断了但思考块没关**是常见情况（见 §9），不收尾就会一直显示"思考中…"。

### 4.3 思考块必须能和正文分开渲染

- 思考块用**更小的字号、更淡的颜色**，默认可能折叠（长思考）。
- 思考块里不要渲染引用角标、不要渲染代码块高亮（思考里的代码往往是伪代码草稿）。chat-demo 给思考块单独一套 Markdown 样式（`thinking.markdown.css`），就是为这个。
- **思考块出现在正文之间时，两侧正文要保留段落分隔**（§8 有细节）。

---

## 5. 把"查"做成可看见的过程

### 5.1 工具调用块：展示"决定"，而不是"参数"

推荐展示形态（按 `kind` 分派）：

| kind / action | 展示 |
|---|---|
| `search` / `deep_search` | 「正在搜索」+ `queries` 的 chip 列表（每个 query 一个 chip） |
| `open_page` / `read_link` | 「正在阅读」+ 域名 + 标题 |
| `find_in_page` | 「页内查找」+ `pattern` |
| `image_search` | 「正在找图」+ query |

- 调用块出现时是 **pending 态**（转圈/呼吸点），对应结果块到达后原地变 **done 态**（对勾 + 结果数）。
- 一个调用块可能对应**多条**结果（`queries` 有 3 个 → 3 组结果），用 `tool_use_id` 关联。
- **不要把工具调用的 JSON 直接丢给用户看**。这是最容易毁掉观感的一步：用户要看的是"它在查什么、查到了什么"，不是 `{"action":"search","queries":[...]}`。

### 5.2 结果块：来源卡，不是文本墙

每条 `WebSearchResult` 渲染成一张卡：`site_icon` + `site_name` + `title` + `snippet` + `page_age`。要点：

- `site_icon` 为 null → 显示域名首字母的色块；图标加载失败 → 同样退化（`FallbackImage`）。
- `snippet` 直接展示，**不要二次截断**（服务端已经控制过长度了，再截就断句）。
- `page_age` 为 null → 不显示时间，不要显示"未知时间"。
- 结果多时**只展示前 3–5 条 + "共 N 条"入口**，全展开会把正文顶到屏幕外，"边想边查"的连续感就断了。

### 5.3 引用锚定：块尾角标

正文块渲染完，若 `citations.length > 0`，在**块尾**追加角标序列（①②③），点击/悬浮弹 `HoverCard`：`title` + `url` + `cited_text`。

因为 §2.4 的"挂过引用就切块"规则，你**不需要**在文本里找插入位置。这是这条协议设计最大的红利。

### 5.4 搜索面板（可选但很加分）

chat-demo 有一个全局的 `SearchPanel`：点任意搜索块，右侧滑出面板，把本次回答涉及的所有工具调用与来源按时间线列出来。适合"回答很长、引用很多"的场景。**第一版可以不做**，但块模型要留好（所有工具块已经在回复数组里了，面板只是换个视图过滤 `kind`）。

---

## 6. 让模型优先用「你的工具」

这是你真正的诉求。三件事决定成败，按影响从大到小排：

### 6.1 声明层：工具数量宁少勿多

- **只声明 1–3 个工具**。声明 10 个工具，模型会犹豫、会调用错、会漏掉你的工具而用它自己记忆里的知识。
- 你的场景建议就两个：
  - `web_search`（`{ queries: string[] }`，必填，2–4 个互补检索式）
  - `read_url`（`{ url: string }`，读用户消息里的链接或搜索结果里的页面）
- **工具名要"像工具"**：`web_search` 比 `my_search_v2` 更能触发调用。别在名字里放公司前缀。
- **参数 schema 里写清"什么时候该用"**（schema 的 description 会进 prompt，是免费的提示位）：

```json
{
  "name": "web_search",
  "description": "检索公开网页。当问题涉及实时信息、新闻、价格、版本号、政策、人物近况、你不确定的事实，或用户明确要求查证时调用。不要用它查你确定知道的通用知识。一次可传 2-4 个互补检索式。",
  "input_schema": {
    "type": "object",
    "properties": {
      "queries": { "type": "array", "items": { "type": "string" }, "minItems": 1, "maxItems": 4 }
    },
    "required": ["queries"]
  }
}
```

### 6.2 提示层：把"优先"写成可执行规则，而不是愿望

系统提示里这几段是**必要的**（顺序也重要，越靠后权重越高）：

```
你可以在思考过程中随时调用工具，而不是必须想完再一次性调用。
当出现以下情况时，先调用 web_search 再继续思考：
- 涉及实时信息（新闻、价格、版本、政策、赛事、天气）
- 涉及具体事实（数字、日期、人名、机构名）而你不完全确定
- 用户要求"查一下""最新""现在""今年"
搜索得到结果后，继续在思考中判断：信息是否足够、是否需要打开某个页面核对、是否需要再搜一次。
不要在没有搜索的情况下断言实时信息；也不要为了显得勤奋而搜索你确定知道的内容。
用户消息里出现 URL 时，优先调用 read_url 读取该页面，而不是猜测其内容。
```

三条要点：

1. **"可以边想边调"必须明说**。很多模型默认"先想完，再一次性调工具"，你要显式允许它在中途插入调用。
2. **给"该搜"的具体触发条件**（实时/数字/日期/用户措辞），比"必要时搜索"有效一个数量级。
3. **给"不该搜"的负例**。只写"要多搜"会导致它对"1+1"也去搜，又慢又蠢。

### 6.3 决策层：并行 queries 是"边想边查"的效率来源

- 强制 `queries` 为数组且 `minItems: 1`，并在 description 里鼓励"2–4 个互补检索式"。
- 提示里加一句：`多个互补检索式请放在同一次调用里并行发出，不要分多次调用。`
- 这样一次往返就能拿到多角度结果，思考块 → 一次调用 → 多组结果 → 继续思考，节奏紧凑。**串行单 query 会把"边想边查"拉成"边等边查"。**

### 6.4 兜底层：模型不调用你的工具怎么办

按成本从低到高：

1. **首次调用强激励**：仅在"用户消息含 URL / 含实时性关键词"时，用一次 `tool_choice: {type:'any'}` 或 `{type:'tool', name:'web_search'}` 强制首轮必调；后续轮放开。这是最有效的一招，且不影响其他场景。
2. **前置路由**：在进模型之前用一个轻量分类（规则或小模型）判断"这轮要不要搜"，要搜就把工具列表里**只留 `web_search`**，从物理上消灭"选错工具"。
3. **事后补救**：正文里出现 `2024年`/`约`/`大概`/`可能` 等不确定词且本轮无工具调用 → 前端给一个"看起来可能过时，要不要查证？"的按钮触发重答（带强制搜索）。**不要自动重跑**，会打断用户。

### 6.5 结果质量决定它愿不愿意再用

这一条最容易被忽视，但它直接决定"边想边查"能不能连续发生：**如果第一次搜索结果又长又水、还慢，模型要么不再调用，要么被噪声带偏。**

- **摘要要短**：每条 `snippet` 控制在 100–200 字，只留与 query 相关的句子。
- **去重**：同域名同路径只留一条；同标题不同 URL 只留最权威的一条。
- **排序**：权威域（gov/官方文档/维基）优先，内容农场降权。
- **给时间**：`page_age` 能拿到就给，模型判断时效全靠它。
- **压延迟**：首字节时间比总时长重要。**流式返回结果**（先回 1 条，再补），比"等齐 10 条再返回"的体感好得多——这是"边想边查"的隐藏前提：用户看到第一个来源卡出现，就知道它在干活了。

---

## 7. 客户端编排版：一个回复 = 多个请求的缝合

如果你的工具只能在前端调（或你暂时不想改造后端），就必须做缝合。以下是必须处理的五件事，漏一件就会露馅。

### 7.1 状态机

```
idle
  → streaming          （发起第 1 个请求）
  → streaming + tool_pending   （收到 tool_use，暂停"打字"，开始执行工具）
  → streaming + tool_running   （工具执行中，UI 显示 pending 块）
  → streaming          （拿到 tool_result，发起第 2 个请求，继续追加到同一回复）
  → done / stopped / error
```

**关键**：`tool_pending` / `tool_running` 期间，回复的 `status` 仍然是 `streaming`。否则"重新生成""停止生成"按钮会闪断，正文也会提前收尾。

### 7.2 块要跨请求累积，不能重置

第 2 个请求返回的块，要**追加**到第 1 个请求已有的块数组后面，而不是替换。所以你的 `applyEvent` 必须接受"已有 blocks"，且 `blockIndexes` 要**每个请求新建一份**（服务端 index 每个请求都从 0 开始）。

### 7.3 必须原样回传 thinking（含 signature）——最容易翻车的一条

客户端编排时，工具往返之间要把上一轮的 assistant 内容（**包括 thinking 块及其 signature**）原样回传，否则：

- 严格实现会直接报错；
- 宽松实现会丢失思考上下文，模型第二次思考"忘了自己刚查过什么"。

**注意：chat-demo 恰恰相反——它回传历史时把 thinking 全部丢掉。** 这不是矛盾，而是因为它走的是服务端编排：工具往返在服务端内部完成，不需要客户端回传思考。**你走客户端编排，就必须回传。**

### 7.4 有工具调用的回复不能"续写"

`canContinueReply` 的判定（照抄）：

```
可续写 = prefix && stop_reason !== 'refusal' && stop_reason !== 'tool_use'
         && 没有任何 server_tool_use / 搜索结果块 / 读页结果块
         && status ∈ {stopped, done 且 stop_reason === 'max_tokens'}
```

理由：续写靠"末尾放 assistant prefill"，而带工具块的历史无法安全地作为 prefill 前缀（服务端会校验失败或语义错乱）。**UI 上直接不显示"继续生成"按钮**，比点了报错好。

### 7.5 伪代码

```ts
async function runAgentTurn(sessionId, question) {
  let reply = createEmptyReply()
  let request = buildRequest(history, question)
  let guard = 0

  while (guard++ < 8) {                       // 硬上限，防死循环
    const pendingTools = []

    for await (const event of streamMessage(request)) {
      reply = applyEvent(reply, event, Date.now(), newBlockIndexes())
      reply = appendToolUseIfAny(reply, event, pendingTools)   // 收集本轮 tool_use
      scheduleFlush(reply)                                      // rAF 合批
    }

    if (pendingTools.length === 0) break                      // 没有工具调用 → 收工

    reply.status = 'streaming'                                // 保持流式中，UI 不闪
    const results = await Promise.all(pendingTools.map(execute))  // 优先并行！§6.3

    reply = appendToolResults(reply, results)                 // 结果块追加到同一回复
    request = buildRequest(history, question, reply)          // 带上 thinking + tool_use + tool_result
  }

  reply = settleReply(reply, 'done', Date.now())
  persist(reply)
}
```

三个补充：

- **工具执行失败也要回 `tool_result`**，内容里写错误原因，让模型自己决定"换 query 再搜"还是"告诉用户查不到"。**不要抛异常中断整轮**——那样"边想边做"就变成了"报错"。
- **`guard` 上限**（8 左右）必须有，否则模型可能反复调用同一个工具。
- 用户点"停止"时，abort 当前请求 **并把 `guard` 置满**，防止 await 返回后继续发下一个请求。

---

## 8. 上下文重建：六条规则，照抄不炸

每次请求前，把历史消息重建成请求体。以下六条是 chat-demo 的成品规则，直接用：

1. **只保留"提问 + 有可用回答"成对的轮次。** 报错、中断、没有正文、被安全拒绝的轮次整对丢弃。比"原样回传并让模型忽略"干净得多，也省 token。
2. **历史 thinking 不回传**（服务端编排时）。规则：只有当它是"当前轮正在续写的那条回复"时才保留思考及其 signature。
3. **搜索结果不回传**，只有"成功的 read_link 调用 + 带 signature 的读页结果"原样回传。搜索结果是易变的中间产物，回传只会膨胀上下文。
4. **签名缺失的 read_link，调用块和结果块一起丢。** 缺一半会让服务端协议校验失败。
5. **相邻正文拼接的规则**：连续 text 块直接拼；被思考或工具隔开的 text 块用**空行**连接（`\n\n`）。这样省略思考后，段落结构仍然正确。
6. **服务端模板提示不进入上下文**（比如"抱歉，我无法回答"这类系统注入的兜底文案），否则模型会把它当自己的话继续模仿。

另外两条与安全相关的：

- **`stop_reason === 'refusal'`** 时，已下发的 thinking 与 text **由客户端主动丢弃**，只保留工具块。这是协议要求（拒绝后不能保留"推理过程"），也是合规要求。
- **`role: 'latest_reminder'`** 这类虚拟角色消息（日期/星期/地区/平台/语言快照），插在每轮 user 之前，**内容没变就不重复插入**（chat-demo 用 `previousReminder` 去重）。它让长会话知道"今天是几号"，是搜索类场景的必要上下文。

---

## 9. 错误与边界：这七种情况必须显式处理

| 情况 | 表现 | 处理 |
|---|---|---|
| 非 2xx | 请求失败 | 读 `body.error.message` 展示原文；无 JSON 时用 statusText |
| 流中错误帧 | `event: error` 或无名 data 帧含 `error` | 按流错误中断，已收到内容保留 |
| 断网/跨域 | `fetch` 抛错 | 归为 network，提示"检查网络" |
| **流在 `message_stop` 前结束** | 无 stop_reason | **若已有 stop_reason → 当正常完成**；否则报"生成中断" |
| 工具失败 | 结果块里是 `*_error` | 展示"检索失败"，**仍把结果回传**让模型决策 |
| 搜索零结果 | `content: []` | 展示"未找到相关来源"，不要报错 |
| 读页被拦 | `url_not_allowed` / `url_not_accessible` | 展示"无法访问该页面"，模型可换源重试 |

**第 4 条是 chat-demo 里最实用的一个判断**：`interrupted` 且已有 `stop_reason` → 判成 `done` 而不是 `error`。因为 `message_delta` 先于 `message_stop` 到达，很多"断了"的情况其实结束原因已经收到了，只是缺了个收尾帧。**不这么判，用户会看到"生成失败"但内容明明全都在。**

---

## 10. 落地清单

### 协议 / 后端
- [ ] 一条流承载完整一轮（含多次工具往返），以 `message_stop` 结束
- [ ] 工具调用块、结果块**原子下发**（start 即全量）
- [ ] 思考块 / 正文块**增量下发**
- [ ] 工具块 id 编码能力类型；`kind` 与 `name` 分离
- [ ] `queries` 支持数组，鼓励并行
- [ ] 结果块 `content` = 结果数组 或 单元素错误数组
- [ ] 正文引用"挂引用即切块"，块内按 url 去重
- [ ] 读页结果带加密 `signature` 供下一轮回传
- [ ] `usage.server_tool_use.web_search_requests` 统计检索次数

### 前端
- [ ] `applyEvent` 是纯函数，块数组只增不改
- [ ] `Map<服务端index, 本地index>` 做索引映射
- [ ] rAF 合批渲染，后台标签页退化为立即刷
- [ ] 思考块：计时（≥1s）、独立样式、结束兜底
- [ ] 工具块：pending → done 状态，按 kind 换文案，不暴露 JSON
- [ ] 结果块：来源卡（icon 可退化、snippet 不二次截断、page_age 可空）
- [ ] 引用角标挂在块尾，点击弹卡片
- [ ] 停止生成：abort + 阻止后续调度
- [ ] 带工具块的回复**不显示"继续生成"**

### 提示 / 工具
- [ ] 只声明 1–3 个工具，名字"像工具"
- [ ] 工具 description 写明"何时用 / 何时不用"
- [ ] 系统提示显式允许"边想边调"，并给出该搜/不该搜的条件
- [ ] 并行 queries 的提示
- [ ] 兜底：`tool_choice` 强制首调 / 前置路由 / 事后补救按钮

---

## 11. 不建议做的四件事

1. **不要给用户"思考强度"滑块。** "边想边做"的价值是它自己判断深浅；给档位等于把判断权交回用户，效果反而更差。
2. **不要把工具调用参数做打字机效果。** 决策是原子的。逐字流动会让"它想清楚了"变成"它在敲键盘"。
3. **不要等所有结果齐了再渲染。** 第一张来源卡出现的时刻，决定了用户觉得它"活着"还是"卡了"。
4. **不要在客户端编排时把 N 个请求暴露成 N 条消息。** 用户心智里这是**一次回答**，不是 5 次。块要跨请求累积到同一条回复上，中间状态保持 `streaming`。
