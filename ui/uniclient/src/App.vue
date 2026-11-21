<script>
/**
 * 应用入口：恢复登录态，未登录直接跳登录页。
 */
import { session, isLoggedIn, setSession, clearSession } from '@/store/session'
import { goLogin } from '@/utils/nav'
import { fetchMe } from '@/api/auth'

export default {
  onLaunch() {
    // 仅供本地预览页（pages/mp-preview）跳过登录重定向
    // #ifdef H5
    if (typeof location !== 'undefined' && location.hash.includes('mp-preview')) return
    // #endif
    // 启动阶段不做跳转：微信端首屏路由还没完成时 reLaunch 会报
    // "routeDone with a webviewId N is not found"，交给首屏页面自己判断
    if (!isLoggedIn()) return
    // 校验 token 是否还有效，并刷新额度信息
    fetchMe()
      .then((data) => setSession(session.token, data.user))
      .catch(() => {
        // 会话失效：清掉本地登录态，跳转统一交给 goLogin 去重
        clearSession()
        goLogin()
      })
  },
}
</script>

<style>
/* 全局基础样式：nvue 页面只依赖这里与组件内的类选择器 */
page {
  height: 100%;
  background-color: #ffffff;
}

.page {
  flex: 1;
  height: 100%;
  background-color: #ffffff;
}

/* H5 端页面容器默认高度是 auto，这里补齐才能让消息区撑满、输入框贴底 */
/* #ifdef H5 */
uni-page-body {
  height: 100%;
}
/* #endif */
</style>
