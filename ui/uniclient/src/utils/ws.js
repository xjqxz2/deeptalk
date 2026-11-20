/**
 * WebSocket 流式客户端。
 *
 * 为什么不用 SSE：小程序（微信 / 支付宝等）没有 EventSource，
 * 也无法消费分块响应，官方方案就是 WebSocket。
 *
 * 协议：客户端发 {type:'stream'|'stop'|'ping'}，
 * 服务端回 {type:<事件名>, payload:<与 SSE 一致的数据>}。
 */
import { getToken } from '@/store/session'
import { wsURL } from '@/config/env'

const HEARTBEAT_MS = 25000
const RECONNECT_MAX = 30000

class ChatSocket {
  constructor() {
    this.task = null
    this.status = 'idle' // idle / connecting / open / closed
    this.retry = 0
    this.heartbeat = null
    this.reconnectTimer = null
    this.handlers = new Map()
    this.listeners = new Set()
    this.manualClose = false
  }

  /** 注册事件回调；返回取消函数。 */
  on(event, handler) {
    if (!this.handlers.has(event)) this.handlers.set(event, new Set())
    this.handlers.get(event).add(handler)
    return () => this.handlers.get(event)?.delete(handler)
  }

  /** 订阅连接状态变化。 */
  onStatus(listener) {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  emitStatus() {
    this.listeners.forEach((fn) => fn(this.status))
  }

  /** 建立连接；已连接时直接返回。 */
  connect() {
    if (this.status === 'open' || this.status === 'connecting') return
    const token = getToken()
    if (!token) return
    this.manualClose = false
    this.status = 'connecting'
    this.emitStatus()
    const task = uni.connectSocket({ url: wsURL(token), complete: () => {} })
    this.task = task
    task.onOpen(() => {
      this.status = 'open'
      this.retry = 0
      this.emitStatus()
      this.startHeartbeat()
    })
    task.onMessage((res) => this.handleMessage(res.data))
    task.onClose(() => {
      this.stopHeartbeat()
      this.status = 'idle'
      this.emitStatus()
      if (!this.manualClose) this.scheduleReconnect()
    })
    task.onError(() => {
      this.stopHeartbeat()
      this.status = 'idle'
      this.emitStatus()
      if (!this.manualClose) this.scheduleReconnect()
    })
  }

  /** 断线重连，指数退避，最长 30 秒。 */
  scheduleReconnect() {
    if (this.reconnectTimer) return
    this.retry += 1
    const delay = Math.min(RECONNECT_MAX, 1000 * 2 ** Math.min(this.retry, 5))
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
    }, delay)
  }

  /** 心跳：服务端 120 秒无消息会断开，这里 25 秒一次。 */
  startHeartbeat() {
    this.stopHeartbeat()
    this.heartbeat = setInterval(() => this.send({ type: 'ping' }), HEARTBEAT_MS)
  }

  stopHeartbeat() {
    if (this.heartbeat) clearInterval(this.heartbeat)
    this.heartbeat = null
  }

  /** 解析一帧数据并分发。 */
  handleMessage(raw) {
    let frame
    try {
      frame = JSON.parse(raw)
    } catch {
      return
    }
    if (!frame?.type) return
    const handlers = this.handlers.get(frame.type)
    if (!handlers) return
    handlers.forEach((fn) => fn(frame.payload ?? {}))
  }

  /** 发送一帧；未连接时先连上再发。 */
  send(frame) {
    const payload = JSON.stringify(frame)
    if (this.status === 'open' && this.task) {
      this.task.send({ data: payload })
      return true
    }
    this.connect()
    // 等连接建立后补发，最多等 3 秒
    const startedAt = Date.now()
    const timer = setInterval(() => {
      if (this.status === 'open' && this.task) {
        clearInterval(timer)
        this.task.send({ data: payload })
        return
      }
      if (Date.now() - startedAt > 3000) clearInterval(timer)
    }, 120)
    return false
  }

  /** 主动关闭（退出登录时调用）。 */
  close() {
    this.manualClose = true
    this.stopHeartbeat()
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    this.reconnectTimer = null
    this.task?.close({})
    this.task = null
    this.status = 'idle'
    this.emitStatus()
  }
}

export const chatSocket = new ChatSocket()
