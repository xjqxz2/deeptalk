package c

import (
	"errors"
	"io"
	"mime"
	"net/http"

	"deeptalk/apis/middleware"
	"deeptalk/apis/responses"
	"deeptalk/internal/attachment"

	"github.com/gin-gonic/gin"
)

// AttachmentController 提供图片上传与读取。
type AttachmentController struct {
	attachments *attachment.Service
}

// NewAttachmentController 创建附件控制器。
func NewAttachmentController(attachments *attachment.Service) *AttachmentController {
	return &AttachmentController{attachments: attachments}
}

// Upload 接收 multipart 的 file 字段，保存后返回附件元数据。
func (h *AttachmentController) Upload(c *gin.Context) {
	current := middleware.CurrentUser(c)
	if h.attachments == nil {
		responses.Fail(c, responses.CodeInternal, "附件服务未启用")
		return
	}
	// 多留 1MB 给 multipart 边界与表单字段，真正的图片大小由 service 再校验
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.attachments.MaxBytes()+1<<20)
	file, err := c.FormFile("file")
	if err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "请选择要上传的图片")
		return
	}
	opened, err := file.Open()
	if err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "无法读取上传文件")
		return
	}
	defer opened.Close()
	data, err := io.ReadAll(io.LimitReader(opened, h.attachments.MaxBytes()+1))
	if err != nil {
		responses.Fail(c, responses.CodeInvalidArgument, "读取上传文件失败")
		return
	}
	asset, err := h.attachments.Save(c.Request.Context(), current.ID, file.Filename, data)
	if err != nil {
		switch {
		case errors.Is(err, attachment.ErrTooLarge):
			responses.Fail(c, responses.CodeInvalidArgument, "图片过大，请压缩后重试")
		case errors.Is(err, attachment.ErrUnsupported):
			responses.Fail(c, responses.CodeInvalidArgument, "仅支持 PNG / JPEG / GIF / WebP 图片")
		case errors.Is(err, attachment.ErrInvalidImage):
			responses.Fail(c, responses.CodeInvalidArgument, "图片内容无法识别")
		default:
			_ = c.Error(err)
			responses.Fail(c, responses.CodeInternal, "图片保存失败，请稍后重试")
		}
		return
	}
	responses.OK(c, gin.H{"attachment": asset.Info})
}

// Get 输出当前用户自己的图片。
func (h *AttachmentController) Get(c *gin.Context) {
	current := middleware.CurrentUser(c)
	id, ok := parseID(c.Param("id"))
	if !ok {
		responses.Fail(c, responses.CodeInvalidArgument, "附件 ID 不合法")
		return
	}
	row, path, err := h.attachments.Open(c.Request.Context(), current.ID, id)
	if err != nil {
		if errors.Is(err, attachment.ErrNotFound) {
			responses.Fail(c, responses.CodeNotFound, "附件不存在")
			return
		}
		_ = c.Error(err)
		responses.Fail(c, responses.CodeInternal, "附件读取失败")
		return
	}
	c.Header("Content-Type", row.MIME)
	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": row.Name}))
	c.File(path)
}
