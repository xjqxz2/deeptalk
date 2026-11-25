package sshmcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"deeptalk/internal/mcp"
)

// TestKeyStoreGeneratesAndReuses 校验密钥只生成一次，公钥与指纹可重复读取。
func TestKeyStoreGeneratesAndReuses(t *testing.T) {
	dir := t.TempDir()
	store := newKeyStore(dir, 2048)

	first, err := store.ensure("1")
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if !strings.HasPrefix(first.PublicKey, "ssh-rsa ") {
		t.Fatalf("public key = %q", first.PublicKey)
	}
	if !strings.HasPrefix(first.Fingerprint, "SHA256:") {
		t.Fatalf("fingerprint = %q", first.Fingerprint)
	}
	info, err := os.Stat(first.Path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("private key mode = %v", info.Mode().Perm())
	}

	second, err := store.ensure("1")
	if err != nil {
		t.Fatalf("ensure again: %v", err)
	}
	if second.PublicKey != first.PublicKey || second.Fingerprint != first.Fingerprint {
		t.Fatal("key should be reused, not regenerated")
	}

	// 不同 owner 必须是不同密钥，避免跨用户共用身份
	other, err := store.ensure("2")
	if err != nil {
		t.Fatalf("ensure other: %v", err)
	}
	if other.PublicKey == first.PublicKey {
		t.Fatal("different owners must not share a key")
	}
}

// TestSafeOwner 校验 owner 不会造成路径穿越。
func TestSafeOwner(t *testing.T) {
	if got := safeOwner("../../etc"); strings.Contains(got, "/") || strings.Contains(got, "..") {
		t.Fatalf("safeOwner escaped: %q", got)
	}
	if got := safeOwner(""); got != "anonymous" {
		t.Fatalf("empty owner = %q", got)
	}
}

// TestLiveSSH 是真实链路测试：需要一台已配置好公钥的机器。
//
// 运行方式（示例）：
//
//	DEEPTALK_SSH_LIVE=1 DEEPTALK_SSH_BIN=/tmp/deeptalk \
//	DEEPTALK_SSH_DATA=/tmp/ssh-mcp-data DEEPTALK_SSH_HOST=example.com \
//	DEEPTALK_SSH_USER=root DEEPTALK_SSH_WORKDIR=/root/ssh-test \
//	go test ./internal/sshmcp -run Live -v
func TestLiveSSH(t *testing.T) {
	if os.Getenv("DEEPTALK_SSH_LIVE") != "1" {
		t.Skip("set DEEPTALK_SSH_LIVE=1 to run the live SSH test")
	}
	bin := envOr("DEEPTALK_SSH_BIN", "/tmp/deeptalk")
	dataDir := envOr("DEEPTALK_SSH_DATA", "/tmp/ssh-mcp-data")
	host := os.Getenv("DEEPTALK_SSH_HOST")
	user := envOr("DEEPTALK_SSH_USER", "root")
	workdir := os.Getenv("DEEPTALK_SSH_WORKDIR")

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	client, err := mcp.Start(ctx, mcp.Config{
		Command: bin,
		Args:    []string{"ssh-mcp"},
		Env:     map[string]string{"DEEPTALK_DATA_DIR": dataDir},
		Timeout: 60 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("start mcp: %v", err)
	}
	defer client.Close()

	pub, err := client.CallTool(ctx, "ssh_public_key", map[string]any{"owner": "1"})
	if err != nil {
		t.Fatalf("ssh_public_key: %v", err)
	}
	t.Logf("public key: %s", truncate(pub, 200))

	connected, err := client.CallTool(ctx, "ssh_connect", map[string]any{
		"owner": "1", "host": host, "port": 22, "username": user, "workdir": workdir,
	})
	if err != nil {
		t.Fatalf("ssh_connect: %v", err)
	}
	t.Logf("connect: %s", truncate(connected, 300))
	var conn struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(connected), &conn); err != nil || conn.SessionID == "" {
		t.Fatalf("parse session id: %v (%s)", err, connected)
	}

	out, err := client.CallTool(ctx, "ssh_exec", map[string]any{
		"owner": "1", "session_id": conn.SessionID, "command": "pwd && id -un",
	})
	if err != nil {
		t.Fatalf("ssh_exec pwd: %v", err)
	}
	t.Logf("pwd: %s", out)
	if workdir != "" && !strings.Contains(out, workdir) {
		t.Fatalf("expected workdir %q in output: %s", workdir, out)
	}

	write, err := client.CallTool(ctx, "ssh_exec", map[string]any{
		"owner": "1", "session_id": conn.SessionID,
		"command": "echo hello-from-deeptalk > note.txt && ls -l && cat note.txt",
	})
	if err != nil {
		t.Fatalf("ssh_exec write: %v", err)
	}
	t.Logf("write: %s", write)
	if !strings.Contains(write, "hello-from-deeptalk") {
		t.Fatalf("expected written content: %s", write)
	}

	// 非零退出码应原样返回，而不是当作错误
	code, err := client.CallTool(ctx, "ssh_exec", map[string]any{
		"owner": "1", "session_id": conn.SessionID, "command": "exit 3",
	})
	if err != nil {
		t.Fatalf("ssh_exec exit: %v", err)
	}
	t.Logf("exit: %s", code)
	if !strings.Contains(code, `"exit_code":3`) {
		t.Fatalf("expected exit_code 3: %s", code)
	}

	// 跨 owner 不能复用会话
	if _, err := client.CallTool(ctx, "ssh_exec", map[string]any{
		"owner": "2", "session_id": conn.SessionID, "command": "echo nope",
	}); err == nil {
		t.Fatal("expected owner isolation error")
	}

	if _, err := client.CallTool(ctx, "ssh_disconnect", map[string]any{
		"owner": "1", "session_id": conn.SessionID,
	}); err != nil {
		t.Fatalf("ssh_disconnect: %v", err)
	}
}

// envOr 读取环境变量，缺省时返回默认值。
func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// truncate 截断日志输出，避免刷屏。
func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}

// TestSessionDirIsolation 确保 owner 目录互不相同。
func TestSessionDirIsolation(t *testing.T) {
	manager := NewManager(Config{DataDir: filepath.Join(t.TempDir(), "data"), KeyBits: 2048})
	if manager.sessionDir("1") == manager.sessionDir("2") {
		t.Fatal("session dirs must differ per owner")
	}
}
