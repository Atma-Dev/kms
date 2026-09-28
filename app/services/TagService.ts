import { useFlashStore } from '~/stores/flash'
import type Tag from '~/types/entities/tag'
import type {
  ResponseData,
  ResponseDataCollectionWithoutPagination,
  ResponseSingleData,
} from '~/types/response'

const failure = (message: string) => ({ code: 'ERROR', message, status_code: 500, timestamp: '' })

export const useTagService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const tags = ref<Tag[]>([])
  const loading = ref(false)
  const submitting = ref(false)

  const getAll = async (search = '') => {
    loading.value = true
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<Tag>>('api/tags', {
        method: 'GET',
        query: search ? { search } : undefined,
      })
      tags.value = response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load tags'))
    } finally {
      loading.value = false
    }
  }

  const addNew = async (name: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<Tag>>('api/tags', { method: 'POST', body: { name } })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to create tag'))
      return false
    } finally {
      submitting.value = false
    }
  }

  // Renaming relabels every document already carrying the tag. A rename onto
  // a name that exists is refused by the API rather than merged — the flash
  // message says so, so this just surfaces it.
  const renameById = async (id: string, name: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<Tag>>(`api/tags/${id}`, {
        method: 'PUT',
        body: { name },
      })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to rename tag'))
      return false
    } finally {
      submitting.value = false
    }
  }

  // Deleting a tag removes the label from every document that carried it,
  // which is why the master list shows the count first.
  const deleteById = async (id: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseData>(`api/tags/${id}`, { method: 'DELETE' })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to delete tag'))
      return false
    } finally {
      submitting.value = false
    }
  }

  return { tags, loading, submitting, getAll, addNew, renameById, deleteById }
}
