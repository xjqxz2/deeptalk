package chat

import (
	"context"
	"strings"
	"time"

	"deeptalk/internal/provider"
)

// titlePrompt 让模型把用户问题压缩成一个短标题，风格对齐 DeepSeek：
// 只给结论，不带引号或标点，长度控制在十来个字。
const titlePrompt = `你是会话标题生成器。根据用户的问题，生成一个 6 到 14 个字的中文标题。
要求：
1. 只输出标题本身，一行，不要换行；
2. 不要引号、书名号、句号、冒号、分点符号；
3. 不要解释、不要重复问题原句、不要以“关于”“询问”开头；
4. 概括主题而不是回答内容，例如“小红书挂商品链接流程”。`

// isDefaultTitle 判断会话是否还停留在默认标题。
func isDefaultTitle(title string) bool {
	trimmed := strings.TrimSpace(title)
	return trimmed == "" || trimmed == "新对话"
}

// defaultTitleSeed 取用户输入作为标题种子；纯图片对话用固定文案。
func defaultTitleSeed(content string, hasImages bool) string {
	text := strings.TrimSpace(content)
	if text == "" {
		if hasImages {
			return "图片对话"
		}
		return ""
	}
	// 只把问题前 400 个字交给模型，避免长粘贴拖慢标题生成
	runes := []rune(text)
	if len(runes) > 400 {
		runes = runes[:400]
	}
	return string(runes)
}

// startTitleSummary 在回答流式输出的同时，并行让模型生成一个短标题。
// 返回 nil 表示当前会话已有标题（非首轮），不需要生成。
// 结果通过带缓冲的 channel 返回，调用方超时取不到就退回本地启发式标题。
func (s *Service) startTitleSummary(ctx context.Context, prov *provider.Provider, modelID, seed string) <-chan string {
	if prov == nil || strings.TrimSpace(seed) == "" {
		return nil
	}
	out := make(chan string, 1)
	go func() {
		// 生成标题不应该被用户点“停止生成”打断：标题仍然有意义
		runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		// 模型偶尔只吐两三个字，重试一次通常就正常；两次都不行就让调用方用兜底标题
		for attempt := 0; attempt < 2; attempt++ {
			if title := s.requestTitle(runCtx, prov, modelID, seed); len([]rune(title)) >= 4 {
				out <- title
				return
			}
			if runCtx.Err() != nil {
				break
			}
		}
		out <- ""
	}()
	return out
}

// requestTitle 发起一次标题请求并清洗结果。
func (s *Service) requestTitle(ctx context.Context, prov *provider.Provider, modelID, seed string) string {
	var b strings.Builder
	_, err := s.providers.StreamCompletion(ctx, prov, modelID, []provider.ChatMessage{
		{Role: "system", Content: titlePrompt},
		{Role: "user", Content: seed},
	}, false, "", 64, 8*time.Second, func(ev provider.StreamEvent) error {
		if ev.Kind != "content" {
			return nil
		}
		b.WriteString(ev.Delta)
		// 标题很短，拿到足够内容就收手，省一次等待
		if len([]rune(b.String())) >= 40 {
			return errTitleEnough
		}
		return nil
	})
	if err != nil && !isTitleEnough(err) {
		return ""
	}
	return cleanTitle(b.String())
}

// errTitleEnough 用于提前结束标题请求的流式读取。
var errTitleEnough = titleEnoughError{}

type titleEnoughError struct{}

func (titleEnoughError) Error() string { return "title enough" }

func isTitleEnough(err error) bool {
	_, ok := err.(titleEnoughError)
	return ok
}

// cleanTitle 清洗模型输出：折叠空白、去掉引号与标点、限制长度。
func cleanTitle(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	// 模型偶尔会分行输出（"便宜好" + 换行 + 补充说明），直接取第一行会丢掉语义，
	// 这里把所有空白折叠成单个空格，再整体限长。
	text = strings.Join(strings.Fields(text), " ")
	text = strings.TrimPrefix(text, "标题：")
	text = strings.TrimPrefix(text, "标题:")
	// 引号和句号常常交替出现（“标题”。），反复清理直到稳定
	for range 4 {
		before := text
		text = strings.Trim(text, " \t\"'“”‘’《》〈〉「」『』【】[]（）()")
		text = strings.TrimRight(text, "。．.,，、；;：:！!？?")
		text = strings.TrimSpace(text)
		if text == before {
			break
		}
	}
	runes := []rune(strings.TrimSpace(text))
	if len(runes) < 2 {
		// 太短的结果没有信息量，交给本地兜底标题
		return ""
	}
	if len(runes) > 24 {
		runes = runes[:24]
	}
	return strings.TrimSpace(string(runes))
}
