package bootstrap

import (
	"context"
	"time"

	"deeptalk/internal/wechat"
)

// startWeChat 初始化小程序 access_token 管理器。
// 没有配置凭据时跳过；获取失败只告警，不影响对话等主链路。
func (a *App) startWeChat(ctx context.Context) error {
	cfg := a.Config.WeChat
	if !cfg.Enabled {
		a.Log.Info("wechat access_token 管理未启用（缺少 app_id / secret）")
		return nil
	}
	manager := wechat.New(wechat.Config{
		AppID:   cfg.AppID,
		Secret:  cfg.Secret,
		Buffer:  time.Duration(cfg.TokenBuffer) * time.Second,
		Timeout: time.Duration(cfg.TimeoutSec) * time.Second,
	}, a.Log)
	manager.Start(ctx)
	a.WeChat = manager
	return nil
}

// stopWeChat 停止后台刷新协程。
func (a *App) stopWeChat(ctx context.Context) error {
	if a.WeChat == nil {
		return nil
	}
	a.WeChat.Stop()
	a.WeChat = nil
	return nil
}
