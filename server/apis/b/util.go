package b

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
)

// parseID 解析路径中的正整数主键。
func parseID(raw string) (uint, bool) {
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || n == 0 {
		return 0, false
	}
	return uint(n), true
}

// parseIDParam 解析 :id 路径参数。
func parseIDParam(c *gin.Context) (uint, error) {
	id, ok := parseID(c.Param("id"))
	if !ok {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

// parsePositive 解析正整数查询参数，非法时返回错误由调用方回退默认值。
func parsePositive(raw string) (int, error) {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, strconv.ErrSyntax
	}
	return n, nil
}
