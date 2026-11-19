import fs from 'node:fs';
import { C, SANS, MONO, text, rect, circle, line, path, icon, brandMark } from '../requirements/req-lib.mjs';

const n = (v) => Number.isInteger(v) ? String(v) : String(Number(v.toFixed(2)));
const wrap = (tag, ...parts) => '<' + tag + '>' + parts.join('') + '</' + tag + '>';
export const g = (...parts) => wrap('g', ...parts);

export const CANVAS_W = 1600;
export const CANVAS_H = 1030;
export const FX = 80;
export const FY = 96;
export const FW = 1440;
export const FH = 900;

export function defs() {
  return [
    '<defs>',
    '<pattern id="dsgGrid" width="26" height="26" patternUnits="userSpaceOnUse"><circle cx="1" cy="1" r="1" fill="#D6DCE6" opacity="0.65"/></pattern>',
    '<linearGradient id="brandGrad" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#6D86FF"/><stop offset="1" stop-color="#3F5BF6"/></linearGradient>',
    '<linearGradient id="authBlob" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#E7EDFF"/><stop offset="1" stop-color="#F5F8FF"/></linearGradient>',
    '<filter id="dsgShadow" x="-20%" y="-20%" width="140%" height="150%"><feDropShadow dx="0" dy="14" stdDeviation="26" flood-color="#101828" flood-opacity="0.09"/></filter>',
    '<filter id="dsgSoft" x="-30%" y="-30%" width="160%" height="170%"><feDropShadow dx="0" dy="6" stdDeviation="9" flood-color="#101828" flood-opacity="0.05"/></filter>',
    '<filter id="dsgPop" x="-30%" y="-30%" width="160%" height="170%"><feDropShadow dx="0" dy="10" stdDeviation="18" flood-color="#101828" flood-opacity="0.10"/></filter>',
    '</defs>',
  ].join('');
}

export function canvasOpen(w = CANVAS_W, h = CANVAS_H) {
  return '<svg xmlns="http://www.w3.org/2000/svg" width="' + w + '" height="' + h + '" viewBox="0 0 ' + w + ' ' + h + '" role="img">'
    + defs()
    + '<rect width="' + w + '" height="' + h + '" fill="#EDF0F5"/>'
    + '<rect width="' + w + '" height="' + h + '" fill="url(#dsgGrid)"/>';
}

/* sheetHeader is defined at the bottom of this file with the spec-sheet layout. */

export function frame(x, y, w, h, id, inner, o = {}) {
  const rx = o.rx ?? 20;
  return g(
    o.shadow === false ? '' : rect(x, y, w, h, { fill: '#FFFFFF', rx, filter: 'dsgShadow' }),
    '<clipPath id="' + id + '">' + rect(x, y, w, h, { rx }) + '</clipPath>',
    '<g clip-path="url(#' + id + ')">' + '<g transform="translate(' + n(x) + ' ' + n(y) + ')">' + inner + '</g>' + '</g>',
    rect(x, y, w, h, { fill: 'none', stroke: '#DCE2EB', rx }),
  );
}

export function btn(x, y, w, h, label, o = {}) {
  const primary = !!o.primary;
  const iconName = o.iconName;
  const iconSize = o.iconSize ?? 16;
  const size = o.size ?? 12.5;
  const radius = o.radius ?? 10;
  const fill = primary ? C.primary : '#FFFFFF';
  const color = primary ? '#FFFFFF' : C.ink2;
  const stroke = primary ? undefined : (o.stroke || '#DDE2EA');
  const gap = iconName ? 8 : 0;
  const iconW = iconName ? iconSize : 0;
  const contentW = iconW + gap + label.length * size * 1.0;
  const startX = x + (w - contentW) / 2;
  const inner = (iconName ? icon(iconName, startX, y + (h - iconSize) / 2, iconSize, primary ? '#fff' : C.ink2, 2) : '')
    + text(startX + iconW + gap, y + h / 2 + size * 0.36, label, { size, weight: 600, fill: color });
  return g(
    rect(x, y, w, h, { fill, rx: radius, stroke }),
    inner,
  );
}

export function iconBtn(x, y, s, name, o = {}) {
  const color = o.color || C.muted;
  const size = o.size ?? 18;
  const bg = o.bg;
  const stroke = o.stroke;
  const radius = o.radius ?? 9;
  const sw = o.sw ?? 1.8;
  return g(
    bg ? rect(x, y, s, s, { fill: bg, rx: radius, stroke }) : '',
    icon(name, x + (s - size) / 2, y + (s - size) / 2, size, color, sw),
  );
}

