import type {
  ResponseData,
  ResponseDataCollectionWithoutPagination,
  ResponseSingleData,
} from '~/types/response'
import type Role from '~/types/entities/role'
import { useFlashStore } from '~/stores/flash'

export interface RoleRequest {
  name: string
  permission_ids: string[]
}

export const useRoleService = () => {
  const { $api } = useNuxtApp()
  const rolesCollection = ref<Role[]>([])
  const loading = ref(false)
  const submitting = ref(false)
  const flashStore = useFlashStore()

  const getAll = async () => {
    loading.value = true
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<Role>>(
        'api/management/roles',
        { method: 'GET' },
      )
      rolesCollection.value = response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? { code: 'ERROR', message: 'Failed to load roles', status_code: 500, timestamp: '' })
    } finally {
      loading.value = false
    }
  }

  const addNew = async (request: RoleRequest): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<Role>>('api/management/roles', {
        method: 'POST',
        body: request,
      })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? { code: 'ERROR', message: 'Failed to create role', status_code: 500, timestamp: '' })
      return false
    } finally {
      submitting.value = false
    }
  }

  const updateById = async (id: string, request: RoleRequest): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<Role>>(`api/management/roles/${id}`, {
        method: 'PATCH',
        body: request,
      })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? { code: 'ERROR', message: 'Failed to update role', status_code: 500, timestamp: '' })
      return false
    } finally {
      submitting.value = false
    }
  }

  const deleteById = async (id: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseData>(`api/management/roles/${id}`, {
        method: 'DELETE',
      })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? { code: 'ERROR', message: 'Failed to delete role', status_code: 500, timestamp: '' })
      return false
    } finally {
      submitting.value = false
    }
  }

  return {
    rolesCollection,
    loading,
    submitting,
    getAll,
    updateById,
    deleteById,
    addNew,
  }
}