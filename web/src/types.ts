export interface User {
  id: number
  username: string
  nickname: string
  avatar: string
  signature?: string
}

export interface Message {
  id: number
  conversationId: number
  senderId: number
  type: 'text' | 'image'
  content: string
  createdAt: number
  sender?: User
}

export type ConversationType = 'direct' | 'group'

export interface Conversation {
  id: number
  type: ConversationType
  name: string
  ownerId: number
  createdAt: number
  members: User[]
  lastMessage: Message | null
  unread: number
  lastReadMessageId: number
}

export interface MomentComment {
  id: number
  user: User
  content: string
  createdAt: number
}

export interface Moment {
  id: number
  user: User
  content: string
  images: string[]
  likes: User[]
  comments: MomentComment[]
  createdAt: number
  likedByMe: boolean
}

export type WsEvent =
  | { type: 'ready' }
  | { type: 'message'; payload: Message }
  | { type: 'typing'; payload: { conversationId: number; userId: number } }

export type Tab = 'chats' | 'contacts' | 'discover' | 'me'
