# 灵感备忘录（Memo）

对话里随时记下灵感，需要时再取出来。两种接入方式共用同一份数据和同一套用户隔离。

## 1. 对话内工具（默认开启）

模型可用两个工具：

| 工具 | 触发场景 | 参数 |
| --- | --- | --- |
| `memo_save` | 用户说「记一下 / 帮我记住 / 存起来」 | `title?`、`content`、`tags?` |
| `memo_search` | 用户问「上次那个想法 / 之前记的关于 X」 | `query?`、`limit?` |

模型会把内容整理成完整句子再保存，不会直接复制原话。

## 2. REST（给客户端做备忘录页面）

需要 C 端会话 Bearer；写请求走签名。

```
GET    /api/v1/c/memos?q=<关键词>     列出 / 检索
POST   /api/v1/c/memos                { title?, content, tags? }
PATCH  /api/v1/c/memos/:id            { title?, content?, tags? }
DELETE /api/v1/c/memos/:id
```

## 3. MCP 端点（给外部 Agent）

```text
POST https://deeptalk.peon.top/api/v1/c/memo/mcp
Authorization: Bearer <C 端 token>
Accept: application/json, text/event-stream
```

Streamable HTTP，先 `initialize` 拿到 `Mcp-Session-Id`，后续请求带上即可。

工具：`memo_save`、`memo_search`、`memo_delete`。

该端点豁免 HMAC 签名（外部 Agent 无法计算），但仍要求 Bearer 会话，
数据按 token 对应的用户隔离。

### DSH 连接示例

```jsonc
{
  "mcpServers": {
    "deeptalk-memo": {
      "url": "https://deeptalk.peon.top/api/v1/c/memo/mcp",
      "headers": { "Authorization": "Bearer <token>" }
    }
  }
}
```
