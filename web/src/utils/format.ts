import type { Conversation } from '@/types'

const WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']

function pad(n: number): string {
  return n < 10 ? `0${n}` : String(n)
}

function isSameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  )
}

function startOfDay(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), d.getDate())
}

export function formatListTime(ts: number): string {
  const d = new Date(ts)
  const now = new Date()
  if (isSameDay(d, now)) return `${pad(d.getHours())}:${pad(d.getMinutes())}`
  const yesterday = new Date(now)
  yesterday.setDate(now.getDate() - 1)
  if (isSameDay(d, yesterday)) return '昨天'
  const diffDays = Math.round((startOfDay(now).getTime() - startOfDay(d).getTime()) / 86400000)
  if (diffDays > 1 && diffDays < 7) return WEEKDAYS[d.getDay()]
  if (d.getFullYear() === now.getFullYear()) return `${d.getMonth() + 1}月${d.getDate()}日`
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`
}

export function formatDayLabel(ts: number): string {
  const d = new Date(ts)
  const now = new Date()
  if (isSameDay(d, now)) return '今天'
  const yesterday = new Date(now)
  yesterday.setDate(now.getDate() - 1)
  if (isSameDay(d, yesterday)) return '昨天'
  if (d.getFullYear() === now.getFullYear()) return `${d.getMonth() + 1}月${d.getDate()}日`
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`
}

export function formatMsgTime(ts: number): string {
  const d = new Date(ts)
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function formatRelative(ts: number): string {
  const diff = Date.now() - ts
  if (diff < 60_000) return '刚刚'
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} 分钟前`
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} 小时前`
  if (diff < 172_800_000) return '昨天'
  if (diff < 604_800_000) return `${Math.floor(diff / 86_400_000)} 天前`
  const d = new Date(ts)
  const now = new Date()
  if (d.getFullYear() === now.getFullYear()) return `${d.getMonth() + 1}月${d.getDate()}日`
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`
}

export function conversationTitle(c: Conversation, meID?: number): string {
  if (c.type === 'group') return c.name || '群聊'
  const other = c.members.find((m) => m.id !== meID)
  return other?.nickname || '会话'
}

export function conversationAvatar(
  c: Conversation,
  meID?: number,
): { name: string; src: string; square: boolean } {
  if (c.type === 'group') {
    return { name: c.name || '群聊', src: '', square: true }
  }
  const other = c.members.find((m) => m.id !== meID)
  return { name: other?.nickname || '会话', src: other?.avatar || '', square: false }
}

export function errorText(err: unknown): string {
  if (err instanceof Error) return err.message
  return '操作失败，请稍后再试'
}
