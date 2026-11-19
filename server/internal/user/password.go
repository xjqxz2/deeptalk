package user

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/crypto/argon2"
)

// Argon2id 参数。
// 说明：64MiB 内存 + 3 次迭代是 OWASP 推荐的交互式登录参数；
// 参数写在哈希串里，未来调整参数仍能校验旧密码。
const (
	argonMemory      = 64 * 1024 // 内存开销，单位 KiB
	argonIterations  = 3         // 迭代次数
	argonParallelism = 2         // 并行度
	argonSaltLength  = 16        // 盐长度（字节）
	argonKeyLength   = 32        // 输出哈希长度（字节）
)

// ErrWeakPassword 表示密码不满足“至少 8 位且包含字母与数字”的规则。
var ErrWeakPassword = errors.New("password must be at least 8 characters and contain letters and digits")

// emailPattern 是邮箱格式校验规则；仅做基本格式校验，不发送验证邮件。
var emailPattern = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// HashPassword 生成 Argon2id 哈希串，格式为：
// $argon2id$v=19$m=65536,t=3,p=2$<base64 salt>$<base64 hash>
// 返回空串与错误表示随机源不可用。
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, argonIterations, argonMemory, argonParallelism, argonKeyLength)
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonIterations, argonParallelism, encodedSalt, encodedHash), nil
}

// VerifyPassword 校验明文密码与哈希串是否匹配。
// 解析失败、算法不匹配或哈希不一致均返回 false，不向调用方泄露具体原因。
func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	// 合法格式由空串 + 6 段组成
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}
	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(want)))
	// 使用常量时间比较，避免通过响应时间推测哈希前缀
	return subtle.ConstantTimeCompare(got, want) == 1
}

// ValidatePassword 校验密码规则：长度 ≥8，且同时包含字母与数字。
// 规则与需求图 REQ-05、设计图 DSG-06 保持一致。
func ValidatePassword(password string) error {
	if len([]rune(password)) < 8 || len(password) > 128 {
		return ErrWeakPassword
	}
	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return ErrWeakPassword
	}
	return nil
}

// NormalizeEmail 统一邮箱格式：去空格、转小写。
// 需求约定邮箱不区分大小写且唯一。
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateEmail 校验邮箱格式。
func ValidateEmail(email string) error {
	if len(email) == 0 || len(email) > 190 || !emailPattern.MatchString(email) {
		return errors.New("invalid email")
	}
	return nil
}

// nicknameFromEmail 用邮箱前缀生成默认昵称，最多保留 12 个字符。
func nicknameFromEmail(email string) string {
	prefix := email
	if idx := strings.Index(email, "@"); idx > 0 {
		prefix = email[:idx]
	}
	runes := []rune(prefix)
	if len(runes) > 12 {
		runes = runes[:12]
	}
	if len(runes) == 0 {
		return "用户"
	}
	return string(runes)
}

// avatarText 取昵称首字符作为头像文字；空昵称回退为 “U”。
func avatarText(nickname string) string {
	runes := []rune(strings.TrimSpace(nickname))
	if len(runes) == 0 {
		return "U"
	}
	return strings.ToUpper(string(runes[0]))
}
