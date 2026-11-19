import fs from 'node:fs';

export const C = {
  bg: '#EEF1F6',
  canvas: '#F4F6FA',
  ink: '#101828',
  ink2: '#344054',
  body: '#475467',
  muted: '#667085',
  subtle: '#98A2B3',
  faint: '#C4CBD6',
  line: '#EAECF0',
  line2: '#E1E5EB',
  panel: '#F7F8FA',
  panel2: '#F2F4F7',
  white: '#FFFFFF',
  primary: '#4D6BFE',
  primary2: '#3B5BDB',
  primarySoft: '#EEF2FF',
  primarySoft2: '#E6ECFF',
  green: '#12B76A',
  greenSoft: '#ECFDF3',
  amber: '#F79009',
  amberSoft: '#FFFAEB',
  red: '#F04438',
  redSoft: '#FEF3F2',
  codeBg: '#171A21',
  codeHeader: '#20242D',
  codeText: '#D6DEEB',
  codeMuted: '#7F8C98',
};

export const SANS = "Inter, 'PingFang SC', 'Microsoft YaHei', 'Noto Sans SC', sans-serif";
export const MONO = "'SFMono-Regular', Menlo, Consolas, 'Liberation Mono', monospace";

export const APP = { x: 32, y: 142, w: 1440, h: 900 };
export const RAIL = { x: 1496, y: 142, w: 272, h: 900 };

export const esc = (s = '') => String(s)
  .replaceAll('&', '&amp;')
  .replaceAll('<', '&lt;')
  .replaceAll('>', '&gt;')
  .replaceAll('"', '&quot;');

const n = (v) => Number.isInteger(v) ? String(v) : String(Number(v.toFixed(3)));

export function text(x, y, content, o = {}) {
  const {
    size = 12, fill = C.ink, weight = 400, anchor = 'start',
    family = SANS, opacity, spacing, baseline, raw = false, italic = false,
  } = o;
  return `<text x="${n(x)}" y="${n(y)}" font-family="${family}" font-size="${n(size)}" font-weight="${weight}" fill="${fill}" text-anchor="${anchor}"${opacity != null ? ` opacity="${opacity}"` : ''}${spacing != null ? ` letter-spacing="${n(spacing)}"` : ''}${baseline ? ` dominant-baseline="${baseline}"` : ''}${italic ? ' font-style="italic"' : ''}>${raw ? content : esc(content)}</text>`;
}

export function tspan(content, o = {}) {
  const { fill = C.ink, weight = 400, family = SANS, size } = o;
  return `<tspan fill="${fill}" font-weight="${weight}" font-family="${family}"${size ? ` font-size="${n(size)}"` : ''}>${esc(content)}</tspan>`;
}

export function rect(x, y, w, h, o = {}) {
  const { fill = C.white, stroke, rx = 0, sw = 1, opacity, filter, dash } = o;
  return `<rect x="${n(x)}" y="${n(y)}" width="${n(w)}" height="${n(h)}" rx="${n(rx)}" fill="${fill}"${stroke ? ` stroke="${stroke}" stroke-width="${n(sw)}"` : ''}${dash ? ` stroke-dasharray="${dash}"` : ''}${opacity != null ? ` opacity="${opacity}"` : ''}${filter ? ` filter="url(#${filter})"` : ''}/>`;
}

export function circle(cx, cy, r, o = {}) {
  const { fill = 'none', stroke, sw = 1, opacity } = o;
  return `<circle cx="${n(cx)}" cy="${n(cy)}" r="${n(r)}" fill="${fill}"${stroke ? ` stroke="${stroke}" stroke-width="${n(sw)}"` : ''}${opacity != null ? ` opacity="${opacity}"` : ''}/>`;
}

export function line(x1, y1, x2, y2, o = {}) {
  const { stroke = C.line, sw = 1, opacity, dash } = o;
  return `<line x1="${n(x1)}" y1="${n(y1)}" x2="${n(x2)}" y2="${n(y2)}" stroke="${stroke}" stroke-width="${n(sw)}"${dash ? ` stroke-dasharray="${dash}"` : ''}${opacity != null ? ` opacity="${opacity}"` : ''}/>`;
}

export function path(d, o = {}) {
  const { fill = 'none', stroke, sw = 1.8, cap = 'round', join = 'round', opacity } = o;
  return `<path d="${d}" fill="${fill}"${stroke ? ` stroke="${stroke}" stroke-width="${n(sw)}"` : ''} stroke-linecap="${cap}" stroke-linejoin="${join}"${opacity != null ? ` opacity="${opacity}"` : ''}/>`;
}

