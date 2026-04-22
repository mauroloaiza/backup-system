import { create } from 'zustand'
import {
  fetchNotifications,
  markNotificationRead,
  markAllNotificationsRead,
  type AppNotification,
} from '@/lib/api'

interface NotificationsState {
  notifications: AppNotification[]
  unread: number
  loaded: boolean
  loading: boolean
  load: () => Promise<void>
  markRead: (id: number) => Promise<void>
  markAllRead: () => Promise<void>
  clear: () => void
}

export const useNotifications = create<NotificationsState>((set, get) => ({
  notifications: [],
  unread: 0,
  loaded: false,
  loading: false,

  load: async () => {
    if (get().loading) return
    set({ loading: true })
    try {
      const { notifications, unread } = await fetchNotifications(false)
      set({ notifications, unread, loaded: true })
    } catch {
      // ignore — keep previous state
    } finally {
      set({ loading: false })
    }
  },

  markRead: async (id: number) => {
    const { notifications, unread } = get()
    const target = notifications.find(n => n.id === id)
    if (!target || target.read) return
    // Optimistic update
    set({
      notifications: notifications.map(n => n.id === id ? { ...n, read: true } : n),
      unread: Math.max(0, unread - 1),
    })
    try {
      await markNotificationRead(id)
    } catch {
      // Rollback on error
      get().load()
    }
  },

  markAllRead: async () => {
    const { notifications } = get()
    set({
      notifications: notifications.map(n => ({ ...n, read: true })),
      unread: 0,
    })
    try {
      await markAllNotificationsRead()
    } catch {
      get().load()
    }
  },

  clear: () => set({ notifications: [], unread: 0, loaded: false }),
}))
