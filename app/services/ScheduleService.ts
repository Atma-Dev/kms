import { useFlashStore } from '~/stores/flash'
import type ScheduledJob from '~/types/entities/scheduled_job'
import type { ResponseDataCollectionWithoutPagination, ResponseSingleData } from '~/types/response'

export interface ScheduleUpdateRequest {
  cron_expression: string
  enabled: boolean
}

const failure = (message: string) => ({ code: 'ERROR', message, status_code: 500, timestamp: '' })

// Unpaginated — one row per registered background job, the same "small list
// the app just has" shape as the master pages, not a per-page query.
export const useScheduleService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const schedules = ref<ScheduledJob[]>([])
  const loading = ref(false)
  const submitting = ref(false)

  const getAll = async () => {
    loading.value = true
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<ScheduledJob>>('api/management/schedules', {
        method: 'GET',
      })
      schedules.value = response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load scheduled jobs'))
    } finally {
      loading.value = false
    }
  }

  const updateById = async (id: string, request: ScheduleUpdateRequest): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<ScheduledJob>>(`api/management/schedules/${id}`, {
        method: 'PATCH',
        body: request,
      })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to update scheduled job'))
      return false
    } finally {
      submitting.value = false
    }
  }

  const runNow = async (id: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<ScheduledJob>>(`api/management/schedules/${id}/run`, { method: 'POST' })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to run scheduled job'))
      return false
    } finally {
      submitting.value = false
    }
  }

  return { schedules, loading, submitting, getAll, updateById, runNow }
}
