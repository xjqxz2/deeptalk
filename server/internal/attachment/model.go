// Package attachment 负责会话图片附件的落盘、归属校验与读取。
// 只接受常见位图格式；文件名与存储名分离，存储名由服务端随机生成。
package attachment

import (
	"time"

	"gorm.io/gorm"
)

// MaxFilesPerMessage 限制单条消息的图片数量，避免上游上下文与请求体过大。
const MaxFilesPerMessage = 4

// Attachment 是用户上传的一张图片。
type Attachment struct {
	ID          uint      `gorm:"primaryKey"`                    // 主键
	UserID      uint      `gorm:"index;not null"`                // 上传者
	MessageID   uint      `gorm:"index;not null;default:0"`      // 关联消息；0 表示尚未发送
	Name        string    `gorm:"size:190;not null"`             // 原始文件名（仅展示）
	MIME        string    `gorm:"size:64;not null"`              // 服务端探测的 MIME
	Size        int64     `gorm:"not null;default:0"`            // 字节数
	Width       int       `gorm:"not null;default:0"`            // 像素宽
	Height      int       `gorm:"not null;default:0"`            // 像素高
	StorageName string    `gorm:"size:128;not null;uniqueIndex"` // 落盘文件名，不含路径
	CreatedAt   time.Time `gorm:"index"`                         // 上传时间
}

// Info 是返回给前端的附件元数据。
type Info struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	MIME   string `json:"mime"`
	Size   int64  `json:"size"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URL    string `json:"url"`
}

// Asset 是服务端内部使用的附件（带绝对路径，不直接序列化）。
type Asset struct {
	Info Info
	Path string
}

// Migrate 创建附件表。
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Attachment{})
}
