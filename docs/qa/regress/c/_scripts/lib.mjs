import path from 'node:path'
import { createRequire } from 'node:module'
const require = createRequire('/home/xjqxz/Workplace/deeptalk/ui/package.json')
const { chromium } = require('playwright-core')

const deps = path.resolve('/home/xjqxz/Workplace/deeptalk/.browser-deps')
process.env.LD_LIBRARY_PATH = [path.join(deps,'root/usr/lib/x86_64-linux-gnu'),path.join(deps,'root/usr/lib'),path.join(deps,'root/lib/x86_64-linux-gnu'),process.env.LD_LIBRARY_PATH||''].join(':')
process.env.FONTCONFIG_FILE = path.join(deps,'fonts.conf')
process.env.FONTCONFIG_PATH = deps

export const SHOT_DIR = '/home/xjqxz/Workplace/deeptalk/docs/qa/regress/c'
export const ACCOUNT = { email: 'xjqxz_2@163.com', password: 'Ml87148002' }

export async function launch() {
  return chromium.launch({
    executablePath: `${process.env.HOME}/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell`,
    args: ['--no-sandbox'],
  })
}

export function trackConsole(page, label) {
  const events = []
  page.on('pageerror', (e) => events.push({ kind: 'pageerror', text: String(e && e.message || e) }))
  page.on('console', (m) => {
    if (m.type() === 'error') events.push({ kind: 'console.error', text: m.text().slice(0, 500) })
  })
  page.on('response', (r) => {
    const s = r.status()
    if (s >= 400) events.push({ kind: 'http' + s, text: `${r.request().method()} ${r.url().slice(0, 200)}` })
  })
  return { label, events }
}

export async function shot(page, name) {
  const p = path.join(SHOT_DIR, name + '.png')
  await page.screenshot({ path: p, fullPage: false })
  return p
}

export function sleep(ms) { return new Promise(r => setTimeout(r, ms)) }
