import { useFlashStore } from '~/stores/flash'
import type User from '~/types/entities/user'
import type { ResponseDataCollectionWithoutPagination, ResponseSingleData } from '~/types/response'

/**
 * PATCH api/management/users/:id accepts `role_ids` and `marketing_id`.
 * `role_ids` replaces the user's roles wholesale — sending `[]` clears them;
 * omitting the key would leave them untouched (the backend treats nil as
 * "no change"). `marketing_id` is sent as an empty string to clear it.
 */
export interface UserRequest {
  role_ids: string[]
  marketing_id: string
}

export const useUserService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const users = ref<User[]>([])
  const loading = ref(false)
  const submitting = ref(false)

  // The whole list in one call — small enough that /account/users paginates and
  // filters it in the browser, like the roles and master lists.
  const getAll = async () => {
    loading.value = true
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<User>>(
        'api/management/users',
        { method: 'GET' },
      )
      users.value = response.payload.data
    } catch (e: any) {
      flashStore.setFailed(
        e?.data ?? {
          code: 'ERROR',
          message: 'Failed to load users',
          status_code: 500,
          timestamp: '',
        },
      )
    } finally {
      loading.value = false
    }
  }

  const updateById = async (id: string, request: UserRequest): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<User>>(`api/management/users/${id}`, {
        method: 'PATCH',
        body: request,
      })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(
        e?.data ?? {
          code: 'ERROR',
          message: 'Failed to update user',
          status_code: 500,
          timestamp: '',
        },
      )
      return false
    } finally {
      submitting.value = false
    }
  }

  return {
    users,
    loading,
    submitting,
    getAll,
    updateById,
  }
}
