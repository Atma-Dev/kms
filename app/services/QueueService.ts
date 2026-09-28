import { useFlashStore } from '~/stores/flash'
import type QueueJob from '~/types/entities/queue_job'
import type { QueueStats } from '~/types/entities/queue_job'
import type { ResponseData, ResponseSingleData, ServerPaginationMeta } from '~/types/response'

interface QueueJobListResponse extends ResponseData {
  payload: {
    data: QueueJob[]
    meta: ServerPaginationMeta
  }
}

export interface QueueJobFilters {
  queue?: string
  name?: string
  status?: string
}

const buildQuery = (page: number, perPage: number, filters: QueueJobFilters) => ({
  page,
  per_page: perPage,
  ...(filters.queue ? { queue: filters.queue } : {}),
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

const failure = (message: string) => ({ code: 'ERROR', message, status_code: 500, timestamp: '' })

export const useQueueService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const jobs = ref<QueueJob[]>([])
  const meta = ref<ServerPaginationMeta>({ ...defaultMeta })
  const stats = ref<QueueStats | null>(null)
  const loading = ref(false)
  const submitting = ref(false)

  const getAll = async (page: number = 1, perPage: number = 10, filters: QueueJobFilters = {}) => {
    loading.value = true
    try {
      const response = await $api<QueueJobListResponse>('api/management/queue/jobs', {
        method: 'GET',
        query: buildQuery(page, perPage, filters),
      })
      jobs.value = response.payload.data
      meta.value = response.payload.meta
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load queue jobs'))
    } finally {
      loading.value = false
    }
  }

  const getStats = async () => {
    try {
      const response = await $api<ResponseSingleData<QueueStats>>('api/management/queue/stats', { method: 'GET' })
      stats.value = response.payload.data
    } catch {
      // non-critical — the table above still works without it
    }
  }

  const retry = async (id: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<QueueJob>>(`api/management/queue/jobs/${id}/retry`, { method: 'POST' })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to retry queue job'))
      return false
    } finally {
      submitting.value = false
    }
  }

  const cancel = async (id: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<QueueJob>>(`api/management/queue/jobs/${id}/cancel`, { method: 'POST' })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to cancel queue job'))
      return false
    } finally {
      submitting.value = false
    }
  }

  const retryFailed = async (name?: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<{ retried: number }>>('api/management/queue/jobs/retry-failed', {
        method: 'POST',
        body: name ? { name } : {},
      })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to retry failed queue jobs'))
      return false
    } finally {
      submitting.value = false
    }
  }

  const purge = async (statuses: string[]): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<{ purged: number }>>('api/management/queue/jobs', {
        method: 'DELETE',
        body: { statuses },
      })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to purge queue jobs'))
      return false
    } finally {
      submitting.value = false
    }
  }

  return { jobs, meta, stats, loading, submitting, getAll, getStats, retry, cancel, retryFailed, purge }
}
