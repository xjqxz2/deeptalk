import { launch, trackConsole, shot, ACCOUNT } from './lib.mjs'
const UNI = 'http://127.0.0.1:5331/'
const browser = await launch()
const ctx = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true, colorScheme: 'light' })
const page = await ctx.newPage()
const con = trackConsole(page, 'uni')
const out = {}
await page.goto(UNI, { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(2500)
await page.locator('input[type="email"]').fill(ACCOUNT.email)
await page.locator('input[type="password"]').fill(ACCOUNT.password)
await page.locator('button.submit-btn').click()
await page.waitForTimeout(6500)

const ta = page.locator('textarea.composer-input')
const st = async () => page.evaluate(() => ({
  gen: !!document.querySelector('.stop-btn'),
  users: document.querySelectorAll('.message.user').length,
  assistants: document.querySelectorAll('.message.assistant').length,
  queued: document.querySelectorAll('.message.queued').length,
  pending: (document.querySelector('.interject-pending')||{}).textContent?.trim() || '',
  tags: document.querySelectorAll('.interject-tag').length,
  lastCls: document.querySelector('.message.assistant:last-of-type')?.className || '',
  stats: [...document.querySelectorAll('.message.assistant .stats')].map(e=>e.textContent.trim()).slice(-3),
}))

// ===== 停止生成 =====
await page.locator('button.menu-btn').click(); await page.waitForTimeout(700)
await page.locator('.sidebar-body .new-chat').click(); await page.waitForTimeout(1200)
await ta.fill('写一篇 500 字的短文介绍 SSE')
await page.locator('.send-btn').click()
let sentU = 0
for (let i=0;i<40;i++){ await page.waitForTimeout(500); const s=await st(); if(s.users>0){sentU=s.users;break} }
for (let i=0;i<20;i++){ await page.waitForTimeout(500); if((await st()).gen) break }
await page.waitForTimeout(3000)
await page.locator('.stop-btn').click()
await page.waitForTimeout(2500)
const afterStop = await st()
out.stop = { sentUsers: sentU, ...afterStop }
await shot(page, 'BUG-236528F2-已停止生成')
out.stopVisible = await page.locator('.message.assistant .stats', { hasText: '已停止生成' }).count()
console.log('STOP:', JSON.stringify(out.stop), 'visible=', out.stopVisible)

// ===== 插话 =====
await page.locator('button.menu-btn').click(); await page.waitForTimeout(700)
await page.locator('.sidebar-body .new-chat').click(); await page.waitForTimeout(1200)
await ta.fill('用一句话说明 SSE 是什么')
await page.locator('.send-btn').click()
let gen2 = false
for (let i=0;i<40;i++){ await page.waitForTimeout(500); const s=await st(); if(s.gen){gen2=true;break} }
out.interject = { genStarted: gen2 }
if (gen2) {
  out.interject.hasSendBtn = await page.locator('.send-btn').count()
  out.interject.hasStopBtn = await page.locator('.stop-btn').count()
  await ta.fill('补充一句')
  await page.waitForTimeout(150)
  await ta.press('Enter')       // 触屏无插话按钮，只能用键盘回车
  for (let i=0;i<14;i++){ await page.waitForTimeout(400); const s=await st(); if(s.queued||s.pending){ out.interject.queued=true; out.interject.pendingText=s.pending; break } }
  await shot(page, 'BUG-065C3420-插话排队')
  const before = await st()
  let second=false
  for (let i=0;i<90;i++){ await page.waitForTimeout(1000); const s=await st(); if(s.assistants>before.assistants){second=true;break} }
  out.interject.secondRound = second
  for (let i=0;i<60;i++){ await page.waitForTimeout(1000); if(!(await st()).gen) break }
  out.interject.answers = await page.evaluate(()=>[...document.querySelectorAll('.message.assistant .prose')].map(e=>({len:(e.innerText||'').length, head:(e.innerText||'').slice(0,60)})))
  await page.locator('.messages-scroll').evaluate(el=>{el.scrollTop=el.scrollHeight})
  await page.waitForTimeout(500)
  await shot(page, 'BUG-065C3420-插话后第二轮')
}
console.log('RESULT:', JSON.stringify(out, null, 1))
console.log('CONSOLE:', JSON.stringify(con.events.slice(0,20), null, 1))
await browser.close()
