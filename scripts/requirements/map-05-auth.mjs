import {
  C, text, rect, circle, line, icon, brandMark, pageHeader, callout,
  chip, writeSvg,
} from './req-lib.mjs';

const W = 1800;
const H = 1160;

function field(x, y, label, value, opts = {}) {
  const { iconName = 'mail', helper, password = false, focus = false } = opts;
  let s = text(x, y, label, { size: 10.2, fill: C.body, weight: 600 });
  s += rect(x, y + 10, 320, 44, { fill: '#FBFCFE', stroke: focus ? '#AFC0FF' : '#DCE2EC', rx: 9 });
  if (focus) s += rect(x, y + 10, 3, 44, { fill: C.primary, rx: 1.5 });
  s += icon(iconName, x + 15, y + 23, 16, C.subtle, 1.7);
  s += text(x + 39, y + 38, value, { size: 11.3, fill: value.startsWith('you@') || value.startsWith('至少') || password ? C.subtle : C.ink2, family: password ? 'monospace' : undefined });
  if (password) s += icon('eye', x + 292, y + 23, 16, C.subtle, 1.6);
  if (helper) s += text(x, y + 72, helper, { size: 8.8, fill: C.subtle });
  return s;
}

function loginScreen(x0, y0) {
  const clip = 'authClipLogin';
  let s = `<defs><clipPath id="${clip}"><rect x="${x0}" y="${y0}" width="820" height="820" rx="20"/></clipPath></defs>`;
  s += rect(x0, y0, 820, 820, { fill: '#FCFDFF', rx: 20, stroke: '#DCE1E9', filter: 'shadow' });
  s += `<g clip-path="url(#${clip})">`;
  s += circle(x0 + 710, y0 + 62, 150, { fill: '#EEF2FF', opacity: 0.78 });
  s += circle(x0 + 70, y0 + 780, 128, { fill: '#F4F7FF', opacity: 0.92 });
  s += brandMark(x0 + 28, y0 + 26, 28);
  s += text(x0 + 66, y0 + 45, 'DeepTalk', { size: 16, weight: 730 });
  s += text(x0 + 66, y0 + 61, 'ACCOUNT ACCESS', { size: 7.6, fill: C.subtle, weight: 600, spacing: 1.4 });
  s += chip(x0 + 668, y0 + 24, 124, '无需邮箱验证', { h: 24, size: 8.8, fill: C.greenSoft, color: '#027A48', dot: true });

  s += rect(x0 + 210, y0 + 150, 400, 500, { fill: '#FFFFFF', stroke: '#E7EBF2', rx: 18, filter: 'softShadow' });
  s += circle(x0 + 250, y0 + 190, 18, { fill: '#EEF2FF' });
  s += icon('user', x0 + 241, y0 + 181, 18, C.primary, 1.7);
  s += text(x0 + 250, y0 + 236, '欢迎回来', { size: 25, weight: 730, spacing: -0.5 });
  s += text(x0 + 250, y0 + 260, '登录后继续你的 DeepTalk 工作台。', { size: 10.4, fill: C.muted });

  s += field(x0 + 250, y0 + 300, '邮箱', 'you@company.com', { focus: true });
  s += text(x0 + 570, y0 + 378, '忘记密码？', { size: 9.2, fill: C.primary, weight: 600, anchor: 'end' });
  s += field(x0 + 250, y0 + 378, '密码', '••••••••••', { iconName: 'lock', password: true });

  s += rect(x0 + 250, y0 + 470, 320, 46, { fill: C.primary, rx: 9 });
  s += text(x0 + 410, y0 + 499, '登录', { size: 12.5, fill: '#FFFFFF', weight: 650, anchor: 'middle', spacing: 1.2 });
  s += circle(x0 + 260, y0 + 540, 3, { fill: C.green });
  s += text(x0 + 270, y0 + 544, '登录后进入你的对话空间', { size: 9.1, fill: C.muted });
  s += line(x0 + 250, y0 + 566, x0 + 570, y0 + 566, { stroke: '#EEF0F4' });
  s += text(x0 + 410, y0 + 598, '还没有账号？', { size: 10.2, fill: C.muted, anchor: 'middle' });
  s += text(x0 + 487, y0 + 598, '立即注册', { size: 10.2, fill: C.primary, weight: 650 });

  s += text(x0 + 410, y0 + 700, '无需验证邮件 · 注册后直接使用', { size: 9.2, fill: C.subtle, anchor: 'middle' });
  s += line(x0 + 280, y0 + 746, x0 + 540, y0 + 746, { stroke: '#E8ECF3' });
  s += text(x0 + 280, y0 + 778, 'DeepTalk  ·  PERSONAL AI', { size: 8.2, fill: C.faint, spacing: 0.8 });
  s += text(x0 + 540, y0 + 778, '隐私 · 帮助', { size: 8.8, fill: C.faint, anchor: 'end' });
  s += '</g>';
  return s;
}

