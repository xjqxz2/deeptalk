package c

import (
	"errors"

	"deeptalk/apis/middleware"
	"deeptalk/apis/responses"
	"deeptalk/internal/memo"

	"github.com/gin-gonic/gin"
)

// MemoController 提供 C 端灵感备忘录接口。
type MemoController struct {
	memos *memo.Service
}

// NewMemoController 创建备忘录控制器。
func NewMemoController(memos *memo.Service) *MemoController {
	return &MemoController{memos: memos}
}

// List 返回当前用户的备忘录；带 q 时按关键词检索。
func (h *MemoController) List(c *gin.Context) {
	user := middleware.CurrentUser(c).ID
	limit := 20
	items, err := h.memos.Search(c.Request.Context(), user, c.Query("q"), limit)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{"items": items})
}

// Create 新建一条备忘录。
func (h *MemoController) Create(c *gin.Context) {
	var req memo.SaveInput
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请求内容不合法")
		return
	}
	item, err := h.memos.Save(c.Request.Context(), middleware.CurrentUser(c).ID, req)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{"memo": item})
}

// Update 修改一条备忘录。
func (h *MemoController) Update(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		responses.Fail(c, responses.CodeInvalidArgument, "备忘录 ID 不合法")
		return
	}
	var req memo.SaveInput
	if err := c.ShouldBindJSON(&req); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请求内容不合法")
		return
	}
	item, err := h.memos.Update(c.Request.Context(), middleware.CurrentUser(c).ID, id, req)
	if err != nil {
		writeMemoError(c, err)
		return
	}
	responses.OK(c, gin.H{"memo": item})
}

// Delete 删除一条备忘录。
func (h *MemoController) Delete(c *gin.Context) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		responses.Fail(c, responses.CodeInvalidArgument, "备忘录 ID 不合法")
		return
	}
	if err := h.memos.Delete(c.Request.Context(), middleware.CurrentUser(c).ID, id); err != nil {
		writeMemoError(c, err)
		return
	}
	responses.OK(c, gin.H{"deleted": true})
}

func writeMemoError(c *gin.Context, err error) {
	if errors.Is(err, memo.ErrNotFound) {
		responses.Fail(c, responses.CodeNotFound, "备忘录不存在")
		return
	}
	responses.FromError(c, err)
}
