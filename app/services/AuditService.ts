import { useFlashStore } from '~/stores/flash'
import type Audit from '~/types/entities/audit'
import type { ResponseData, ServerPaginationMeta } from '~/types/response'

interface AuditListResponse extends ResponseData {
  payload: {
    data: Audit[]
    meta: ServerPaginationMeta
  }
}

export interface AuditFilters {
  entry_object_table?: string
  entry_object_id?: string
  actor_id?: string
  action?: string
  date_from?: string
  date_to?: string
}

const buildQuery = (page: number, perPage: number, filters: AuditFilters) => ({
  page,
  per_page: perPage,
  ...(filters.entry_object_table ? { entry_object_table: filters.entry_object_table } : {}),
  ...(filters.entry_object_id ? { entry_object_id: filters.entry_object_id } : {}),
  ...(filters.actor_id ? { actor_id: filters.actor_id } : {}),
  ...(filters.action ? { action: filters.action } : {}),
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

export const useAuditService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const audits = ref<Audit[]>([])
  const meta = ref<ServerPaginationMeta>({ ...defaultMeta })
  const loading = ref(false)

  const getAll = async (page: number = 1, perPage: number = 10, filters: AuditFilters = {}) => {
    loading.value = true
    try {
      const response = await $api<AuditListResponse>('api/management/audits', {
        method: 'GET',
        query: buildQuery(page, perPage, filters),
      })
      audits.value = response.payload.data
      meta.value = response.payload.meta
    } catch (e: any) {
      flashStore.setFailed(
        e?.data ?? { code: 'ERROR', message: 'Failed to load audits', status_code: 500, timestamp: '' },
      )
    } finally {
      loading.value = false
    }
  }

  return {
    audits,
    meta,
    loading,
    getAll,
  }
}
