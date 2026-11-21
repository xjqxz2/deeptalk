package wechat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCode2Session 校验 code 换 openid 的解析与错误分支。
func TestCode2Session(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("js_code") != "good-code" {
			_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 40029, "errmsg": "invalid code"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"openid": "openid-1", "session_key": "sk", "unionid": "union-1",
		})
	}))
	defer server.Close()

	manager := New(Config{AppID: "a", Secret: "b", Code2SessionEndpoint: server.URL}, nil)
	session, err := manager.Code2Session(context.Background(), "good-code")
	if err != nil {
		t.Fatalf("Code2Session: %v", err)
	}
	if session.OpenID != "openid-1" || session.UnionID != "union-1" {
		t.Fatalf("session = %+v", session)
	}
	if _, err := manager.Code2Session(context.Background(), "bad"); err == nil {
		t.Fatal("expected error for invalid code")
	}
}

// TestMsgSecCheckPass 校验正常文本返回 pass。
func TestMsgSecCheckPass(t *testing.T) {
	var gotOpenID string
	var gotScene int
	var gotVersion int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Content string `json:"content"`
			Version int    `json:"version"`
			Scene   int    `json:"scene"`
			OpenID  string `json:"openid"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gotOpenID, gotScene, gotVersion = body.OpenID, body.Scene, body.Version
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errcode": 0, "errmsg": "ok",
			"result":   map[string]any{"suggest": "pass", "label": 100},
			"trace_id": "trace-1",
		})
	}))
	defer server.Close()

	manager := newSecManager(t, server.URL)
	verdict, err := manager.MsgSecCheck(context.Background(), "你好，今天天气不错", "openid-1", SceneComment)
	if err != nil {
		t.Fatalf("MsgSecCheck: %v", err)
	}
	if verdict.Suggest != VerdictPass {
		t.Fatalf("suggest = %q", verdict.Suggest)
	}
	if gotOpenID != "openid-1" || gotScene != SceneComment || gotVersion != 2 {
		t.Fatalf("request = openid %q scene %d version %d", gotOpenID, gotScene, gotVersion)
	}
}

// TestMsgSecCheckRisky 校验违规文本返回 ErrContentRisky。
func TestMsgSecCheckRisky(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errcode": 0,
			"result":  map[string]any{"suggest": "risky", "label": 20001},
			"detail":  []map[string]any{{"strategy": "keyword", "label": 20001}},
		})
	}))
	defer server.Close()

	manager := newSecManager(t, server.URL)
	verdict, err := manager.MsgSecCheck(context.Background(), "违规文本", "openid-1", SceneComment)
	if !errors.Is(err, ErrContentRisky) {
		t.Fatalf("err = %v, want ErrContentRisky", err)
	}
	if verdict == nil || !verdict.Risky() || verdict.Label != 20001 {
		t.Fatalf("verdict = %+v", verdict)
	}
}

// TestMsgSecCheckLegacyRisky 校验老接口 87014 也按违规处理。
func TestMsgSecCheckLegacyRisky(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 87014, "errmsg": "risky content"})
	}))
	defer server.Close()

	manager := newSecManager(t, server.URL)
	if _, err := manager.MsgSecCheck(context.Background(), "违规", "openid-1", SceneComment); !errors.Is(err, ErrContentRisky) {
		t.Fatalf("err = %v, want ErrContentRisky", err)
	}
}

// TestMsgSecCheckRetriesAfterTokenExpired 校验 40001 会刷新 token 后重试。
func TestMsgSecCheckRetriesAfterTokenExpired(t *testing.T) {
	var checkCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tk", "expires_in": 7200})
		case "/check":
			checkCalls++
			if checkCalls == 1 {
				_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 40001, "errmsg": "invalid credential"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 0, "result": map[string]any{"suggest": "pass"}})
		}
	}))
	defer server.Close()

	manager := New(Config{
		AppID: "a", Secret: "b",
		Endpoint:            server.URL + "/token",
		MsgSecCheckEndpoint: server.URL + "/check",
	}, nil)
	verdict, err := manager.MsgSecCheck(context.Background(), "你好", "openid-1", SceneComment)
	if err != nil {
		t.Fatalf("MsgSecCheck: %v", err)
	}
	if verdict.Suggest != VerdictPass || checkCalls != 2 {
		t.Fatalf("suggest=%q calls=%d", verdict.Suggest, checkCalls)
	}
}

// newSecManager 构造一个能取到 token 的管理器。
func newSecManager(t *testing.T, checkURL string) *Manager {
	t.Helper()
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tk", "expires_in": 7200})
	}))
	t.Cleanup(tokenServer.Close)
	return New(Config{
		AppID: "a", Secret: "b",
		Endpoint:            tokenServer.URL,
		MsgSecCheckEndpoint: checkURL,
	}, nil)
}
