# syntax=docker/dockerfile:1.7
# DeepTalk 单镜像：Vue 双端构建 -> Go 内嵌编译 -> Alpine + Obscura 运行。
# 说明：Obscura 官方 Linux 产物是 glibc 动态链接，因此运行层使用 Alpine + glibc 兼容基础镜像。

ARG NODE_VERSION=24
ARG GO_VERSION=1.26
ARG OBSCURA_VERSION=0.2.7
ARG PNPM_VERSION=10.26.0

# ---------- 前端：一次构建 C / B 两个应用 ----------
FROM --platform=$BUILDPLATFORM node:${NODE_VERSION}-alpine AS ui
ARG PNPM_VERSION
WORKDIR /src/ui
RUN npm config set registry https://registry.npmmirror.com \
    && npm install -g pnpm@${PNPM_VERSION}
# 先只拷贝依赖清单，最大化利用 Docker 层缓存
COPY ui/package.json ui/pnpm-lock.yaml ui/pnpm-workspace.yaml ui/.npmrc ./
COPY ui/c/package.json ./c/package.json
COPY ui/b/package.json ./b/package.json
COPY ui/shared/package.json ./shared/package.json
RUN pnpm install --frozen-lockfile
COPY ui/ ./
RUN pnpm build

# ---------- Obscura：按目标架构下载官方静态资源 ----------
FROM --platform=$BUILDPLATFORM alpine:3.21 AS obscura
ARG OBSCURA_VERSION
ARG TARGETARCH=amd64
RUN apk add --no-cache curl tar \
    && case "${TARGETARCH}" in \
         amd64) arch="x86_64" ;; \
         arm64) arch="aarch64" ;; \
         *) echo "unsupported arch: ${TARGETARCH}" >&2; exit 1 ;; \
       esac \
    && curl -fsSL -o /tmp/obscura.tar.gz \
       "https://github.com/h4ckf0r0day/obscura/releases/download/v${OBSCURA_VERSION}/obscura-${arch}-linux.tar.gz" \
    && mkdir -p /opt/obscura \
    && tar -xzf /tmp/obscura.tar.gz -C /opt/obscura \
    && test -x /opt/obscura/obscura \
    && test -x /opt/obscura/obscura-worker

# ---------- glibc 运行库：Obscura 需要 glibc 版 libgcc_s ----------
# Alpine 自带的 libgcc_s 面向 musl，不能安全地加载到 glibc 进程里；
# 从 Ubuntu 取对应架构的 libgcc_s，放进 alpine-glibc 的兼容目录。
FROM --platform=$TARGETPLATFORM ubuntu:22.04 AS glibc-libs
RUN apt-get update \
    && apt-get install -y --no-install-recommends libgcc-s1 \
    && rm -rf /var/lib/apt/lists/* \
    && cp /usr/lib/*-linux-gnu/libgcc_s.so.1 /libgcc_s.so.1

# ---------- 后端：交叉编译，把两个前端产物嵌入二进制 ----------
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS backend
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_TIME=unknown
ARG GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=0
WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    GOPROXY=${GOPROXY} go mod download
COPY server/ ./
COPY --from=ui /src/ui/c/dist ./web/dist/c
COPY --from=ui /src/ui/b/dist ./web/dist/b
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /out/deeptalk .

# ---------- 运行：Alpine + glibc（Obscura 需要）+ 非 root ----------
FROM frolvlad/alpine-glibc:alpine-3.21_glibc-2.41
LABEL org.opencontainers.image.title="DeepTalk" \
      org.opencontainers.image.description="DeepTalk AI 对话工作台与管理后台"

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S -g 10001 deeptalk \
    && adduser -S -D -H -u 10001 -G deeptalk deeptalk

WORKDIR /app
COPY --from=obscura /opt/obscura/obscura /usr/local/bin/obscura
COPY --from=obscura /opt/obscura/obscura-worker /usr/local/bin/obscura-worker
COPY --from=glibc-libs /libgcc_s.so.1 /usr/glibc-compat/lib/libgcc_s.so.1
COPY --from=backend /out/deeptalk /app/deeptalk
COPY server/configs/app.yaml /app/configs/app.yaml
RUN mkdir -p /app/data && chown -R deeptalk:deeptalk /app

# 启动时校验 Obscura 与 glibc 兼容层可用；失败则镜像构建失败，避免运行期才发现
ENV LD_LIBRARY_PATH=/usr/glibc-compat/lib
RUN obscura --version >/dev/null

USER deeptalk
ENV HOME=/app \
    TZ=Asia/Shanghai \
    DEEPTALK_ADDR=0.0.0.0:9821 \
    DEEPTALK_BASE_URL=http://localhost:9821 \
    DEEPTALK_DB=/app/data/deeptalk.db \
    DEEPTALK_SEED=true \
    DEEPTALK_SEARCH_ENABLED=true \
    DEEPTALK_OBSCURA_BIN=obscura \
    DEEPTALK_UPLOAD_DIR=/app/data/uploads

VOLUME ["/app/data"]
EXPOSE 9821
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD wget -qO- http://127.0.0.1:9821/healthz >/dev/null || exit 1
ENTRYPOINT ["/app/deeptalk"]
