/**
 * Ant Design Vue 主题令牌。
 * 目标：与设计稿的字体、圆角、主色与控件高度保持一致，不在页面里覆盖内部类。
 */
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
