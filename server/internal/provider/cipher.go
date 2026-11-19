package provider

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

// deriveKey 将任意长度的配置密钥派生为 32 字节 AES-256 密钥。
// 说明：使用 SHA-256 而不是直接截断，避免短密钥导致安全性下降。
func deriveKey(secret string) []byte {
	sum := sha256.Sum256([]byte(secret))
	return sum[:]
}

// EncryptAPIKey 使用 AES-GCM 加密上游 Key。
// 返回 base64(nonce + ciphertext)；同一明文每次结果不同，避免密文比对。
func EncryptAPIKey(secret, plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	block, err := aes.NewCipher(deriveKey(secret))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plain), nil)
	raw := append(nonce, ciphertext...)
	return base64.RawStdEncoding.EncodeToString(raw), nil
}

// DecryptAPIKey 解密上游 Key；密钥错误或密文损坏时返回错误。
func DecryptAPIKey(secret, encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(deriveKey(secret))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// MaskAPIKey 生成前端可见的 Key 掩码：保留前 3 位与后 4 位。
// 过短的 Key 只显示固定掩码，避免泄露有效位数。
func MaskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	runes := []rune(key)
	if len(runes) <= 10 {
		return "••••••••"
	}
	return string(runes[:3]) + "••••••••" + string(runes[len(runes)-4:])
}
