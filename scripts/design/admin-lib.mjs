import { C, MONO, text, rect, circle, line, icon, brandMark } from '../requirements/req-lib.mjs';
import { g, pill, avatar, toggle, progress } from './design-lib.mjs';

export const ADMIN_NAV = [
  { id: 'overview', label: '概览', icon: 'grid' },
  { id: 'providers', label: '厂商与模型', icon: 'server' },
  { id: 'users', label: '用户管理', icon: 'users' },
  { id: 'usage', label: '用量统计', icon: 'chart' },
  { id: 'settings', label: '系统设置', icon: 'settings' },
];

export function adminShell(active, title) {
  const parts = [];
  parts.push(rect(240, 0, 1200, 900, { fill: '#F7F8FA' }));
  parts.push(rect(0, 0, 240, 900, { fill: '#FBFCFD' }));
  parts.push(line(240, 0, 240, 900, { stroke: '#EDEFF4' }));
  parts.push(brandMark(20, 18, 30));
  parts.push(text(60, 39, 'DeepTalk', { size: 15, weight: 720 }));
  parts.push(pill(156, 24, 46, 18, 'ADMIN', { h: 18, size: 7.6, weight: 700, fill: '#EEF2FF', color: C.primary2 }));
  parts.push(text(20, 88, '控制台', { size: 9.6, weight: 650, fill: C.faint, spacing: 1.2 }));

  let y = 100;
  for (const item of ADMIN_NAV) {
    const on = item.id === active;
    if (on) {
      parts.push(rect(12, y, 216, 40, { fill: '#EEF2FF', rx: 10 }));
      parts.push(rect(12, y + 10, 3, 20, { fill: C.primary, rx: 2 }));
    }
    parts.push(icon(item.icon, 28, y + 11, 18, on ? C.primary2 : C.subtle, on ? 2 : 1.7));
    parts.push(text(56, y + 25, item.label, { size: 12.6, weight: on ? 650 : 450, fill: on ? C.primary2 : C.body }));
    y += 44;
  }

  parts.push(rect(16, 356, 208, 104, { fill: '#FFFFFF', rx: 12, stroke: '#E8EBF0' }));
  parts.push(text(32, 378, '存储用量', { size: 11, weight: 620, fill: C.ink2 }));
  parts.push(text(208, 378, '62%', { size: 11, weight: 720, fill: C.primary, anchor: 'end' }));
  parts.push(progress(32, 388, 176, 6, 0.62));
  parts.push(text(32, 410, '6.2 GB / 10 GB', { size: 9.2, fill: C.muted }));
  parts.push(text(32, 426, 'SQLite · 单实例部署', { size: 9, fill: C.subtle }));
  parts.push(text(32, 442, '本月对话 1,284 条', { size: 9, fill: C.subtle }));

  parts.push(rect(16, 736, 208, 92, { fill: '#FFFFFF', rx: 12, stroke: '#E8EBF0' }));
  parts.push(icon('help', 32, 752, 15, C.primary, 1.7));
  parts.push(text(54, 764, '管理员指南', { size: 11.4, weight: 620, fill: C.ink2 }));
  parts.push(text(32, 786, '查看密钥托管与路由说明', { size: 9.8, fill: C.subtle }));
  parts.push(text(32, 806, 'v0.3 · MVP', { size: 9.2, fill: C.faint, family: MONO }));

  parts.push(avatar(36, 856, 16, 'L'));
  parts.push(text(64, 852, '林默', { size: 12, weight: 620 }));
  parts.push(text(64, 867, '平台管理员', { size: 9.6, fill: C.subtle }));
  parts.push(icon('more', 208, 847, 16, C.subtle, 1.8));

  // topbar
  parts.push(rect(240, 0, 1200, 64, { fill: '#FFFFFF' }));
  parts.push(line(240, 64, 1440, 64, { stroke: '#EFF1F5' }));
  parts.push(text(272, 39, '控制台', { size: 11.4, fill: C.subtle }));
  parts.push(text(316, 39, '/', { size: 11.4, fill: C.faint }));
  parts.push(text(330, 39, title, { size: 11.4, weight: 600, fill: C.ink2 }));
  parts.push(rect(920, 16, 210, 32, { fill: '#F4F6F9', rx: 9 }));
  parts.push(icon('search', 934, 24, 15, C.subtle, 1.8));
  parts.push(text(956, 37, '搜索用户、厂商或模型', { size: 10.6, fill: C.subtle }));
  parts.push(iconBtnBell(1224, 18));
  parts.push(line(1264, 22, 1264, 42, { stroke: '#EDEFF4' }));
  parts.push(avatar(1294, 32, 14, 'L'));
  parts.push(text(1316, 36, '林默', { size: 11.6, weight: 600, fill: C.ink2 }));
  parts.push(icon('chevronDown', 1352, 26, 12, C.faint, 2));
  parts.push(icon('more', 1392, 23, 18, C.subtle, 1.8));
  return g(...parts);
}

function iconBtnBell(x, y) {
  return g(
    icon('message', x, y, 18, C.subtle, 1.7),
    circle(x + 14, y + 2, 3, { fill: C.red }),
  );
}

export function navCard(x, y, w, h, parts) {
  return g(
    rect(x, y, w, h, { fill: '#FFFFFF', rx: 14, stroke: '#E8EBF0' }),
    ...parts,
  );
}

export function statRow(x, y, w, label, value, o = {}) {
  return g(
    text(x, y, label, { size: 9.8, fill: C.subtle }),
    text(x, y + 20, value, { size: 13.5, weight: 680, fill: o.color || C.ink, family: o.mono ? MONO : undefined }),
  );
}

export function tableHeader(x, y, w, cols, h = 34) {
  const parts = [rect(x, y, w, h, { fill: '#F9FAFB' }), line(x, y + h, x + w, y + h, { stroke: '#EDEFF4' })];
  for (const c of cols) parts.push(text(x + c.x, y + h / 2 + 3.6, c.label, { size: 10.2, weight: 600, fill: C.subtle, anchor: c.anchor }));
  return g(...parts);
}
