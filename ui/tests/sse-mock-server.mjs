// 测试用慢速 SSE 服务：模拟带检索活动提示的生成过程。
import http from 'node:http'

const port = Number(process.env.SSE_PORT || 4199)

const frames = [
  ['meta', { assistant_message_id: 9001, model_name: 'DeepSeek-V4', user_message: { id: 2, created_at: '2026-11-20T10:00:02+08:00', input_tokens: 5 }, title: '会话 A' }],
  ['reasoning', { delta: '用户想知道今天的新闻，我需要联网确认。' }],
  ['search', { status: 'searching' }],
  ['activity', { status: 'searching', label: '正在搜索资料…' }],
  ['activity', { status: 'reading', label: '正在阅读网页…', detail: '今日头条', url: 'https://example.com/news' }],
  ['sources', { items: [{ title: '今日头条', url: 'https://example.com/news', snippet: '新闻摘要' }] }],
  ['activity', { status: 'found', label: '已找到 1 个来源' }],
  ['delta', { delta: '今天的主要新闻是' }],
  ['delta', { delta: '这些内容。' }],
  ['usage', { input_tokens: 5, output_tokens: 8, duration_ms: 4200, tokens_per_sec: 2, character_count: 12 }],
  ['done', { assistant_message_id: 9001, status: 'complete', title: '会话 A' }],
]

http
  .createServer((req, res) => {
    if (!req.url?.includes('/stream')) {
      res.writeHead(404).end()
      return
    }
    res.writeHead(200, {
      'content-type': 'text/event-stream',
      'cache-control': 'no-cache',
      'access-control-allow-origin': '*',
    })
    let index = 0
    const tick = setInterval(() => {
      if (index >= frames.length) {
        clearInterval(tick)
        res.end()
        return
      }
      const [event, data] = frames[index]
      res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`)
      index += 1
    }, 450)
    req.on('close', () => clearInterval(tick))
  })
  .listen(port, '127.0.0.1', () => {
    console.log(`sse mock listening on ${port}`)
  })
