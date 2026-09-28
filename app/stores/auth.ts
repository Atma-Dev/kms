import type User from "~/types/entities/user";
import type { ResponseSingleData } from "~/types/response";

const USER_STORAGE_KEY = 'auth_user'

export const useAuthStore = defineStore("authStore", () => {
    const storedUser = import.meta.client ? localStorage.getItem(USER_STORAGE_KEY) : null
    const user = ref<User | null>(storedUser ? JSON.parse(storedUser) : null)

    const accessToken = useCookie('access_token', {
        maxAge: 60 * 60 * 24 * 7,
        sameSite: 'lax',
        secure: false,
        watch: true
    })

    // Falls back to first + last name so the UI still has something to show
    // when the backend omits `full_name`.
    const fullName = computed(() => {
        if (!user.value) return ''
        if (user.value.full_name) return user.value.full_name
        return [user.value.first_name, user.value.last_name].filter(Boolean).join(' ')
    })

    const roleNames = computed(() => (user.value?.roles ?? []).map(role => role.name.toLowerCase()))

    // Case-insensitive — backend role names are mixed case (e.g. "Super-Admin").
    const hasRole = (...names: string[]) =>
        names.some(name => roleNames.value.includes(name.toLowerCase()))

    const permissionNames = computed(() => user.value?.permissions ?? [])

    const hasPermission = (...names: string[]) =>
        names.some(name => permissionNames.value.includes(name))

    const setAuthenticatedUser = async (responseUser: User): Promise<void> => {
        user.value = responseUser
        accessToken.value = responseUser.access_token
        localStorage.setItem('access_token', responseUser.access_token ?? '')
        localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(responseUser))
    }

    // Re-reads the user from the API so data cached in localStorage cannot go
    // stale (an admin changed their role). Keeps the token that is
    // already in play — `api/auth/me` does not return one.
    const fetchMe = async (): Promise<User | null> => {
        const { $api } = useNuxtApp()

        try {
            const response = await $api<ResponseSingleData<User>>('api/profiles/me')
            const freshUser: User = {
                ...response.payload.data,
                access_token: user.value?.access_token ?? accessToken.value ?? undefined,
            }

            user.value = freshUser
            localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(freshUser))

            return freshUser
        } catch {
            // A dead session is already handled by the api plugin (flash + logout)
            return null
        }
    }

    const setUnauthenticatedUser = async () => {
        accessToken.value = null
        localStorage.removeItem('access_token')
        localStorage.removeItem(USER_STORAGE_KEY)
        user.value = null
    }

    return {
      user,
      accessToken,
      fullName,
      hasRole,
      hasPermission,
      fetchMe,
      setAuthenticatedUser,
      setUnauthenticatedUser,
    }
})
