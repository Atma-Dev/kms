import type { ResponseData, ResponseError, ResponseSingleData } from '~/types/response'
import type User from '~/types/entities/user'
import { useAuthStore } from '~/stores/auth'
import { isFetchResponseError } from '~/utils/helper'
import { ResponseCode } from '~/enums/ResponseCode'

export interface AuthRequest {
  email: string
  password: string
}

export const useAuthService = () => {
  const { $api } = useNuxtApp()
  const responseError = ref<ResponseError | null>(null)

  const { setAuthenticatedUser, setUnauthenticatedUser } = useAuthStore()

  /**
   * Email + password login. The API answers with an access token only (the
   * refresh token stays in its httpOnly cookie), so the session is completed
   * the same way an OAuth login is — see finishLogin.
   *
   * Throws on a bad credential so the caller can render the message inline;
   * the api plugin does not flash 401s from this route.
   */
  const login = async (request: AuthRequest): Promise<User> => {
    const response = await $api<ResponseSingleData<{ access_token: string }>>('api/auth/login', {
      method: 'POST',
      body: request,
    })
    return finishLogin(response.payload.data.access_token)
  }

  // Shared by every login path: the OAuth callback page (top-level redirect),
  // the Google popup flow, and password login — all three end up with just an
  // access_token to exchange for the profile that actually completes the
  // session.
  const finishLogin = async (accessToken: string): Promise<User> => {
    localStorage.setItem('access_token', accessToken)
    const response = await $api<ResponseSingleData<User>>('api/profiles/me')
    await setAuthenticatedUser({ ...response.payload.data, access_token: accessToken })
    return response.payload.data
  }

  const logout = async () => {
    try {
      const accessToken = useCookie('access_token').value
      if (accessToken) {
        await $api<ResponseData>('/api/auth/logout', {
          method: 'POST',
        })
        await setUnauthenticatedUser()
        navigateTo('/auth')
        return
      } else {
        await setUnauthenticatedUser()
        navigateTo('/auth')
        return
      }
    } catch (e) {
      if (isFetchResponseError(e)) {
        const error = e._data as ResponseError
        if (error.code === ResponseCode.ERR_ACTION_UNAUTHORIZED) {
          await setUnauthenticatedUser()
          navigateTo('/auth')
        }
      }
    }
  }

  return {
    login,
    finishLogin,
    logout,
    responseError,
  }
}
