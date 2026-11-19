import { C, MONO, text, rect, circle, line, icon, brandMark } from '../requirements/req-lib.mjs';
import { frame, sheetHeader, specRail, pinLabel, writeSvg, g, btn, pill, progress, toggle } from './design-lib.mjs';
import { adminShell, navCard } from './admin-lib.mjs';

const CX = 272;
const CW = 1136;

function providerCard(x, y, p) {
  const w = 365, h = 164;
  const parts = [
    rect(x, y, w, h, { fill: '#FFFFFF', rx: 14, stroke: '#E8EBF0' }),
    rect(x + 18, y + 18, 44, 44, { fill: p.logoBg, rx: 12 }),
    text(x + 40, y + 46, p.logo, { size: 13.5, weight: 750, fill: p.logoColor, anchor: 'middle' }),
    text(x + 74, y + 37, p.name, { size: 13.8, weight: 680 }),
    text(x + 74, y + 56, p.host, { size: 9.6, fill: C.subtle, family: MONO }),
    pill(x + w - 84, y + 18, 66, 22, p.status, { size: 9.8, fill: p.statusBg, color: p.statusColor, dot: true }),
    line(x + 18, y + 80, x + w - 18, y + 80, { stroke: '#EFF1F5' }),
  ];
  const stats = [['模型', p.models], ['超时', p.timeout], ['并发', p.concurrency]];
  stats.forEach((s, i) => {
    parts.push(text(x + 18 + i * 108, y + 102, s[0], { size: 9.4, fill: C.subtle }));
    parts.push(text(x + 18 + i * 108, y + 122, s[1], { size: 12.6, weight: 650, fill: C.ink }));
  });
  parts.push(toggle(x + 18, y + 132, p.enabled));
  if (p.badge) parts.push(pill(x + 60, y + 132, 44, 20, p.badge, { size: 9, fill: C.primarySoft, color: C.primary2 }));
  parts.push(text(x + w - 18, y + 146, p.action, { size: 10.4, weight: 600, fill: C.primary, anchor: 'end' }));
  return g(...parts);
}

function modelsTable() {
  const x = CX, y = 398, w = 820, h = 458;
  const rows = [
    ['deepseek-chat', 'DeepSeek V3.2 · 通用对话', 'DeepSeek', '128K', '8', true],
    ['deepseek-reasoner', 'R1 · 深度思考', 'DeepSeek', '64K', '4', true],
    ['deepseek-coder', '代码补全与重构', 'DeepSeek', '128K', '4', true],
    ['gpt-4o-mini', '低成本多模态', 'OpenAI', '128K', '6', true],
    ['text-embedding-3-small', '向量化 · 检索', 'OpenAI', '8K', '2', false],
    ['llama-3.1-70b', '自建推理 · 内部', 'vLLM', '32K', '2', false],
  ];
  const parts = [
    rect(x, y, w, h, { fill: '#FFFFFF', rx: 14, stroke: '#E8EBF0' }),
    text(x + 18, y + 30, '模型路由', { size: 13.6, weight: 680 }),
    text(x + 18, y + 47, '共 6 个模型 · 4 个已启用', { size: 9.8, fill: C.subtle }),
    btn(x + w - 122, y + 16, 104, 32, '添加模型', { iconName: 'plus', size: 11.4, iconSize: 14 }),
  ];
  parts.push(text(x + 292 - 272 + 18, y + 47, '', { size: 1 }));
  parts.push(rect(x, y + 58, w, 34, { fill: '#F9FAFB' }));
  parts.push(line(x, y + 92, x + w, y + 92, { stroke: '#EDEFF4' }));
  const cols = [
    { label: '模型', x: 20 },
    { label: '厂商', x: 298 },
    { label: '上下文', x: 428 },
    { label: '并发', x: 516 },
    { label: '状态', x: 596 },
  ];
  cols.forEach((c) => parts.push(text(x + c.x, y + 79, c.label, { size: 10.2, weight: 600, fill: C.subtle })));

  rows.forEach((r, i) => {
    const ry = y + 92 + i * 50;
    if (i > 0) parts.push(line(x + 18, ry, x + w - 18, ry, { stroke: '#F1F3F6' }));
    parts.push(circle(x + 26, ry + 25, 4, { fill: r[5] ? C.primary : '#C4CBD6' }));
    parts.push(text(x + 40, ry + 24, r[0], { size: 11.6, weight: 620, fill: r[5] ? C.ink : C.muted, family: MONO }));
    parts.push(text(x + 40, ry + 39, r[1], { size: 9.6, fill: C.subtle }));
    parts.push(pill(x + 298, ry + 14, 84, 22, r[2], { size: 9.8, fill: '#F4F6F9', color: C.body, weight: 600 }));
    parts.push(text(x + 428, ry + 29, r[3], { size: 11, fill: C.ink2 }));
    parts.push(text(x + 516, ry + 29, r[4], { size: 11, fill: C.ink2 }));
    parts.push(toggle(x + 596, ry + 16, r[5]));
    parts.push(text(x + 640, ry + 30, r[5] ? '启用' : '停用', { size: 9.8, fill: r[5] ? C.green : C.subtle }));
  });

  parts.push(line(x, y + h - 64, x + w, y + h - 64, { stroke: '#EFF1F5' }));
  parts.push(text(x + 18, y + h - 30, '每 30 秒同步上游模型列表', { size: 9.6, fill: C.faint }));
  const pages = ['‹', '1', '2', '›'];
  pages.forEach((p, i) => {
    const bx = x + w - 168 + i * 34;
    if (i === 1) parts.push(rect(bx, y + h - 44, 28, 28, { fill: C.primarySoft, rx: 8 }));
    parts.push(text(bx + 14, y + h - 25, p, { size: 10.4, weight: 600, fill: i === 1 ? C.primary : C.subtle, anchor: 'middle' }));
  });
  return g(...parts);
}

