import type { ResponseSingleData } from '~/types/response'

const STORAGE_KEY = 'access_token_ws'

// Fetches a token scoped to WebSocket routes only (see POST /auth/ws-token on
// the API) and stores it under its own key — never reuse the regular
// access_token here, that is exactly what this exists to avoid depending on.
// Called fresh before every (re)connect: the token is short-lived and this
// keeps a stale one from being handed to a reconnect attempt.
export const fetchWsToken = async (): Promise<string | null> => {
  if (!import.meta.client) return null

  const { $api } = useNuxtApp()
  try {
    const response = await $api<ResponseSingleData<{ access_token: string }>>('api/auth/ws-token', {
      method: 'POST',
    })
    const token = response.payload.data.access_token
    localStorage.setItem(STORAGE_KEY, token)
    return token
  } catch {
    return null
  }
}
