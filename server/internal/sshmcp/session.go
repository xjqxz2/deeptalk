package sshmcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// DefaultDialTimeout 是建立 SSH 连接的上限；连不上要尽快失败，别让模型干等。
const DefaultDialTimeout = 15 * time.Second

// Session 是一条已建立的 SSH 连接，绑定 owner，跨请求复用。
type Session struct {
	ID      string    // 会话 ID，模型用它执行后续命令
	Owner   string    // 归属 owner，跨用户不可见
	Host    string    // 主机
	Port    int       // 端口
	User    string    // 登录用户
	Workdir string    // 会话固定工作目录（可选）
	Created time.Time // 建立时间

	client *ssh.Client
}

// Manager 管理密钥、known_hosts 与活跃会话。
type Manager struct {
	cfg  Config
	keys *keyStore

	mu       sync.Mutex
	sessions map[string]*Session
}

// Config 是 SSH MCP 的运行参数。
type Config struct {
	DataDir        string        // 数据根目录
	KeyBits        int           // RSA 位数
	DefaultTimeout time.Duration // 单条命令默认超时
	MaxTimeout     time.Duration // 单条命令最大超时
	MaxOutputBytes int           // 输出截断阈值（stdout/stderr 各自）
}

// Normalize 补齐默认值。
func (c Config) Normalize() Config {
	if strings.TrimSpace(c.DataDir) == "" {
		c.DataDir = "data"
	}
	if c.KeyBits <= 0 {
		c.KeyBits = 4096
	}
	if c.DefaultTimeout <= 0 {
		c.DefaultTimeout = 60 * time.Second
	}
	if c.MaxTimeout <= 0 {
		c.MaxTimeout = 300 * time.Second
	}
	if c.MaxOutputBytes <= 0 {
		c.MaxOutputBytes = 200 * 1024
	}
	return c
}

// NewManager 创建 manager。
func NewManager(cfg Config) *Manager {
	cfg = cfg.Normalize()
	return &Manager{
		cfg:      cfg,
		keys:     newKeyStore(filepath.Join(cfg.DataDir, "ssh"), cfg.KeyBits),
		sessions: make(map[string]*Session),
	}
}

// PublicKey 返回 owner 的公钥与指纹。
func (m *Manager) PublicKey(owner string) (*KeyPair, error) {
	return m.keys.ensure(owner)
}

// sessionDir 返回 owner 的目录（放 known_hosts 等）。
func (m *Manager) sessionDir(owner string) string {
	return m.keys.path(owner)
}

// Connect 建立 SSH 连接，首次连接按 TOFU 记录主机指纹。
func (m *Manager) Connect(ctx context.Context, owner, host string, port int, user, workdir string) (*Session, error) {
	host = strings.TrimSpace(host)
	user = strings.TrimSpace(user)
	if host == "" || user == "" {
		return nil, errors.New("host 和 username 不能为空")
	}
	if port <= 0 || port > 65535 {
		port = 22
	}

	pair, err := m.keys.ensure(owner)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(pair.Path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	hostKeyCB, err := m.hostKeyCallback(owner)
	if err != nil {
		return nil, err
	}

	dialCtx, cancel := context.WithTimeout(ctx, DefaultDialTimeout)
	defer cancel()
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	var dialer net.Dialer
	conn, err := dialer.DialContext(dialCtx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("连接 %s 失败：%w", addr, err)
	}
	// 握手也要有超时，避免半开连接一直挂着
	_ = conn.SetDeadline(time.Now().Add(DefaultDialTimeout))
	clientConn, chans, reqs, err := ssh.NewClientConn(conn, addr, &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCB,
		Timeout:         DefaultDialTimeout,
	})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("SSH 握手失败：%w", err)
	}
	_ = conn.SetDeadline(time.Time{})
	client := ssh.NewClient(clientConn, chans, reqs)

	// 固定工作目录：进入失败直接报错，避免后续命令跑在错误目录里
	if workdir != "" {
		out, probeErr := runOnce(client, fmt.Sprintf("cd %s && pwd", shellQuote(workdir)))
		if probeErr != nil || strings.TrimSpace(out) == "" {
			_ = client.Close()
			return nil, fmt.Errorf("无法进入工作目录 %s：%v", workdir, probeErr)
		}
		workdir = strings.TrimSpace(out)
	}

	id := randomID()
	session := &Session{
		ID:      id,
		Owner:   owner,
		Host:    host,
		Port:    port,
		User:    user,
		Workdir: workdir,
		Created: time.Now(),
		client:  client,
	}
	m.mu.Lock()
	m.sessions[id] = session
	m.mu.Unlock()
	return session, nil
}

