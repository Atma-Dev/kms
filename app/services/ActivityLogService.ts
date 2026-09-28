import { useFlashStore } from '~/stores/flash'
import type ActivityLog from '~/types/entities/activity_log'
import type { ResponseData, ServerPaginationMeta } from '~/types/response'

interface ActivityLogListResponse extends ResponseData {
  payload: {
    data: ActivityLog[]
    meta: ServerPaginationMeta
  }
}

export interface ActivityLogFilters {
  search?: string
  method?: string
  status_code?: string
  user_id?: string
  date_from?: string
  date_to?: string
}

const buildQuery = (page: number, perPage: number, filters: ActivityLogFilters) => ({
  page,
  per_page: perPage,
  ...(filters.search ? { search: filters.search } : {}),
  ...(filters.method ? { method: filters.method } : {}),
  ...(filters.status_code ? { status_code: filters.status_code } : {}),
  ...(filters.user_id ? { user_id: filters.user_id } : {}),
  ...(filters.date_from ? { date_from: filters.date_from } : {}),
  ...(filters.date_to ? { date_to: filters.date_to } : {}),
})

const defaultMeta: ServerPaginationMeta = {
  current_page: 1,
  per_page: 10,
  total: 0,
  last_page: 1,
  from: 0,
  to: 0,
}

export const useActivityLogService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const logs = ref<ActivityLog[]>([])
  const meta = ref<ServerPaginationMeta>({ ...defaultMeta })
  const loading = ref(false)

  const getAll = async (page: number = 1, perPage: number = 10, filters: ActivityLogFilters = {}) => {
    loading.value = true
    try {
      const response = await $api<ActivityLogListResponse>('api/management/activity-logs', {
        method: 'GET',
        query: buildQuery(page, perPage, filters),
      })
      logs.value = response.payload.data
      meta.value = response.payload.meta
    } catch (e: any) {
      flashStore.setFailed(
        e?.data ?? { code: 'ERROR', message: 'Failed to load activity logs', status_code: 500, timestamp: '' },
      )
    } finally {
      loading.value = false
    }
  }

  return {
    logs,
    meta,
    loading,
    getAll,
  }
}
