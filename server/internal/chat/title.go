package chat

import (
	"context"
	"strings"
	"time"

	"deeptalk/internal/provider"
)

// titlePrompt 让模型把用户问题压缩成一个短标题，风格对齐 DeepSeek：
// 只给结论，不带引号或标点，长度控制在十来个字。
const titlePrompt = `你是会话标题生成器。根据用户的问题，输出一个不超过 12 个字的中文标题，概括主题。
要求：只输出标题本身；不要引号、书名号、句号、冒号；不要解释；不要以“关于”“询问”开头；不要换行。`

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

		var b strings.Builder
		_, err := s.providers.StreamCompletion(runCtx, prov, modelID, []provider.ChatMessage{
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
			out <- ""
			return
		}
		out <- cleanTitle(b.String())
	}()
	return out
}

// errTitleEnough 用于提前结束标题请求的流式读取。
var errTitleEnough = titleEnoughError{}

type titleEnoughError struct{}

func (titleEnoughError) Error() string { return "title enough" }

func isTitleEnough(err error) bool {
	_, ok := err.(titleEnoughError)
	return ok
}

// cleanTitle 清洗模型输出：取第一行、去掉引号与标点、限制长度。
func cleanTitle(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	if index := strings.IndexAny(text, "\r\n"); index >= 0 {
		text = text[:index]
	}
	text = strings.TrimSpace(text)
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
	runes := []rune(text)
	if len(runes) > 24 {
		runes = runes[:24]
	}
	return strings.TrimSpace(string(runes))
}
