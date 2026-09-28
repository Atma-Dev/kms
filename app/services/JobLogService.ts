import { useFlashStore } from '~/stores/flash'
import type JobLog from '~/types/entities/job_log'
import type { JobDefinition, JobLogStats } from '~/types/entities/job_log'
import type { ResponseData, ResponseSingleData, ServerPaginationMeta } from '~/types/response'

interface JobLogListResponse extends ResponseData {
  payload: {
    data: JobLog[]
    meta: ServerPaginationMeta
  }
}

export interface JobLogFilters {
  type?: string
  name?: string
  status?: string
}

const buildQuery = (page: number, perPage: number, filters: JobLogFilters) => ({
  page,
  per_page: perPage,
  ...(filters.type ? { type: filters.type } : {}),
  ...(filters.name ? { name: filters.name } : {}),
  ...(filters.status ? { status: filters.status } : {}),
})

const defaultMeta: ServerPaginationMeta = {
  current_page: 1,
  per_page: 10,
  total: 0,
  last_page: 1,
  from: 0,
  to: 0,
}

export const useJobLogService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const logs = ref<JobLog[]>([])
  const meta = ref<ServerPaginationMeta>({ ...defaultMeta })
  const definitions = ref<JobDefinition[]>([])
  const stats = ref<JobLogStats | null>(null)
  const loading = ref(false)

  const getAll = async (page: number = 1, perPage: number = 10, filters: JobLogFilters = {}) => {
    loading.value = true
    try {
      const response = await $api<JobLogListResponse>('api/management/job-logs', {
        method: 'GET',
        query: buildQuery(page, perPage, filters),
      })
      logs.value = response.payload.data
      meta.value = response.payload.meta
    } catch (e: any) {
      flashStore.setFailed(
        e?.data ?? { code: 'ERROR', message: 'Failed to load job logs', status_code: 500, timestamp: '' },
      )
    } finally {
      loading.value = false
    }
  }

  const getDefinitions = async () => {
    try {
      const response = await $api<ResponseSingleData<JobDefinition[]>>('api/management/job-logs/definitions', {
        method: 'GET',
      })
      definitions.value = response.payload.data
    } catch {
      // Non-critical for the page (only feeds the "registered jobs" panel and
      // the name filter's options) — the run history table above still works.
    }
  }

  // range: '24h' | '7d' | '30d'. Non-critical for the page, same reason as
  // getDefinitions — the run history table doesn't depend on it.
  const getStats = async (range: string = '24h') => {
    try {
      const response = await $api<ResponseSingleData<JobLogStats>>('api/management/job-logs/stats', {
        method: 'GET',
        query: { range },
      })
      stats.value = response.payload.data
    } catch {
      // ignore — see getDefinitions' comment
    }
  }

  return {
    logs,
    meta,
    definitions,
    stats,
    loading,
    getAll,
    getDefinitions,
    getStats,
  }
}
