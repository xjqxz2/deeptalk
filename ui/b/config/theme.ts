/**
 * Ant Design Vue 主题令牌。
 * 目标：与设计稿的字体、圆角、主色与控件高度保持一致，不在页面里覆盖内部类。
 */
import { theme as antdAlgorithms } from 'ant-design-vue'
import type { ThemeConfig } from 'ant-design-vue/es/config-provider/context'

/** 设计稿字体栈：Inter 优先，中文回退到苹方 / 雅黑。 */
export const FONT_FAMILY =
  "Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', 'Noto Sans SC', sans-serif"

/** 全局主题。 */
export const antdTheme: ThemeConfig = {
  token: {
    colorPrimary: '#4D6BFE',
    colorInfo: '#4D6BFE',
    colorSuccess: '#12B76A',
    colorWarning: '#F79009',
    colorError: '#F04438',
    colorText: '#101828',
    colorTextSecondary: '#475467',
    colorTextTertiary: '#667085',
    colorTextQuaternary: '#98A2B3',
    colorBorder: '#EAECF0',
    colorBorderSecondary: '#EFF1F5',
    colorBgLayout: '#F7F8FA',
    colorBgContainer: '#FFFFFF',
    colorFillSecondary: '#F4F6FA',
    borderRadius: 10,
    borderRadiusLG: 14,
    borderRadiusSM: 8,
    controlHeight: 36,
    fontFamily: FONT_FAMILY,
    fontSize: 13.4,
    lineHeight: 1.55,
    boxShadow: '0 1px 2px rgba(16, 24, 40, 0.04)',
    boxShadowSecondary: '0 14px 26px rgba(16, 24, 40, 0.09)',
  },
  components: {
    Button: {
      borderRadius: 10,
    },
    Input: {
      borderRadius: 10,
    },
    Select: {
      borderRadius: 10,
    },
    Modal: {
      borderRadiusLG: 16,
    },
    Tooltip: {
      colorBgSpotlight: '#101828',
    },
  },
}

/**
 * 深色主题：跟随系统（prefers-color-scheme）。
 * 用 antd 的 darkAlgorithm 覆盖默认算法，再把令牌换成与 theme.css 深色变量一致的值，
 * 这样 antd 组件（输入框、下拉、表格、弹窗）与页面自定义样式是同一套色板。
 */
export const darkAntdTheme: ThemeConfig = {
  ...antdTheme,
  algorithm: antdAlgorithms.darkAlgorithm,
  token: {
    ...antdTheme.token,
    colorPrimary: '#7C93FF',
    colorInfo: '#7C93FF',
    colorSuccess: '#3DDC97',
    colorWarning: '#F7B955',
    colorError: '#FF6B6B',
    colorText: '#E6E9EF',
    colorTextSecondary: '#B6BDCB',
    colorTextTertiary: '#9AA3B2',
    colorTextQuaternary: '#7B8494',
    colorBorder: '#2A2F3A',
    colorBorderSecondary: '#23272F',
    colorBgLayout: '#12151B',
    colorBgContainer: '#191C23',
    colorBgElevated: '#1F232B',
    colorFillSecondary: '#1F232B',
    boxShadow: '0 1px 2px rgba(0, 0, 0, 0.4)',
    boxShadowSecondary: '0 14px 26px rgba(0, 0, 0, 0.45)',
  },
  components: {
    ...antdTheme.components,
    Tooltip: { colorBgSpotlight: '#E6E9EF', colorTextLightSolid: '#12151B' },
  },
}

/**
 * 监听系统深浅色：
 * 返回当前是否深色，并在系统主题变化时回调，组件里据此切换 antdTheme。
 */
export function watchColorScheme(onChange: (dark: boolean) => void): () => void {
  if (typeof window === 'undefined' || !window.matchMedia) return () => {}
  const query = window.matchMedia('(prefers-color-scheme: dark)')
  const apply = () => onChange(query.matches)
  apply()
  query.addEventListener('change', apply)
  return () => query.removeEventListener('change', apply)
}
