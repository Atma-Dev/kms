import { useFlashStore } from '~/stores/flash'
import type M2MClient from '~/types/entities/m2m_client'
import type {
  ResponseData,
  ResponseDataCollectionWithoutPagination,
  ResponseSingleData,
} from '~/types/response'

export interface M2MClientRequest {
  client: string
  scope: string[]
}

// Store's response only: the plaintext secret is generated once server-side
// and never recoverable after this — see M2MClientResource on the API.
export interface M2MClientCreated extends M2MClient {
  secret: string
}

const failure = (message: string) => ({ code: 'ERROR', message, status_code: 500, timestamp: '' })

export const useM2MClientService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const clients = ref<M2MClient[]>([])
  const loading = ref(false)
  const submitting = ref(false)

  const getAll = async () => {
    loading.value = true
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<M2MClient>>(
        'api/management/m2m-clients',
        { method: 'GET' },
      )
      clients.value = response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load m2m clients'))
    } finally {
      loading.value = false
    }
  }

  const addNew = async (request: M2MClientRequest): Promise<M2MClientCreated | null> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<M2MClientCreated>>('api/management/m2m-clients', {
        method: 'POST',
        body: request,
      })
      flashStore.setSuccess(response)
      return response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to create m2m client'))
      return null
    } finally {
      submitting.value = false
    }
  }

  const updateById = async (id: string, request: M2MClientRequest): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<M2MClient>>(`api/management/m2m-clients/${id}`, {
        method: 'PATCH',
        body: request,
      })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to update m2m client'))
      return false
    } finally {
      submitting.value = false
    }
  }

  const regenerateSecret = async (id: string): Promise<M2MClientCreated | null> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<M2MClientCreated>>(
        `api/management/m2m-clients/${id}/regenerate-secret`,
        { method: 'POST' },
      )
      flashStore.setSuccess(response)
      return response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to regenerate m2m client secret'))
      return null
    } finally {
      submitting.value = false
    }
  }

  const revokeById = async (id: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseData>(`api/management/m2m-clients/${id}`, { method: 'DELETE' })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to revoke m2m client'))
      return false
    } finally {
      submitting.value = false
    }
  }

  return { clients, loading, submitting, getAll, addNew, updateById, regenerateSecret, revokeById }
}
