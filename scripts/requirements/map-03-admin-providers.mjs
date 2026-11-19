import {
  C, text, rect, circle, line, path, icon, brandMark, pageHeader, appFrame,
  callout, annotationRail, writeSvg, chip,
} from './req-lib.mjs';

function adminSidebar() {
  let s = rect(0, 0, 220, 900, { fill: '#FBFCFD' });
  s += line(220, 0, 220, 900, { stroke: '#E8EBF0' });
  s += brandMark(20, 20, 28);
  s += text(58, 40, 'DeepTalk', { size: 16, weight: 730, spacing: -0.2 });
  s += chip(132, 24, 44, 'ADMIN', { h: 18, size: 7.5, fill: '#F2F4F7', color: C.muted });

  s += rect(16, 78, 188, 40, { fill: '#F5F7FA', stroke: '#EAECF0', rx: 9 });
  s += text(30, 102, '平台控制台', { size: 11.5, fill: C.ink2, weight: 600 });
  s += icon('chevronDown', 182, 90, 14, C.subtle, 1.8);

  s += text(24, 148, '管理', { size: 9.5, fill: C.subtle, weight: 600, spacing: 0.8 });
  const nav = [
    ['grid', '概览', 160, false],
    ['users', '用户管理', 202, false],
    ['server', '模型厂商', 244, true],
    ['layers', '模型映射', 286, false],
  ];
  for (const [ic, label, y, active] of nav) {
    if (active) {
      s += rect(12, y, 196, 38, { fill: C.primarySoft, rx: 9 });
      s += rect(12, y + 9, 3, 20, { fill: C.primary, rx: 1.5 });
    }
    s += icon(ic, 28, y + 10, 17, active ? C.primary : C.subtle, 1.7);
    s += text(54, y + 24, label, { size: 11.8, fill: active ? C.primary2 : C.body, weight: active ? 650 : 450 });
  }

  s += text(24, 344, '观测', { size: 9.5, fill: C.subtle, weight: 600, spacing: 0.8 });
  for (const [ic, label, y] of [['chart', '用量与日志', 358], ['settings', '系统设置', 400]]) {
    s += icon(ic, 28, y + 10, 17, C.subtle, 1.7);
    s += text(54, y + 24, label, { size: 11.8, fill: C.body });
  }

  s += rect(16, 760, 188, 40, { fill: '#F7F8FA', stroke: '#EAECF0', rx: 9 });
  s += icon('arrowLeft', 30, 772, 16, C.primary, 1.8);
  s += text(54, 785, '返回对话工作台', { size: 10.7, fill: C.primary2, weight: 550 });
  s += icon('external', 184, 772, 13, C.subtle, 1.6);

  s += circle(34, 840, 16, { fill: '#E9EDF5' });
  s += text(34, 845, 'L', { size: 11.5, fill: C.ink2, weight: 700, anchor: 'middle' });
  s += text(58, 835, '林默', { size: 11.5, fill: C.ink2, weight: 600 });
  s += text(58, 851, '平台管理员', { size: 9.5, fill: C.subtle });
  s += icon('more', 188, 832, 16, C.subtle, 1.7);
  s += text(20, 888, 'DeepTalk Admin  ·  v0.1', { size: 8.8, fill: C.faint });
  return s;
}

function adminTopbar() {
  let s = rect(0, 0, 1220, 64, { fill: '#FFFFFF' });
  s += line(0, 64, 1220, 64, { stroke: '#EEF0F4' });
  s += text(24, 38, '平台管理', { size: 10.5, fill: C.subtle });
  s += text(77, 38, '/', { size: 10.5, fill: C.faint });
  s += text(91, 38, '模型厂商', { size: 11.5, fill: C.ink2, weight: 600 });
  s += chip(1028, 18, 88, '生产环境', { h: 26, size: 9.8, fill: C.greenSoft, color: '#027A48', dot: true });
  s += icon('help', 1136, 22, 18, C.subtle, 1.7);
  s += circle(1182, 32, 16, { fill: '#E9EDF5' });
  s += text(1182, 36, 'L', { size: 11, fill: C.ink2, weight: 700, anchor: 'middle' });
  return s;
}

function statCard(x, label, value, note, noteColor = C.green) {
  let s = rect(x, 158, 224, 78, { fill: '#FFFFFF', stroke: '#EAECF0', rx: 12 });
  s += text(x + 16, 181, label, { size: 10.5, fill: C.muted });
  s += text(x + 16, 211, value, { size: 22, weight: 720, fill: C.ink });
  s += text(x + 16, 227, note, { size: 9.3, fill: noteColor });
  return s;
}

