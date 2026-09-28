import type AppNotification from '~/types/entities/notification'
import { useNotificationService } from '~/services/NotificationService'

export const useNotificationStore = defineStore('notificationStore', () => {
  const items = ref<AppNotification[]>([])
  const unreadCount = computed(() => items.value.filter((n) => !n.is_read).length)

  const fetch = async () => {
    const { getAll } = useNotificationService()
    items.value = await getAll(1, 10)
  }

  const markRead = async (id: string) => {
    const notification = items.value.find((n) => n.id === id)
    if (!notification || notification.is_read) return
    notification.is_read = true
    await useNotificationService().markRead(id)
  }

  const markAllRead = async () => {
    items.value.forEach((n) => (n.is_read = true))
    await useNotificationService().markAllRead()
  }

  // Called by useNotificationSocket when a live event arrives over the WS.
  const receive = (notification: AppNotification) => {
    items.value.unshift(notification)
  }

  return { items, unreadCount, fetch, markRead, markAllRead, receive }
})