const ICONS = {
  plus: { s: ['M12 5v14', 'M5 12h14'] },
  search: { s: ['M11 18a7 7 0 1 0 0-14 7 7 0 0 0 0 14Z', 'm16.5 16.5 4.2 4.2'] },
  panelLeft: { s: ['M4 5.5A1.5 1.5 0 0 1 5.5 4h13A1.5 1.5 0 0 1 20 5.5v13a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 4 18.5v-13Z', 'M9 4v16'] },
  chevronDown: { s: ['m7 10 5 5 5-5'] },
  chevronUp: { s: ['m7 14 5-5 5 5'] },
  chevronRight: { s: ['m10 7 5 5-5 5'] },
  chevronLeft: { s: ['m14 7-5 5 5 5'] },
  settings: { s: ['M4 7h16', 'M4 12h16', 'M4 17h16'], c: [[9, 7, 2], [15, 12, 2], [7, 17, 2]] },
  spark: { f: ['M12 2.6l1.9 5.5L19.4 10l-5.5 1.9L12 17.4l-1.9-5.5L4.6 10l5.5-1.9L12 2.6Z', 'M18.6 16.2l.8 2.2 2.2.8-2.2.8-.8 2.2-.8-2.2-2.2-.8 2.2-.8.8-2.2Z'] },
  globe: { s: ['M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Z', 'M3.5 9h17', 'M3.5 15h17', 'M12 3c2.3 2.4 3.5 5.4 3.5 9S14.3 18.6 12 21c-2.3-2.4-3.5-5.4-3.5-9S9.7 5.4 12 3Z'] },
  paperclip: { s: ['M20.8 11.2l-8.2 8.2a5.7 5.7 0 0 1-8.1-8.1l8.5-8.5a3.8 3.8 0 0 1 5.4 5.4l-8.5 8.5a1.9 1.9 0 0 1-2.7-2.7l7.8-7.8'] },
  arrowUp: { s: ['M12 19V5', 'm5 12 7-7 7 7'] },
  arrowRight: { s: ['M5 12h14', 'm13 6 6 6-6 6'] },
  arrowLeft: { s: ['M19 12H5', 'm11 18-6-6 6-6'] },
  stop: { rects: [[7, 7, 10, 10, 2]] },
  copy: { s: ['M8 8h10.5A1.5 1.5 0 0 1 20 9.5V20a1.5 1.5 0 0 1-1.5 1.5H8A1.5 1.5 0 0 1 6.5 20V9.5A1.5 1.5 0 0 1 8 8Z', 'M15.5 8V4.5A1.5 1.5 0 0 0 14 3H5.5A1.5 1.5 0 0 0 4 4.5V16'] },
  refresh: { s: ['M20 11a8 8 0 1 0-2.3 6.7', 'M20 4v5h-5'] },
  thumbUp: { s: ['M7 10v10H4.8A1.8 1.8 0 0 1 3 18.2V11.8A1.8 1.8 0 0 1 4.8 10H7Z', 'M7 10l3.1-6.1A1.5 1.5 0 0 1 11.5 3h.4a1.5 1.5 0 0 1 1.5 1.5V9h4.1a2 2 0 0 1 2 2.4l-1 7A2 2 0 0 1 16.5 20H7'] },
  thumbDown: { s: ['M17 14V4h2.2A1.8 1.8 0 0 1 21 5.8v6.4a1.8 1.8 0 0 1-1.8 1.8H17Z', 'M17 14l-3.1 6.1a1.5 1.5 0 0 1-1.4.9h-.4a1.5 1.5 0 0 1-1.5-1.5V15H6.5a2 2 0 0 1-2-2.4l1-7A2 2 0 0 1 7.5 4H17'] },
  more: { sf: [[5, 12, 1.6], [12, 12, 1.6], [19, 12, 1.6]] },
  share: { s: ['M15 5.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5Z', 'M6 14.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5Z', 'M18 23.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5Z', 'M8.2 11 15.7 7', 'M8.2 13l7.5 4'] },
  key: { s: ['M14.5 14.5 21 21', 'M18.2 17.8 21 15', 'M19.7 19.3 22.5 16.5', 'M9.5 16.5a5.5 5.5 0 1 0 0-11 5.5 5.5 0 0 0 0 11Z', 'M9.5 13.5a2.5 2.5 0 1 0 0-5 2.5 2.5 0 0 0 0 5Z'] },
  users: { s: ['M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2', 'M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z', 'M22 21v-2a4 4 0 0 0-3-3.9', 'M16 3.1a4 4 0 0 1 0 7.8'] },
  server: { s: ['M4 5.5A1.5 1.5 0 0 1 5.5 4h13A1.5 1.5 0 0 1 20 5.5v4A1.5 1.5 0 0 1 18.5 11h-13A1.5 1.5 0 0 1 4 9.5v-4Z', 'M4 14.5A1.5 1.5 0 0 1 5.5 13h13a1.5 1.5 0 0 1 1.5 1.5v4a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 4 18.5v-4Z'], sf: [[8, 7.5, 1], [8, 16.5, 1]] },
  grid: { s: ['M4 4h6v6H4V4Z', 'M14 4h6v6h-6V4Z', 'M4 14h6v6H4v-6Z', 'M14 14h6v6h-6v-6Z'] },
  chart: { s: ['M4 20V10', 'M10 20V4', 'M16 20v-7', 'M22 20H2'] },
  logout: { s: ['M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4', 'M16 17l5-5-5-5', 'M21 12H9'] },
  check: { s: ['M20 6 9 17l-5-5'] },
  close: { s: ['M6 6l12 12', 'M18 6 6 18'] },
  eye: { s: ['M2.5 12s3.5-6.5 9.5-6.5S21.5 12 21.5 12 18 18.5 12 18.5 2.5 12 2.5 12Z', 'M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z'] },
  edit: { s: ['M12 20h9', 'M16.5 3.5a2.1 2.1 0 0 1 3 3L8 18l-4 1 1-4L16.5 3.5Z'] },
  clock: { s: ['M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Z', 'M12 7v5l3 2'] },
  alert: { s: ['M12 3 2.8 20h18.4L12 3Z', 'M12 9v5', 'M12 17.4v.1'] },
  download: { s: ['M12 3v12', 'm7 10 5 5 5-5', 'M5 21h14'] },
  lock: { s: ['M5 11h14v10H5V11Z', 'M8 11V7a4 4 0 0 1 8 0v4'] },
  code: { s: ['m8 8-5 4 5 4', 'm16 8 5 4-5 4', 'm14 4-4 16'] },
  file: { s: ['M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6Z', 'M14 2v6h6'] },
  pen: { s: ['M12 20h9', 'M16.5 3.5a2.1 2.1 0 0 1 3 3L8 18l-4 1 1-4L16.5 3.5Z'] },
  database: { s: ['M12 8c5 0 9-1.3 9-3s-4-3-9-3-9 1.3-9 3 4 3 9 3Z', 'M3 5v7c0 1.7 4 3 9 3s9-1.3 9-3V5', 'M3 12v7c0 1.7 4 3 9 3s9-1.3 9-3v-7'] },
  zap: { s: ['M13 2 3 14h8l-1 8 11-13h-8l1-7Z'] },
  shield: { s: ['M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z', 'm8.5 12 2.5 2.5 4.5-5'] },
  filter: { s: ['M3 5h18l-7 8v6l-4 2v-8L3 5Z'] },
  help: { s: ['M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18Z', 'M9.5 9a2.5 2.5 0 1 1 4 2c-.9.7-1.5 1.3-1.5 2.5', 'M12 17.2v.1'] },
  external: { s: ['M14 3h7v7', 'M10 14 21 3', 'M21 14v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5'] },
  activity: { s: ['M3 12h4l2.5-7 5 14L17 12h4'] },
  layers: { s: ['M12 2 3 7l9 5 9-5-9-5Z', 'm3 12 9 5 9-5', 'm3 17 9 5 9-5'] },
  mail: { s: ['M4 5h16a1 1 0 0 1 1 1v12a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1Z', 'm3.5 6.5 8.5 6 8.5-6'] },
  user: { s: ['M20 21a8 8 0 0 0-16 0', 'M12 13a5 5 0 1 0 0-10 5 5 0 0 0 0 10Z'] },
  message: { s: ['M21 11.5a8.4 8.4 0 0 1-9 8.4 9.9 9.9 0 0 1-3.8-.8L3 21l1.9-5.2A8.4 8.4 0 1 1 21 11.5Z'] },
  send: { s: ['M22 2 11 13', 'M22 2 15 22l-4-9-9-4 20-7Z'] },
};

