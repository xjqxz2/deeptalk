#!/usr/bin/env bash
# 服务器端部署脚本：由 GitHub Actions 通过 SSH 调用，也可手动执行。
# 用法：./deploy.sh v1.0.0
set -euo pipefail

TAG="${1:?usage: deploy.sh <image-tag>}"
if [[ ! "$TAG" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "非法镜像 tag: $TAG" >&2
  exit 2
fi

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$APP_DIR"

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.prod.yml}"
ENV_FILE=".env"
ENV_DATA_DIR="$(grep -E '^DEEPTALK_DATA_DIR=' "$ENV_FILE" 2>/dev/null | tail -1 | cut -d= -f2- || true)"
ENV_PORT="$(grep -E '^DEEPTALK_PORT=' "$ENV_FILE" 2>/dev/null | tail -1 | cut -d= -f2- || true)"
DATA_DIR="${DEEPTALK_DATA_DIR:-${ENV_DATA_DIR:-$APP_DIR/data}}"
PORT="${DEEPTALK_PORT:-${ENV_PORT:-9821}}"

[ -f "$COMPOSE_FILE" ] || { echo "缺少 $APP_DIR/$COMPOSE_FILE" >&2; exit 1; }
[ -f "$ENV_FILE" ] || { echo "缺少 $APP_DIR/.env" >&2; exit 1; }

PREV_TAG="$(grep -E '^DEEPTALK_IMAGE_TAG=' "$ENV_FILE" | tail -1 | cut -d= -f2- || true)"
echo "当前 tag: ${PREV_TAG:-none}；目标 tag: $TAG"

# SQLite 的 db + wal + shm 必须在停容器后一起备份，避免复制出不一致快照
if [ -f "$DATA_DIR/deeptalk.db" ]; then
  mkdir -p "$APP_DIR/backups"
  BACKUP="$APP_DIR/backups/deeptalk-$(date +%Y%m%d-%H%M%S).tar.gz"
  echo "停止容器以备份数据..."
  docker compose -f "$COMPOSE_FILE" stop >/dev/null 2>&1 || true
  tar -czf "$BACKUP" -C "$DATA_DIR" .
  docker compose -f "$COMPOSE_FILE" start >/dev/null 2>&1 || true
  echo "已备份: $BACKUP"
  # 只保留最近 14 天的备份
  find "$APP_DIR/backups" -name 'deeptalk-*.tar.gz' -mtime +14 -delete 2>/dev/null || true
fi

echo "拉取镜像 $TAG..."
DEEPTALK_IMAGE_TAG="$TAG" docker compose -f "$COMPOSE_FILE" pull

echo "启动新版本..."
DEEPTALK_IMAGE_TAG="$TAG" docker compose -f "$COMPOSE_FILE" up -d --remove-orphans

echo "等待健康检查..."
ok=0
for _ in $(seq 1 30); do
  if curl -fsS "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1; then
    ok=1
    break
  fi
  sleep 2
done

if [ "$ok" -ne 1 ]; then
  echo "健康检查失败，尝试回滚..." >&2
  if [ -n "$PREV_TAG" ] && [ "$PREV_TAG" != "$TAG" ]; then
    DEEPTALK_IMAGE_TAG="$PREV_TAG" docker compose -f "$COMPOSE_FILE" up -d --remove-orphans || true
  fi
  exit 1
fi

# 健康后才写入新 tag，下一次回滚有目标
if grep -qE '^DEEPTALK_IMAGE_TAG=' "$ENV_FILE"; then
  sed -i "s/^DEEPTALK_IMAGE_TAG=.*/DEEPTALK_IMAGE_TAG=${TAG}/" "$ENV_FILE"
else
  printf '\nDEEPTALK_IMAGE_TAG=%s\n' "$TAG" >> "$ENV_FILE"
fi

echo "部署成功: $TAG"
docker compose -f "$COMPOSE_FILE" ps
