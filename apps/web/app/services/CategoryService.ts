import { useFlashStore } from '~/stores/flash'
import type Category from '~/types/entities/category'
import type { CategoryTree } from '~/types/entities/category'
import type {
  ResponseData,
  ResponseSingleData,
} from '~/types/response'

export interface CategoryRequest {
  name: string
  parent_id: string | null
}

const failure = (message: string) => ({ code: 'ERROR', message, status_code: 500, timestamp: '' })

export const useCategoryService = (spaceId: MaybeRefOrGetter<string>) => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const tree = ref<Category[]>([])
  // Totals for the two rows that have no node of their own.
  const totalDocumentCount = ref(0)
  const uncategorisedCount = ref(0)
  const loading = ref(false)
  const submitting = ref(false)

  const base = () => `api/spaces/${toValue(spaceId)}/categories`

  const getTree = async () => {
    loading.value = true
    try {
      const response = await $api<ResponseSingleData<CategoryTree>>(base(), { method: 'GET' })
      tree.value = response.payload.data.categories
      totalDocumentCount.value = response.payload.data.total_document_count
      uncategorisedCount.value = response.payload.data.uncategorised_count
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load categories'))
    } finally {
      loading.value = false
    }
  }

  const addNew = async (request: CategoryRequest): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<Category>>(base(), { method: 'POST', body: request })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to create category'))
      return false
    } finally {
      submitting.value = false
    }
  }

  const updateById = async (id: string, request: CategoryRequest): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<Category>>(`api/categories/${id}`, {
        method: 'PUT',
        body: request,
      })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to update category'))
      return false
    } finally {
      submitting.value = false
    }
  }

  const deleteById = async (id: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseData>(`api/categories/${id}`, { method: 'DELETE' })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to delete category'))
      return false
    } finally {
      submitting.value = false
    }
  }

  /**
   * Sends the whole ordered sibling list, which is what the API takes — it
   * renumbers 0..n-1 in one transaction. The response is the whole tree, so
   * `tree` is replaced from it rather than patched: positions shifted across
   * the level, and reproducing that here would be a second implementation of
   * the ordering rules.
   */
  const reorder = async (parentId: string | null, ids: string[]): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<CategoryTree>>(`${base()}/order`, {
        method: 'PATCH',
        body: { parent_id: parentId, ids },
      })
      tree.value = response.payload.data.categories
      totalDocumentCount.value = response.payload.data.total_document_count
      uncategorisedCount.value = response.payload.data.uncategorised_count
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to reorder categories'))
      return false
    } finally {
      submitting.value = false
    }
  }

  return {
    tree, totalDocumentCount, uncategorisedCount,
    loading, submitting, getTree, addNew, updateById, deleteById, reorder,
  }
}
