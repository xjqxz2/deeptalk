package wechat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// 微信开放接口地址。
const (
	code2SessionURL = "https://api.weixin.qq.com/sns/jscode2session"
	msgSecCheckURL  = "https://api.weixin.qq.com/wxa/msg_sec_check"
)

// 内容安全的判定结果。
const (
	VerdictPass   = "pass"
	VerdictReview = "review"
	VerdictRisky  = "risky"
)

// 内容安全的业务场景（微信要求上报，影响检测模型）。
const (
	SceneProfile = 1 // 资料
	SceneComment = 2 // 评论
	SceneForum   = 3 // 论坛
	SceneSocial  = 4 // 社交日志
)

// ErrContentRisky 表示文本被判定为违规。
var ErrContentRisky = errors.New("wechat: content is risky")

// Session 是 code2session 的结果。
type Session struct {
	OpenID     string // 小程序内用户唯一标识
	UnionID    string // 开放平台唯一标识（绑定后才有）
	SessionKey string // 会话密钥，仅用于解密开放数据
}

// Verdict 是内容安全的检测结论。
type Verdict struct {
	Suggest string   // pass / review / risky
	Label   int      // 命中标签，0 表示未命中
	TraceID string   // 便于向微信侧排查
	Details []string // 细粒度标签描述，仅用于审计
}

// Risky 表示需要拦截。
func (v Verdict) Risky() bool { return v.Suggest == VerdictRisky }

// Review 表示可疑，建议人工复核。
func (v Verdict) Review() bool { return v.Suggest == VerdictReview }

// Code2Session 用小程序 wx.login 的 code 换 openid。
func (m *Manager) Code2Session(ctx context.Context, code string) (*Session, error) {
	if !m.Enables() {
		return nil, ErrDisabled
	}
	if strings.TrimSpace(code) == "" {
		return nil, errors.New("wechat: js_code is required")
	}
	query := url.Values{}
	query.Set("appid", m.cfg.AppID)
	query.Set("secret", m.cfg.Secret)
	query.Set("js_code", code)
	query.Set("grant_type", "authorization_code")

	endpoint := m.cfg.Code2SessionEndpoint
	if endpoint == "" {
		endpoint = code2SessionURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	body, err := m.do(req)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		OpenID     string `json:"openid"`
		UnionID    string `json:"unionid"`
		SessionKey string `json:"session_key"`
		ErrCode    int    `json:"errcode"`
		ErrMsg     string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("wechat: decode code2session: %w", err)
	}
	if parsed.ErrCode != 0 {
		return nil, fmt.Errorf("wechat: code2session errcode=%d errmsg=%s", parsed.ErrCode, parsed.ErrMsg)
	}
	if parsed.OpenID == "" {
		return nil, errors.New("wechat: code2session returned empty openid")
	}
	return &Session{OpenID: parsed.OpenID, UnionID: parsed.UnionID, SessionKey: parsed.SessionKey}, nil
}

// MsgSecCheck 检测一段文本是否含有违法违规内容。
// scene 取值见 Scene* 常量；openid 为该用户在小程序内的标识（v2 接口必填）。
func (m *Manager) MsgSecCheck(ctx context.Context, content, openid string, scene int) (*Verdict, error) {
	if !m.Enables() {
		return nil, ErrDisabled
	}
	if strings.TrimSpace(content) == "" {
		return &Verdict{Suggest: VerdictPass}, nil
	}
	if scene <= 0 {
		scene = SceneComment
	}

	verdict, err := m.msgSecCheckOnce(ctx, content, openid, scene)
	// token 失效时微信返回 40001/42001，强制刷新后再试一次
	if err != nil && needsTokenRefresh(err) {
		m.Invalidate()
		return m.msgSecCheckOnce(ctx, content, openid, scene)
	}
	return verdict, err
}

// msgSecCheckOnce 发起一次文本检测。
func (m *Manager) msgSecCheckOnce(ctx context.Context, content, openid string, scene int) (*Verdict, error) {
	token, err := m.Token(ctx)
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(map[string]any{
		"content": content,
		"version": 2,
		"scene":   scene,
		"openid":  openid,
	})
	if err != nil {
		return nil, err
	}
	endpoint := m.cfg.MsgSecCheckEndpoint
	if endpoint == "" {
		endpoint = msgSecCheckURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"?access_token="+url.QueryEscape(token), strings.NewReader(string(payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	body, err := m.do(req)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		ErrCode int `json:"errcode"`
		ErrMsg  string
		Result  struct {
			Suggest string `json:"suggest"`
			Label   int    `json:"label"`
		} `json:"result"`
		Detail []struct {
			Strategy string `json:"strategy"`
			Label    int    `json:"label"`
			Keyword  string `json:"keyword"`
		} `json:"detail"`
		TraceID string `json:"trace_id"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("wechat: decode msg_sec_check: %w", err)
	}

	// 老版本接口用 87014 表示违规
	if parsed.ErrCode == 87014 {
		return &Verdict{Suggest: VerdictRisky, TraceID: parsed.TraceID}, ErrContentRisky
	}
	if parsed.ErrCode != 0 {
		return nil, &APIError{Code: parsed.ErrCode, Message: parsed.ErrMsg, Endpoint: "msg_sec_check"}
	}

	verdict := &Verdict{Suggest: parsed.Result.Suggest, Label: parsed.Result.Label, TraceID: parsed.TraceID}
	if verdict.Suggest == "" {
		verdict.Suggest = VerdictPass
	}
	for _, item := range parsed.Detail {
		verdict.Details = append(verdict.Details, fmt.Sprintf("%s:%d", item.Strategy, item.Label))
	}
	if verdict.Risky() {
		return verdict, ErrContentRisky
	}
	return verdict, nil
}

// APIError 表示微信接口返回的业务错误。
type APIError struct {
	Code     int
	Message  string
	Endpoint string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("wechat: %s errcode=%d errmsg=%s", e.Endpoint, e.Code, e.Message)
}

// needsTokenRefresh 判断错误是否是 access_token 失效。
func needsTokenRefresh(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.Code == 40001 || apiErr.Code == 42001 || apiErr.Code == 40014
}

// do 发送请求并读取响应体。
func (m *Manager) do(req *http.Request) ([]byte, error) {
	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<18))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("wechat: http %d", resp.StatusCode)
	}
	return body, nil
}
