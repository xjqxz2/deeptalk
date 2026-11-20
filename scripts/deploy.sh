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

# 镜像仓库名从 .env 读取，用于部署后的旧镜像清理
ENV_IMAGE="$(grep -E '^DEEPTALK_IMAGE=' "$ENV_FILE" 2>/dev/null | tail -1 | cut -d= -f2- || true)"
IMAGE_REPO="${DEEPTALK_IMAGE:-${ENV_IMAGE:-}}"
IMAGE_REPO="${IMAGE_REPO%:*}"

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


# prune_images 只删除本项目的旧镜像：
# - 按创建时间从新到旧排序，保留最新的 KEEP_IMAGES 个（默认 3：当前 + 两个可回滚版本）；
# - 正在被容器使用的镜像永不删除；
# - 只匹配本项目仓库名，不会碰到服务器上的其它服务。
prune_images() {
  local keep="${KEEP_IMAGES:-3}"
  if [ -z "${IMAGE_REPO:-}" ]; then
    echo "未识别镜像仓库名，跳过旧镜像清理"
    return 0
  fi

  local removed=0 kept=0 index=0 id
  while read -r id; do
    [ -n "$id" ] || continue
    index=$((index + 1))
    if [ "$index" -le "$keep" ]; then
      kept=$((kept + 1))
      continue
    fi
    # ancestor 过滤能同时匹配短 ID，容器在用的镜像一律跳过
    if docker ps -a --filter "ancestor=$id" --format '{{.ID}}' | grep -q .; then
      continue
    fi
    if docker rmi "$id" >/dev/null 2>&1; then
      removed=$((removed + 1))
    fi
  done < <(
    docker images --filter "reference=${IMAGE_REPO}*" --format '{{.ID}} {{.CreatedAt}}' \
      | sort -k2,3 -r \
      | awk '{print $1}' \
      | awk '!seen[$0]++'
  )
  echo "旧镜像清理：删除 ${removed} 个，保留 ${kept} 个（KEEP_IMAGES=${keep}）"
}

echo "部署成功: $TAG"
docker compose -f "$COMPOSE_FILE" ps
prune_images