export function pill(x, y, w, h, label, o = {}) {
  const fill = o.fill || C.primarySoft;
  const color = o.color || C.primary;
  const size = o.size ?? 10.5;
  const weight = o.weight ?? 600;
  const dot = !!o.dot;
  const iconName = o.iconName;
  const iconSize = o.iconSize ?? 13;
  const dotW = dot ? 12 : 0;
  const icW = iconName ? iconSize + 6 : 0;
  const total = dotW + icW + label.length * size * 1.0;
  let cx = x + (w - total) / 2;
  let inner = '';
  if (dot) { inner += circle(cx + 3, y + h / 2, 3, { fill: color }); cx += dotW; }
  if (iconName) { inner += icon(iconName, cx, y + (h - iconSize) / 2, iconSize, color, 1.9); cx += icW; }
  inner += text(cx, y + h / 2 + size * 0.36, label, { size, weight, fill: color });
  return g(rect(x, y, w, h, { fill, stroke: o.stroke, rx: h / 2 }), inner);
}

export function avatar(cx, cy, r, initials, o = {}) {
  const fill = o.fill || '#E9EDFB';
  const color = o.color || C.primary2;
  const size = o.size ?? r * 0.9;
  return g(
    circle(cx, cy, r, { fill }),
    text(cx, cy + size * 0.35, initials, { size, weight: 650, fill: color, anchor: 'middle' }),
  );
}

export function progress(x, y, w, h, pct, o = {}) {
  const fill = o.fill || '#EDF0F4';
  const color = o.color || C.primary;
  return g(
    rect(x, y, w, h, { fill, rx: h / 2 }),
    rect(x, y, Math.max(h, w * pct), h, { fill: color, rx: h / 2 }),
  );
}

export function toggle(x, y, on, o = {}) {
  const w = o.w ?? 34;
  const h = o.h ?? 20;
  return g(
    rect(x, y, w, h, { fill: on ? C.primary : '#E4E7EC', rx: h / 2 }),
    circle(x + (on ? w - h / 2 : h / 2), y + h / 2, h / 2 - 2.5, { fill: '#fff' }),
  );
}

export function field(x, y, w, label, o = {}) {
  const h = o.h ?? 44;
  const iconName = o.iconName ?? 'mail';
  const value = o.value || '';
  const placeholder = o.placeholder || '';
  const focus = !!o.focus;
  const radius = o.radius ?? 10;
  const color = value ? C.ink : C.faint;
  let inner = '';
  if (iconName) inner += icon(iconName, x + 14, y + (h - 16) / 2, 16, focus ? C.primary : C.subtle, 1.7);
  inner += text(x + 40, y + h / 2 + 4.5, value || placeholder, { size: 12.5, fill: color, family: o.mono ? MONO : SANS });
  if (focus) inner += rect(x + 2, y + 9, 2, h - 18, { fill: C.primary, rx: 1 });
  if (o.trailing) inner += o.trailing;
  return g(
    label ? text(x, y - 10, label, { size: 11.4, weight: 600, fill: C.ink2 }) : '',
    focus ? rect(x - 3, y - 3, w + 6, h + 6, { fill: 'none', rx: radius + 3, stroke: '#C9D4FF', sw: 2 }) : '',
    rect(x, y, w, h, { fill: '#FFFFFF', rx: radius, stroke: focus ? C.primary : '#DDE2EA', sw: focus ? 1.6 : 1 }),
    inner,
  );
}

/* ---------- chat shell ---------- */

export const CHAT_HISTORY = [
  { group: '今天', items: ['Go JWT 中间件设计', 'React 流式消息渲染', 'DeepSeek 风格交互清单'] },
  { group: '7 天内', items: ['用户会话隔离方案', 'SSE 超时与重连策略'] },
  { group: '30 天内', items: ['OpenAI 兼容接口调研', 'LangGraph 工作流笔记'] },
];

