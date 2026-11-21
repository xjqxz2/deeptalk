// Package compliance 负责小程序端的内容安全校验。
//
// 微信要求文本在服务端用 msgSecCheck 检测，且 v2 接口必须带用户 openid。
// 本项目只对"来自微信小程序"的请求做检测（PWA / APP 走自己的策略），
// 未上架的小程序每天只有 100 次额度，因此所有失败都按配置的策略降级，
// 不会因为微信侧限流把对话整体打挂。
package compliance

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"unicode/utf8"

	"deeptalk/internal/wechat"
)

// 平台标识：客户端通过 X-Client 头声明。
const PlatformMPWeChat = "mp-weixin"

// 业务错误。
var (
	ErrBlocked     = errors.New("内容包含违规信息，请修改后重试")
	ErrNotBound    = errors.New("尚未完成小程序授权，请重新进入小程序")
	ErrUnavailable = errors.New("内容安全服务暂时不可用，请稍后重试")
)

// 微信单次检测的文本上限（文档：2500 字）。
const maxCheckRunes = 2500

// OpenIDStore 提供用户绑定的 openid。
type OpenIDStore interface {
	WeChatOpenID(ctx context.Context, userID uint) string
}

// Logger 只依赖标准库的 slog。
type Logger = *slog.Logger

// Config 是内容安全的配置。
type Config struct {
	Enabled     bool // 是否启用
	Scene       int  // 场景：1 资料 / 2 评论 / 3 论坛 / 4 社交日志
	FailOpen    bool // 微信接口异常时是否放行（默认放行，避免误伤）
	RequireBind bool // 没有 openid 时是否拦截（默认只告警）
}

// Service 实现文本检测。
type Service struct {
	cfg   Config
	wx    *wechat.Manager
	users OpenIDStore
	log   Logger
}

// New 创建服务；wx 或 users 为空时 CheckText 直接放行。
func New(cfg Config, wx *wechat.Manager, users OpenIDStore, log Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	if cfg.Scene <= 0 {
		cfg.Scene = wechat.SceneComment
	}
	return &Service{cfg: cfg, wx: wx, users: users, log: log}
}

// Enabled 表示是否会对小程序端请求做检测。
func (s *Service) Enabled() bool {
	return s != nil && s.cfg.Enabled && s.wx != nil && s.wx.Enables()
}

// CheckText 检测用户输入；只在平台为小程序且启用时生效。
// 返回 ErrBlocked 表示内容违规，调用方应中止本次请求。
func (s *Service) CheckText(ctx context.Context, userID uint, text, platform string) error {
	if !s.Enabled() || platform != PlatformMPWeChat {
		return nil
	}
	content := strings.TrimSpace(text)
	if content == "" {
		return nil
	}

	openid := ""
	if s.users != nil {
		openid = s.users.WeChatOpenID(ctx, userID)
	}
	if openid == "" {
		if s.cfg.RequireBind {
			return ErrNotBound
		}
		s.log.Warn("小程序内容安全缺少 openid，已跳过检测", "user_id", userID)
		return nil
	}

	verdict, err := s.wx.MsgSecCheck(ctx, clampRunes(content, maxCheckRunes), openid, s.cfg.Scene)
	if errors.Is(err, wechat.ErrContentRisky) {
		s.log.Warn("小程序内容安全命中违规", "user_id", userID, "label", verdictLabel(verdict), "trace_id", traceID(verdict))
		return ErrBlocked
	}
	if err != nil {
		s.log.Warn("小程序内容安全检测失败", "user_id", userID, "error", err)
		if s.cfg.FailOpen {
			return nil
		}
		return ErrUnavailable
	}
	if verdict != nil && verdict.Review() {
		// review 不拦截，但记录便于人工复核
		s.log.Info("小程序内容安全建议复核", "user_id", userID, "label", verdict.Label, "trace_id", verdict.TraceID)
	}
	return nil
}

// clampRunes 按字符截断，避免超过微信单次 2500 字的上限。
func clampRunes(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	return string(runes[:limit])
}

func verdictLabel(v *wechat.Verdict) int {
	if v == nil {
		return 0
	}
	return v.Label
}

func traceID(v *wechat.Verdict) string {
	if v == nil {
		return ""
	}
	return v.TraceID
}
