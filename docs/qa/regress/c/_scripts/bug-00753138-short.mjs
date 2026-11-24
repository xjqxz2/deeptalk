import { launch, trackConsole, shot, ACCOUNT } from './lib.mjs'
const browser = await launch()
const ctx = await browser.newContext({ viewport: { width: 1440, height: 900 } })
const page = await ctx.newPage()
const con = trackConsole(page, 'pwa')
await page.goto('https://deeptalk.peon.top/login', { waitUntil: 'domcontentloaded' })
await page.waitForTimeout(1200)
await page.locator('input[placeholder="you@company.com"]').fill(ACCOUNT.email)
await page.locator('input[placeholder="请输入密码"]').fill(ACCOUNT.password)
await page.locator('button.submit-btn').click()
await page.waitForURL((u) => !u.pathname.startsWith('/login'), { timeout: 30000 })
await page.waitForTimeout(2000)
const state = async () => page.evaluate(() => ({
  gen: !!document.querySelector('button[aria-label="插话"]'),
  users: document.querySelectorAll('.message.user').length,
  assistants: document.querySelectorAll('.message.assistant').length,
  queued: document.querySelectorAll('.message.queued').length,
  pendingText: (document.querySelector('.interject-pending')||{}).textContent?.trim() || '',
}))
async function setText(t){ await page.evaluate((v)=>{const el=document.querySelector('textarea[aria-label="消息输入框"]');const s=Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype,'value').set;s.call(el,v);el.dispatchEvent(new Event('input',{bubbles:true}))}, t) }
const out = []
for (let run=1; run<=2; run++) {
  await page.locator('button.new-chat').click(); await page.waitForTimeout(900)
  await setText('用一句话说明 SSE 是什么'); await page.waitForTimeout(120)
  await page.locator('button[aria-label="发送消息"]').click()
  let gen=false
  for (let i=0;i<40;i++){ await page.waitForTimeout(400); const s=await state(); if(s.gen){gen=true;break} }
  if(!gen){ out.push({run,gen:false}); continue }
  await setText('补充一句'); await page.waitForTimeout(100)
  await page.locator('button[aria-label="插话"]').click()
  let q=false
  for (let i=0;i<10;i++){ await page.waitForTimeout(300); const s=await state(); if(s.queued||s.pendingText){q=true;break} }
  const before = await state()
  let second=false
  for (let i=0;i<90;i++){ await page.waitForTimeout(1000); const s=await state(); if(s.assistants>before.assistants){second=true;break} }
  for (let i=0;i<60;i++){ await page.waitForTimeout(1000); const s=await state(); if(!s.gen) break }
  const answers = await page.evaluate(()=>[...document.querySelectorAll('.message.assistant')].map(e=>{const t=(e.textContent||'').replace(/\s+/g,' ').trim();return {len:t.length,fail:/生成回答失败/.test(t),text:t.slice(0,100)}}))
  out.push({run,q,second,secondAnswer: answers[1]&&{len:answers[1].len,fail:answers[1].fail,text:answers[1].text}})
  await page.evaluate(()=>{const m=document.querySelector('.chat-content,.messages,.chat-main');if(m)m.scrollTop=m.scrollHeight})
  await shot(page, `_short-interject-run${run}`)
}
console.log('SHORT INTERJECT:', JSON.stringify(out,null,1))
console.log('CONSOLE:', JSON.stringify(con.events.filter(e=>!/auth\/me/.test(e.text))))
await browser.close()