function providerRow(y, opts) {
  const { mark, markBg, markColor, name, url, model, status, statusColor, statusBg, latency, defaultTag } = opts;
  let s = line(244, y, 956, y, { stroke: '#F0F2F5' });
  s += rect(264, y + 16, 32, 32, { fill: markBg, rx: 8 });
  s += text(280, y + 37, mark, { size: 12.5, fill: markColor, weight: 750, anchor: 'middle' });
  s += text(306, y + 28, name, { size: 11.8, fill: C.ink2, weight: 650 });
  s += text(306, y + 44, url, { size: 9.2, fill: C.subtle });
  s += text(470, y + 35, model, { size: 10.8, fill: C.body });
  if (defaultTag) s += chip(650, y + 24, 38, '默认', { h: 18, size: 8.2, fill: C.primarySoft, color: C.primary });
  s += chip(700, y + 23, 58, status, { h: 20, size: 9, fill: statusBg, color: statusColor, dot: true });
  s += text(790, y + 36, latency, { size: 10.3, fill: C.muted });
  s += icon('edit', 922, y + 26, 15, C.subtle, 1.6);
  return s;
}

function mainContent() {
  let s = adminSidebar();
  s += `<g transform="translate(220 0)">`;
  s += adminTopbar();
  s += rect(0, 64, 1220, 836, { fill: '#F7F8FA' });
  s += text(24, 112, '模型厂商', { size: 23, weight: 730, fill: C.ink });
  s += text(24, 134, '对接上游模型服务，统一为 OpenAI 兼容协议；用户只看到已启用的模型。', { size: 11.5, fill: C.muted });
  s += `<g transform="translate(-220 0)">`;
  s += rect(842, 88, 118, 38, { fill: C.primary, rx: 9 });
  s += icon('plus', 854, 98, 16, '#FFFFFF', 2);
  s += text(876, 112, '添加厂商', { size: 11.5, fill: '#FFFFFF', weight: 600 });

  s += statCard(244, '已启用厂商', '2', '+1 本周', C.green);
  s += statCard(480, '可用模型', '12', '+2 新增', C.primary);
  s += statCard(716, '今日请求', '12,480', '99.2% 成功', C.green);

  s += rect(244, 254, 300, 38, { fill: '#FFFFFF', stroke: '#E1E5EB', rx: 9 });
  s += icon('search', 258, 265, 15, C.subtle, 1.7);
  s += text(282, 278, '搜索厂商或模型', { size: 10.5, fill: C.subtle });
  s += rect(558, 254, 232, 38, { fill: '#F2F4F7', rx: 9 });
  s += rect(560, 256, 66, 34, { fill: '#FFFFFF', stroke: '#EAECF0', rx: 7 });
  s += text(593, 278, '全部 4', { size: 9.6, fill: C.ink2, weight: 600, anchor: 'middle' });
  s += text(667, 278, '已启用 2', { size: 9.6, fill: C.muted, anchor: 'middle' });
  s += text(748, 278, '异常 1', { size: 9.6, fill: C.muted, anchor: 'middle' });
  s += icon('refresh', 930, 265, 15, C.subtle, 1.7);

  s += rect(244, 308, 712, 386, { fill: '#FFFFFF', stroke: '#EAECF0', rx: 12 });
  s += text(264, 334, '厂商', { size: 10, fill: C.subtle, weight: 600 });
  s += text(470, 334, '默认模型', { size: 10, fill: C.subtle, weight: 600 });
  s += text(700, 334, '状态', { size: 10, fill: C.subtle, weight: 600 });
  s += text(790, 334, '首字延迟', { size: 10, fill: C.subtle, weight: 600 });
  s += line(244, 348, 956, 348, { stroke: '#EEF0F4' });

  s += providerRow(348, {
    mark: 'D', markBg: '#E9EEFF', markColor: C.primary, name: 'DeepSeek 官方',
    url: 'api.deepseek.com/v1', model: 'DeepSeek-V3.2 · R1', status: '已启用',
    statusColor: '#027A48', statusBg: C.greenSoft, latency: '320 ms', defaultTag: true,
  });
  s += providerRow(416, {
    mark: 'A', markBg: '#FFF4E5', markColor: '#F79009', name: '阿里云百炼',
    url: 'dashscope.aliyuncs.com/compatible-mode', model: 'Qwen3-Max', status: '已启用',
    statusColor: '#027A48', statusBg: C.greenSoft, latency: '580 ms',
  });
  s += providerRow(484, {
    mark: 'V', markBg: '#F2F4F7', markColor: C.body, name: '自定义 · vLLM',
    url: '10.0.0.12:8000/v1', model: 'Llama-3.3-70B', status: '已停用',
    statusColor: '#667085', statusBg: '#F2F4F7', latency: '—',
  });

  s += providerRow(552, {
    mark: 'O', markBg: '#FEF3F2', markColor: '#F04438', name: '自建 · Ollama',
    url: 'ollama.internal:11434/v1', model: 'Qwen2.5-32B', status: '连接失败',
    statusColor: '#B42318', statusBg: '#FEF3F2', latency: '—',
  });
  s += rect(264, 634, 672, 38, { fill: '#FBFCFE', stroke: '#DDE5FF', rx: 8, dash: '5 4' });
  s += icon('plus', 282, 644, 15, C.primary, 1.8);
  s += text(306, 658, '添加自定义 OpenAI 兼容接口', { size: 10.6, fill: C.primary2, weight: 550 });
  s += text(914, 658, '支持 /chat/completions', { size: 9, fill: C.subtle, anchor: 'end' });

  s += rect(244, 720, 712, 68, { fill: '#F8FAFF', stroke: '#DDE5FF', rx: 12 });
  s += circle(272, 754, 16, { fill: '#E9EEFF' });
  s += icon('shield', 263, 745, 18, C.primary, 1.7);
  s += text(300, 746, '统一网关与密钥隔离', { size: 11.3, fill: C.ink2, weight: 650 });
  s += text(300, 765, '所有请求经服务端代理；厂商 Key AES-GCM 加密，浏览器不接触上游凭证。', { size: 10.1, fill: C.muted });
  s += `</g>`;
  s += `</g>`;

  // drawer
  s += rect(980, 64, 460, 836, { fill: '#FFFFFF', filter: 'drawerShadow' });
  s += line(980, 64, 980, 900, { stroke: '#E4E8EF' });
  s += text(1004, 101, '添加自定义厂商', { size: 17, weight: 720 });
  s += text(1004, 120, '接入任意 OpenAI 兼容服务，保存后即可被用户选择。', { size: 9.8, fill: C.subtle });
  s += icon('close', 1404, 86, 18, C.muted, 1.7);
  s += line(1004, 138, 1416, 138, { stroke: '#EEF0F4' });

  s += text(1004, 160, '接入方式', { size: 10, fill: C.muted, weight: 550 });
  s += rect(1004, 168, 412, 42, { fill: '#F2F4F7', rx: 9 });
  s += text(1104, 194, '预置厂商', { size: 10.6, fill: C.muted, anchor: 'middle' });
  s += rect(1204, 170, 210, 38, { fill: '#FFFFFF', stroke: '#DDE5FF', rx: 7 });
  s += icon('check', 1218, 181, 13, C.primary, 2);
  s += text(1238, 193, '自定义兼容 API', { size: 10.4, fill: C.primary2, weight: 650 });

  s += text(1004, 232, '显示名称', { size: 10, fill: C.muted });
  s += rect(1004, 240, 412, 40, { fill: '#FFFFFF', stroke: '#E1E5EB', rx: 8 });
  s += text(1020, 265, 'My Gateway', { size: 11.2, fill: C.ink2 });

  s += text(1004, 300, 'Base URL', { size: 10, fill: C.muted });
  s += rect(1004, 308, 412, 40, { fill: '#FFFFFF', stroke: '#E1E5EB', rx: 8 });
  s += text(1020, 333, 'https://gateway.example.com/v1', { size: 10.8, fill: C.ink2 });
  s += text(1004, 363, '需兼容 /v1/chat/completions 与 /v1/models', { size: 8.8, fill: C.subtle });

  s += text(1004, 391, 'API Key', { size: 10, fill: C.muted });
  s += rect(1004, 399, 412, 40, { fill: '#FFFFFF', stroke: '#E1E5EB', rx: 8 });
  s += text(1020, 424, 'sk-••••••••••••••••••••••••', { size: 10.8, fill: C.ink2, family: 'monospace' });
  s += icon('eye', 1386, 411, 16, C.subtle, 1.6);

  s += text(1004, 463, '模型列表', { size: 10, fill: C.muted });
  s += chip(1004, 472, 78, 'gpt-4.1', { h: 24, size: 9.2, fill: '#F5F7FF', color: C.primary2, stroke: '#DDE5FF' });
  s += chip(1090, 472, 92, 'qwen3-max', { h: 24, size: 9.2, fill: '#F5F7FF', color: C.primary2, stroke: '#DDE5FF' });
  s += rect(1190, 472, 92, 24, { fill: '#FFFFFF', stroke: '#D0D5DD', rx: 12, dash: '4 3' });
  s += text(1236, 488, '+ 添加模型', { size: 8.8, fill: C.muted, anchor: 'middle' });
  s += text(1004, 514, '仅启用的模型会出现在用户侧；可稍后编辑能力标签。', { size: 8.8, fill: C.subtle });

  s += text(1004, 544, '超时（秒）', { size: 10, fill: C.muted });
  s += rect(1004, 552, 196, 38, { fill: '#FFFFFF', stroke: '#E1E5EB', rx: 8 });
  s += text(1020, 576, '120', { size: 11, fill: C.ink2 });
  s += text(1220, 544, '最大并发', { size: 10, fill: C.muted });
  s += rect(1220, 552, 196, 38, { fill: '#FFFFFF', stroke: '#E1E5EB', rx: 8 });
  s += text(1236, 576, '20', { size: 11, fill: C.ink2 });

  s += text(1004, 618, '启用该厂商', { size: 10.4, fill: C.ink2 });
  s += text(1004, 634, '保存后立即参与模型路由', { size: 8.8, fill: C.subtle });
  s += rect(1374, 612, 40, 22, { fill: C.primary, rx: 11 });
  s += circle(1403, 623, 8, { fill: '#FFFFFF' });
  s += text(1220, 618, '设为平台默认', { size: 10.4, fill: C.muted });
  s += text(1220, 634, '新用户默认使用', { size: 8.8, fill: C.subtle });
  s += rect(1374, 630, 40, 22, { fill: '#E4E7EC', rx: 11 });
  s += circle(1385, 641, 8, { fill: '#FFFFFF' });

  s += rect(1004, 660, 412, 78, { fill: C.greenSoft, stroke: '#ABEFC6', rx: 10 });
  s += circle(1028, 690, 12, { fill: '#D1FADF' });
  s += icon('check', 1021, 683, 14, '#039855', 2.2);
  s += text(1048, 686, '连接成功', { size: 11, fill: '#027A48', weight: 650 });
  s += text(1048, 706, '延迟 284ms  ·  发现 2 个模型', { size: 9.5, fill: '#027A48' });
  s += text(1384, 694, '重新测试', { size: 9.5, fill: C.primary, weight: 600, anchor: 'end' });

  s += line(1004, 786, 1416, 786, { stroke: '#EEF0F4' });
  s += rect(1004, 812, 112, 42, { fill: '#FFFFFF', stroke: '#D0D5DD', rx: 8 });
  s += text(1060, 838, '取消', { size: 11, fill: C.body, anchor: 'middle' });
  s += rect(1128, 812, 288, 42, { fill: C.primary, rx: 8 });
  s += text(1272, 838, '保存并启用', { size: 11.3, fill: '#FFFFFF', weight: 650, anchor: 'middle' });
  s += text(1004, 878, '保存前会再次测试 /models；失败不会覆盖现有配置。', { size: 8.6, fill: C.subtle });
  return s;
}

