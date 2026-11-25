package c

import (
	"errors"

	"deeptalk/apis/middleware"
	"deeptalk/apis/requests"
	"deeptalk/apis/responses"
	"deeptalk/internal/memo"

	"github.com/gin-gonic/gin"
)

// MemoController 提供 C 端小橘灵感录接口。
type MemoController struct {
	memos *memo.Service
}

// NewMemoController 创建灵感录控制器。
func NewMemoController(memos *memo.Service) *MemoController {
	return &MemoController{memos: memos}
}

// List 返回当前用户的灵感录；带 q 时按关键词检索。
func (h *MemoController) List(c *gin.Context) {
	var req requests.MemoListRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "查询参数不合法")
		return
	}
	items, err := h.memos.Search(c.Request.Context(), middleware.CurrentUser(c).ID, req.Query, req.Limit)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, responses.MemoListResponse{Items: responses.NewMemos(items)})
}

// Create 新建一条灵感录。
func (h *MemoController) Create(c *gin.Context) {
	var req requests.MemoCreateRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请求内容不合法")
		return
	}
	item, err := h.memos.Save(c.Request.Context(), middleware.CurrentUser(c).ID, memo.SaveInput{
		Title:          req.Title,
		Content:        req.Content,
		Tags:           req.Tags,
		ConversationID: req.ConversationID,
	})
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, gin.H{"memo": responses.NewMemo(item)})
}

// Update 修改一条灵感录。
func (h *MemoController) Update(c *gin.Context) {
	var idReq requests.MemoIDRequest
	if err := idReq.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "灵感录 ID 不合法")
		return
	}
	var req requests.MemoUpdateRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请求内容不合法")
		return
	}
	item, err := h.memos.Update(c.Request.Context(), middleware.CurrentUser(c).ID, idReq.ID, memo.SaveInput{
		Title:   req.Title,
		Content: req.Content,
		Tags:    req.Tags,
	})
	if err != nil {
		writeMemoError(c, err)
		return
	}
	responses.OK(c, gin.H{"memo": responses.NewMemo(item)})
}

// Delete 删除一条灵感录。
func (h *MemoController) Delete(c *gin.Context) {
	var req requests.MemoIDRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "灵感录 ID 不合法")
		return
	}
	if err := h.memos.Delete(c.Request.Context(), middleware.CurrentUser(c).ID, req.ID); err != nil {
		writeMemoError(c, err)
		return
	}
	responses.OK(c, gin.H{"deleted": true})
}

// writeMemoError 把灵感录领域错误翻译成响应码。
func writeMemoError(c *gin.Context, err error) {
	if errors.Is(err, memo.ErrNotFound) {
		responses.Fail(c, responses.CodeNotFound, "灵感录不存在")
		return
	}
	responses.FromError(c, err)
}
