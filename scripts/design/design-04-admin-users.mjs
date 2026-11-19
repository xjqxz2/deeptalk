import { C, MONO, text, rect, circle, line, icon } from '../requirements/req-lib.mjs';
import { frame, sheetHeader, specRail, pinLabel, writeSvg, g, btn, pill, progress, toggle, avatar, field } from './design-lib.mjs';
import { adminShell } from './admin-lib.mjs';

const CX = 272;
const TW = 800;
const DX = 1096;
const DW = 312;

const USERS = [
  ['林默', 'lmo@deeptalk.app', '管理员', '免费版', 0.32, '1.28M / 4M', '刚刚', 'normal'],
  ['陈亦航', 'chen.yh@acme.io', '专业版', '专业版', 0.42, '8.4M / 20M', '3 分钟前', 'normal'],
  ['苏晓', 'su.xiao@mail.com', '普通用户', '免费版', 0.98, '3.9M / 4M', '12 分钟前', 'warn'],
  ['周予安', 'zhou.ya@studio.cn', '团队版', '团队版', 0.24, '12M / 50M', '1 小时前', 'normal'],
  ['何季', 'he.ji@corp.cn', '普通用户', '免费版', 0.10, '0.4M / 4M', '3 小时前', 'blocked'],
  ['唐可', 'tang.ke@mail.com', '专业版', '专业版', 0.31, '6.1M / 20M', '昨天 22:10', 'normal'],
  ['顾明', 'gu.ming@dev.io', '普通用户', '免费版', 0.55, '2.2M / 4M', '2 天前', 'normal'],
  ['沈知', 'shen.zhi@lab.cn', '团队版', '团队版', 0.20, '9.8M / 50M', '3 天前', 'normal'],
];

const STATUS = {
  normal: { label: '正常', color: '#067647', bg: C.greenSoft, dot: C.green },
  warn: { label: '额度将尽', color: '#B54708', bg: C.amberSoft, dot: C.amber },
  blocked: { label: '已限制', color: '#B42318', bg: C.redSoft, dot: C.red },
};

const PLAN = {
  '免费版': { bg: '#F4F6F9', color: C.body },
  '专业版': { bg: C.primarySoft, color: C.primary2 },
  '团队版': { bg: '#EEF0F4', color: C.ink2 },
};

function statCard(x, y, label, value, delta, o = {}) {
  return g(
    rect(x, y, 272, 92, { fill: '#FFFFFF', rx: 14, stroke: '#E8EBF0' }),
    text(x + 20, y + 28, label, { size: 10.4, fill: C.muted, weight: 500 }),
    text(x + 20, y + 66, value, { size: 27, weight: 720, spacing: -0.6 }),
    text(x + 20, y + 82, delta, { size: 9.8, fill: o.color || C.subtle }),
  );
}

function table() {
  const x = CX, y = 320, w = TW;
  const parts = [
    rect(x, y, w, 536, { fill: '#FFFFFF', rx: 14, stroke: '#E8EBF0' }),
    rect(x, y, w, 40, { fill: '#F9FAFB' }),
    line(x, y + 40, x + w, y + 40, { stroke: '#EDEFF4' }),
    text(x + 20, y + 25, '用户', { size: 10.2, weight: 600, fill: C.subtle }),
    text(x + 248, y + 25, '套餐', { size: 10.2, weight: 600, fill: C.subtle }),
    text(x + 348, y + 25, '本月用量', { size: 10.2, weight: 600, fill: C.subtle }),
    text(x + 546, y + 25, '状态', { size: 10.2, weight: 600, fill: C.subtle, anchor: 'middle' }),
    text(x + w - 20, y + 25, '操作', { size: 10.2, weight: 600, fill: C.subtle, anchor: 'end' }),
  ];
  USERS.forEach((u, i) => {
    const ry = y + 40 + i * 56;
    if (i > 0) parts.push(line(x + 20, ry, x + w - 20, ry, { stroke: '#F2F4F7' }));
    const st = STATUS[u[7]];
    parts.push(avatar(x + 38, ry + 28, 15, u[0][0]));
    parts.push(text(x + 66, ry + 24, u[0], { size: 12, weight: 620 }));
    parts.push(text(x + 66, ry + 40, u[1], { size: 9.4, fill: C.subtle, family: MONO }));
    const pl = PLAN[u[3]];
    parts.push(pill(x + 248, ry + 17, 62, 22, u[3], { size: 9.6, fill: pl.bg, color: pl.color }));
    parts.push(progress(x + 348, ry + 21, 64, 6, u[4]));
    parts.push(text(x + 422, ry + 27, u[5], { size: 9.8, fill: C.muted }));
    parts.push(text(x + 422, ry + 40, Math.round(u[4] * 100) + '%', { size: 8.6, fill: C.faint }));
    parts.push(pill(x + 518, ry + 17, 56, 22, st.label, { size: 9.2, fill: st.bg, color: st.color, dot: true }));
    parts.push(icon('more', x + w - 36, ry + 19, 16, C.subtle, 1.8));
  });
  parts.push(line(x, y + 488, x + w, y + 488, { stroke: '#EFF1F5' }));
  parts.push(text(x + 20, y + 516, '共 128 位用户 · 第 1 / 16 页', { size: 9.8, fill: C.faint }));
  const pages = ['‹', '1', '2', '3', '›'];
  pages.forEach((p, i) => {
    const bx = x + w - 196 + i * 34;
    if (i === 1) parts.push(rect(bx, y + 500, 28, 26, { fill: C.primarySoft, rx: 8 }));
    parts.push(text(bx + 14, y + 518, p, { size: 10.2, weight: 600, fill: i === 1 ? C.primary : C.subtle, anchor: 'middle' }));
  });
  return g(...parts);
}

