import type { ResponseSingleData } from '~/types/response'
import { Permission } from '~/enums/Permission'

/** One select-option: id and name, nothing else. */
export interface LookupOption {
  id: string
  name: string
}

// Every dropdown-backing list in the app. Each answers to its own
// lookups.* permission, deliberately separate from the *.show permission of
// the underlying resource — a page should not need the right to manage users
// just to put their names in a filter.
export const LOOKUP_PERMISSION = {
  users: Permission.LOOKUPS_USERS_SHOW,
  roles: Permission.LOOKUPS_ROLES_SHOW,
  permissions: Permission.LOOKUPS_PERMISSIONS_SHOW,
  spaces: Permission.LOOKUPS_SPACES_SHOW,
  categories: Permission.LOOKUPS_CATEGORIES_SHOW,
  tags: Permission.LOOKUPS_TAGS_SHOW,
} as const

export type LookupResource = keyof typeof LOOKUP_PERMISSION

/**
 * Loads one dropdown's options.
 *
 * `query` carries the scope a lookup needs — `categories` is per space, and
 * without a `space_id` it would offer another space's categories to file a
 * document under.
 *
 * `can` reports whether the user may use this dropdown at all. A caller must
 * hide the control when it is false — the matching filter parameter is
 * refused with a 403 by the API, not ignored, so sending it anyway would fail
 * the whole request.
 */
export const useLookupService = (resource: LookupResource, query?: Record<string, string>) => {
  const { $api } = useNuxtApp()
  const { hasPermission } = useAuthStore()

  const options = ref<LookupOption[]>([])
  const loading = ref(false)
  const can = hasPermission(LOOKUP_PERMISSION[resource])

  const load = async () => {
    // Skipped rather than left to 403: the page around this dropdown is
    // usually fine, and the api plugin would redirect all of it to /403.
    if (!can || loading.value) return
    loading.value = true
    try {
      const response = await $api<ResponseSingleData<LookupOption[]>>(`api/lookups/${resource}`, { query })
      options.value = response.payload.data
    } finally {
      loading.value = false
    }
  }

  return { options, loading, can, load }
}
