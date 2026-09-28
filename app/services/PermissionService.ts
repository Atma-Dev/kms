import type { ResponseDataCollectionWithoutPagination } from '~/types/response'
import type Permission from '~/types/entities/permission'

export const usePermissionService = () => {
  const { $api } = useNuxtApp()
  const permissionsCollection = ref<Permission[]>([])
  const loading = ref(false)

  const getAll = async () => {
    loading.value = true
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<Permission>>(
        'api/management/permissions',
        { method: 'GET' },
      )
      permissionsCollection.value = response.payload.data
    } finally {
      loading.value = false
    }
  }

  return {
    permissionsCollection,
    loading,
    getAll,
  }
}
