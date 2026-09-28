import { fetchWsToken } from '~/composables/useWsToken'

/**
 * One token per session, not per component.
 *
 * A document with an attachment list and four inline images has six callers of
 * this; without sharing, opening it would ask the API for six tokens that are
 * all equally valid. The promise is module-level so the second caller awaits
 * the first one's request instead of starting another.
 */
let tokenRequest: Promise<string | null> | null = null

const requestToken = (): Promise<string | null> => {
  if (!import.meta.client) return Promise.resolve(null)
  // Cleared on failure so a token fetch that failed once (offline, expired
  // session) is retried by the next caller rather than cached as "no token".
  tokenRequest ??= fetchWsToken().then((token) => {
    if (!token) tokenRequest = null
    return token
  })
  return tokenRequest
}

/**
 * Turns the bare `/api/attachments/{id}/download` path the API stores into a
 * URL the browser can actually fetch.
 *
 * Two things have to be added. The API lives on its own origin, so the path
 * needs the base in front of it; and a browser-issued request — an `<img src>`
 * or an `<a download>` — carries no Authorization header, so the token rides
 * in the query instead (see TokenOrHeaderAuthMiddleware on the API).
 *
 * The token is the short-lived WS-scoped one, never the access token: a URL
 * leaks easily (a referer, a pasted link, shell history) and this one stops
 * working in minutes rather than granting full API access. It is never written
 * into a document's content — only into the `src` of a rendered tag.
 */
export const useAttachmentUrl = () => {
  const config = useRuntimeConfig()
  const token = ref<string | null>(null)

  const ready = requestToken().then((value) => {
    token.value = value
    return value
  })

  const resolve = (path: string): string => {
    if (!path) return path
    const base = String(config.public.apiBase ?? '').replace(/\/$/, '')
    const absolute = path.startsWith('http') ? path : base + path
    if (!token.value) return absolute
    return absolute + (absolute.includes('?') ? '&' : '?') + 'token=' + encodeURIComponent(token.value)
  }

  return { token, ready, resolve }
}
