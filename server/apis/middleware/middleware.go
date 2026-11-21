// Package middleware 提供请求 ID、日志、恢复、CORS 与登录态校验等共享中间件。
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"deeptalk/apis/responses"
	"deeptalk/internal/user"

	"github.com/gin-gonic/gin"
)

// 上下文键名，集中定义避免拼写错误。
const (
	contextUser    = "current_user"  // 当前登录用户
	contextToken   = "session_token" // 当前会话 token
	contextReqID   = "request_id"    // 请求 ID
	contextSignKey = "sign_key_id"   // 会话签名密钥标识
)

// Config 是中间件所需的配置。
type Config struct {
	CookieName     string        // 会话 Cookie 名
	CookieSecure   bool          // 生产环境仅 HTTPS
	SessionTTL     time.Duration // 会话有效期
	AllowedOrigins []string      // 允许跨域的来源
}

// RequestID 为每个请求生成唯一 ID，响应头与日志共用。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-Id")
		if id == "" {
			raw := make([]byte, 8)
			if _, err := rand.Read(raw); err == nil {
				id = hex.EncodeToString(raw)
			} else {
				id = time.Now().Format("20060102150405")
			}
		}
		c.Set(contextReqID, id)
		c.Header("X-Request-Id", id)
		c.Next()
	}
}

// Recovery 捕获 panic，返回统一 500，避免连接直接断开。
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic recovered", "request_id", RequestIDFrom(c), "panic", rec, "stack", string(debug.Stack()))
				if !c.Writer.Written() {
					responses.Fail(c, responses.CodeInternal, "服务暂时不可用，请稍后重试")
				}
				c.Abort()
			}
		}()
		c.Next()
	}
}

// AccessLog 记录访问日志；不打印请求体与敏感头。
func AccessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		// 健康检查与静态资源不记录，避免日志噪声
		path := c.Request.URL.Path
		if path == "/healthz" || strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/admin/assets/") {
			return
		}
		log.Info("http_request",
			"request_id", RequestIDFrom(c),
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"ip", c.ClientIP(),
			"user_id", UserIDFrom(c),
		)
	}
}

// CORS 允许开发期 Vite 端口携带 Cookie 访问；生产同源部署时不会触发。
func CORS(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[strings.TrimRight(origin, "/")] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := strings.TrimRight(c.GetHeader("Origin"), "/")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Request-Id")
				c.Header("Access-Control-Max-Age", "600")
			}
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// Auth 校验会话 Cookie，并把当前用户写入上下文。
// 被限制登录的用户会立即失效，即使 Cookie 尚未过期。
func Auth(users *user.Service, cfg Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := sessionTokenFrom(c, cfg)
		if token == "" {
			responses.Fail(c, responses.CodeUnauthenticated, "登录已失效，请重新登录")
			c.Abort()
			return
		}
		current, err := users.Authenticate(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, user.ErrDisabled) {
				ClearSessionCookie(c, cfg)
				responses.Fail(c, responses.CodeLocked, "该账号已被限制登录，请联系管理员")
				c.Abort()
				return
			}
			responses.Fail(c, responses.CodeUnauthenticated, "登录已失效，请重新登录")
			c.Abort()
			return
		}
		c.Set(contextUser, current)
		c.Set(contextToken, token)
		if keyID, keyErr := users.SessionSignKey(c.Request.Context(), token); keyErr == nil {
			c.Set(contextSignKey, keyID)
		}
		c.Next()
	}
}

// sessionTokenFrom 依次从 Cookie、Authorization: Bearer、X-Session-Token 取会话令牌。
// 小程序 / APP 无法可靠携带 Cookie，用请求头更稳；Web 端仍走 HttpOnly Cookie。
func sessionTokenFrom(c *gin.Context, cfg Config) string {
	if cookie, err := c.Cookie(cfg.CookieName); err == nil && cookie != "" {
		return cookie
	}
	if header := strings.TrimSpace(c.GetHeader("Authorization")); strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	if header := strings.TrimSpace(c.GetHeader("X-Session-Token")); header != "" {
		return header
	}
	return ""
}

// AdminOnly 校验管理员身份；必须在 Auth 之后使用。
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		current := CurrentUser(c)
		if current == nil || current.Role != user.RoleAdmin {
			responses.Fail(c, responses.CodeForbidden, "只有平台管理员可以访问管理后台")
			c.Abort()
			return
		}
		c.Next()
	}
}

// SetSessionCookie 下发 HttpOnly 会话 Cookie。
func SetSessionCookie(c *gin.Context, cfg Config, token string) {
	maxAge := int(cfg.SessionTTL.Seconds())
	if maxAge <= 0 {
		maxAge = 7 * 24 * 3600
	}
	c.SetSameSite(http.SameSiteLaxMode)
	// domain 留空表示仅当前站点；SameSite=Lax 可防御大部分 CSRF
	c.SetCookie(cfg.CookieName, token, maxAge, "/", "", cfg.CookieSecure, true)
}

// ClearSessionCookie 清除会话 Cookie，用于退出与密码变更。
func ClearSessionCookie(c *gin.Context, cfg Config) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cfg.CookieName, "", -1, "/", "", cfg.CookieSecure, true)
}

// CurrentUser 返回上下文中的当前用户；未登录返回 nil。
func CurrentUser(c *gin.Context) *user.User {
	value, ok := c.Get(contextUser)
	if !ok {
		return nil
	}
	current, ok := value.(*user.User)
	return current
}

// SessionToken 返回当前请求的会话 token。
func SessionToken(c *gin.Context) string {
	value, _ := c.Get(contextToken)
	token, _ := value.(string)
	return token
}

// RequestIDFrom 返回请求 ID。
func RequestIDFrom(c *gin.Context) string {
	value, _ := c.Get(contextReqID)
	id, _ := value.(string)
	return id
}

// UserIDFrom 返回当前用户 ID，未登录时为 0。
func UserIDFrom(c *gin.Context) uint {
	if current := CurrentUser(c); current != nil {
		return current.ID
	}
	return 0
}