function routingCard() {
  const x = 1112, y = 398, w = 296, h = 190;
  const parts = [
    rect(x, y, w, h, { fill: '#FFFFFF', rx: 14, stroke: '#E8EBF0' }),
    text(x + 18, y + 30, '默认路由', { size: 13.4, weight: 680 }),
    text(x + 18, y + 62, '默认模型', { size: 9.6, weight: 600, fill: C.subtle }),
    rect(x + 18, y + 70, w - 36, 38, { fill: '#FFFFFF', rx: 9, stroke: '#DDE2EA' }),
    circle(x + 32, y + 89, 3.4, { fill: C.primary }),
    text(x + 44, y + 93, 'deepseek-chat', { size: 11.4, weight: 600, fill: C.ink2, family: MONO }),
    icon('chevronDown', x + w - 40, y + 83, 12, C.faint, 2),
    text(x + 18, y + 128, '兜底模型', { size: 9.6, weight: 600, fill: C.subtle }),
    rect(x + 18, y + 136, w - 36, 38, { fill: '#FFFFFF', rx: 9, stroke: '#DDE2EA' }),
    circle(x + 32, y + 155, 3.4, { fill: C.subtle }),
    text(x + 44, y + 159, 'gpt-4o-mini', { size: 11.4, weight: 600, fill: C.ink2, family: MONO }),
    icon('chevronDown', x + w - 40, y + 149, 12, C.faint, 2),
  ];
  return g(...parts);
}

function healthCard() {
  const x = 1112, y = 604, w = 296, h = 120;
  const rows = [
    ['上游可用', '3 / 3', C.green],
    ['近 1 小时错误', '0 次', C.ink2],
    ['首字节 P95', '620 ms', C.ink2],
  ];
  const parts = [
    rect(x, y, w, h, { fill: '#FFFFFF', rx: 14, stroke: '#E8EBF0' }),
    text(x + 18, y + 28, '接入健康', { size: 13.4, weight: 680 }),
    text(x + w - 18, y + 28, '实时', { size: 9.4, fill: C.faint, anchor: 'end' }),
  ];
  rows.forEach((r, i) => {
    const ry = y + 54 + i * 22;
    parts.push(circle(x + 21, ry - 4, 3, { fill: r[2] === C.green ? C.green : C.faint }));
    parts.push(text(x + 32, ry, r[0], { size: 10.2, fill: C.muted }));
    parts.push(text(x + w - 18, ry, r[1], { size: 10.4, weight: 620, fill: r[2], anchor: 'end', family: MONO }));
  });
  return g(...parts);
}

function changeCard() {
  const x = 1112, y = 740, w = 296, h = 116;
  const items = [
    ['14:22', '更新 deepseek-chat 并发 8 → 12'],
    ['11:40', 'OpenAI 连接测试通过'],
    ['昨日', '新增自定义厂商 vLLM'],
  ];
  const parts = [
    rect(x, y, w, h, { fill: '#FFFFFF', rx: 14, stroke: '#E8EBF0' }),
    text(x + 18, y + 28, '最近变更', { size: 13.4, weight: 680 }),
    text(x + w - 18, y + 28, '全部 ›', { size: 9.4, fill: C.primary, anchor: 'end' }),
  ];
  items.forEach((it, i) => {
    const ry = y + 54 + i * 22;
    parts.push(text(x + 18, ry, it[0], { size: 9.4, fill: C.faint, family: MONO }));
    parts.push(text(x + 62, ry, it[1], { size: 9.8, fill: C.muted }));
  });
  return g(...parts);
}