export function chatSidebar(o = {}) {
  const active = o.active || 'Go JWT 中间件设计';
  const usage = o.usage ?? 0.32;
  const parts = [
    rect(0, 0, 280, 900, { fill: '#FAFBFC' }),
    line(280, 0, 280, 900, { stroke: '#EDEFF4' }),
    brandMark(20, 18, 28),
    text(58, 38, 'DeepTalk', { size: 15, weight: 720, spacing: -0.2 }),
    iconBtn(232, 17, 30, 'panelLeft', { size: 17 }),
    rect(16, 62, 248, 40, { fill: '#FFFFFF', rx: 11, stroke: '#E3E7EE', filter: 'dsgSoft' }),
    icon('plus', 32, 74, 16, C.primary2, 2),
    text(56, 87, '新建对话', { size: 12.6, weight: 620, fill: C.ink2 }),
    text(246, 87, 'Ctrl K', { size: 9.6, fill: C.faint, anchor: 'end', family: MONO }),
    rect(16, 112, 248, 36, { fill: '#F1F3F7', rx: 10 }),
    icon('search', 30, 122, 15, C.subtle, 1.8),
    text(52, 135, '搜索对话', { size: 11.8, fill: C.subtle }),
  ];
  let y = 178;
  for (const grp of CHAT_HISTORY) {
    parts.push(text(20, y, grp.group, { size: 10.2, weight: 650, fill: C.subtle, spacing: 0.6 }));
    y += 12;
    for (const item of grp.items) {
      const on = item === active;
      if (on) parts.push(rect(12, y + 3, 256, 32, { fill: '#EEF2FF', rx: 9 }));
      parts.push(icon('message', 26, y + 11, 14, on ? C.primary2 : C.subtle, 1.7));
      parts.push(text(50, y + 22, item, { size: 12.4, fill: on ? C.primary2 : C.body, weight: on ? 620 : 400 }));
      y += 34;
    }
    y += 20;
  }
  const cardY = 730;
  parts.push(
    rect(16, cardY, 248, 74, { fill: '#FFFFFF', rx: 12, stroke: '#E8EBF0' }),
    text(30, cardY + 24, '本月用量', { size: 11.4, weight: 620, fill: C.ink2 }),
    text(250, cardY + 24, '32%', { size: 11.8, weight: 720, fill: C.primary, anchor: 'end' }),
    progress(30, cardY + 34, 220, 6, usage),
    text(30, cardY + 60, '1.28M / 4M tokens', { size: 10.2, fill: C.muted }),
    text(250, cardY + 60, '重置 12 天', { size: 10.2, fill: C.subtle, anchor: 'end' }),
    line(16, 822, 264, 822, { stroke: '#EDEFF4' }),
    avatar(30, 844, 15, 'L'),
    text(56, 840, '林默', { size: 12.2, weight: 620 }),
    text(56, 855, 'lmo@deeptalk.app', { size: 9.6, fill: C.subtle, family: MONO }),
    iconBtn(230, 829, 30, 'settings', { size: 16 }),
  );
  return g(...parts);
}

export function chatTopbar(o = {}) {
  const title = o.title || '新对话';
  const model = o.model || 'DeepSeek-V3.2';
  const connected = o.connected || '已连接';
  const parts = [
    rect(280, 0, 1160, 64, { fill: '#FFFFFF' }),
    line(280, 64, 1440, 64, { stroke: '#EFF1F5' }),
    iconBtn(296, 17, 30, 'panelLeft', { size: 17 }),
    text(334, 39, title, { size: 13.6, weight: 650 }),
  ];
  if (o.draft !== false) {
    parts.push(circle(394, 35, 3, { fill: C.green }), text(404, 39, '草稿已保存', { size: 10.2, fill: C.subtle }));
  }
  parts.push(
    rect(1044, 14, 198, 36, { fill: '#FFFFFF', rx: 10, stroke: '#E3E7EE' }),
    circle(1062, 32, 3.4, { fill: C.primary }),
    text(1074, 36, model, { size: 11.8, weight: 620, fill: C.ink2 }),
    icon('chevronDown', 1212, 26, 13, C.subtle, 2),
    rect(1258, 19, 78, 26, { fill: C.greenSoft, rx: 13 }),
    circle(1271, 32, 3, { fill: C.green }),
    text(1280, 36, connected, { size: 10.2, weight: 600, fill: '#067647' }),
    avatar(1358, 32, 14, 'L'),
    iconBtn(1388, 17, 30, 'more', { size: 18 }),
  );
  return g(...parts);
}

/* ---------- spec sheet ---------- */

export const SHEET = { w: 1800, h: 1080, sx: 32, sy: 142, sw: 1440, sh: 900, rx: 1496, ry: 142, rw: 272, rh: 900 };

