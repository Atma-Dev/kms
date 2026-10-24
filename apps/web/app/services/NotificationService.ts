import { useFlashStore } from '~/stores/flash'
import type AppNotification from '~/types/entities/notification'
import type { ResponseData, ServerPaginationMeta } from '~/types/response'

interface NotificationListResponse extends ResponseData {
  payload: {
    data: AppNotification[]
    meta: ServerPaginationMeta
  }
}

const defaultMeta: ServerPaginationMeta = {
  current_page: 1,
  per_page: 10,
  total: 0,
  last_page: 1,
  from: 0,
  to: 0,
}

const failure = (message: string) => ({ code: 'ERROR', message, status_code: 500, timestamp: '' })

export const useNotificationService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const notifications = ref<AppNotification[]>([])
  const meta = ref<ServerPaginationMeta>({ ...defaultMeta })
  const loading = ref(false)

  const getAll = async (page: number = 1, perPage: number = 10) => {
    loading.value = true
    try {
      const response = await $api<NotificationListResponse>('api/notifications', {
        method: 'GET',
        query: { page, per_page: perPage },
      })
      notifications.value = response.payload.data
      meta.value = response.payload.meta
      return response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load notifications'))
      return []
    } finally {
      loading.value = false
    }
  }

  const markRead = async (id: string) => {
    try {
      await $api(`api/notifications/${id}/read`, { method: 'PATCH' })
    } catch {
      // api plugin already flashed the backend message.
    }
  }

  const markAllRead = async () => {
    try {
      await $api('api/notifications/read-all', { method: 'PATCH' })
    } catch {
      // api plugin already flashed the backend message.
    }
  }

  return { notifications, meta, loading, getAll, markRead, markAllRead }
}