function detail() {
  const x = DX, y = 320, w = DW, h = 536;
  const parts = [
    rect(x, y, w, h, { fill: '#FFFFFF', rx: 14, stroke: '#E8EBF0' }),
    avatar(x + 40, y + 44, 22, '林'),
    text(x + 72, y + 38, '林默', { size: 15, weight: 700 }),
    text(x + 72, y + 56, 'lmo@deeptalk.app', { size: 9.6, fill: C.subtle, family: MONO }),
    pill(x + w - 72, y + 28, 52, 22, '正常', { size: 9.4, fill: C.greenSoft, color: '#067647', dot: true }),
    line(x + 20, y + 86, x + w - 20, y + 86, { stroke: '#EFF1F5' }),
    text(x + 20, y + 112, '账号', { size: 9.4, weight: 650, fill: C.faint, spacing: 1 }),
  ];
  const rows = [['角色', '平台管理员'], ['注册时间', '2026-10-02'], ['最近登录', '3 分钟前 · 上海']];
  rows.forEach((r, i) => {
    const ry = y + 136 + i * 22;
    parts.push(text(x + 20, ry, r[0], { size: 10, fill: C.muted }));
    parts.push(text(x + w - 20, ry, r[1], { size: 10, fill: C.ink2, anchor: 'end', family: i === 2 ? undefined : MONO }));
  });
  parts.push(line(x + 20, y + 208, x + w - 20, y + 208, { stroke: '#EFF1F5' }));
  parts.push(text(x + 20, y + 234, '套餐与额度', { size: 9.4, weight: 650, fill: C.faint, spacing: 1 }));
  parts.push(rect(x + 20, y + 246, w - 40, 38, { fill: '#FFFFFF', rx: 9, stroke: '#DDE2EA' }));
  parts.push(text(x + 32, y + 269, '免费版 · 4M tokens / 月', { size: 10.6, weight: 600, fill: C.ink2 }));
  parts.push(icon('chevronDown', x + w - 44, y + 257, 12, C.faint, 2));
  parts.push(progress(x + 20, y + 300, w - 40, 6, 0.32));
  parts.push(text(x + 20, y + 322, '已用 1.28M / 4M · 32%', { size: 9.4, fill: C.muted }));
  parts.push(text(x + w - 20, y + 322, '重置 12 天', { size: 9.4, fill: C.faint, anchor: 'end' }));
  parts.push(text(x + 20, y + 348, '调整额度 ›', { size: 10, weight: 600, fill: C.primary }));
  parts.push(line(x + 20, y + 368, x + w - 20, y + 368, { stroke: '#EFF1F5' }));
  parts.push(text(x + 20, y + 394, '安全', { size: 9.4, weight: 650, fill: C.faint, spacing: 1 }));
  parts.push(text(x + 20, y + 420, '限制登录', { size: 10.6, weight: 600, fill: C.ink2 }));
  parts.push(text(x + 20, y + 436, '开启后已签发会话立即失效', { size: 9, fill: C.subtle }));
  parts.push(toggle(x + w - 54, y + 406, false));
  parts.push(btn(x + 20, y + 452, w - 40, 34, '重置密码', { iconName: 'lock', size: 10.8, iconSize: 14, radius: 9 }));
  parts.push(text(x + w / 2, y + 510, '强制退出全部设备', { size: 10, fill: C.primary, anchor: 'middle' }));
  return g(...parts);
}

