import type { WsEvent } from '@/types'

type Handler = (event: WsEvent) => void

class ChatSocket {
  private ws: WebSocket | null = null
  private handlers = new Set<Handler>()
  private retry = 0
  private closedByUser = false
  private timer: number | null = null

  connect(): void {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return
    }
    const token = localStorage.getItem('mw_token')
    if (!token) return

    this.closedByUser = false
    const protocol = location.protocol === 'https:' ? 'wss' : 'ws'
    const ws = new WebSocket(`${protocol}://${location.host}/ws?token=${encodeURIComponent(token)}`)
    this.ws = ws

    ws.onopen = () => {
      this.retry = 0
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data as string) as WsEvent
        this.handlers.forEach((handler) => handler(data))
      } catch {
        // Ignore malformed frames.
      }
    }

    ws.onclose = () => {
      this.ws = null
      if (this.closedByUser) return
      const delay = Math.min(1000 * 2 ** this.retry, 10_000)
      this.retry += 1
      this.timer = window.setTimeout(() => this.connect(), delay)
    }
  }

  send(payload: unknown): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(payload))
    }
  }

  on(handler: Handler): () => void {
    this.handlers.add(handler)
    return () => this.handlers.delete(handler)
  }

  disconnect(): void {
    this.closedByUser = true
    if (this.timer) {
      window.clearTimeout(this.timer)
      this.timer = null
    }
    this.ws?.close()
    this.ws = null
  }
}

export const chatSocket = new ChatSocket()
