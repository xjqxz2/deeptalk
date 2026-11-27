// Package b 提供 B 端（平台管理后台）接口控制器。
package b

import (
	"deeptalk/apis/middleware"
	"deeptalk/internal/audit"

	"github.com/gin-gonic/gin"
)

// recordAudit 写入后台操作审计。
// 审计失败不阻断主流程，但会静默丢失一条记录，因此只用于可容忍的辅助信息。
func recordAudit(c *gin.Context, recorder *audit.Service, action, targetType string, targetID uint, detail string) {
	current := middleware.CurrentUser(c)
	if recorder == nil || current == nil {
		return
	}
	_ = recorder.Record(c.Request.Context(), audit.Record{
		ActorID:    current.ID,
		ActorName:  current.Nickname,
		ActorEmail: current.Email,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
		IP:         middleware.ClientIP(c),
	})
}