const inner = g(
  adminShell('providers', '厂商与模型'),
  text(CX, 112, '厂商与模型', { size: 22, weight: 730, spacing: -0.3 }),
  text(CX, 134, '统一管理上游接入、密钥与默认路由；Key 只存服务端，前端永不回显。', { size: 11.6, fill: C.muted }),
  btn(1116, 88, 128, 38, '测试连通性', { iconName: 'zap', size: 11.6, iconSize: 15, radius: 10 }),
  btn(1258, 88, 118, 38, '新建厂商', { primary: true, iconName: 'plus', size: 11.6, iconSize: 15, radius: 10 }),

  rect(CX, 160, 268, 36, { fill: '#EFF1F5', rx: 10 }),
  rect(CX + 3, 163, 92, 30, { fill: '#FFFFFF', rx: 8 }),
  text(CX + 49, 182, '全部 6', { size: 10.8, weight: 620, fill: C.ink2, anchor: 'middle' }),
  text(CX + 135, 182, '预置 4', { size: 10.8, fill: C.muted, anchor: 'middle' }),
  text(CX + 220, 182, '自定义 2', { size: 10.8, fill: C.muted, anchor: 'middle' }),

  providerCard(CX, 210, { logo: 'DS', logoBg: '#EEF2FF', logoColor: C.primary2, name: '深度求索', host: 'api.deepseek.com', status: '已启用', statusBg: C.greenSoft, statusColor: '#067647', models: '3 个', timeout: '60s', concurrency: '8', enabled: true, badge: '默认', action: '配置 ›' }),
  providerCard(CX + 385, 210, { logo: 'OA', logoBg: '#F2F4F7', logoColor: C.ink2, name: 'OpenAI', host: 'api.openai.com/v1', status: '已启用', statusBg: C.greenSoft, statusColor: '#067647', models: '2 个', timeout: '90s', concurrency: '6', enabled: true, action: '配置 ›' }),
  providerCard(CX + 770, 210, { logo: 'vL', logoBg: '#FFF6E8', logoColor: '#B54708', name: '内部 vLLM', host: 'llm.internal.corp/v1', status: '待验证', statusBg: C.amberSoft, statusColor: '#B54708', models: '1 个', timeout: '45s', concurrency: '2', enabled: false, badge: '自定义', action: '测试连接 ›' }),

  modelsTable(),
  routingCard(),
  healthCard(),
  changeCard(),

  pinLabel(1, 246, 420),
  pinLabel(2, 640, 386),
  pinLabel(3, 1092, 520),
  pinLabel(4, 1404, 412),
  pinLabel(5, 1404, 626),
  pinLabel(6, 1390, 126),
);

const notes = [
  { n: 1, title: '后台骨架', lines: ['240 侧栏 + 面包屑顶栏；内容', '区留白 32，卡片间距 20。'] },
  { n: 2, title: '厂商卡片', lines: ['状态胶囊区分启用 / 待验证；', '关闭开关需二次确认。'] },
  { n: 3, title: '模型路由表', lines: ['行高 50，双行=模型 ID + 用途；', '停用行降透明度、不可设为默认。'] },
  { n: 4, title: '默认路由', lines: ['默认与兜底分离；兜底仅在', '默认失败或超时时触发一次。'] },
  { n: 5, title: '接入健康', lines: ['3/3 上游可用、1h 错误数、', '首字节 P95，30s 自动刷新。'] },
  { n: 6, title: '密钥安全', lines: ['Key 只写不读、前端不回显；', '测试连接由服务端发起。'] },
];

writeSvg('docs/design/03-admin-providers.svg', sheetHeader({
  title: '管理后台 · 厂商与模型',
  sub: '把上游接入做成可运营的路由：厂商、密钥、模型与兜底一屏管完',
  code: 'DSG-03 / ADMIN-PROVIDERS',
}) + frame(32, 142, 1440, 900, 'screen03', inner) + specRail({
  items: notes,
  footerLines: ['空态：插画 + 新建厂商引导', '危险操作二次确认，Toast 3s', '上游失败不自动跨厂商重试'],
}), 1800, 1080);
