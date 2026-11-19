package chat

import (
	"strings"
	"unicode/utf8"
)

// demoAnswer 在未配置上游 Key 或上游不可用时，生成一段结构与真实回答一致的演示内容。
// 返回值分别为深度思考摘要与回答正文；正文使用 Markdown，便于前端完整验证排版与代码块。
func demoAnswer(prompt string, deepThinking bool) (reasoning, content string) {
	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(prompt, "SSE") || strings.Contains(lower, "sse"):
		reasoning = "先确认 SSE 的取消语义与 flush 时机，再检查上游响应头与缓冲。\n方案：独立 goroutine 读上游 → 事件通道 → 客户端断开时 cancel。"
		content = "把超时、重试与退避抽成统一配置，代理层只负责按配置执行；\n流中断后重连用 `Last-Event-ID` 续传，避免重复计费。\n\n" +
			"- 超时按 连接 10s / 首字节 30s / 空闲 45s 分段配置\n" +
			"- 指数退避 1s → 2s → 4s，最多 3 次；4xx 不重试\n" +
			"- 单测覆盖断流、超时、半包与客户端主动取消\n\n" +
			"```go\ninternal/sse/proxy.go\n\nfunc (p *Proxy) Stream(w http.ResponseWriter, r *http.Request) {\n" +
			"    cfg := p.cfg.SSE // 超时 / 退避 / 重试统一配置\n" +
			"    ctx, cancel := context.WithCancel(r.Context())\n" +
			"    defer cancel()\n\n" +
			"    up, err := p.connect(ctx, cfg)\n" +
			"    if err != nil { p.fail(w, err); return }\n" +
			"    defer up.Close()\n\n" +
			"    w.Header().Set(\"Content-Type\", \"text/event-stream\")\n" +
			"    w.Header().Set(\"X-Accel-Buffering\", \"no\")\n" +
			"    return relay(ctx, w, up, cfg.Retry)\n}\n```"
	case strings.Contains(prompt, "JWT") || strings.Contains(lower, "jwt"):
		reasoning = "先确认鉴权边界：JWT 只承载身份，不承载权限；再用中间件统一注入上下文。"
		content = "可以在 Gin 中用一个轻量中间件完成校验，并把用户信息放进请求上下文：\n\n" +
			"```go\nfunc Auth(secret []byte) gin.HandlerFunc {\n    return func(c *gin.Context) {\n" +
			"        raw := c.GetHeader(\"Authorization\")\n" +
			"        token := strings.TrimPrefix(raw, \"Bearer \")\n" +
			"        claims, err := Parse(token, secret)\n" +
			"        if err != nil {\n" +
			"            c.AbortWithStatusJSON(401, gin.H{\"code\": 1002})\n" +
			"            return\n" +
			"        }\n" +
			"        c.Set(\"user_id\", claims.Subject)\n" +
			"        c.Next()\n    }\n}\n```\n\n" +
			"要点：\n\n" +
			"1. 校验 `exp`、`nbf` 与签名算法，拒绝 `none`；\n" +
			"2. 密钥只从环境变量读取，不写进代码；\n" +
			"3. 刷新令牌与访问令牌分离，退出登录时撤销服务端会话。"
	case strings.Contains(prompt, "React") || strings.Contains(prompt, "流式消息"):
		reasoning = "流式渲染的关键不是“收到就 setState”，而是合帧与滚动策略，避免 React 每帧重排。"
		content = "流式渲染推荐“增量缓冲 + 16ms 合帧”，并让滚动跟随只在底部时生效：\n\n" +
			"```tsx\nconst buffer = useRef(\"\")\nconst [, force] = useReducer((x) => x + 1, 0)\n\n" +
			"useEffect(() => {\n  const timer = setInterval(() => {\n" +
			"    if (!buffer.current) return\n    setText((prev) => prev + buffer.current)\n" +
			"    buffer.current = \"\"\n  }, 16)\n  return () => clearInterval(timer)\n}, [])\n```\n\n" +
			"- 用 `AbortController` 处理停止生成；\n" +
			"- 代码块按语言懒加载 highlighter，避免首屏体积膨胀；\n" +
			"- 用户上滚后暂停自动跟随，右下角出现“回到底部”。"
	case strings.Contains(prompt, "隔离") || strings.Contains(prompt, "会话"):
		reasoning = "多用户数据隔离必须在查询层强制带 user_id，而不是只在接口入口校验一次。"
		content = "建议把隔离做成“默认安全”的数据访问函数，任何会话查询都必须同时带 `user_id`：\n\n" +
			"```go\nfunc dalFindConversation(ctx context.Context, db *gorm.DB, userID, id uint) (*Conversation, error) {\n" +
			"    var c Conversation\n    err := db.WithContext(ctx).\n" +
			"        Where(\"id = ? AND user_id = ?\", id, userID).\n" +
			"        First(&c).Error\n    return &c, err\n}\n```\n\n" +
			"同时：\n\n" +
			"- 消息表冗余 `user_id`，便于审计与越权排查；\n" +
			"- 删除会话做软删除，避免误删后无法追溯；\n" +
			"- 管理后台只读账号与用量字段，不暴露对话正文。"
	case strings.Contains(prompt, "OpenAI") || strings.Contains(prompt, "兼容"):
		reasoning = "兼容层的价值在于统一协议，而不是适配每个厂商的私有字段；先抽象最小公共集。"
		content = "OpenAI 兼容接入只需要统一三件事：\n\n" +
			"1. 请求：`POST /v1/chat/completions`，`stream=true`；\n" +
			"2. 响应：`data:` 行 + `choices[0].delta.content`，以 `[DONE]` 结束；\n" +
			"3. 用量：`stream_options.include_usage=true` 时最后一帧带 `usage`。\n\n" +
			"```json\n{\"model\":\"deepseek-flash\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}],\n" +
			" \"stream\":true,\"stream_options\":{\"include_usage\":true}}\n```\n\n" +
			"把厂商差异收敛到 `Provider` 接口，业务层只依赖 `StreamCompletion(messages)`。"
	case strings.Contains(prompt, "LangGraph") || strings.Contains(prompt, "工作流"):
		reasoning = "工作流编排先确定状态机，再决定是否需要图；避免为了框架而框架。"
		content = "LangGraph 适合“有状态、可回放”的多步骤 Agent。落地时建议：\n\n" +
			"- 每个节点输入输出都显式声明，便于单测；\n" +
			"- 用 checkpoint 保存中间态，失败从最近节点恢复；\n" +
			"- 人工确认节点独立出来，不要混进工具调用；\n" +
			"- 给每次运行生成 trace id，与日志、用量对齐。\n\n" +
			"如果你只是做线性 RAG，普通函数编排会更简单、更好排查。"
	default:
		reasoning = "先把问题拆成目标、约束与验收三步，再给出可执行的方案。"
		content = "可以，我们先把问题拆开：\n\n" +
			"**目标**：明确你要的结果形态（代码、文档还是决策建议）。\n\n" +
			"**约束**：现有技术栈、上线时间、可接受的复杂度。\n\n" +
			"**验收**：什么情况下算完成，如何验证。\n\n" +
			"围绕「" + truncateRunes(strings.TrimSpace(prompt), 40) + "」，我建议先做一个最小闭环：\n\n" +
			"1. 用最小数据跑通主流程；\n" +
			"2. 补上错误分支与边界条件；\n" +
			"3. 最后再抽组件、加缓存与监控。\n\n" +
			"你希望我先从哪一部分展开？"
	}
	if !deepThinking {
		reasoning = ""
	}
	return reasoning, content
}

// chunkRunes 按固定字符数切分文本，用于模拟流式 token 输出。
// 说明：按 rune 切分避免中文被截断成乱码。
func chunkRunes(text string, size int) []string {
	if size <= 0 {
		size = 16
	}
	runes := []rune(text)
	chunks := make([]string, 0, len(runes)/size+1)
	for start := 0; start < len(runes); start += size {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}

// estimateTokens 估算 tokens 数量。
// 说明：中文约 1 字 ≈ 0.6 token，这里按 rune/2 粗估，仅用于未返回 usage 的场景。
func estimateTokens(text string) int {
	n := utf8.RuneCountInString(text)
	if n == 0 {
		return 0
	}
	return n/2 + 1
}

// truncateRunes 按字符数截断，避免演示回答里出现超长原文。
func truncateRunes(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max]) + "…"
}
