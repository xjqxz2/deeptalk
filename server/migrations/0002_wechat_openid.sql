-- 小程序 openid：用于内容安全（msgSecCheck）按用户维度检测
ALTER TABLE users ADD COLUMN wechat_open_id TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_wechat_open_id ON users (wechat_open_id);
