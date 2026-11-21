// Package responses 定义统一响应信封与各端响应结构，控制器不得自行拼装 JSON 信封。
package responses

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"deeptalk/internal/billing"
	"deeptalk/internal/chat"
	"deeptalk/internal/provider"
	"deeptalk/internal/user"

	"github.com/gin-gonic/gin"
)

// 错误码。code == 0 表示成功，>0 表示失败。
const (
	CodeOK              = 0    // 成功
	CodeInvalidArgument = 1001 // 参数不合法
	CodeUnauthenticated = 1002 // 未登录或会话失效
	CodeForbidden       = 1003 // 无权限
	CodeNotFound        = 1004 // 资源不存在
	CodeConflict        = 1005 // 冲突（邮箱已注册等）
	CodeLocked          = 1006 // 账号锁定或登录受限
	CodeQuotaExceeded   = 1007 // 额度不足
	CodeRegistrationOff = 1008 // 注册已关闭
	CodeInternal        = 2001 // 服务端错误
	CodeUpstream        = 3001 // 上游调用失败
)

// Envelope 是所有 JSON 接口的统一信封。
type Envelope struct {
	Code int    `json:"code"` // 业务码
	Msg  string `json:"msg"`  // 提示消息
	Data any    `json:"data"` // 业务数据
}

// OK 返回成功响应。
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Code: CodeOK, Msg: "ok", Data: data})
}

// Fail 返回失败响应，并按业务码映射 HTTP 状态。
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(httpStatus(code), Envelope{Code: code, Msg: msg, Data: nil})
}

// FromError 将领域错误映射为统一错误码与可展示文案。
// 未知错误统一返回 2001，避免把内部细节暴露给前端；调用方应同时记录日志。
// billingAsLimit 取出套餐限额错误，便于给出具体原因与恢复时间。
func billingAsLimit(err error) *billing.LimitError {
	var limit *billing.LimitError
	if errors.As(err, &limit) {
		return limit
	}
	return nil
}

// limitMessage 生成用户可读的限额提示。
func limitMessage(limit *billing.LimitError) string {
	when := ""
	if !limit.ResetsAt.IsZero() {
		when = "，约 " + limit.ResetsAt.Format("01-02 15:04") + " 恢复"
	}
	switch limit.Window {
	case "5h":
		return fmt.Sprintf("5 小时内的请求已达上限（%d 次）%s", limit.Limit, when)
	case "week":
		return fmt.Sprintf("本周请求已达上限（%d 次）%s", limit.Limit, when)
	case "month":
		if limit.Kind == billing.ErrSpendLimit {
			return fmt.Sprintf("本月费用额度已用完（%.2f 元）%s", float64(limit.Limit)/100, when)
		}
		return fmt.Sprintf("本月请求已达上限（%d 次）%s", limit.Limit, when)
	}
	return limit.Kind.Error()
}

func FromError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, user.ErrInvalidCredentials):
		Fail(c, CodeInvalidArgument, "邮箱或密码不正确")
	case errors.Is(err, user.ErrLocked):
		Fail(c, CodeLocked, "登录失败次数过多，请 10 分钟后再试")
	case errors.Is(err, user.ErrDisabled):
		Fail(c, CodeLocked, "该账号已被限制登录，请联系管理员")
	case errors.Is(err, user.ErrEmailTaken):
		Fail(c, CodeConflict, "该邮箱已注册，请直接登录")
	case errors.Is(err, user.ErrRegistrationClosed):
		Fail(c, CodeRegistrationOff, "注册已关闭，请联系管理员")
	case errors.Is(err, user.ErrUnauthenticated):
		Fail(c, CodeUnauthenticated, "登录已失效，请重新登录")
	case errors.Is(err, user.ErrForbidden):
		Fail(c, CodeForbidden, "没有权限执行该操作")
	case errors.Is(err, user.ErrWeakPassword):
		Fail(c, CodeInvalidArgument, "密码至少 8 位，且需同时包含字母与数字")
	case errors.Is(err, user.ErrNotFound), errors.Is(err, chat.ErrNotFound), errors.Is(err, chat.ErrMessageNotOwn),
		errors.Is(err, provider.ErrNotFound), errors.Is(err, provider.ErrModelNotFound):
		Fail(c, CodeNotFound, "资源不存在或无权访问")
	case errors.Is(err, chat.ErrNotGenerating):
		Fail(c, CodeInvalidArgument, "当前没有正在生成的回答，可直接发送")
	case errors.Is(err, chat.ErrEmptyInterject):
		Fail(c, CodeInvalidArgument, "插话内容不能为空")
	case billingAsLimit(err) != nil:
		limit := billingAsLimit(err)
		Fail(c, CodeQuotaExceeded, limitMessage(limit))
	case errors.Is(err, billing.ErrPlanNotFound):
		Fail(c, CodeNotFound, "套餐不存在或已被删除")
	case errors.Is(err, chat.ErrQuotaExceeded):
		Fail(c, CodeQuotaExceeded, "本月额度已用完，请联系管理员调整额度")
	case errors.Is(err, chat.ErrNoModel):
		Fail(c, CodeUpstream, "当前没有可用模型，请联系管理员")
	case errors.Is(err, provider.ErrBuiltinDelete):
		Fail(c, CodeForbidden, "预置厂商不可删除，可将其停用")
	case errors.Is(err, provider.ErrInvalidBaseURL):
		Fail(c, CodeInvalidArgument, "Base URL 不合法")
	case strings.Contains(err.Error(), "invalid email"), strings.Contains(err.Error(), "email"):
		Fail(c, CodeInvalidArgument, "邮箱格式不正确")
	case strings.Contains(err.Error(), "password"):
		Fail(c, CodeInvalidArgument, "密码至少 8 位，且需同时包含字母与数字")
	default:
		// 通用参数类错误使用 1001，其余按内部错误处理
		msg := err.Error()
		if isArgumentError(msg) {
			Fail(c, CodeInvalidArgument, msg)
			return
		}
		Fail(c, CodeInternal, "服务暂时不可用，请稍后重试")
	}
}

// httpStatus 将业务码映射为 HTTP 状态码，便于网关与前端统一处理。
func httpStatus(code int) int {
	switch code {
	case CodeOK:
		return http.StatusOK
	case CodeInvalidArgument:
		return http.StatusBadRequest
	case CodeUnauthenticated:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeLocked:
		return http.StatusLocked
	case CodeQuotaExceeded:
		return http.StatusPaymentRequired
	case CodeRegistrationOff:
		return http.StatusForbidden
	case CodeUpstream:
		return http.StatusBadGateway
	default:
		return http.StatusInternalServerError
	}
}

// isArgumentError 判断错误是否属于可由用户修正的参数问题。
func isArgumentError(msg string) bool {
	keywords := []string{
		"required", "invalid", "too long", "too short", "must be", "至少", "不能", "不可用", "not found",
		"name is", "content", "quota", "password", "email", "title", "无需",
	}
	for _, kw := range keywords {
		if contains(msg, kw) {
			return true
		}
	}
	return false
}

// contains 是大小写无关的子串判断，避免为一次判断引入额外依赖。
func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

// indexOf 返回子串位置，未找到返回 -1。
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