export function icon(name, x, y, size = 18, color = C.muted, sw = 1.8) {
  const def = ICONS[name];
  if (!def) throw new Error(`Unknown icon: ${name}`);
  const scale = size / 24;
  let body = '';
  for (const d of def.s || []) body += `<path d="${d}"/>`;
  for (const [cx, cy, r] of def.c || []) body += `<circle cx="${cx}" cy="${cy}" r="${r}"/>`;
  for (const d of def.f || []) body += `<path d="${d}" fill="${color}" stroke="none"/>`;
  for (const [cx, cy, r] of def.sf || []) body += `<circle cx="${cx}" cy="${cy}" r="${r}" fill="${color}" stroke="none"/>`;
  for (const [rx, ry, rw, rh, rr] of def.rects || []) body += `<rect x="${rx}" y="${ry}" width="${rw}" height="${rh}" rx="${rr}" fill="${color}" stroke="none"/>`;
  return `<g transform="translate(${n(x)} ${n(y)}) scale(${n(scale)})" fill="none" stroke="${color}" stroke-width="${n(sw)}" stroke-linecap="round" stroke-linejoin="round">${body}</g>`;
}

export function brandMark(x, y, size = 30) {
  const s = size / 24;
  return `<g transform="translate(${n(x)} ${n(y)})">
    <rect width="${n(size)}" height="${n(size)}" rx="${n(size * 0.28)}" fill="url(#brandGrad)"/>
    <g transform="scale(${n(s)})" fill="none" stroke="#fff" stroke-width="2.1" stroke-linecap="round" stroke-linejoin="round">
      <path d="M7.2 7.1h4.7a4.9 4.9 0 0 1 0 9.8H7.2V7.1Z"/>
      <path d="M7.2 7.1v9.8"/>
      <circle cx="17.2" cy="8.1" r="1.15" fill="#fff" stroke="none"/>
    </g>
  </g>`;
}

