import type { Component } from 'vue'
import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    /** 浏览器标题与面包屑文案 */
    title?: string
    /** 无需管理员身份即可访问 */
    public?: boolean
    /** 是否出现在左侧控制台菜单 */
    menu?: boolean
    /** 菜单图标组件 */
    icon?: Component
  }
}
