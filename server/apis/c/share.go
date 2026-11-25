package c

import (
	"errors"

	"deeptalk/apis/middleware"
	"deeptalk/apis/responses"
	"deeptalk/internal/chat"

	"github.com/gin-gonic/gin"
)

// GetShare 返回当前会话的分享链接；没有分享时返回空对象。
func (h *ChatController) GetShare(c *gin.Context) {
	current := middleware.CurrentUser(c)
	id, err := h.chats.ResolveRef(c.Request.Context(), current.ID, c.Param("id"))
	if err != nil {
		responses.FromError(c, err)
		return
	}
	share, err := h.chats.CurrentShare(c.Request.Context(), current.ID, id)
	if err != nil {
		if errors.Is(err, chat.ErrShareNotFound) {
			responses.OK(c, gin.H{"share": nil})
			return
		}
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{"share": responses.NewShare(share)})
}

// CreateShare 创建（或复用）分享链接。
func (h *ChatController) CreateShare(c *gin.Context) {
	current := middleware.CurrentUser(c)
	id, err := h.chats.ResolveRef(c.Request.Context(), current.ID, c.Param("id"))
	if err != nil {
		responses.FromError(c, err)
		return
	}
	share, err := h.chats.CreateShare(c.Request.Context(), current.ID, id)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{"share": responses.NewShare(share)})
}

// RevokeShare 撤销分享，旧链接立即失效。
func (h *ChatController) RevokeShare(c *gin.Context) {
	current := middleware.CurrentUser(c)
	id, err := h.chats.ResolveRef(c.Request.Context(), current.ID, c.Param("id"))
	if err != nil {
		responses.FromError(c, err)
		return
	}
	if err := h.chats.RevokeShare(c.Request.Context(), current.ID, id); err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, nil)
}

// PublicShare 是无需登录的只读分享页数据。
// 只返回消息，不提供任何写入能力；访客不能继续对话。
func (h *ChatController) PublicShare(c *gin.Context) {
	shared, err := h.chats.GetShared(c.Request.Context(), c.Param("token"))
	if err != nil {
		if errors.Is(err, chat.ErrShareNotFound) {
			responses.Fail(c, responses.CodeNotFound, "分享不存在或已失效")
			return
		}
		responses.FromError(c, err)
		return
	}
	responses.OK(c, responses.NewSharedConversation(*shared))
}
