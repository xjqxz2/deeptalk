// Package security 提供请求签名能力：防重放、防篡改、防无凭证伪造。
//
// 方案（B）：每个会话在登录时分配一个 key id，真正的签名密钥由服务端主密钥
// 派生（HMAC-SHA256(secret, "sign:"+keyID)），数据库里只存 key id，即使库被
// 拖走也拿不到可用的签名密钥。
//
// 客户端对写请求计算：
//
//	HMAC-SHA256(key, METHOD\nPATH\nSORTED_QUERY\nSHA256(BODY)\nTS\nNONCE)
//
// 服务端校验时间窗、nonce 未使用、签名一致。
package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// 签名相关的请求头。
const (
	HeaderTimestamp = "X-DT-Ts"
	HeaderNonce     = "X-DT-Nonce"
	HeaderKeyID     = "X-DT-Key"
	HeaderSignature = "X-DT-Sign"
)

// 校验结果。
var (
	ErrMissing      = errors.New("signature required")
	ErrBadFormat    = errors.New("signature malformed")
	ErrExpired      = errors.New("signature timestamp out of window")
	ErrReplay       = errors.New("signature nonce reused")
	ErrMismatch     = errors.New("signature mismatch")
	ErrUnsupported  = errors.New("signature not supported for this request")
	DefaultSkew     = 120 * time.Second
	DefaultNonceTTL = 5 * time.Minute
)

// Verifier 校验请求签名。
type Verifier struct {
	secret   []byte
	skew     time.Duration
	now      func() time.Time
	nonces   *nonceStore
	disabled bool
}

// NewVerifier 创建校验器；secret 为空表示不校验。
func NewVerifier(secret string) *Verifier {
	return &Verifier{
		secret: []byte(secret),
		skew:   DefaultSkew,
		now:    time.Now,
		nonces: newNonceStore(DefaultNonceTTL),
	}
}

// Disabled 表示未配置主密钥，直接放行。
func (v *Verifier) Disabled() bool { return v == nil || len(v.secret) == 0 }

// NewKeyID 生成会话签名密钥标识。
func NewKeyID() string {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(raw)
}

// DeriveKey 由会话 key id 派生签名密钥。
func (v *Verifier) DeriveKey(keyID string) []byte {
	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte("sign:" + keyID))
	return mac.Sum(nil)
}

// Sign 计算签名，供测试与客户端参考实现使用。
func (v *Verifier) Sign(keyID, method, path, rawQuery, body string, ts int64, nonce string) string {
	mac := hmac.New(sha256.New, v.DeriveKey(keyID))
	mac.Write([]byte(Canonical(method, path, rawQuery, body, ts, nonce)))
	return hex.EncodeToString(mac.Sum(nil))
}

// Canonical 生成签名原文：方法、路径、排序后的查询串、请求体摘要、时间戳、随机串。
func Canonical(method, path, rawQuery, body string, ts int64, nonce string) string {
	bodySum := sha256.Sum256([]byte(body))
	return strings.Join([]string{
		strings.ToUpper(method),
		path,
		sortedQuery(rawQuery),
		hex.EncodeToString(bodySum[:]),
		strconv.FormatInt(ts, 10),
		nonce,
	}, "\n")
}

// sortedQuery 把 query 参数按名字排序后拼回，避免客户端顺序差异。
func sortedQuery(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		items := append([]string(nil), values[key]...)
		sort.Strings(items)
		for _, item := range items {
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(item))
		}
	}
	return strings.Join(parts, "&")
}

// Verify 校验一次签名；keyID 来自会话记录。
func (v *Verifier) Verify(keyID, method, path, rawQuery string, body []byte, tsRaw, nonce, signature string) error {
	if v.Disabled() {
		return nil
	}
	if tsRaw == "" || nonce == "" || signature == "" {
		return ErrMissing
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(tsRaw), 10, 64)
	if err != nil {
		return ErrBadFormat
	}
	now := v.now().Unix()
	skew := int64(v.skew / time.Second)
	if ts < now-skew || ts > now+skew {
		return ErrExpired
	}
	if len(nonce) < 16 || len(nonce) > 64 {
		return ErrBadFormat
	}
	expected := v.Sign(keyID, method, path, rawQuery, string(body), ts, nonce)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(strings.ToLower(strings.TrimSpace(signature)))) != 1 {
		return ErrMismatch
	}
	// 签名正确后再占用 nonce，避免无效请求把缓存打满
	if !v.nonces.Remember(keyID + ":" + nonce) {
		return ErrReplay
	}
	return nil
}

// nonceStore 记录已使用的 nonce，TTL 后自动过期。
type nonceStore struct {
	mu   sync.Mutex
	ttl  time.Duration
	seen map[string]time.Time
	last time.Time
}

func newNonceStore(ttl time.Duration) *nonceStore {
	return &nonceStore{ttl: ttl, seen: make(map[string]time.Time)}
}

// Remember 记录 nonce；已存在返回 false。
func (s *nonceStore) Remember(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if now.Sub(s.last) > time.Minute {
		for item, expiry := range s.seen {
			if now.After(expiry) {
				delete(s.seen, item)
			}
		}
		s.last = now
	}
	if expiry, ok := s.seen[key]; ok && now.Before(expiry) {
		return false
	}
	s.seen[key] = now.Add(s.ttl)
	return true
}

// GenerateNonce 生成客户端用的随机串（调试与文档示例）。
func GenerateNonce() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(raw)
}
