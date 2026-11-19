import {
  C, text, rect, circle, line, icon, brandMark, pageHeader, appFrame,
  callout, annotationRail, writeSvg, chip, MONO,
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
    ['users', '用户管理', 202, true],
    ['server', '模型厂商', 244, false],
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
  s += text(91, 38, '用户管理', { size: 11.5, fill: C.ink2, weight: 600 });
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

function userRow(y, opts) {
  const { initial, avatarBg, avatarColor, name, email, registered, usage, plan, status, statusColor, statusBg, lastSeen, admin, selected } = opts;
  let s = line(24, y, 736, y, { stroke: '#F0F2F5' });
  if (selected) s += rect(24, y + 1, 712, 61, { fill: '#F8FAFF' });
  s += circle(48, y + 31, 15, { fill: avatarBg });
  s += text(48, y + 35, initial, { size: 10.5, fill: avatarColor, weight: 700, anchor: 'middle' });
  s += text(72, y + 25, name, { size: 11.2, fill: C.ink2, weight: 650 });
  if (admin) s += chip(154, y + 15, 42, '管理员', { h: 16, size: 7.2, fill: C.primarySoft, color: C.primary });
  s += text(72, y + 41, email, { size: 8.9, fill: C.subtle });
  s += text(262, y + 35, registered, { size: 9.4, fill: C.body });
  s += text(360, y + 35, usage, { size: 9.4, fill: C.body });
  const planW = plan === 'Pro' ? 38 : 48;
  s += chip(468, y + 22, planW, plan, { h: 19, size: 8.3, fill: plan === 'Pro' ? '#F5F3FF' : '#F2F4F7', color: plan === 'Pro' ? '#6941C6' : C.muted });
  s += chip(552, y + 23, 50, status, { h: 19, size: 8.4, fill: statusBg, color: statusColor, dot: true });
  s += text(630, y + 35, lastSeen, { size: 9.3, fill: C.muted });
  s += icon('more', 710, y + 23, 15, C.subtle, 1.7);
  return s;
}

function mainContent() {
  let s = adminSidebar();
  s += `<g transform="translate(220 0)">`;
  s += adminTopbar();
  s += rect(0, 64, 1220, 836, { fill: '#F7F8FA' });
  s += text(24, 112, '用户管理', { size: 23, weight: 730, fill: C.ink });
  s += text(24, 134, 'C 端账号、额度与访问状态；注册开放，无邮箱验证。', { size: 11.5, fill: C.muted });

  s += text(470, 112, '开放注册', { size: 10.2, fill: C.body, weight: 550 });
  s += rect(536, 94, 38, 22, { fill: C.primary, rx: 11 });
  s += circle(563, 105, 8, { fill: '#FFFFFF' });
  s += rect(622, 88, 114, 38, { fill: '#FFFFFF', stroke: '#D0D5DD', rx: 9 });
  s += icon('download', 636, 99, 15, C.body, 1.8);
  s += text(658, 112, '导出 CSV', { size: 10.6, fill: C.body, weight: 550 });

  s += statCard(24, '用户总数', '1,248', '+42 本周', C.green);
  s += statCard(264, '今日活跃', '386', '31% 日活', C.primary);
  s += statCard(504, '本月消息', '48.2K', '+18% 环比', C.green);

  s += rect(24, 254, 300, 38, { fill: '#FFFFFF', stroke: '#E1E5EB', rx: 9 });
  s += icon('search', 38, 265, 15, C.subtle, 1.7);
  s += text(62, 278, '搜索昵称或邮箱', { size: 10.5, fill: C.subtle });
  s += rect(338, 254, 232, 38, { fill: '#F2F4F7', rx: 9 });
  s += rect(340, 256, 66, 34, { fill: '#FFFFFF', stroke: '#EAECF0', rx: 7 });
  s += text(373, 278, '全部 1,248', { size: 9.4, fill: C.ink2, weight: 600, anchor: 'middle' });
  s += text(447, 278, '正常 1,241', { size: 9.4, fill: C.muted, anchor: 'middle' });
  s += text(528, 278, '受限 7', { size: 9.4, fill: C.muted, anchor: 'middle' });
  s += icon('refresh', 710, 265, 15, C.subtle, 1.7);

  s += rect(24, 308, 712, 386, { fill: '#FFFFFF', stroke: '#EAECF0', rx: 12 });
  s += text(44, 334, '用户', { size: 10, fill: C.subtle, weight: 600 });
  s += text(262, 334, '注册时间', { size: 10, fill: C.subtle, weight: 600 });
  s += text(360, 334, '本月用量', { size: 10, fill: C.subtle, weight: 600 });
  s += text(468, 334, '套餐', { size: 10, fill: C.subtle, weight: 600 });
  s += text(552, 334, '状态', { size: 10, fill: C.subtle, weight: 600 });
  s += text(630, 334, '最近活跃', { size: 10, fill: C.subtle, weight: 600 });
  s += line(24, 348, 736, 348, { stroke: '#EEF0F4' });

  s += userRow(348, {
    initial: 'L', avatarBg: '#DFE6FF', avatarColor: C.primary2, name: '林默', email: 'lin@deeptalk.app',
    registered: '2026-10-02', usage: '320K / 500K', plan: '免费版', status: '正常',
    statusColor: '#027A48', statusBg: C.greenSoft, lastSeen: '2 分钟前', admin: true,
  });
  s += userRow(410, {
    initial: '陈', avatarBg: '#FFF4E5', avatarColor: '#B54708', name: '陈一', email: 'chenyi@example.com',
    registered: '2026-10-18', usage: '128K / 2M', plan: 'Pro', status: '正常',
    statusColor: '#027A48', statusBg: C.greenSoft, lastSeen: '1 小时前', selected: true,
  });
  s += userRow(472, {
    initial: '苏', avatarBg: '#ECFDF3', avatarColor: '#027A48', name: '苏打', email: 'soda@example.com',
    registered: '2026-10-21', usage: '62K / 500K', plan: '免费版', status: '正常',
    statusColor: '#027A48', statusBg: C.greenSoft, lastSeen: '昨天',
  });
  s += userRow(534, {
    initial: '王', avatarBg: '#F5F3FF', avatarColor: '#6941C6', name: '王可', email: 'wangke@example.com',
    registered: '2026-11-01', usage: '480K / 2M', plan: 'Pro', status: '正常',
    statusColor: '#027A48', statusBg: C.greenSoft, lastSeen: '3 天前',
  });
  s += userRow(596, {
    initial: 'T', avatarBg: '#F2F4F7', avatarColor: C.muted, name: 'test_01', email: 'test@example.com',
    registered: '2026-11-12', usage: '12K / 500K', plan: '免费版', status: '受限',
    statusColor: '#B42318', statusBg: C.redSoft, lastSeen: '7 天前',
  });

  s += rect(24, 720, 712, 68, { fill: '#F8FAFF', stroke: '#DDE5FF', rx: 12 });
  s += circle(52, 754, 16, { fill: '#E9EEFF' });
  s += icon('shield', 43, 745, 18, C.primary, 1.7);
  s += text(80, 746, 'C 端隐私边界', { size: 11.3, fill: C.ink2, weight: 650 });
  s += text(80, 765, '管理员只查看账号、用量与最近登录信息，不读取对话正文；CSV 仅导出账号字段。', { size: 10.1, fill: C.muted });
  s += `</g>`;

  // User detail drawer
  s += rect(980, 64, 460, 836, { fill: '#FFFFFF', filter: 'drawerShadow' });
  s += line(980, 64, 980, 900, { stroke: '#E4E8EF' });
  s += text(1004, 101, '用户详情', { size: 17, weight: 720 });
  s += text(1004, 120, '账号、额度与访问状态；不展示对话内容。', { size: 9.8, fill: C.subtle });
  s += icon('close', 1404, 86, 18, C.muted, 1.7);
  s += line(1004, 138, 1416, 138, { stroke: '#EEF0F4' });

  s += rect(1004, 150, 412, 96, { fill: '#F8FAFF', stroke: '#DDE5FF', rx: 10 });
  s += circle(1034, 184, 20, { fill: '#FFF4E5' });
  s += text(1034, 189, '陈', { size: 12, fill: '#B54708', weight: 700, anchor: 'middle' });
  s += text(1066, 179, '陈一', { size: 13, fill: C.ink2, weight: 680 });
  s += text(1066, 197, 'chenyi@example.com', { size: 9.3, fill: C.subtle });
  s += text(1066, 215, 'usr_7d41c9e2  ·  2026-10-18 注册', { size: 8.7, fill: C.muted, family: MONO });
  s += chip(1348, 162, 52, '普通用户', { h: 19, size: 7.8, fill: '#F2F4F7', color: C.muted });
  s += chip(1348, 188, 38, 'Pro', { h: 19, size: 8.2, fill: '#F5F3FF', color: '#6941C6' });

  const mini = [
    { x: 1004, label: '对话', value: '42' },
    { x: 1144, label: '本月 Tokens', value: '128K' },
    { x: 1284, label: '当前额度', value: '2M' },
  ];
  for (const m of mini) {
    s += rect(m.x, 262, 128, 74, { fill: '#FFFFFF', stroke: '#EAECF0', rx: 10 });
    s += text(m.x + 14, 284, m.label, { size: 9.2, fill: C.subtle });
    s += text(m.x + 14, 313, m.value, { size: 17, fill: C.ink, weight: 720 });
  }

  s += rect(1004, 352, 412, 92, { fill: '#FFFFFF', stroke: '#EAECF0', rx: 10 });
  s += text(1020, 376, '本月额度', { size: 9.8, fill: C.muted });
  s += text(1400, 376, '128K / 2M', { size: 9.6, fill: C.ink2, weight: 600, anchor: 'end' });
  s += rect(1020, 390, 380, 7, { fill: '#EAECF0', rx: 3.5 });
  s += rect(1020, 390, 24, 7, { fill: C.primary, rx: 3.5 });
  s += text(1020, 414, '6% 已用', { size: 8.8, fill: C.subtle });
  s += text(1400, 414, '调整额度', { size: 9.3, fill: C.primary, weight: 600, anchor: 'end' });
  s += text(1020, 432, '额度用尽后仍可登录，但无法发起新对话。', { size: 8.6, fill: C.subtle });

  s += text(1004, 476, '访问权限', { size: 10, fill: C.muted, weight: 600 });
  const toggles = [
    ['允许登录', '关闭后所有会话立即失效', true, true],
    ['使用 Pro 模型', '跟随 Pro 套餐', true, true],
    ['上传文件', 'MVP 暂未开放', false, false],
  ];
  toggles.forEach(([label, hint, on, enabled], i) => {
    const y = 490 + i * 36;
    s += text(1004, y, label, { size: 10.3, fill: enabled ? C.ink2 : C.subtle, weight: 550 });
    s += text(1004, y + 14, hint, { size: 8.3, fill: C.faint });
    s += rect(1376, y - 7, 40, 22, { fill: on ? (enabled ? C.primary : '#AFC0FF') : '#E4E7EC', rx: 11, opacity: enabled ? 1 : 0.65 });
    s += circle(on ? 1405 : 1387, y + 4, 8, { fill: '#FFFFFF' });
  });

  s += rect(1004, 608, 200, 40, { fill: '#FFFFFF', stroke: '#D0D5DD', rx: 8 });
  s += icon('refresh', 1022, 619, 15, C.body, 1.7);
  s += text(1046, 633, '重置密码', { size: 10.3, fill: C.body, weight: 550 });
  s += rect(1216, 608, 200, 40, { fill: '#FFFFFF', stroke: '#FDA29B', rx: 8 });
  s += icon('lock', 1234, 619, 15, '#B42318', 1.7);
  s += text(1258, 633, '限制登录', { size: 10.3, fill: '#B42318', weight: 550 });

  s += line(1004, 668, 1416, 668, { stroke: '#EEF0F4' });
  s += text(1004, 692, '账号信息', { size: 9.8, fill: C.muted, weight: 600 });
  const info = [
    ['注册时间', '2026-10-18 20:14'],
    ['最近登录', '1 小时前 · 192.168.1.8'],
    ['注册方式', '邮箱注册 · 无验证'],
  ];
  info.forEach(([k, v], i) => {
    const y = 714 + i * 20;
    s += text(1004, y, k, { size: 9.1, fill: C.subtle });
    s += text(1400, y, v, { size: 9.1, fill: C.body, anchor: 'end' });
  });
  s += text(1004, 786, '删除账号', { size: 9.6, fill: '#B42318', weight: 550 });
  s += text(1090, 786, '7 天冷静期 · P1', { size: 8.4, fill: C.faint });
  s += rect(1004, 812, 412, 42, { fill: C.primary, rx: 8 });
  s += text(1210, 838, '完成', { size: 11.5, fill: '#FFFFFF', weight: 650, anchor: 'middle' });
  s += text(1004, 878, '管理员操作会记录到审计日志。', { size: 8.5, fill: C.subtle });
  return s;
}

const items = [
  { n: 1, title: '用户即账号', lines: ['邮箱唯一，注册即用，无验证邮件。', '对话、文件与用量按账号隔离。'] },
  { n: 2, title: '角色', lines: ['首个注册账号为平台管理员。', '其余均为普通用户，账号平权。'] },
  { n: 3, title: '套餐与额度', lines: ['按用户统计 tokens；免费 / Pro 仅影响额度与模型权限。', '额度用尽仍可登录，但不能发起新对话。'] },
  { n: 4, title: '访问控制', lines: ['限制登录立即生效，已签发会话同时失效。', '管理员可恢复；恢复后需重新登录。'] },
  { n: 5, title: '密码管理', lines: ['无邮件找回；管理员重置或 CLI 强制重置。', '重置后旧会话全部失效。'] },
  { n: 6, title: '隐私边界', lines: ['只展示账号、用量与最近登录。', '不读取对话正文，CSV 仅导出账号字段。'] },
  { n: 7, title: '删除账号', lines: ['7 天冷静期，期间可恢复。', '到期清除账号、会话与消息（P1）。'] },
  { n: 8, title: '审计', lines: ['记录限制、重置、额度调整、删除。', '包含操作者、IP 与时间。'] },
];

const content = `${pageHeader({
  title: '管理后台 · 用户管理',
  subtitle: 'C 端账号、用量与访问状态 · 单实例 · 用户自助注册',
  code: 'REQ-04',
})}
${appFrame(mainContent())}
${callout(1, 244, 364)}
${callout(2, 996, 530)}
${callout(3, 1456, 568)}
${callout(4, 1456, 650)}
${callout(5, 1456, 760)}
${annotationRail(items, {
  header: '用户与账号',
  badge: 'ADMIN',
  footerTitle: '验收标准',
  footerLines: ['注册 → 登录 → 对话；受限用户无法登录。', '管理员可重置密码、限制登录、调整额度。'],
  footerNote: 'P1：套餐/支付、邀请、导出、注销自助。',
})}`;

writeSvg('docs/requirements/04-admin-users.svg', content);