export function baseDefs() {
  return `<defs>
    <pattern id="dotgrid" width="24" height="24" patternUnits="userSpaceOnUse">
      <circle cx="1" cy="1" r="1" fill="#D5DBE5" opacity="0.7"/>
    </pattern>
    <linearGradient id="brandGrad" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#6D86FF"/>
      <stop offset="1" stop-color="#3F5BF6"/>
    </linearGradient>
    <linearGradient id="fadeWhite" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#FFFFFF" stop-opacity="0"/>
      <stop offset="1" stop-color="#FFFFFF" stop-opacity="1"/>
    </linearGradient>
    <filter id="shadow" x="-20%" y="-20%" width="140%" height="150%">
      <feDropShadow dx="0" dy="12" stdDeviation="22" flood-color="#101828" flood-opacity="0.10"/>
    </filter>
    <filter id="softShadow" x="-30%" y="-30%" width="160%" height="170%">
      <feDropShadow dx="0" dy="6" stdDeviation="10" flood-color="#101828" flood-opacity="0.06"/>
    </filter>
    <filter id="drawerShadow" x="-30%" y="-10%" width="160%" height="130%">
      <feDropShadow dx="-10" dy="0" stdDeviation="24" flood-color="#101828" flood-opacity="0.12"/>
    </filter>
    <clipPath id="appClip"><rect x="${APP.x}" y="${APP.y}" width="${APP.w}" height="${APP.h}" rx="20"/></clipPath>
  </defs>`;
}

export function pageHeader({ title, subtitle, code, status = '待评审', version = 'v0.1', date = '2026.11.18' }) {
  return `<g>
    ${brandMark(32, 26, 30)}
    ${text(72, 45, 'DeepTalk', { size: 18, weight: 720 })}
    ${text(72, 62, 'MVP REQUIREMENT MAP', { size: 9.5, fill: C.subtle, weight: 600, spacing: 1.35 })}
    ${rect(1592, 26, 82, 24, { fill: C.amberSoft, stroke: '#FEDF89', rx: 12 })}
    ${circle(1605, 38, 3.2, { fill: C.amber })}
    ${text(1614, 42, status, { size: 10.5, fill: '#B54708', weight: 600 })}
    ${text(1768, 41, `${date}  ·  ${version}`, { size: 10.5, fill: C.muted, anchor: 'end' })}
    ${text(32, 101, title, { size: 26, weight: 730, spacing: -0.4 })}
    ${text(32, 122, subtitle, { size: 12.5, fill: C.muted })}
    ${text(1768, 104, code, { size: 25, fill: '#CBD2DD', weight: 750, anchor: 'end', spacing: 1 })}
  </g>`;
}

