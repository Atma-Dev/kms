import type { ResponseData, ResponseDataCollectionWithoutPagination } from '~/types/response'
import { useFlashStore } from '~/stores/flash'

export interface CacheEntry {
  key: string
  count: number
  expires_in_seconds: number
}

export const useCacheService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const entries = ref<CacheEntry[]>([])
  const loading = ref(false)
  // Holds the key being cleared so only that row shows a busy state; '' is the
  // clear-everything button.
  const clearingKey = ref<string | null>(null)

  const getAll = async () => {
    loading.value = true
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<CacheEntry>>(
        'api/management/cache',
        { method: 'GET' },
      )
      entries.value = response.payload.data
    } finally {
      loading.value = false
    }
  }

  // No key clears everything — the backend treats the key as a prefix, so an
  // empty one matches all of them.
  const clear = async (key = ''): Promise<boolean> => {
    clearingKey.value = key
    try {
      const response = await $api<ResponseData>('api/management/cache', {
        method: 'DELETE',
        query: key ? { key } : undefined,
      })
      flashStore.setSuccess(response)
      return true
    } catch {
      // The api plugin already flashed the backend message.
      return false
    } finally {
      clearingKey.value = null
    }
  }

  return {
    entries,
    loading,
    clearingKey,
    getAll,
    clear,
  }
}
