// Package wechat 封装微信小程序的接口凭据（access_token）管理。
//
// 微信的 access_token 是全局唯一的接口凭据，有效期 7200 秒，且有调用频率限制。
// 这里统一用稳定版接口（stable_token）换取，并缓存在进程内存里：
//   - 启动时取一次，失败只告警不阻断服务
//   - 到期前提前刷新（默认提前 5 分钟）
//   - 业务侧遇到 token 失效（40001/42001 等）时调用 Invalidate，下一次取用会强制刷新
package wechat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// 默认参数。
const (
	stableTokenURL   = "https://api.weixin.qq.com/cgi-bin/stable_token"
	defaultTimeout   = 10 * time.Second
	defaultBuffer    = 5 * time.Minute  // 提前刷新窗口
	defaultStaleLife = 30 * time.Minute // 刷新失败时可继续使用的旧 token 上限
)

// Config 是 access_token 管理器所需配置。
type Config struct {
	AppID     string        // 小程序 AppID
	Secret    string        // 小程序 AppSecret
	Endpoint  string        // 覆盖 stable_token 地址，测试用
	Timeout   time.Duration // 单次请求超时
	Buffer    time.Duration // 提前刷新窗口
	StaleLife time.Duration // 过期后仍可兜底使用的时间

	Code2SessionEndpoint string // 覆盖 code2session 地址，测试用
	MsgSecCheckEndpoint  string // 覆盖内容安全地址，测试用
}

// Manager 负责获取与刷新 access_token。
type Manager struct {
	cfg    Config
	log    *slog.Logger
	client *http.Client
	now    func() time.Time // 便于测试注入

	mu        sync.Mutex
	token     string    // 当前 access_token
	expiresAt time.Time // 过期时间
	refreshAt time.Time // 下次应刷新的时间
	fetching  *call     // 进行中的刷新，合并并发请求
	invalid   bool      // 业务侧报告 token 失效，下次刷新要求微信强制换新

	stopOnce sync.Once
	stop     chan struct{}
	done     chan struct{}
	started  bool
}

// call 表示一次进行中的刷新，其他协程等待它的结果。
type call struct {
	done  chan struct{}
	token string
	err   error
}

// stableTokenRequest 是稳定版接口的请求体。
type stableTokenRequest struct {
	GrantType    string `json:"grant_type"`
	AppID        string `json:"appid"`
	Secret       string `json:"secret"`
	ForceRefresh bool   `json:"force_refresh"`
}

// stableTokenResponse 是稳定版接口的响应体。
type stableTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// ErrDisabled 表示没有配置 AppID/Secret，管理器不参与工作。
var ErrDisabled = errors.New("wechat: credentials are not configured")