const inner = g(
  adminShell('users', '用户管理'),
  text(CX, 112, '用户管理', { size: 22, weight: 730, spacing: -0.3 }),
  text(CX, 134, '账号、套餐与额度；后台不读取对话正文，导出的 CSV 仅含账号字段。', { size: 11.6, fill: C.muted }),
  btn(1110, 88, 118, 38, '导出 CSV', { iconName: 'download', size: 11.6, iconSize: 15 }),
  btn(1244, 88, 128, 38, '新建账号', { primary: true, iconName: 'plus', size: 11.6, iconSize: 15 }),

  statCard(CX, 160, '总用户', '128', '本月新增 24 · 全部邮箱注册', { color: C.muted }),
  statCard(CX + 288, 160, '7 日活跃', '63', '49.2% 周活跃率', { color: C.muted }),
  statCard(CX + 576, 160, '本月用量', '38.6M', 'tokens · 较上月 +18%', { color: C.muted }),
  statCard(CX + 864, 160, '已限制', '2', '需关注 · 可随时解除', { color: C.amber }),

  rect(CX, 268, 300, 36, { fill: '#FFFFFF', rx: 9, stroke: '#E3E7EE' }),
  icon('search', CX + 12, 278, 15, C.subtle, 1.8),
  text(CX + 34, 291, '搜索邮箱或昵称', { size: 10.6, fill: C.subtle }),
  rect(592, 268, 352, 36, { fill: '#EFF1F5', rx: 10 }),
  rect(595, 271, 86, 30, { fill: '#FFFFFF', rx: 8 }),
  text(638, 290, '全部 128', { size: 10.4, weight: 620, fill: C.ink2, anchor: 'middle' }),
  text(724, 290, '正常 124', { size: 10.4, fill: C.muted, anchor: 'middle' }),
  text(810, 290, '已限制 2', { size: 10.4, fill: C.muted, anchor: 'middle' }),
  text(900, 290, '额度将尽 2', { size: 10.4, fill: C.muted, anchor: 'middle' }),
  text(1180, 291, '最近活跃', { size: 10.4, fill: C.muted }),
  icon('chevronDown', 1240, 282, 12, C.faint, 2),
  text(1396, 291, '筛选', { size: 10.4, fill: C.muted, anchor: 'end' }),
  icon('filter', 1352, 278, 14, C.subtle, 1.7),

  table(),
  detail(),

  pinLabel(1, 1390, 214),
  pinLabel(2, 930, 286),
  pinLabel(3, 1064, 470),
  pinLabel(4, 1088, 330),
  pinLabel(5, 1404, 790),
  pinLabel(6, 1396, 118),
);

const notes = [
  { n: 1, title: '统计卡', lines: ['四等分 272；口径=总用户 / 7 日', '活跃 / 本月 tokens / 已限制。'] },
  { n: 2, title: '工具栏', lines: ['搜索 300 + 防抖 300ms；状态', '筛选用分段控件，右侧排序。'] },
  { n: 3, title: '用户表', lines: ['行高 56：头像 + 邮箱、套餐、', '额度条 + 百分比、状态点。'] },
  { n: 4, title: '账号详情', lines: ['312 常驻右栏；额度可调，限制', '登录后已签发会话立即失效。'] },
  { n: 5, title: '危险操作', lines: ['限登 / 删除二次确认；重置密码', '生成一次性密码，只显示一次。'] },
  { n: 6, title: '隐私边界', lines: ['不展示对话正文；CSV 仅导出', '账号字段，操作写审计日志。'] },
];

writeSvg('docs/design/04-admin-users.svg', sheetHeader({
  title: '管理后台 · 用户管理',
  sub: '列表做运营、右栏做处置：额度、限登与重置密码都在一屏内完成',
  code: 'DSG-04 / ADMIN-USERS',
}) + frame(32, 142, 1440, 900, 'screen04', inner) + specRail({
  items: notes,
  footerLines: ['空态提示「没有匹配的用户」', '批量操作最多 50 条', '状态色：绿正常 / 橙将尽 / 红限制'],
}), 1800, 1080);
