import type { RuntimeConfig } from 'nuxt/schema'
import type { FetchResponse } from 'ofetch'

// $fetch is Nitro-typed, not ofetch-typed — its options narrow `method` to a union
type ApiOptions = NonNullable<Parameters<typeof $fetch>[1]> & {
  // Set by a caller that fires several independent, partial-permission calls
  // at once (e.g. dashboard widgets) and wants to handle a missing
  // permission itself instead of the whole page showing the 403 screen.
  silent403?: boolean
}
import type { ResponseError } from '~/types/response'
import { useFlashStore } from '~/stores/flash'

interface RefreshTokenResponse {
  payload: { data: { tokens: { access_token: string } } }
}

// Module-level — survives across calls within the same page session
let isRefreshing = false
let pendingRequests: Array<(token: string | null) => void> = []

const flushPending = (token: string | null) => {
  pendingRequests.forEach(cb => cb(token))
  pendingRequests = []
}

export default defineNuxtPlugin(() => {
  const config: RuntimeConfig = useRuntimeConfig()

  // Tells the backend to use httpOnly verifier/refresh cookies rather than
  // returning tokens bare (that path is for mobile, which has no cookie jar).
  const CLIENT_TYPE_HEADER = { 'X-Client-Type': 'web' }

  // FormData bodies (file uploads) must NOT get a Content-Type here — the
  // browser sets multipart/form-data with the correct boundary itself only
  // when the header is left unset.
  const buildHeaders = (token: string | null, isFormData: boolean, extra?: HeadersInit): HeadersInit => ({
    ...(isFormData ? {} : { 'Content-Type': 'application/json' }),
    Accept: 'application/json',
    Authorization: token ? `Bearer ${token}` : '',
    ...CLIENT_TYPE_HEADER,
    ...(extra ?? {}),
  })

  // Calls the refresh endpoint directly with $fetch (bypasses the wrapper to avoid loops)
  const tryRefresh = async (): Promise<string | null> => {
    try {
      const res = await $fetch<RefreshTokenResponse>('api/auth/refresh', {
        baseURL: config.public.apiBase,
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json', ...CLIENT_TYPE_HEADER },
      })
      return res.payload?.data?.tokens?.access_token ?? null
    } catch {
      return null
    }
  }

  // ERR_ACTION_UNAUTHORIZED (403) means the caller is authenticated but
  // lacks permission. A GET means a whole page's data is unreachable — show
  // the 403 error screen instead of leaving a broken/empty page behind. A
  // mutating action (POST/PATCH/DELETE) means they tried something from a
  // page they can otherwise use, so that stays a flash — not a reason to
  // navigate away. Only a 401 (handled separately below) means the token
  // itself is no good and ends the session.
  const handleNon401Error = (response: FetchResponse<ResponseError>, method?: string, silent403?: boolean) => {
    if (response.status === 403) {
      // Caller asked to handle a missing permission itself — no redirect, no flash.
      if (silent403) return
      if ((method ?? 'GET').toUpperCase() === 'GET') {
        showError({ statusCode: 403 })
        return
      }
    }
    const flashStore = useFlashStore()
    flashStore.clearMessages()
    if (response._data) flashStore.setFailed(response._data)
  }

  // Persist the refreshed token everywhere the old one lived — the cookie too,
  // since the route middleware and logout read it instead of localStorage.
  const applyNewToken = (newToken: string) => {
    const authStore = useAuthStore()
    localStorage.setItem('access_token', newToken)
    authStore.accessToken = newToken
    if (authStore.user) {
      authStore.user.access_token = newToken
      localStorage.setItem('auth_user', JSON.stringify(authStore.user))
    }
  }

  const forceLogout = async () => {
    await useAuthStore().setUnauthenticatedUser()
    navigateTo('/auth')
  }

  // Replays a request with the refreshed token. A second 401 means the fresh
  // token is not accepted either, so the session is genuinely dead — log out.
  const retryWithToken = async <T>(url: string, opts: ApiOptions): Promise<T> => {
    const { silent403, ...fetchOpts } = opts
    try {
      return await $fetch<T>(url, fetchOpts)
    } catch (retryError: any) {
      const retryResponse = retryError?.response as FetchResponse<ResponseError> | undefined

      if (retryResponse?.status === 401) {
        await forceLogout()
        throw retryResponse
      }

      if (retryResponse) handleNon401Error(retryResponse, opts.method as string | undefined, silent403)
      throw retryResponse ?? retryError
    }
  }

  const api = async <T = any>(url: string, options: ApiOptions = {}): Promise<T> => {
    const { silent403, ...callOptions } = options
    const token = import.meta.client ? localStorage.getItem('access_token') : null

    const isFormData = callOptions.body instanceof FormData

    const buildOpts = (withToken: string | null): ApiOptions => ({
      ...callOptions,
      silent403,
      baseURL: config.public.apiBase,
      credentials: 'include',
      headers: buildHeaders(withToken, isFormData, callOptions.headers as HeadersInit),
    })

    let response: FetchResponse<ResponseError> | undefined

    try {
      return await $fetch<T>(url, buildOpts(token))
    } catch (error: any) {
      response = error?.response as FetchResponse<ResponseError> | undefined
      const status = response?.status

      if (status !== 401 || url.includes('api/auth/refresh')) {
        if (response) handleNon401Error(response, callOptions.method as string | undefined, silent403)
        throw response ?? error
      }

      // ── 401 handling ──────────────────────────────────────────────────────

      // Another refresh is already in flight — queue this request
      if (isRefreshing) {
        return new Promise<T>((resolve, reject) => {
          pendingRequests.push((newToken) => {
            if (!newToken) return reject(response ?? error)
            retryWithToken<T>(url, buildOpts(newToken)).then(resolve, reject)
          })
        })
      }

      isRefreshing = true

      let newToken: string | null
      try {
        newToken = await tryRefresh()
      } finally {
        // Cleared before the replay below, otherwise a request that 401s while
        // the replay is still running would queue onto an already-flushed list.
        isRefreshing = false
      }

      if (!newToken) {
        flushPending(null)
        await forceLogout()
        throw response ?? error
      }

      applyNewToken(newToken)
      flushPending(newToken)

      return await retryWithToken<T>(url, buildOpts(newToken))
    }
  }

  return {
    provide: { api },
  }
})