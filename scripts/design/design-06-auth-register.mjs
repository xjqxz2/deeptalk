import { C, text, rect, circle, line, icon } from '../requirements/req-lib.mjs';
import { frame, sheetHeader, specRail, pinLabel, writeSvg, g, field, btn } from './design-lib.mjs';
import { authLeft, authRight, topLink } from './auth-lib.mjs';

const FX = 840, FW = 380, FR = FX + FW;

function form() {
  const parts = [];
  parts.push(topLink('已有账号？', '去登录'));
  parts.push(text(FX, 300, 'CREATE ACCOUNT', { size: 8.8, weight: 620, fill: C.subtle, spacing: 1.5 }));
  parts.push(text(FX, 344, '创建账号', { size: 30, weight: 730, spacing: -0.5 }));
  parts.push(text(FX, 372, '使用邮箱注册，注册后直接进入工作台。', { size: 12.2, fill: C.muted }));

  parts.push(field(FX, 434, FW, '邮箱', { iconName: 'mail', value: 'you@company.com', focus: false }));
  parts.push(field(FX, 520, FW, '密码', {
    iconName: 'lock',
    value: '••••••••',
    trailing: icon('eye', FR - 34, 535, 16, C.subtle, 1.7),
  }));
  parts.push(field(FX, 606, FW, '确认密码', {
    iconName: 'lock',
    value: '••••••••',
    trailing: icon('eye', FR - 34, 621, 16, C.subtle, 1.7),
  }));

  const checks = [['至少 8 位', 0], ['字母 + 数字', 82], ['无需邮箱验证', 178]];
  checks.forEach(([label, dx]) => {
    parts.push(icon('check', FX + dx, 676, 12, C.green, 2.4));
    parts.push(text(FX + dx + 16, 686, label, { size: 9.2, fill: C.muted }));
  });

  parts.push(btn(FX, 708, FW, 46, '创建账号并进入', { primary: true, radius: 11, size: 13.2 }));
  parts.push(icon('alert', FX + 1, 770, 13, C.faint, 1.8));
  parts.push(text(FX + 19, 780, '注册成功即登录；首个账号为平台管理员。', { size: 9.2, fill: C.subtle }));

  parts.push(line(FX, 802, FR, 802, { stroke: '#F0F2F6' }));
  parts.push(icon('lock', FX, 820, 13, C.faint, 1.8));
  parts.push(text(FX + 18, 830, '不发送验证邮件 · 忘记密码由管理员重置', { size: 9.2, fill: C.faint }));

  parts.push(pinLabel(1, 608, 300));
  parts.push(pinLabel(3, 826, 616));
  parts.push(pinLabel(2, 1234, 682));
  parts.push(pinLabel(4, 1234, 776));
  parts.push(pinLabel(5, 1234, 826));
  return g(...parts);
}

const inner = g(authLeft(), authRight(form()));

const notes = [
  { n: 1, title: '品牌区', lines: ['与登录页复用同一左栏，仅切换', '右侧表单，保证两页视觉连续。'] },
  { n: 2, title: '分步校验', lines: ['密码实时校验 ≥8 位且含字母与', '数字，未满足时按钮禁用。'] },
  { n: 3, title: '确认密码', lines: ['失焦即校验一致性，错误定位到', '字段下方，不弹窗打断。'] },
  { n: 4, title: '注册即登录', lines: ['成功直接进工作台，无验证邮件；', '首个账号写为平台管理员。'] },
  { n: 5, title: '关闭注册', lines: ['后台关闭后该页显示「注册已关闭，', '请联系管理员」，隐藏表单。'] },
];

writeSvg('docs/design/06-auth-register.svg', sheetHeader({
  title: '注册 · 邮箱密码',
  sub: '无验证邮件的注册：填完即进入工作台，首个账号自动成为管理员',
  code: 'DSG-06 / AUTH-REGISTER',
  meta: '1440 × 900 · Web',
}) + frame(32, 142, 1440, 900, 'screen06', inner) + specRail({
  items: notes,
  footerLines: ['邮箱不区分大小写且唯一', '密码 Argon2id，前端不明文留存', '提交防抖，避免重复创建'],
}), 1800, 1080);