export function canvasStart(w = 1800, h = 1080) {
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${w}" height="${h}" viewBox="0 0 ${w} ${h}" role="img">
    ${baseDefs()}
    <rect width="${w}" height="${h}" fill="${C.canvas}"/>
    <rect width="${w}" height="${h}" fill="url(#dotgrid)"/>
  `;
}

export function appFrame(inner) {
  return `<g>
    ${rect(APP.x, APP.y, APP.w, APP.h, { fill: C.white, rx: 20, filter: 'shadow' })}
    <g clip-path="url(#appClip)">
      <g transform="translate(${APP.x} ${APP.y})">${inner}</g>
    </g>
    ${rect(APP.x, APP.y, APP.w, APP.h, { fill: 'none', stroke: '#DCE1E9', rx: 20 })}
  </g>`;
}

export function chip(x, y, w, label, o = {}) {
  const { h = 22, fill = C.primarySoft, color = C.primary, size = 10, weight = 600, stroke, dot = false } = o;
  return `<g>
    ${rect(x, y, w, h, { fill, stroke, rx: h / 2 })}
    ${dot ? circle(x + 10, y + h / 2, 3, { fill: color }) : ''}
    ${text(x + (dot ? 18 : w / 2), y + h / 2 + 3.9, label, { size, fill: color, weight, anchor: dot ? 'start' : 'middle' })}
  </g>`;
}

export function callout(num, x, y) {
  return `<g>
    ${circle(x, y, 12.5, { fill: '#FFFFFF', opacity: 0.96 })}
    ${circle(x, y, 10.2, { fill: C.primary })}
    ${text(x, y + 3.6, String(num), { size: 10.5, fill: '#fff', weight: 700, anchor: 'middle' })}
  </g>`;
}

export function annotationRail(items, o = {}) {
  const { header = '交互与规则', badge = 'MVP / P0', footerTitle = '验收标准', footerLines = [], footerNote } = o;
  let out = `<g>
    ${rect(RAIL.x, RAIL.y, RAIL.w, RAIL.h, { fill: C.white, rx: 18, stroke: '#E0E5EC', filter: 'shadow' })}
    ${text(RAIL.x + 20, RAIL.y + 34, header, { size: 17, weight: 720 })}
    ${chip(RAIL.x + 194, RAIL.y + 16, 58, badge, { h: 22, size: 9, fill: C.primarySoft })}
    ${text(RAIL.x + 20, RAIL.y + 56, 'PC Web · 1440 × 900 基准', { size: 10.5, fill: C.subtle })}
    ${line(RAIL.x + 20, RAIL.y + 72, RAIL.x + RAIL.w - 20, RAIL.y + 72, { stroke: C.line })}
  `;
  let cy = RAIL.y + 100;
  for (const item of items) {
    out += circle(RAIL.x + 25, cy - 4, 9.5, { fill: C.primary });
    out += text(RAIL.x + 25, cy - 0.4, String(item.n), { size: 9.5, fill: '#fff', weight: 700, anchor: 'middle' });
    out += text(RAIL.x + 45, cy, item.title, { size: 12, weight: 700, fill: C.ink2 });
    let ly = cy + 18;
    for (const desc of item.lines) {
      out += text(RAIL.x + 45, ly, desc, { size: 10.4, fill: C.muted });
      ly += 15.2;
    }
    cy = ly + 13.5;
  }
  const footerY = RAIL.y + RAIL.h - 100;
  out += line(RAIL.x + 20, footerY, RAIL.x + RAIL.w - 20, footerY, { stroke: C.line });
  out += text(RAIL.x + 20, footerY + 22, footerTitle, { size: 10.5, weight: 700, fill: C.primary });
  let fy = footerY + 42;
  for (const fl of footerLines) {
    out += text(RAIL.x + 20, fy, fl, { size: 10.2, fill: C.muted });
    fy += 14.8;
  }
  if (footerNote) {
    out += text(RAIL.x + 20, fy + 2, footerNote, { size: 9.8, fill: C.subtle });
  }
  out += '</g>';
  return out;
}

export function writeSvg(file, content, width = 1800, height = 1080) {
  const svg = `${canvasStart(width, height)}${content}\n</svg>\n`;
  fs.writeFileSync(file, svg, 'utf8');
  console.log(`wrote ${file} (${(svg.length / 1024).toFixed(1)} KB)`);
}
