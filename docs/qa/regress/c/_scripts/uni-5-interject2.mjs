import { launch, trackConsole, shot, ACCOUNT } from './lib.mjs'
const UNI = 'http://127.0.0.1:5331/'
const browser = await launch()
const ctx = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 2, isMobile: true, hasTouch: true, colorScheme: 'light' })
const page = await ctx.newPage()
const con = trackConsole(page, 'uni')
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
  lastAssistantCls: (() => { const a=[...document.querySelectorAll('.message.assistant')]; return a.length? a[a.length-1].className : '' })(),
  lastText: (() => { const a=[...document.querySelectorAll('.message.assistant')]; if(!a.length) return ''; return (a[a.length-1].innerText||'').replace(/\s+/g,' ').slice(0,120) })(),
  notices: [...document.querySelectorAll('.notice, .error-box, .error-text')].map(e=>e.textContent.trim().slice(0,60)),
}))
const out = []
for (let run=1; run<=2; run++) {
  await page.locator('button.menu-btn').click(); await page.waitForTimeout(700)
  await page.locator('.sidebar-body .new-chat').click(); await page.waitForTimeout(1200)
  await ta.fill('用一句话说明 SSE 是什么')
  await page.locator('.send-btn').click()
  let gen=false
  for (let i=0;i<40;i++){ await page.waitForTimeout(400); const s=await st(); if(s.gen){gen=true;break} }
  if(!gen){ out.push({run,gen:false}); continue }
  await ta.fill('补充一句'); await page.waitForTimeout(120)
  await ta.press('Enter')
  let q=false
  for (let i=0;i<12;i++){ await page.waitForTimeout(300); const s=await st(); if(s.queued||s.pending){q=true;break} }
  const before = await st()
  const timeline = []
  let second=false
  for (let i=0;i<90;i++){
    await page.waitForTimeout(1000)
    const s = await st()
    if (i%10===0) timeline.push({t:i, gen:s.gen, a:s.assistants, q:s.queued, cls:s.lastAssistantCls, text:s.lastText.slice(0,40)})
    if (s.assistants>before.assistants) { second=true; break }
  }
  const fin = await st()
  out.push({run, q, before:{a:before.assistants}, second, fin, timeline})
  await shot(page, `_uni-interject2-run${run}`)
  console.log('RUN', run, JSON.stringify({q, second, fin, timeline}))
}
console.log('RESULT:', JSON.stringify(out, null, 1))
console.log('CONSOLE:', JSON.stringify(con.events.slice(0,20)))
await browser.close()
