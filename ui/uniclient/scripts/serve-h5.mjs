/**
 * 本地预览 H5 构建产物：静态文件 + /api 反向代理（含 WebSocket 升级）。
 *
 * 用法：
 *   npm run build:h5           # 产物在 dist/build/h5
 *   node scripts/serve-h5.mjs  # 默认 0.0.0.0:5274
 *
 * 需要代理到别的后端时用 PROXY_TARGET 覆盖（默认生产环境）。
 */
import http from 'node:http'
import https from 'node:https'
import { createReadStream, existsSync, statSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const ROOT = process.env.SERVE_ROOT || path.resolve(here, '../dist/build/h5')
const PORT = Number(process.env.PORT || 5274)
const HOST = process.env.HOST || '0.0.0.0'
const TARGET = process.env.PROXY_TARGET || 'https://deeptalk.peon.top'
const target = new URL(TARGET)

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.mjs': 'text/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.svg': 'image/svg+xml',
  '.ico': 'image/x-icon',
  '.woff2': 'font/woff2',
  '.ttf': 'font/ttf',
  '.map': 'application/json',
}

/** 代理 /api 请求，包括 SSE 与 WebSocket 升级。 */
function proxy(req, res) {
  const client = target.protocol === 'https:' ? https : http
  const upstream = client.request(
    {
      protocol: target.protocol,
      hostname: target.hostname,
      port: target.port || (target.protocol === 'https:' ? 443 : 80),
      method: req.method,
      path: req.url,
      headers: { ...req.headers, host: target.host },
    },
    (upstreamRes) => {
      res.writeHead(upstreamRes.statusCode || 502, upstreamRes.headers)
      upstreamRes.pipe(res)
    },
  )
  upstream.on('error', (error) => {
    res.writeHead(502, { 'Content-Type': 'text/plain; charset=utf-8' })
    res.end(`代理失败：${error.message}`)
  })
  req.pipe(upstream)
}

const server = http.createServer((req, res) => {
  if (req.url.startsWith('/api/')) {
    proxy(req, res)
    return
  }
  const url = new URL(req.url, `http://${req.headers.host}`)
  let filePath = path.join(ROOT, decodeURIComponent(url.pathname))
  if (!filePath.startsWith(ROOT)) {
    res.writeHead(403).end('forbidden')
    return
  }
  if (!existsSync(filePath) || statSync(filePath).isDirectory()) {
    filePath = path.join(ROOT, 'index.html')
  }
  const ext = path.extname(filePath).toLowerCase()
  res.writeHead(200, { 'Content-Type': MIME[ext] || 'application/octet-stream' })
  createReadStream(filePath).pipe(res)
})

// WebSocket 升级交给同一个后端（小程序端的流式回答）
server.on('upgrade', (req, socket, head) => {
  const client = target.protocol === 'https:' ? https : http
  const upstream = client.request({
    protocol: target.protocol,
    hostname: target.hostname,
    port: target.port || (target.protocol === 'https:' ? 443 : 80),
    method: 'GET',
    path: req.url,
    headers: { ...req.headers, host: target.host },
  })
  // 鉴权失败时后端会直接回 HTTP 响应而不是 101，这里要把状态码透传给客户端
  upstream.on('response', (upRes) => {
    const headers = Object.entries(upRes.headers)
      .map(([key, value]) => `${key}: ${value}`)
      .join('\r\n')
    socket.write(`HTTP/1.1 ${upRes.statusCode} ${upRes.statusMessage || ''}\r\n${headers}\r\n\r\n`)
    upRes.pipe(socket)
  })
  upstream.on('upgrade', (upRes, upSocket, upHead) => {
    socket.write(
      `HTTP/1.1 101 Switching Protocols\r\n${Object.entries(upRes.headers)
        .map(([key, value]) => `${key}: ${value}`)
        .join('\r\n')}\r\n\r\n`,
    )
    if (upHead?.length) socket.write(upHead)
    upSocket.pipe(socket)
    socket.pipe(upSocket)
  })
  upstream.on('error', () => {
    if (!socket.destroyed) {
      socket.write('HTTP/1.1 502 Bad Gateway\r\n\r\n')
      socket.destroy()
    }
  })
  upstream.end()
  if (head?.length) socket.write(head)
})

server.listen(PORT, HOST, () => {
  console.log(`DeepTalk 小程序端 H5 预览：http://${HOST}:${PORT}`)
  console.log(`接口代理：${TARGET}`)
})
