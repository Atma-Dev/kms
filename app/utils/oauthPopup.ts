// Shared with app/pages/auth/callback.vue: when that page detects it was
// opened by openOAuthPopup below (not a top-level navigation), it posts the
// OAuth result back as this message instead of finishing the login itself.
export const OAUTH_CALLBACK_MESSAGE_TYPE = 'oauth-callback'

export type OAuthPopupResult =
  | { status: 'blocked' }
  | { status: 'cancelled' }
  | { status: 'done'; access_token?: string; error?: string }

const POPUP_WIDTH = 480
const POPUP_HEIGHT = 640

/**
 * Opens `url` in a centered popup and resolves once /auth/callback (loaded
 * inside it) posts the OAuth result back, or the popup is blocked/closed.
 * 'cancelled' (popup closed without finishing) is kept distinct from
 * 'blocked' so the caller only falls back to a full-page redirect when
 * there was never a popup to begin with — not when the user chose to close it.
 */
export const openOAuthPopup = (url: string): Promise<OAuthPopupResult> => {
  const left = window.screenX + (window.outerWidth - POPUP_WIDTH) / 2
  const top = window.screenY + (window.outerHeight - POPUP_HEIGHT) / 2
  const popup = window.open(
    url,
    'oauth-login',
    `width=${POPUP_WIDTH},height=${POPUP_HEIGHT},left=${left},top=${top}`,
  )

  if (!popup) return Promise.resolve({ status: 'blocked' })

  return new Promise((resolve) => {
    const cleanup = () => {
      window.removeEventListener('message', onMessage)
      window.clearInterval(closedCheck)
    }

    const onMessage = (event: MessageEvent) => {
      if (event.origin !== window.location.origin) return
      if (event.data?.type !== OAUTH_CALLBACK_MESSAGE_TYPE) return
      cleanup()
      resolve({ status: 'done', access_token: event.data.access_token, error: event.data.error })
    }

    const closedCheck = window.setInterval(() => {
      if (popup.closed) {
        cleanup()
        resolve({ status: 'cancelled' })
      }
    }, 500)

    window.addEventListener('message', onMessage)
  })
}