// New 创建管理器；AppID 或 Secret 为空时 Enables() 返回 false。
func New(cfg Config, log *slog.Logger) *Manager {
	if cfg.Endpoint == "" {
		cfg.Endpoint = stableTokenURL
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTimeout
	}
	if cfg.Buffer <= 0 {
		cfg.Buffer = defaultBuffer
	}
	if cfg.StaleLife <= 0 {
		cfg.StaleLife = defaultStaleLife
	}
	if log == nil {
		log = slog.Default()
	}
	return &Manager{
		cfg: cfg,
		log: log,
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		now:  time.Now,
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
}

// Enables 表示是否配置了可用凭据。
func (m *Manager) Enables() bool {
	return m != nil && strings.TrimSpace(m.cfg.AppID) != "" && strings.TrimSpace(m.cfg.Secret) != ""
}

// Start 启动后台刷新：先同步取一次 token（失败只告警），再起协程按到期时间刷新。
func (m *Manager) Start(ctx context.Context) {
	if !m.Enables() || m.started {
		return
	}
	m.started = true
	if _, err := m.Token(ctx); err != nil {
		m.log.Warn("wechat access_token 初始获取失败，稍后自动重试", "error", err)
	} else {
		m.log.Info("wechat access_token 已缓存")
	}
	go m.loop()
}

// Stop 停止后台刷新。
func (m *Manager) Stop() {
	if m == nil || !m.started {
		return
	}
	m.stopOnce.Do(func() {
		close(m.stop)
		<-m.done
	})
}

// Token 返回可用的 access_token；临近过期或已过期时自动刷新。
func (m *Manager) Token(ctx context.Context) (string, error) {
	if !m.Enables() {
		return "", ErrDisabled
	}

	m.mu.Lock()
	token, expiresAt, refreshAt, fetching := m.token, m.expiresAt, m.refreshAt, m.fetching
	m.mu.Unlock()

	now := m.now()
	// 缓存可用且还没到刷新时间，直接返回
	if token != "" && now.Before(refreshAt) {
		return token, nil
	}
	// 已经过期太久，不再兜底
	if token != "" && now.After(expiresAt.Add(m.cfg.StaleLife)) {
		token = ""
	}

	// 已有刷新在进行，等它的结果（避免并发打爆接口频率限制）
	if fetching != nil {
		select {
		case <-fetching.done:
			if fetching.err == nil {
				return fetching.token, nil
			}
			if token != "" && now.Before(expiresAt.Add(m.cfg.StaleLife)) {
				return token, nil
			}
			return "", fetching.err
		case <-ctx.Done():
			if token != "" && now.Before(expiresAt.Add(m.cfg.StaleLife)) {
				return token, nil
			}
			return "", ctx.Err()
		}
	}

	// 由当前协程发起刷新
	refreshed, err := m.refresh(ctx, false)
	if err == nil {
		return refreshed, nil
	}
	// 刷新失败但旧 token 还没彻底死，先兜底用
	if token != "" && now.Before(expiresAt.Add(m.cfg.StaleLife)) {
		m.log.Warn("wechat access_token 刷新失败，暂用旧值", "error", err)
		return token, nil
	}
	return "", err
}

// Invalidate 标记当前 token 失效：下一次 Token 调用会强制刷新。
func (m *Manager) Invalidate() {
	if m == nil {
		return
	}
	m.mu.Lock()
	m.refreshAt = time.Time{}
	m.invalid = true
	m.mu.Unlock()
}

// Refresh 立即强制刷新一次。
func (m *Manager) Refresh(ctx context.Context) (string, error) {
	if !m.Enables() {
		return "", ErrDisabled
	}
	return m.refresh(ctx, true)
}

// refresh 发起一次刷新，force 决定是否要求微信强制换新。
func (m *Manager) refresh(ctx context.Context, force bool) (string, error) {
	m.mu.Lock()
	// 业务侧标记过失效时，即使调用方没显式要求，也要让微信换新
	if m.invalid {
		force = true
	}
	if m.fetching != nil {
		waitFor := m.fetching
		m.mu.Unlock()
		select {
		case <-waitFor.done:
			return waitFor.token, waitFor.err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	current := &call{done: make(chan struct{})}
	m.fetching = current
	m.mu.Unlock()

	token, expiresIn, err := m.request(ctx, force)

	m.mu.Lock()
	current.token, current.err = token, err
	if err == nil {
		m.invalid = false
		now := m.now()
		m.token = token
		m.expiresAt = now.Add(time.Duration(expiresIn) * time.Second)
		m.refreshAt = m.expiresAt.Add(-m.cfg.Buffer)
		if m.refreshAt.Before(now) {
			// 有效期比刷新窗口还短时，按一半有效期刷新，避免空转
			m.refreshAt = now.Add(time.Duration(expiresIn) * time.Second / 2)
		}
	}
	m.fetching = nil
	close(current.done)
	m.mu.Unlock()

	return token, err
}

// request 调用稳定版接口换取 token。
func (m *Manager) request(ctx context.Context, force bool) (string, int, error) {
	payload, err := json.Marshal(stableTokenRequest{
		GrantType:    "client_credential",
		AppID:        m.cfg.AppID,
		Secret:       m.cfg.Secret,
		ForceRefresh: force,
	})
	if err != nil {
		return "", 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.cfg.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("wechat: request stable_token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return "", 0, fmt.Errorf("wechat: read stable_token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("wechat: stable_token http %d", resp.StatusCode)
	}

	var parsed stableTokenResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", 0, fmt.Errorf("wechat: decode stable_token response: %w", err)
	}
	if parsed.ErrCode != 0 {
		return "", 0, fmt.Errorf("wechat: stable_token errcode=%d errmsg=%s", parsed.ErrCode, parsed.ErrMsg)
	}
	if parsed.AccessToken == "" {
		return "", 0, errors.New("wechat: stable_token returned empty token")
	}
	expiresIn := parsed.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 7200
	}
	return parsed.AccessToken, expiresIn, nil
}

// loop 按刷新时间自动续期，直到 Stop。
func (m *Manager) loop() {
	defer close(m.done)
	for {
		m.mu.Lock()
		next := m.refreshAt
		m.mu.Unlock()

		wait := time.Until(next)
		if wait < time.Second {
			wait = time.Second
		}
		timer := time.NewTimer(wait)
		select {
		case <-m.stop:
			timer.Stop()
			return
		case <-timer.C:
		}

		ctx, cancel := context.WithTimeout(context.Background(), m.cfg.Timeout)
		if _, err := m.refresh(ctx, false); err != nil {
			m.log.Warn("wechat access_token 自动刷新失败，稍后重试", "error", err)
			// 失败后一分钟重试，避免立刻打爆频率限制
			m.mu.Lock()
			m.refreshAt = time.Now().Add(time.Minute)
			m.mu.Unlock()
		}
		cancel()
	}
}
