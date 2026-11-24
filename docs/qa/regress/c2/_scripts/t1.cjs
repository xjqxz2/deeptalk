const { launch, wire, logs, H5, SHOT_DIR, USER, PASS } = require('./lib.cjs')

const T0 = Date.now()
const ts = () => ((Date.now() - T0) / 1000).toFixed(1) + 's'

;(async () => {
  const b = await launch()
  const ctx = await b.newContext({ viewport: { width: 420, height: 860 }, deviceScaleFactor: 2 })
  const page = await ctx.newPage()
  wire(page, 'item1')
  await page.goto(H5 + '#/pages/login/login', { waitUntil: 'networkidle' }).catch(() => {})
  await page.waitForTimeout(1500)
  await page.locator('input[type=email]').fill(USER)
  await page.locator('input[type=password]').fill(PASS)
  await page.locator('button.submit-btn').click()
  await page.waitForTimeout(3500)
  console.log(ts(), 'logged in', page.url())

  // new conversation
  await page.locator('button.menu-btn').click()
  await page.waitForTimeout(900)
  const newBtn = page.getByText('新建对话').first()
  console.log(ts(), 'new-btn count', await page.getByText('新建对话').count())
  await newBtn.click()
  await page.waitForTimeout(1800)
  console.log(ts(), 'after new conv url', page.url(), 'title=', await page.locator('h1.title').innerText().catch(() => 'n/a'))

  const state = () =>
    page.evaluate(() => {
      const q = (s) => [...document.querySelectorAll(s)]
      const asst = q('.message.assistant').map((e) => ({
        cls: e.className,
        len: (e.querySelector('.dt-prose')?.innerText || '').trim().length,
        txt: (e.querySelector('.dt-prose')?.innerText || '').replace(/\s+/g, ' ').trim().slice(0, 120),
      }))
      const user = q('.message.user').map((e) => ({
        cls: e.className,
        txt: (e.querySelector('.user-bubble')?.innerText || '').replace(/\s+/g, ' ').trim().slice(0, 60),
        tag: (e.querySelector('.interject-tag')?.innerText || '').trim(),
        pending: (e.querySelector('.interject-pending')?.innerText || '').trim(),
      }))
      return {
        asst,
        user,
        placeholder: document.querySelector('textarea.composer-input')?.getAttribute('placeholder'),
        busy: /生成中/.test(document.body.innerText),
        queueText: (document.body.innerText.match(/排队中[^\n]*/g) || []).slice(0, 3),
        interjectTag: document.querySelectorAll('.interject-tag').length,
      }
    })

  // send round 1
  const ta = page.locator('textarea.composer-input')
  await ta.click()
  await ta.fill('用一句话说明 SSE 是什么')
  await page.waitForTimeout(300)
  await page.locator('button.send-btn').click()
  console.log(ts(), 'sent round1')

  // wait for streaming to start
  let started = false
  for (let i = 0; i < 40; i++) {
    await page.waitForTimeout(500)
    const s = await state()
    if (s.asst.length > 0 && s.asst.some((a) => a.len > 0 || /streaming/.test(a.cls)) || /生成中/.test(s.placeholder || '')) {
      started = true
      console.log(ts(), 'streaming detected', JSON.stringify(s.asst))
      break
    }
  }
  console.log(ts(), 'STARTED=', started)
  await page.screenshot({ path: `${SHOT_DIR}/_1-生成中.png` })

  // interject quickly
  await ta.click()
  await ta.fill('补充一句')
  await page.waitForTimeout(200)
  await page.keyboard.press('Enter')
  console.log(ts(), 'interjected (Enter)')

  // observe queued state up to 8s
  let sawQueued = false
  for (let i = 0; i < 16; i++) {
    await page.waitForTimeout(500)
    const s = await state()
    if (s.interjectTag > 0 || s.queueText.length > 0) {
      sawQueued = true
      console.log(ts(), 'QUEUED_SEEN', JSON.stringify({ queueText: s.queueText, tag: s.interjectTag, user: s.user }))
      break
    }
  }
  await page.screenshot({ path: `${SHOT_DIR}/1-插话排队中.png` })
  console.log(ts(), 'SAW_QUEUED=', sawQueued)

  // wait up to 120s for round 2
  const deadline = Date.now() + 120000
  let final = null
  let round2 = false
  while (Date.now() < deadline) {
    await page.waitForTimeout(2000)
    const s = await state()
    const real = s.asst.filter((a) => a.len > 20)
    const streaming = s.asst.some((a) => /streaming/.test(a.cls))
    console.log(ts(), 'poll asst=', s.asst.length, 'real=', real.length, 'streaming=', streaming, 'ph=', (s.placeholder || '').slice(0, 12))
    if (real.length >= 2 && !streaming) {
      round2 = true
      final = s
      break
    }
    if (!streaming && s.asst.length >= 2 && real.length >= 2) {
      round2 = true
      final = s
      break
    }
    final = s
  }
  await page.waitForTimeout(1500)
  const end = await state()
  console.log(ts(), 'FINAL:', JSON.stringify(end, null, 2))
  await page.screenshot({ path: `${SHOT_DIR}/1-第二轮回答.png`, fullPage: false })
  console.log(ts(), 'ROUND2_RESULT=', JSON.stringify({ round2, asstCount: end.asst.length, realCount: end.asst.filter((a) => a.len > 20).length, queueText: end.queueText, interjectTag: end.interjectTag }))
  console.log('LOGS:\n' + logs.join('\n'))
  await b.close()
})()