function registerScreen(x0, y0) {
  const clip = 'authClipRegister';
  let s = `<defs><clipPath id="${clip}"><rect x="${x0}" y="${y0}" width="820" height="820" rx="20"/></clipPath></defs>`;
  s += rect(x0, y0, 820, 820, { fill: '#FCFDFF', rx: 20, stroke: '#DCE1E9', filter: 'shadow' });
  s += `<g clip-path="url(#${clip})">`;
  s += circle(x0 + 112, y0 + 62, 156, { fill: '#F0F4FF', opacity: 0.9 });
  s += circle(x0 + 744, y0 + 748, 136, { fill: '#EEF2FF', opacity: 0.82 });
  s += brandMark(x0 + 28, y0 + 26, 28);
  s += text(x0 + 66, y0 + 45, 'DeepTalk', { size: 16, weight: 730 });
  s += text(x0 + 66, y0 + 61, 'CREATE ACCOUNT', { size: 7.6, fill: C.subtle, weight: 600, spacing: 1.4 });
  s += chip(x0 + 646, y0 + 24, 146, '注册即登录 · 无验证', { h: 24, size: 8.5, fill: C.greenSoft, color: '#027A48', dot: true });

  s += rect(x0 + 210, y0 + 110, 400, 585, { fill: '#FFFFFF', stroke: '#E7EBF2', rx: 18, filter: 'softShadow' });
  s += text(x0 + 250, y0 + 170, '创建账号', { size: 25, weight: 730, spacing: -0.5 });
  s += text(x0 + 250, y0 + 194, '使用邮箱注册，创建后直接进入工作台。', { size: 10.4, fill: C.muted });

  s += field(x0 + 250, y0 + 238, '邮箱', 'you@company.com', { focus: true });
  s += field(x0 + 250, y0 + 320, '密码', '至少 8 位，包含字母与数字', { iconName: 'lock', password: true });
  s += text(x0 + 250, y0 + 407, '确认密码', { size: 10.2, fill: C.body, weight: 600 });
  s += rect(x0 + 250, y0 + 417, 320, 44, { fill: '#FBFCFE', stroke: '#DCE2EC', rx: 9 });
  s += icon('lock', x0 + 265, y0 + 430, 16, C.subtle, 1.7);
  s += text(x0 + 289, y0 + 445, '••••••••••', { size: 11.3, fill: C.subtle, family: 'monospace' });
  s += icon('eye', x0 + 542, y0 + 430, 16, C.subtle, 1.6);

  s += icon('check', x0 + 251, y0 + 480, 12, C.green, 2.2);
  s += text(x0 + 269, y0 + 490, '至少 8 位', { size: 9.2, fill: C.muted });
  s += icon('check', x0 + 351, y0 + 480, 12, C.green, 2.2);
  s += text(x0 + 369, y0 + 490, '字母 + 数字', { size: 9.2, fill: C.muted });
  s += icon('check', x0 + 469, y0 + 480, 12, C.green, 2.2);
  s += text(x0 + 487, y0 + 490, '无需邮箱验证', { size: 9.2, fill: C.muted });

  s += rect(x0 + 250, y0 + 516, 320, 46, { fill: C.primary, rx: 9 });
  s += text(x0 + 410, y0 + 545, '创建账号并进入', { size: 12.3, fill: '#FFFFFF', weight: 650, anchor: 'middle' });
  s += text(x0 + 410, y0 + 590, '注册后直接进入对话', { size: 9.2, fill: C.subtle, anchor: 'middle' });
  s += line(x0 + 250, y0 + 614, x0 + 570, y0 + 614, { stroke: '#EEF0F4' });
  s += text(x0 + 410, y0 + 648, '已有账号？', { size: 10.2, fill: C.muted, anchor: 'middle' });
  s += text(x0 + 480, y0 + 648, '去登录', { size: 10.2, fill: C.primary, weight: 650 });

  s += text(x0 + 410, y0 + 730, '不发送验证邮件，注册成功即创建会话', { size: 9.2, fill: C.subtle, anchor: 'middle' });
  s += line(x0 + 280, y0 + 762, x0 + 540, y0 + 762, { stroke: '#E8ECF3' });
  s += text(x0 + 280, y0 + 794, 'DeepTalk  ·  PERSONAL AI', { size: 8.2, fill: C.faint, spacing: 0.8 });
  s += text(x0 + 540, y0 + 794, '隐私 · 帮助', { size: 8.8, fill: C.faint, anchor: 'end' });
  s += '</g>';
  return s;
}

