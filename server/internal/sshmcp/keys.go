package sshmcp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

// KeyPair 是一个 owner 的 SSH 身份。
type KeyPair struct {
	Path        string // 私钥路径
	PublicKey   string // authorized_keys 行（ssh-rsa AAAA… 注释）
	Fingerprint string // SHA256 指纹
}

// keyStore 按 owner 管理密钥：同一 owner 只生成一次，之后直接复用。
type keyStore struct {
	dir  string // 根目录，通常是 <data>/ssh
	bits int    // RSA 位数
	mu   sync.Mutex
}

// newKeyStore 创建密钥仓库。
func newKeyStore(dir string, bits int) *keyStore {
	if bits < 2048 {
		bits = 4096
	}
	return &keyStore{dir: dir, bits: bits}
}

// ensure 返回 owner 的密钥；不存在时生成 RSA 私钥与 .pub。
func (s *keyStore) ensure(owner string) (*KeyPair, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.path(owner)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create key dir: %w", err)
	}
	keyPath := filepath.Join(dir, "id_rsa")
	pubPath := keyPath + ".pub"

	// 已存在：优先读 .pub，缺失时从私钥推导，保证两者永远一致
	if raw, err := os.ReadFile(keyPath); err == nil {
		signer, parseErr := ssh.ParsePrivateKey(raw)
		if parseErr != nil {
			return nil, fmt.Errorf("parse private key: %w", parseErr)
		}
		pair := &KeyPair{
			Path:        keyPath,
			PublicKey:   strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey()))),
			Fingerprint: ssh.FingerprintSHA256(signer.PublicKey()),
		}
		if _, statErr := os.Stat(pubPath); statErr != nil {
			if writeErr := os.WriteFile(pubPath, []byte(pair.PublicKey), 0o644); writeErr != nil {
				return nil, fmt.Errorf("write public key: %w", writeErr)
			}
		}
		return pair, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	// 生成新密钥：PEM(PKCS#1) 是 OpenSSH 普遍支持的格式
	key, err := rsa.GenerateKey(rand.Reader, s.bits)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(block), 0o600); err != nil {
		return nil, fmt.Errorf("write private key: %w", err)
	}
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		return nil, fmt.Errorf("build signer: %w", err)
	}
	// 统一不带注释：复用分支从私钥推导出的内容必须与这里完全一致，
	// 否则同一把密钥两次返回的字符串不同，用户会以为换了密钥。
	pubLine := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
	if err := os.WriteFile(pubPath, []byte(pubLine+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("write public key: %w", err)
	}
	return &KeyPair{
		Path:        keyPath,
		PublicKey:   pubLine,
		Fingerprint: ssh.FingerprintSHA256(signer.PublicKey()),
	}, nil
}

// path 返回 owner 的目录。
func (s *keyStore) path(owner string) string {
	return filepath.Join(s.dir, safeOwner(owner))
}

// safeOwner 把 owner 收敛成安全的目录名，避免路径穿越。
func safeOwner(owner string) string {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return "anonymous"
	}
	var b strings.Builder
	for _, r := range owner {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
