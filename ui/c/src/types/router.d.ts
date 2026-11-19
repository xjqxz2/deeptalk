import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    /** 浏览器标题 */
    title?: string
    /** 无需登录即可访问 */
    public?: boolean
    /** 需要登录 */
    requiresAuth?: boolean
    /** 布局标识：auth 使用认证布局 */
    layout?: string
  }
}
