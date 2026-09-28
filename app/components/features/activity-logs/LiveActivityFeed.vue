<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Badge } from '~/base/ui/badge'
import { useActivityLogService } from '~/services/ActivityLogService'
import { timeAgo } from '~/utils/helper'
import type ActivityLog from '~/types/entities/activity_log'

const emit = defineEmits<{ view: [log: ActivityLog] }>()

const MAX_ITEMS = 15
const RECONNECT_DELAY_MS = 3000

// A separate service instance from the main table below — this one always
// shows the newest, unfiltered activity rather than whatever the table's
// filters currently narrow it to. Used once on mount to seed the panel so it
// isn't empty until the next live event arrives.
const { logs, loading, getAll } = useActivityLogService()

const connected = ref(false)
let socket: WebSocket | null = null
let closedByUs = false

// Mirrors useNotificationSocket's connect/reconnect shape — this one is
// page-scoped (only ever used here), so it's inlined rather than pulled into
// its own composable.
const connect = async () => {
  if (!localStorage.getItem('access_token')) return

  // A dedicated token scoped to WS routes only — see useWsToken. Fetched
  // fresh on every (re)connect rather than the regular access_token: that
  // one depends on the access_token_verifier cookie crossing to the API's
  // subdomain during the handshake, which browsers do not reliably do.
  const token = await fetchWsToken()
  if (!token) return

  const config = useRuntimeConfig()
  const wsBase = config.public.apiBase.replace(/^http/, 'ws')
  socket = new WebSocket(`${wsBase}/api/management/activity-logs/ws?token=${encodeURIComponent(token)}`)

  socket.onopen = () => {
    connected.value = true
  }
  socket.onmessage = (event) => {
    try {
      const log: ActivityLog = JSON.parse(event.data)
      logs.value = [log, ...logs.value].slice(0, MAX_ITEMS)
    } catch {
      // ignore malformed frames
    }
  }
  // ponytail: fixed-delay reconnect, no exponential backoff/jitter — upgrade
  // if this ever causes a reconnect storm.
  socket.onclose = () => {
    connected.value = false
    if (!closedByUs) setTimeout(connect, RECONNECT_DELAY_MS)
  }
}

onMounted(() => {
  getAll(1, MAX_ITEMS)
  connect()
})
onUnmounted(() => {
  closedByUs = true
  socket?.close()
})

const statusVariant = (status: number) => {
  if (status >= 500) return 'danger'
  if (status >= 400) return 'warning'
  return 'success'
}
</script>

<template>
  <Box class="h-full">
    <div class="mb-3 flex items-center gap-2">
      <span class="relative flex size-2">
        <span
          v-if="connected"
          class="absolute inline-flex h-full w-full animate-ping rounded-full bg-success opacity-75"
        />
        <span class="relative inline-flex size-2 rounded-full" :class="connected ? 'bg-success' : 'bg-foreground/30'" />
      </span>
      <p class="text-sm font-semibold">{{ $t('activityLogs.liveFeed.title') }}</p>
      <p class="text-xs text-foreground/40">{{ connected ? $t('activityLogs.liveFeed.connected') : $t('activityLogs.liveFeed.reconnecting') }}</p>
    </div>

    <div v-if="loading && logs.length === 0" class="py-10 text-center text-sm text-foreground/40">
      {{ $t('common.loading') }}
    </div>
    <div v-else-if="logs.length === 0" class="py-10 text-center text-sm text-foreground/40">
      {{ $t('activityLogs.liveFeed.noActivityYet') }}
    </div>

    <TransitionGroup
      v-else
      tag="div"
      name="live-row"
      class="flex flex-col gap-2 max-h-[70vh] overflow-y-auto"
    >
      <button
        v-for="log in logs"
        :key="log.id"
        type="button"
        class="w-full rounded-lg border border-foreground/10 p-2.5 text-left text-xs hover:bg-foreground/5 transition-colors"
        @click="emit('view', log)"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-1.5 min-w-0">
            <Badge variant="secondary" look="outline" class="text-[10px] shrink-0">{{ log.method }}</Badge>
            <span class="truncate font-medium">{{ log.path }}</span>
          </div>
          <Badge :variant="statusVariant(log.status_code)" look="outline" class="text-[10px] shrink-0">
            {{ log.status_code }}
          </Badge>
        </div>
        <div class="mt-1.5 flex items-center justify-between text-foreground/40">
          <span>{{ log.user?.full_name ?? $t('activityLogs.liveFeed.anonymous') }}</span>
          <span>{{ timeAgo(log.created_at) }}</span>
        </div>
      </button>
    </TransitionGroup>
  </Box>
</template>

<style scoped>
.live-row-enter-active {
  transition: all 0.3s ease;
}
.live-row-enter-from {
  opacity: 0;
  transform: translateY(-6px);
}
</style>
