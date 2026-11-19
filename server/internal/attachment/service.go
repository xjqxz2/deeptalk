package attachment

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	_ "golang.org/x/image/webp"

	"gorm.io/gorm"
)

// 业务错误。
var (
	ErrNotFound     = errors.New("attachment not found")
	ErrTooLarge     = errors.New("attachment too large")
	ErrUnsupported  = errors.New("unsupported image type")
	ErrInvalidImage = errors.New("invalid image")
	ErrTooMany      = errors.New("too many attachments")
)

// 允许的图片类型与落盘扩展名。
var allowedMIME = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// Service 管理附件元数据与文件。
type Service struct {
	db       *gorm.DB
	dir      string
	maxBytes int64
}

// NewService 创建附件服务，目录不存在时自动创建。
func NewService(db *gorm.DB, dir string, maxBytes int64) (*Service, error) {
	if strings.TrimSpace(dir) == "" {
		dir = "data/uploads"
	}
	if maxBytes <= 0 {
		maxBytes = 8 << 20
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, err
	}
	return &Service{db: db, dir: abs, maxBytes: maxBytes}, nil
}

// MaxBytes 返回单张图片的字节上限。
func (s *Service) MaxBytes() int64 { return s.maxBytes }

// Save 校验并保存一张图片，返回可展示的元数据。
func (s *Service) Save(ctx context.Context, userID uint, name string, data []byte) (*Asset, error) {
	if int64(len(data)) == 0 {
		return nil, ErrInvalidImage
	}
	if int64(len(data)) > s.maxBytes {
		return nil, ErrTooLarge
	}
	mime := http.DetectContentType(data)
	ext, ok := allowedMIME[mime]
	if !ok {
		return nil, ErrUnsupported
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	// 过小的图片在部分视觉模型上会被判为无效，也会浪费一次上传
	if err != nil || cfg.Width < 16 || cfg.Height < 16 {
		return nil, ErrInvalidImage
	}
	if !formatAllowed(format) {
		return nil, ErrUnsupported
	}
	storage, err := randomName(ext)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(s.dir, storage)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, err
	}
	row := &Attachment{
		UserID:      userID,
		Name:        safeName(name),
		MIME:        mime,
		Size:        int64(len(data)),
		Width:       cfg.Width,
		Height:      cfg.Height,
		StorageName: storage,
		CreatedAt:   time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	return &Asset{Info: infoOf(row), Path: path}, nil
}

// Load 按 ID 顺序加载当前用户的附件；任意一个不存在都视为非法请求。
func (s *Service) Load(ctx context.Context, userID uint, ids []uint) ([]Asset, error) {
	ids = uniqueIDs(ids)
	if len(ids) == 0 {
		return nil, nil
	}
	if len(ids) > MaxFilesPerMessage {
		return nil, ErrTooMany
	}
	var rows []Attachment
	if err := s.db.WithContext(ctx).Where("user_id = ? AND id IN ?", userID, ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	byID := make(map[uint]Attachment, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	out := make([]Asset, 0, len(ids))
	for _, id := range ids {
		row, ok := byID[id]
		if !ok {
			return nil, ErrNotFound
		}
		out = append(out, Asset{Info: infoOf(&row), Path: filepath.Join(s.dir, row.StorageName)})
	}
	return out, nil
}

// Attach 把本次发送的附件关联到用户消息；重复调用是幂等的。
func (s *Service) Attach(ctx context.Context, userID, messageID uint, ids []uint) error {
	ids = uniqueIDs(ids)
	if len(ids) == 0 {
		return nil
	}
	result := s.db.WithContext(ctx).Model(&Attachment{}).
		Where("user_id = ? AND id IN ?", userID, ids).
		Update("message_id", messageID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != int64(len(ids)) {
		return ErrNotFound
	}
	return nil
}

// Open 返回附件元数据与绝对路径，供 HTTP 层输出图片。
func (s *Service) Open(ctx context.Context, userID, id uint) (*Attachment, string, error) {
	var row Attachment
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	path := filepath.Join(s.dir, row.StorageName)
	if _, err := os.Stat(path); err != nil {
		return nil, "", ErrNotFound
	}
	return &row, path, nil
}

// DataURL 把图片转成 OpenAI 兼容的 data URL，供上游视觉输入使用。
func (a Asset) DataURL() (string, error) {
	data, err := os.ReadFile(a.Path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("data:%s;base64,%s", a.Info.MIME, base64.StdEncoding.EncodeToString(data)), nil
}

func infoOf(row *Attachment) Info {
	return Info{
		ID:     row.ID,
		Name:   row.Name,
		MIME:   row.MIME,
		Size:   row.Size,
		Width:  row.Width,
		Height: row.Height,
		URL:    fmt.Sprintf("/api/v1/c/attachments/%d", row.ID),
	}
}

func uniqueIDs(ids []uint) []uint {
	out := make([]uint, 0, len(ids))
	seen := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func randomName(ext string) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw) + ext, nil
}

func safeName(name string) string {
	name = strings.TrimSpace(filepath.Base(strings.ReplaceAll(name, "\\", "/")))
	name = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' || r == 0 {
			return -1
		}
		return r
	}, name)
	if name == "" || name == "." {
		return "image"
	}
	if utf8.RuneCountInString(name) > 120 {
		name = string([]rune(name)[:120])
	}
	return name
}

func formatAllowed(format string) bool {
	switch format {
	case "png", "jpeg", "gif", "webp":
		return true
	default:
		return false
	}
}
