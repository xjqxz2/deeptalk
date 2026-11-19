import { chromium } from '/home/xjqxz/Workplace/xhs/web/node_modules/playwright/index.mjs';
import fs from 'node:fs';
import path from 'node:path';

const files = process.argv.slice(2);
const outDir = 'docs/design/previews';
fs.mkdirSync(outDir, { recursive: true });
const browser = await chromium.launch({
  executablePath: '/home/xjqxz/.cache/ms-playwright/chromium_headless_shell-1248/chrome-headless-shell-linux64/chrome-headless-shell',
  args: ['--no-sandbox'],
});
for (const file of files) {
  const abs = path.resolve(file);
  const svg = fs.readFileSync(abs, 'utf8');
  const m = svg.match(/width="(\d+)" height="(\d+)"/);
  const w = Number(m[1]), h = Number(m[2]);
  const page = await (await browser.newContext({ viewport: { width: w, height: h }, deviceScaleFactor: 1.5 })).newPage();
  await page.goto('file://' + abs);
  await page.waitForTimeout(250);
  const png = path.join(outDir, path.basename(file, '.svg') + '.png');
  await page.screenshot({ path: png });
  await page.close();
  console.log('rendered', png, w, h);
}
await browser.close();
