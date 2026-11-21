import path from 'node:path'
import { chromium } from 'playwright-core'
const deps = path.resolve('/home/xjqxz/Workplace/deeptalk/.browser-deps')
process.env.LD_LIBRARY_PATH = [path.join(deps,'root/usr/lib/x86_64-linux-gnu'),path.join(deps,'root/usr/lib'),path.join(deps,'root/lib/x86_64-linux-gnu'),process.env.LD_LIBRARY_PATH||''].join(':')
process.env.FONTCONFIG_FILE = path.join(deps,'fonts.conf'); process.env.FONTCONFIG_PATH = deps
const browser = await chromium.launch({ executablePath: `${process.env.HOME}/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell`, args:['--no-sandbox'] })
const page = await browser.newPage({ viewport: { width: 390, height: 844 } })
const bad = []
page.on('response', r => { if (r.status() >= 400) bad.push(`${r.status()} ${r.url().slice(0,90)}`) })
await page.goto('http://127.0.0.1:5303/', { waitUntil: 'networkidle' })
await page.waitForTimeout(800)
await page.locator('input').first().fill('qa-test@deeptalk.local')
await page.locator('input').nth(1).fill('Qa-pass-2026')
await page.locator('.submit-btn, .submit').first().click()
await page.waitForTimeout(2500)
const storage = await page.evaluate(() => { try { return Object.keys(localStorage).filter(k => k.includes('sign')) } catch { return [] } })
const body = await page.locator('body').innerText()
console.log(JSON.stringify({ loggedIn: body.includes('继续追问') || body.includes('新对话'), signKeys: storage, bad: bad.slice(0,3) }))
await browser.close()
