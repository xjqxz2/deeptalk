package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"deeptalk/apis/responses"
	"deeptalk/internal/security"

	"github.com/gin-gonic/gin"
)

// SignMode 是签名校验模式。
const (
	SignModeOff     = "off"     // 不校验
	SignModeLog     = "log"     // 只记录不拦截，用于灰度观察
	SignModeEnforce = "enforce" // 缺少或错误一律拒绝
)

// SignatureMaxBody 是参与签名的请求体上限；超过的请求不做签名校验（例如大文件上传）。
const SignatureMaxBody = 8 << 20

// Signature 校验写请求的 HMAC 签名（时间窗 + 一次性 nonce + 请求体摘要）。
// 必须在 Auth 之后挂载：密钥按会话分配。
func Signature(verifier *security.Verifier, mode string, log *slog.Logger) gin.HandlerFunc {
	enforce := mode == SignModeEnforce
	if verifier.Disabled() || mode == SignModeOff {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		// MCP 端点由外部 Agent 直接调用，它们无法计算我们的 HMAC；
		// 该端点仍要求 Bearer 会话，并在服务端按用户隔离数据。
		if strings.HasSuffix(c.Request.URL.Path, "/memo/mcp") {
			c.Next()
			return
		}
		// 文件上传走 multipart，整体哈希代价过高，且已被会话与大小限制保护
		if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
			c.Next()
			return
		}
		if c.Request.ContentLength > SignatureMaxBody {
			c.Next()
			return
		}

		keyID, _ := c.Get(contextSignKey)
		key, _ := keyID.(string)
		if key == "" {
			rejectSignature(c, enforce, log, security.ErrMissing)
			return
		}

		body, err := io.ReadAll(io.LimitReader(c.Request.Body, SignatureMaxBody+1))
		if err != nil {
			rejectSignature(c, enforce, log, security.ErrBadFormat)
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		verifyErr := verifier.Verify(
			key,
			c.Request.Method,
			c.Request.URL.Path,
			c.Request.URL.RawQuery,
			body,
			c.GetHeader(security.HeaderTimestamp),
			c.GetHeader(security.HeaderNonce),
			c.GetHeader(security.HeaderSignature),
		)
		if verifyErr != nil {
			rejectSignature(c, enforce, log, verifyErr)
			return
		}
		c.Next()
	}
}

// rejectSignature 按模式记录或拒绝。
func rejectSignature(c *gin.Context, enforce bool, log *slog.Logger, err error) {
	if log != nil {
		log.Warn("request signature rejected",
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"mode", map[bool]string{true: SignModeEnforce, false: SignModeLog}[enforce],
			"error", err.Error(),
			"request_id", RequestIDFrom(c),
		)
	}
	if !enforce {
		c.Next()
		return
	}
	message := "请求签名校验失败"
	switch err {
	case security.ErrMissing:
		message = "请求缺少签名参数，请更新客户端"
	case security.ErrExpired:
		message = "请求已过期，请检查设备时间后重试"
	case security.ErrReplay:
		message = "请求已提交过，请勿重复发送"
	case security.ErrBadFormat:
		message = "请求签名格式不正确"
	}
	responses.Fail(c, responses.CodeInvalidArgument, message)
	c.Abort()
}