const items = [
  { n: 1, title: '厂商抽象', lines: ['统一 Provider 接口，屏蔽协议差异。', '预置 DeepSeek / OpenAI / 阿里百炼 /', 'Moonshot / 智谱 / SiliconFlow。'] },
  { n: 2, title: '自定义兼容', lines: ['仅需 Base URL + Key + 模型名。', '校验 /v1/chat/completions 与流式 SSE 返回。'] },
  { n: 3, title: '安全存储', lines: ['Key 使用 AES-GCM 加密入库，接口只回掩码。', '日志脱敏；浏览器永不接触上游 Key。'] },
  { n: 4, title: '连接测试', lines: ['GET /models 或最小请求，10s 超时。', '成功展示延迟与模型数，失败给出可读原因。'] },
  { n: 5, title: '模型目录', lines: ['从测试结果勾选，设置展示名、上下文与能力标签。', '仅启用的模型下发给用户。'] },
  { n: 6, title: '默认与路由', lines: ['平台默认 + 用户切换；请求按模型映射路由。', 'MVP 不跨厂商自动重试，避免重复计费。'] },
  { n: 7, title: '网络与 SSRF', lines: ['生产禁止私网地址，Base URL 必须 HTTPS。', '本地开发可白名单 127.0.0.1。'] },
  { n: 8, title: '审计与轮换', lines: ['记录配置变更人、时间与前后值。', 'Key 轮换不中断历史会话。'] },
];

const content = `${pageHeader({
  title: '管理后台 · 厂商与模型',
  subtitle: '预置厂商、自定义 OpenAI 兼容接口、连接测试与模型启用 · MVP / P0',
  code: 'REQ-03',
})}
${appFrame(mainContent())}
${callout(1, 244, 408)}
${callout(2, 996, 780)}
${callout(4, 1434, 806)}
${callout(5, 1434, 934)}
${annotationRail(items, {
  header: '配置规则',
  badge: 'ADMIN',
  footerTitle: '验收标准',
  footerLines: ['新增自定义厂商 → 测试成功 → 用户可选模型。', '上游 Key 不出服务端，日志中不可检索。'],
  footerNote: 'P1：按模型计费、失败重试、负载均衡。',
})}`;

writeSvg('docs/requirements/03-admin-providers.svg', content);
