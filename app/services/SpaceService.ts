import { useFlashStore } from '~/stores/flash'
import type Space from '~/types/entities/space'
import type {
  ResponseData,
  ResponseSingleData,
  ServerPaginationMeta,
} from '~/types/response'

interface SpaceListResponse extends ResponseData {
  payload: {
    data: Space[]
    meta: ServerPaginationMeta
  }
}

export interface SpaceRequest {
  name: string
  description: string | null
  icon: string | null
  is_archived?: boolean
}

export interface SpaceFilters {
  search?: string
  include_archived?: string
}

const buildQuery = (page: number, perPage: number, filters: SpaceFilters) => ({
  page,
  per_page: perPage,
  ...(filters.search ? { search: filters.search } : {}),
  ...(filters.include_archived ? { include_archived: filters.include_archived } : {}),
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

export const useSpaceService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const spaces = ref<Space[]>([])
  const space = ref<Space | null>(null)
  const meta = ref<ServerPaginationMeta>({ ...defaultMeta })
  const loading = ref(false)
  const submitting = ref(false)

  const getAll = async (page: number, perPage: number, filters: SpaceFilters = {}) => {
    loading.value = true
    try {
      const response = await $api<SpaceListResponse>('api/spaces', {
        method: 'GET',
        query: buildQuery(page, perPage, filters),
      })
      spaces.value = response.payload.data
      meta.value = response.payload.meta
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load spaces'))
    } finally {
      loading.value = false
    }
  }

  const getById = async (id: string) => {
    loading.value = true
    try {
      const response = await $api<ResponseSingleData<Space>>(`api/spaces/${id}`, { method: 'GET' })
      space.value = response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load space'))
    } finally {
      loading.value = false
    }
  }

  const addNew = async (request: SpaceRequest): Promise<Space | null> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<Space>>('api/spaces', { method: 'POST', body: request })
      flashStore.setSuccess(response)
      return response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to create space'))
      return null
    } finally {
      submitting.value = false
    }
  }

  const updateById = async (id: string, request: SpaceRequest): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<Space>>(`api/spaces/${id}`, { method: 'PUT', body: request })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to update space'))
      return false
    } finally {
      submitting.value = false
    }
  }

  // DELETE archives rather than removes — a space owns documents. The method
  // is named for what it does, not for the verb it sends.
  const archiveById = async (id: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseData>(`api/spaces/${id}`, { method: 'DELETE' })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to archive space'))
      return false
    } finally {
      submitting.value = false
    }
  }

  // Archiving has its own verb (DELETE) but un-archiving does not — it is an
  // ordinary update. Named here so the caller says what it means rather than
  // spelling out a whole space payload with one flag flipped.
  const unarchiveById = async (space: Space): Promise<boolean> =>
    updateById(space.id, {
      name: space.name,
      description: space.description,
      icon: space.icon,
      is_archived: false,
    })

  return {
    spaces, space, meta, loading, submitting,
    getAll, getById, addNew, updateById, archiveById, unarchiveById,
  }
}
