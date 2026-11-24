const { launch, wire, logs, H5, SHOT_DIR, USER, PASS } = require('./lib.cjs')
const fs = require('fs')

const T0 = Date.now()
const ts = () => ((Date.now() - T0) / 1000).toFixed(1) + 's'

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 420, height: 860 }, deviceScaleFactor: 2 })
  const page = await ctx.newPage()
  wire(page, 'item1b')

  const wsEvents = []
  page.on('websocket', (ws) => {
    wsEvents.push({ dir: 'open', url: ws.url(), t: ts() })
    ws.on('framesent', (f) => {
      const p = typeof f.payload === 'string' ? f.payload : '<bin>'
      wsEvents.push({ dir: 'sent', t: ts(), payload: p.slice(0, 500) })
    })
    ws.on('framereceived', (f) => {
      const p = typeof f.payload === 'string' ? f.payload : '<bin>'
      wsEvents.push({ dir: 'recv', t: ts(), payload: p.slice(0, 400) })
    })
    ws.on('close', () => wsEvents.push({ dir: 'close', t: ts() }))
  })

  await page.goto(H5 + '#/pages/login/login', { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(1500)
  await page.locator('input[type=email]').fill(USER)
  await page.locator('input[type=password]').fill(PASS)
  await page.locator('button.submit-btn').click()
  await page.waitForTimeout(3500)

  await page.locator('button.menu-btn').click()
  await page.waitForTimeout(900)
  await page.getByText('新建对话').first().click()
  await page.waitForTimeout(1800)

  // ordered snapshot of the message column
  const snap = () =>
    page.evaluate(() => {
      const col = document.querySelector('.messages-column')
      const rows = [...(col ? col.children : [])].map((e) => {
        const cls = (e.className || '').toString()
        if (!/message/.test(cls)) return { cls, role: 'other', len: 0, txt: '' }
        const role = /assistant/.test(cls) ? 'assistant' : 'user'
        const body = e.querySelector('.dt-prose, .user-bubble')
        const t = (body?.innerText || '').replace(/\s+/g, ' ').trim()
        return {
          role,
          status: (/streaming/.test(cls) && 'streaming') || (/(^|\s)complete/.test(cls) && 'complete') || cls,
          len: t.length,
          head: t.slice(0, 40),
          tag: (e.querySelector('.interject-tag')?.innerText || '').trim(),
          pending: /排队中/.test(e.innerText || ''),
        }
      })
      return { rows, ph: document.querySelector('textarea.composer-input')?.getAttribute('placeholder') || '' }
    })
  const show = (label, s) => {
    console.log(`${label} :: ` + s.rows.map((r, i) => `${i}:${r.role}/${r.status}/len${r.len}/${r.tag || '-'}${r.pending ? '/排队中' : ''} "${r.head}"`).join('  |  '))
  }

  console.log(ts(), 'NEW_CONV_EMPTY:', JSON.stringify(await snap()))

  const ta = page.locator('textarea.composer-input')
  await ta.click()
  await ta.fill('用一句话说明 SSE 是什么')
  await page.waitForTimeout(200)
  await page.locator('button.send-btn').click()
  const sendAt = Date.now()

  let streamStart = null
  for (let i = 0; i < 60; i++) {
    await page.waitForTimeout(400)
    const s = await snap()
    if (s.rows.some((r) => r.role === 'assistant' && (r.status === 'streaming' || r.len > 0))) { streamStart = Date.now() - sendAt; show(ts(), s); break }
  }
  console.log(ts(), 'ROUND1_STREAM_START_MS=', streamStart)
  await page.screenshot({ path: `${SHOT_DIR}/_1b-生成中.png` })

  await ta.click()
  await ta.fill('补充一句')
  await page.waitForTimeout(150)
  await page.keyboard.press('Enter')
  console.log(ts(), 'INTERJECT_SENT')
  await page.waitForTimeout(400)
  show(ts() + ' JUST_AFTER_INTERJECT', await snap())
  await page.screenshot({ path: `${SHOT_DIR}/1-插话排队中.png` })

  const deadline = Date.now() + 140000
  let lastRows = null, stableFor = 0
  while (Date.now() < deadline) {
    await page.waitForTimeout(2500)
    const s = await snap()
    const streaming = s.rows.some((r) => r.status === 'streaming')
    show(ts(), s)
    const key = JSON.stringify(s.rows.map((r) => [r.role, r.status, r.len]))
    if (key === lastRows) stableFor++; else { stableFor = 0; lastRows = key }
    if (!streaming && stableFor >= 1) break
  }
  const end = await snap()
  console.log(ts(), 'END:', JSON.stringify(end, null, 2))
  await page.screenshot({ path: `${SHOT_DIR}/1-第二轮回答.png` })

  fs.writeFileSync(`${SHOT_DIR}/_scripts/ws_item1.json`, JSON.stringify(wsEvents, null, 2))
  console.log('WS_EVENTS=' + wsEvents.length)
  console.log(wsEvents.filter((e) => e.dir !== 'recv').map((e) => `${e.t} ${e.dir} ${(e.payload || '').slice(0, 200)}`).join('\n'))
  console.log('WS_RECV_TYPES:', wsEvents.filter((e) => e.dir === 'recv').map((e) => `${e.t} ${(e.payload || '').slice(0, 160)}`).join('\n'))
  console.log('LOGS:\n' + logs.join('\n'))
  await b.close()
})()