const x1 = 64;
const x2 = 916;
const y0 = 150;

let content = pageHeader({
  title: '登录与注册 · 邮箱密码',
  subtitle: 'C 端产品 · 无邮箱验证 · 注册即登录 · 用户数据按账号隔离',
  code: 'REQ-05',
});

content += loginScreen(x1, y0);
content += registerScreen(x2, y0);

// Screen callouts
content += callout(1, x1 + 620, y0 + 236);
content += callout(2, x2 + 620, y0 + 170);
content += callout(3, x2 + 650, y0 + 60);
content += callout(4, x1 + 620, y0 + 544);
content += callout(5, x2 + 620, y0 + 596);

// Bottom rules band
const bandX = 64;
const bandY = 998;
const bandW = 1672;
const bandH = 134;
content += rect(bandX, bandY, bandW, bandH, { fill: '#FFFFFF', stroke: '#E0E5EC', rx: 16, filter: 'softShadow' });
content += text(bandX + 24, bandY + 31, '账号与安全规则', { size: 16, weight: 720 });
content += chip(bandX + bandW - 82, bandY + 14, 58, 'AUTH / P0', { h: 22, size: 9, fill: C.primarySoft, color: C.primary });
content += line(bandX + 24, bandY + 44, bandX + bandW - 24, bandY + 44, { stroke: C.line });

const rules = [
  { n: 1, title: '账号模型', lines: ['邮箱唯一，密码 Argon2id 哈希。', '首个注册账号为平台管理员。', '其余用户注册即用，无需邀请码。'] },
  { n: 2, title: '注册即登录', lines: ['不发送验证邮件，也不做邮箱验证。', '注册成功直接创建会话并进入工作台。', '后台可关闭注册，关闭后仅管理员建号。'] },
  { n: 3, title: '登录安全', lines: ['HttpOnly + SameSite 会话，7 天有效。', '连续 5 次失败锁定 10 分钟。', '错误提示不区分邮箱不存在或密码错误。'] },
  { n: 4, title: '密码与重置', lines: ['至少 8 位，包含字母与数字。', '无邮件找回；管理员或 CLI 强制重置。', '改密后旧会话立即失效。'] },
  { n: 5, title: 'C 端账号', lines: ['Web 只认邮箱账号；普通用户不接触 API Key。', '用户数据按账号隔离，上游 Key 永不下发。', '同一邮箱不可重复注册。'] },
];
const colW = 312;
const colGap = 14;
const startX = bandX + 24;
for (let i = 0; i < rules.length; i++) {
  const r = rules[i];
  const x = startX + i * (colW + colGap);
  content += circle(x + 9, bandY + 66, 9.5, { fill: C.primary });
  content += text(x + 9, bandY + 69.5, String(r.n), { size: 9.3, fill: '#FFFFFF', weight: 700, anchor: 'middle' });
  content += text(x + 27, bandY + 71, r.title, { size: 11.5, weight: 700, fill: C.ink2 });
  let yy = bandY + 90;
  for (const ln of r.lines) {
    content += text(x + 27, yy, ln, { size: 9.5, fill: C.muted });
    yy += 14.2;
  }
}

writeSvg('docs/requirements/05-auth.svg', content, W, H);
