package requests

import (
	"github.com/gin-gonic/gin"
)

// IDRequest 是通用路径参数：/xxx/:id。
// 控制器统一用它绑定，避免各端各写一套字符串转数字。
type IDRequest struct {
	ID uint `uri:"id" binding:"required,min=1"` // 资源 ID
}

// Bind 绑定路径参数。
func (r *IDRequest) Bind(c *gin.Context) error {
	return c.ShouldBindUri(r)
}
