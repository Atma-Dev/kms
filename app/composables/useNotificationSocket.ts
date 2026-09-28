import { toaster } from '~/base/ui/toast'
import { useNotificationStore } from '~/stores/notification'
import type AppNotification from '~/types/entities/notification'

const RECONNECT_DELAY_MS = 3000

// Where clicking the toast takes the user — the invoice list for invoice.*,
// otherwise the feed. Everything the socket delivers is informational today,
// so every toast is type 'info'.
const pageForNotification = (type: string) =>
  type.startsWith('invoice.') ? '/invoices' : '/notifications'

// Opens a live connection to the notification feed. Called once from
// app/layouts/default.vue, which only mounts for authenticated pages.
export const useNotificationSocket = () => {
  if (!import.meta.client) return () => {}

  const config = useRuntimeConfig()
  const store = useNotificationStore()
  let socket: WebSocket | null = null
  let closedByUs = false

  const connect = async () => {
    if (!localStorage.getItem('access_token')) return

    // A dedicated token scoped to WS routes only — see useWsToken. Fetched
    // fresh on every (re)connect rather than the regular access_token: that
    // one depends on the access_token_verifier cookie crossing to the API's
    // subdomain during the handshake, which browsers do not reliably do.
    const token = await fetchWsToken()
    if (!token) return

    // http -> ws, https -> wss. A browser WebSocket handshake can't carry a
    // custom Authorization header, so the token travels as a query param.
    const wsBase = config.public.apiBase.replace(/^http/, 'ws')
    socket = new WebSocket(`${wsBase}/api/notifications/ws?token=${encodeURIComponent(token)}`)

    socket.onmessage = (event) => {
      try {
        const frame = JSON.parse(event.data)

        // Control frames carry an "event" key; notifications never do. The
        // backend sends this when a role the user holds changes, because the
        // permission list lives in localStorage and would otherwise keep
        // drawing the old menu until a full page load.
        if (frame?.event === 'permissions.changed') {
          useAuthStore().fetchMe()
          return
        }

        const notification = frame as AppNotification
        store.receive(notification)
        toaster.create({
          title: notification.title,
          description: notification.body,
          type: 'info',
          meta: { href: pageForNotification(notification.type) },
        })
      } catch {
        // ignore malformed frames
      }
    }

    // ponytail: fixed-delay reconnect, no exponential backoff/jitter —
    // upgrade if this ever causes a reconnect storm.
    socket.onclose = () => {
      if (!closedByUs) setTimeout(connect, RECONNECT_DELAY_MS)
    }
  }

  connect()

  return () => {
    closedByUs = true
    socket?.close()
  }
}
