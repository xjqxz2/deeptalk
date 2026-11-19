-- DeepTalk MVP v1.0.0 · SQLite 初始结构
-- 说明：MVP 启动时使用 GORM AutoMigrate 便于首次运行；生产环境请以此脚本为准。
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
  id                INTEGER PRIMARY KEY AUTOINCREMENT,
  email             TEXT    NOT NULL,
  nickname          TEXT    NOT NULL,
  password_hash     TEXT    NOT NULL,
  role              TEXT    NOT NULL DEFAULT 'user',
  plan              TEXT    NOT NULL DEFAULT 'free',
  quota_tokens      INTEGER NOT NULL DEFAULT 4000000,
  used_tokens       INTEGER NOT NULL DEFAULT 0,
  disabled          INTEGER NOT NULL DEFAULT 0,
  failed_attempts   INTEGER NOT NULL DEFAULT 0,
  locked_until      DATETIME,
  last_login_at     DATETIME,
  last_login_ip     TEXT    NOT NULL DEFAULT '',
  last_login_city   TEXT    NOT NULL DEFAULT '',
  created_at        DATETIME NOT NULL,
  updated_at        DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_disabled ON users(disabled);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);
CREATE INDEX IF NOT EXISTS idx_users_last_login_at ON users(last_login_at);
CREATE INDEX IF NOT EXISTS idx_users_locked_until ON users(locked_until);

CREATE TABLE IF NOT EXISTS sessions (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id     INTEGER NOT NULL,
  token_hash  TEXT    NOT NULL,
  expires_at  DATETIME NOT NULL,
  revoked_at  DATETIME,
  ip          TEXT    NOT NULL DEFAULT '',
  user_agent  TEXT    NOT NULL DEFAULT '',
  created_at  DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_revoked_at ON sessions(revoked_at);

CREATE TABLE IF NOT EXISTS conversations (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id    INTEGER NOT NULL,
  title      TEXT    NOT NULL DEFAULT '新对话',
  model_id   INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  deleted_at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON conversations(updated_at);
CREATE INDEX IF NOT EXISTS idx_conversations_deleted_at ON conversations(deleted_at);

CREATE TABLE IF NOT EXISTS messages (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  conversation_id INTEGER NOT NULL,
  user_id         INTEGER NOT NULL,
  role            TEXT    NOT NULL,
  content         TEXT    NOT NULL DEFAULT '',
  reasoning       TEXT    NOT NULL DEFAULT '',
  status          TEXT    NOT NULL DEFAULT 'complete',
  model_name      TEXT    NOT NULL DEFAULT '',
  input_tokens    INTEGER NOT NULL DEFAULT 0,
  output_tokens   INTEGER NOT NULL DEFAULT 0,
  duration_ms     INTEGER NOT NULL DEFAULT 0,
  feedback        TEXT    NOT NULL DEFAULT '',
  sources_json    TEXT    NOT NULL DEFAULT '',
  attachments_json TEXT   NOT NULL DEFAULT '',
  created_at      DATETIME NOT NULL,
  updated_at      DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_user_id ON messages(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

CREATE TABLE IF NOT EXISTS attachments (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id      INTEGER NOT NULL,
  message_id   INTEGER NOT NULL DEFAULT 0,
  name         TEXT    NOT NULL,
  mime         TEXT    NOT NULL,
  size         INTEGER NOT NULL DEFAULT 0,
  width        INTEGER NOT NULL DEFAULT 0,
  height       INTEGER NOT NULL DEFAULT 0,
  storage_name TEXT    NOT NULL,
  created_at   DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_attachments_storage_name ON attachments(storage_name);
CREATE INDEX IF NOT EXISTS idx_attachments_user_id ON attachments(user_id);
CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id);

CREATE TABLE IF NOT EXISTS providers (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  name            TEXT    NOT NULL,
  code            TEXT    NOT NULL,
  kind            TEXT    NOT NULL DEFAULT 'custom',
  base_url        TEXT    NOT NULL,
  api_key_enc     TEXT    NOT NULL DEFAULT '',
  api_key_masked  TEXT    NOT NULL DEFAULT '',
  timeout_sec     INTEGER NOT NULL DEFAULT 60,
  max_concurrency INTEGER NOT NULL DEFAULT 4,
  enabled         INTEGER NOT NULL DEFAULT 0,
  verified        INTEGER NOT NULL DEFAULT 0,
  is_builtin      INTEGER NOT NULL DEFAULT 0,
  is_default      INTEGER NOT NULL DEFAULT 0,
  sort_order      INTEGER NOT NULL DEFAULT 0,
  last_tested_at  DATETIME,
  last_latency_ms INTEGER NOT NULL DEFAULT 0,
  last_error      TEXT    NOT NULL DEFAULT '',
  created_at      DATETIME NOT NULL,
  updated_at      DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_providers_code ON providers(code);
CREATE INDEX IF NOT EXISTS idx_providers_enabled ON providers(enabled);

CREATE TABLE IF NOT EXISTS models (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  provider_id     INTEGER NOT NULL,
  model_id        TEXT    NOT NULL,
  display_name    TEXT    NOT NULL,
  description     TEXT    NOT NULL DEFAULT '',
  context_window  INTEGER NOT NULL DEFAULT 32768,
  max_concurrency INTEGER NOT NULL DEFAULT 2,
  enabled         INTEGER NOT NULL DEFAULT 1,
  is_default      INTEGER NOT NULL DEFAULT 0,
  is_fallback     INTEGER NOT NULL DEFAULT 0,
  sort_order      INTEGER NOT NULL DEFAULT 0,
  created_at      DATETIME NOT NULL,
  updated_at      DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_models_provider_model ON models(provider_id, model_id);
CREATE INDEX IF NOT EXISTS idx_models_enabled ON models(enabled);

CREATE TABLE IF NOT EXISTS settings (
  key        TEXT PRIMARY KEY,
  value      TEXT NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS logs (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_id    INTEGER NOT NULL,
  actor_name  TEXT    NOT NULL,
  actor_email TEXT    NOT NULL,
  action      TEXT    NOT NULL,
  target_type TEXT    NOT NULL,
  target_id   INTEGER NOT NULL DEFAULT 0,
  detail      TEXT    NOT NULL DEFAULT '',
  ip          TEXT    NOT NULL DEFAULT '',
  created_at  DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_logs_actor_id ON logs(actor_id);
CREATE INDEX IF NOT EXISTS idx_logs_created_at ON logs(created_at);
