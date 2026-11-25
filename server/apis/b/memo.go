package b

import (
	"errors"

	"deeptalk/apis/requests"
	"deeptalk/apis/responses"
	"deeptalk/internal/audit"
	"deeptalk/internal/memo"

	"github.com/gin-gonic/gin"
)

// MemoController 处理后台「小橘灵感录」：统计、列表与删除。
type MemoController struct {
	memos  *memo.Service
	audits *audit.Service
}

// NewMemoController 创建灵感录控制器。
func NewMemoController(memos *memo.Service, audits *audit.Service) *MemoController {
	return &MemoController{memos: memos, audits: audits}
}

// Stats 返回总数、今日、近 7 天与每日趋势。
func (h *MemoController) Stats(c *gin.Context) {
	stats, err := h.memos.StatsOf(c.Request.Context())
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, stats)
}

// List 分页列出全部用户的灵感。
func (h *MemoController) List(c *gin.Context) {
	var req requests.AdminMemoListRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "查询参数不合法")
		return
	}
	items, total, err := h.memos.AdminList(c.Request.Context(), req.Query, req.Page, req.Size)
	if err != nil {
		responses.FromError(c, err)
		return
	}
	responses.OK(c, responses.AdminMemoListResponse{
		Items: responses.NewAdminMemos(items),
		Total: total,
		Page:  req.Page,
		Size:  req.Size,
	})
}

// Delete 删除任意一条灵感。
func (h *MemoController) Delete(c *gin.Context) {
	var req requests.MemoIDRequest
	if err := req.Bind(c); err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "灵感 ID 不合法")
		return
	}
	if err := h.memos.AdminDelete(c.Request.Context(), req.ID); err != nil {
		if errors.Is(err, memo.ErrNotFound) {
			responses.Fail(c, responses.CodeNotFound, "灵感不存在")
			return
		}
		responses.FromError(c, err)
		return
	}
	recordAudit(c, h.audits, "memo.delete", "memo", req.ID, "删除灵感")
	responses.OK(c, gin.H{"deleted": true})
}