// Exec 在指定会话上执行命令，返回 stdout / stderr / 退出码。
func (m *Manager) Exec(ctx context.Context, owner, sessionID, command string, timeout time.Duration) (Result, error) {
	session, err := m.get(owner, sessionID)
	if err != nil {
		return Result{}, err
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return Result{}, errors.New("command 不能为空")
	}
	if timeout <= 0 {
		timeout = m.cfg.DefaultTimeout
	}
	if timeout > m.cfg.MaxTimeout {
		timeout = m.cfg.MaxTimeout
	}
	if session.Workdir != "" {
		command = fmt.Sprintf("cd %s && %s", shellQuote(session.Workdir), command)
	}

	done := make(chan struct{})
	var out Result
	var runErr error
	go func() {
		defer close(done)
		out, runErr = runCommand(session.client, command, m.cfg.MaxOutputBytes)
	}()
	select {
	case <-ctx.Done():
		_ = session.client.Close()
		m.remove(session.ID)
		return Result{}, ctx.Err()
	case <-time.After(timeout):
		// 超时后主动断开：SSH 没有可移植的「只杀掉远端命令」的手段
		_ = session.client.Close()
		m.remove(session.ID)
		return Result{}, fmt.Errorf("命令超过 %s 未返回，已断开该会话", timeout)
	case <-done:
	}
	if runErr != nil {
		var exitErr *ssh.ExitError
		if errors.As(runErr, &exitErr) {
			out.ExitCode = exitErr.ExitStatus()
			return out, nil
		}
		return out, runErr
	}
	return out, nil
}

// Disconnect 关闭并移除会话，返回被关闭会话的快照（用于生成清理提醒）。
func (m *Manager) Disconnect(owner, sessionID string) (Session, error) {
	session, err := m.get(owner, sessionID)
	if err != nil {
		return Session{}, err
	}
	_ = session.client.Close()
	m.remove(sessionID)
	return *session, nil
}

// Sessions 列出该 owner 的会话。
func (m *Manager) Sessions(owner string) []Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := make([]Session, 0, len(m.sessions))
	for _, item := range m.sessions {
		if item.Owner == owner {
			list = append(list, *item)
		}
	}
	return list
}

// Close 关闭全部会话。
func (m *Manager) Close() {
	m.mu.Lock()
	list := make([]*Session, 0, len(m.sessions))
	for _, item := range m.sessions {
		list = append(list, item)
	}
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()
	for _, item := range list {
		_ = item.client.Close()
	}
}

// get 取出会话并校验归属：A 用户不能操作 B 用户的会话。
func (m *Manager) get(owner, sessionID string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[sessionID]
	if !ok || session.Owner != owner {
		return nil, fmt.Errorf("会话 %s 不存在或无权访问，请重新 ssh_connect", sessionID)
	}
	return session, nil
}

// remove 删除会话记录。
func (m *Manager) remove(sessionID string) {
	m.mu.Lock()
	delete(m.sessions, sessionID)
	m.mu.Unlock()
}

// hostKeyCallback 返回带 TOFU 的主机指纹校验。
func (m *Manager) hostKeyCallback(owner string) (ssh.HostKeyCallback, error) {
	dir := m.sessionDir(owner)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create known_hosts dir: %w", err)
	}
	path := filepath.Join(dir, "known_hosts")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if writeErr := os.WriteFile(path, nil, 0o600); writeErr != nil {
			return nil, fmt.Errorf("create known_hosts: %w", writeErr)
		}
	}
	base, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("load known_hosts: %w", err)
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		verifyErr := base(hostname, remote, key)
		if verifyErr == nil {
			return nil
		}
		var keyErr *knownhosts.KeyError
		if errors.As(verifyErr, &keyErr) && len(keyErr.Want) == 0 {
			// 首次见到这台机器：记录指纹后放行（TOFU）
			line := knownhosts.Line([]string{hostname}, key)
			file, openErr := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
			if openErr != nil {
				return fmt.Errorf("记录主机指纹失败：%w", openErr)
			}
			defer file.Close()
			if _, writeErr := file.WriteString(line + "\n"); writeErr != nil {
				return fmt.Errorf("记录主机指纹失败：%w", writeErr)
			}
			return nil
		}
		return fmt.Errorf("主机指纹与已记录的不一致，可能存在中间人攻击：%w", verifyErr)
	}, nil
}

// Result 是一条命令的执行结果。
type Result struct {
	Stdout    string `json:"stdout"`
	Stderr    string `json:"stderr"`
	ExitCode  int    `json:"exit_code"`
	Truncated bool   `json:"truncated"`
}

// runOnce 执行一次命令并返回 stdout（用于探测工作目录）。
func runOnce(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	out, err := session.Output(command)
	return string(out), err
}

// runCommand 在客户端上执行命令，分别收集 stdout/stderr。
func runCommand(client *ssh.Client, command string, limit int) (Result, error) {
	session, err := client.NewSession()
	if err != nil {
		return Result{}, err
	}
	defer session.Close()
	stdout := &limitedBuffer{limit: limit}
	stderr := &limitedBuffer{limit: limit}
	session.Stdout = stdout
	session.Stderr = stderr
	err = session.Run(command)
	return Result{
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		Truncated: stdout.truncated || stderr.truncated,
	}, err
}

// limitedBuffer 是有上限的输出缓冲，超出后丢弃并打标记。
type limitedBuffer struct {
	limit     int
	buf       strings.Builder
	truncated bool
}

// Write 实现 io.Writer。
func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > b.limit {
		remain := b.limit - b.buf.Len()
		if remain > 0 {
			b.buf.Write(p[:remain])
		}
		b.truncated = true
		return len(p), nil
	}
	b.buf.Write(p)
	return len(p), nil
}

// String 返回已收集内容。
func (b *limitedBuffer) String() string { return b.buf.String() }

// shellQuote 把参数包成安全的一层 shell 引号。
func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

// randomID 生成 16 字节十六进制会话 ID。
func randomID() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
