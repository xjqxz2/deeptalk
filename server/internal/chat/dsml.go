package chat

import (
	"encoding/json"
	"regexp"
	"strings"

	"deeptalk/internal/provider"
)

// 部分上游模型（如 DeepSeek 的 DSML 系列）不会返回 OpenAI 的 tool_calls，
// 而是把工具调用写成正文里的标记：
//
//	<｜｜DSML｜｜ invoke name="web_search">
//	<｜｜DSML｜｜ parameter name="query" string="true">关键词</｜｜DSML｜｜ parameter>
//	</｜｜DSML｜｜ invoke>
//
// 这里把这类标记从正文里剥掉，并转换成标准工具调用，让搜索照常执行。

const dsmlBar = "｜｜"

var (
	// dsmlMarkerRe 匹配标记前缀，兼容全角与半角竖线
	dsmlMarkerRe = regexp.MustCompile(`<[|｜]{2}DSML[|｜]{2}`)
	// dsmlInvokeRe 匹配一个 invoke 块
	dsmlInvokeRe = regexp.MustCompile(`(?s)<[|｜]{2}DSML[|｜]{2}\s*invoke\s+name="([^"]+)"\s*>(.*?)</[|｜]{2}DSML[|｜]{2}\s*invoke\s*>`)
	// dsmlParamRe 匹配一个 parameter 块
	dsmlParamRe = regexp.MustCompile(`(?s)<[|｜]{2}DSML[|｜]{2}\s*parameter\s+name="([^"]+)"(?:\s+string="(true|false)")?\s*>(.*?)</[|｜]{2}DSML[|｜]{2}\s*parameter\s*>`)
	// dsmlTagRe 兜底清理残留的单个标记
	dsmlTagRe = regexp.MustCompile(`</?[|｜]{2}DSML[|｜]{2}[^>]*>`)
)

// HasDSML 判断文本中是否出现 DSML 标记。
func HasDSML(text string) bool {
	return dsmlMarkerRe.MatchString(text)
}

// ParseDSMLCalls 从正文中提取工具调用，并返回去掉标记后的正文。
func ParseDSMLCalls(text string) ([]provider.ToolCall, string) {
	if !HasDSML(text) {
		return nil, text
	}
	var calls []provider.ToolCall
	for index, match := range dsmlInvokeRe.FindAllStringSubmatch(text, -1) {
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		args := map[string]any{}
		for _, param := range dsmlParamRe.FindAllStringSubmatch(match[2], -1) {
			key := strings.TrimSpace(param[1])
			if key == "" {
				continue
			}
			raw := strings.TrimSpace(param[3])
			if param[2] == "false" {
				var decoded any
				if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
					args[key] = decoded
					continue
				}
			}
			args[key] = raw
		}
		encoded, err := json.Marshal(args)
		if err != nil {
			continue
		}
		calls = append(calls, provider.ToolCall{
			ID:        "dsml_call_" + itoa(index+1),
			Name:      name,
			Arguments: string(encoded),
		})
	}
	return calls, SanitizeDSML(text)
}

// SanitizeDSML 去掉正文里的 DSML 标记与调用块，避免把内部协议展示给用户。
func SanitizeDSML(text string) string {
	if !HasDSML(text) {
		return text
	}
	cleaned := dsmlInvokeRe.ReplaceAllString(text, "")
	// 没有完整 invoke 块时，至少把零散标记删掉
	cleaned = dsmlTagRe.ReplaceAllString(cleaned, "")
	cleaned = strings.TrimRight(cleaned, " \t\r\n")
	if strings.TrimSpace(cleaned) == "" {
		return ""
	}
	return cleaned
}

// itoa 只用于生成稳定的调用 ID。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// dsmlStream 过滤流式正文：
// 遇到 DSML 标记就停止向用户输出，把标记内容攒下来交给 ParseDSMLCalls。
type dsmlStream struct {
	emit    func(kind, delta string) error // 向上游之外输出安全正文
	pending strings.Builder                // 尚未确认安全的尾巴
	block   strings.Builder                // DSML 块内容
	spoken  strings.Builder                // 已经下发给用户的正文
	inside  bool                           // 是否已进入 DSML 块
}

// Spoken 返回已经输出给用户的正文，用于把调用前后的文本写回上下文。
func (d *dsmlStream) Spoken() string { return d.spoken.String() }

func newDSMLStream(emit func(kind, delta string) error) *dsmlStream {
	return &dsmlStream{emit: emit}
}

// Write 消费一段正文增量，只把安全部分转发给客户端。
func (d *dsmlStream) Write(delta string) error {
	if delta == "" {
		return nil
	}
	if d.inside {
		d.block.WriteString(delta)
		return nil
	}
	d.pending.WriteString(delta)
	text := d.pending.String()
	if index := dsmlMarkerRe.FindStringIndex(text); index != nil {
		if index[0] > 0 {
			d.spoken.WriteString(text[:index[0]])
			if err := d.emit("content", text[:index[0]]); err != nil {
				return err
			}
		}
		d.inside = true
		d.block.WriteString(text[index[0]:])
		d.pending.Reset()
		return nil
	}
	// 没命中标记：保留可能是标记前缀的尾巴，其余立即下发
	keep := markerPrefixSuffixLen(text)
	if safe := len(text) - keep; safe > 0 {
		d.spoken.WriteString(text[:safe])
		if err := d.emit("content", text[:safe]); err != nil {
			return err
		}
		d.pending.Reset()
		d.pending.WriteString(text[safe:])
	}
	return nil
}

// Finish 在流结束后清空尾部，并返回解析出的工具调用。
func (d *dsmlStream) Finish() []provider.ToolCall {
	if !d.inside {
		if rest := d.pending.String(); rest != "" {
			d.spoken.WriteString(rest)
			_ = d.emit("content", rest)
		}
		d.pending.Reset()
		return nil
	}
	calls, cleaned := ParseDSMLCalls(d.block.String())
	// 块里可能夹着少量正常文字，补发出去
	if strings.TrimSpace(cleaned) != "" {
		d.spoken.WriteString(cleaned)
		_ = d.emit("content", cleaned)
	}
	d.block.Reset()
	d.inside = false
	return calls
}

// markerPrefixSuffixLen 返回尾部可能是标记前缀的长度，用于延迟下发。
func markerPrefixSuffixLen(text string) int {
	limit := len("<|｜|DSML") + 8
	if limit > len(text) {
		limit = len(text)
	}
	for size := limit; size > 0; size-- {
		tail := text[len(text)-size:]
		if strings.HasPrefix("<｜｜DSML｜｜", tail) || strings.HasPrefix("<||DSML||", tail) {
			return size
		}
	}
	return 0
}
