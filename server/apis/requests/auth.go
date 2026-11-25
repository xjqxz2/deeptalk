package requests

import (
	"github.com/gin-gonic/gin"
)

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=190"`    // 邮箱
	Password string `json:"password" binding:"required,min=8,max=128"` // 密码
}

// Bind 绑定并校验注册请求。
func (r *RegisterRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// LoginRequest 是 C 端登录请求。
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=190"`    // 邮箱
	Password string `json:"password" binding:"required,min=1,max=128"` // 密码
}

// Bind 绑定登录请求。
func (r *LoginRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// ChangePasswordRequest 是修改密码请求。
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`               // 当前密码
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"` // 新密码
}

// Bind 绑定修改密码请求。
func (r *ChangePasswordRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// ConversationCreateRequest 是新建会话请求，标题可空。

// WeChatLoginRequest 是小程序 wx.login 授权请求。
type WeChatLoginRequest struct {
	Code string `json:"code" binding:"required"` // wx.login 的 code
}

// Bind 绑定小程序授权请求。
func (r *WeChatLoginRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// UpdateProfileRequest 是小程序授权后补昵称头像的请求。
type UpdateProfileRequest struct {
	Nickname  string `json:"nickname" binding:"max=40"`    // 昵称
	AvatarURL string `json:"avatar_url" binding:"max=500"` // 头像地址
}

// Bind 绑定资料更新请求。
func (r *UpdateProfileRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}

// CompleteProfileRequest 是补全邮箱密码的请求。
type CompleteProfileRequest struct {
	Email    string `json:"email" binding:"required,email,max=190"`    // 邮箱
	Password string `json:"password" binding:"required,min=8,max=128"` // 密码
}

// Bind 绑定补全资料请求。
func (r *CompleteProfileRequest) Bind(c *gin.Context) error {
	return c.ShouldBindJSON(r)
}