export function sheetHeader(o = {}) {
  return g(
    brandMark(32, 26, 30),
    text(72, 45, 'DeepTalk', { size: 18, weight: 730, spacing: -0.3 }),
    text(72, 62, 'DESIGN SPEC · 界面与交互说明', { size: 9, fill: C.subtle, weight: 620, spacing: 1.1 }),
    rect(1604, 26, 76, 24, { fill: '#EEF2FF', rx: 12 }),
    circle(1617, 38, 3.2, { fill: C.primary }),
    text(1626, 42, o.status || '待开发', { size: 10, fill: C.primary2, weight: 600 }),
    text(1768, 41, '2026.11.18  ·  v1', { size: 10.4, fill: C.muted, anchor: 'end' }),
    text(32, 101, o.title || '', { size: 26, weight: 730, spacing: -0.4 }),
    text(32, 122, o.sub || '', { size: 12.2, fill: C.muted }),
    text(1768, 104, o.code || 'DSG-01', { size: 24, fill: '#CBD2DD', weight: 750, anchor: 'end', spacing: 1 }),
    text(1768, 122, o.meta || '1440 × 900 · PC Web', { size: 9.6, fill: C.subtle, anchor: 'end', spacing: 0.4 }),
  );
}

export function specRail(o = {}) {
  const x = SHEET.rx, y = SHEET.ry, w = SHEET.rw, h = SHEET.rh;
  const items = o.items || [];
  const parts = [
    rect(x, y, w, h, { fill: '#FFFFFF', rx: 18, stroke: '#E0E5EC', filter: 'dsgShadow' }),
    text(x + 20, y + 34, '设计说明', { size: 17, weight: 720 }),
    pill(x + w - 78, y + 16, 58, 22, 'UI / UX', { size: 9, fill: C.primarySoft, color: C.primary2 }),
    text(x + 20, y + 56, '1440 × 900 · 8pt 网格', { size: 10.2, fill: C.subtle }),
    line(x + 20, y + 72, x + w - 20, y + 72, { stroke: C.line }),
  ];
  let cy = y + 96;
  for (const item of items) {
    parts.push(circle(x + 25, cy - 4, 9.5, { fill: C.primary }));
    parts.push(text(x + 25, cy - 0.4, String(item.n), { size: 9.4, fill: '#fff', weight: 700, anchor: 'middle' }));
    parts.push(text(x + 44, cy, item.title, { size: 11.8, weight: 700, fill: C.ink2 }));
    let ly = cy + 17.5;
    for (const line of item.lines) {
      parts.push(text(x + 44, ly, line, { size: 10.2, fill: C.muted }));
      ly += 14.6;
    }
    cy = ly + 13;
  }
  const ty = y + 694;
  parts.push(line(x + 20, ty - 22, x + w - 20, ty - 22, { stroke: C.line }));
  parts.push(text(x + 20, ty, '设计令牌', { size: 10, weight: 650, fill: C.subtle, spacing: 1 }));
  const toks = o.tokens || [['主色', C.primary], ['文字', C.ink], ['辅助', C.muted], ['成功', C.green], ['警告', C.amber]];
  toks.forEach((t, i) => {
    const tx = x + 20 + (i % 3) * 82;
    const tyy = ty + 22 + Math.floor(i / 3) * 26;
    parts.push(rect(tx, tyy - 9, 12, 12, { fill: t[1], rx: 3 }));
    parts.push(text(tx + 18, tyy + 1, t[0], { size: 9.4, fill: C.muted }));
  });
  const fy = y + h - 132;
  parts.push(line(x + 20, fy - 14, x + w - 20, fy - 14, { stroke: C.line }));
  parts.push(text(x + 20, fy, '落地约定', { size: 10, weight: 650, fill: C.subtle, spacing: 1 }));
  let ly2 = fy + 20;
  for (const line of o.footerLines || []) {
    parts.push(text(x + 20, ly2, line, { size: 10.2, fill: C.muted }));
    ly2 += 14.8;
  }
  return g(...parts);
}

export function pinLabel(n, x, y) {
  return g(
    circle(x, y, 12.5, { fill: '#FFFFFF', opacity: 0.96 }),
    circle(x, y, 10.2, { fill: C.primary }),
    text(x, y + 3.5, String(n), { size: 10.2, fill: '#fff', weight: 700, anchor: 'middle' }),
  );
}

export function writeSvg(file, body, w = CANVAS_W, h = CANVAS_H) {
  const svg = canvasOpen(w, h) + '\n' + body + '\n</svg>\n';
  fs.writeFileSync(file, svg, 'utf8');
  console.log('wrote ' + file + ' (' + (svg.length / 1024).toFixed(1) + ' KB)');
}
