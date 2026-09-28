<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Badge } from '~/base/ui/badge'
import { useJobLogService } from '~/services/JobLogService'
import { timeAgo } from '~/utils/helper'
import type JobLog from '~/types/entities/job_log'

const MAX_ITEMS = 15
const RECONNECT_DELAY_MS = 3000

// Separate service instance from the main table below, same reason as
// activity-logs' LiveActivityFeed: this always shows the newest run regardless of
// whatever the table's filters currently narrow it to.
const { logs, loading, getAll } = useJobLogService()

const connected = ref(false)
let socket: WebSocket | null = null
let closedByUs = false

const connect = async () => {
  if (!localStorage.getItem('access_token')) return

  const token = await fetchWsToken()
  if (!token) return

  const config = useRuntimeConfig()
  const wsBase = config.public.apiBase.replace(/^http/, 'ws')
  socket = new WebSocket(`${wsBase}/api/management/job-logs/ws?token=${encodeURIComponent(token)}`)

  socket.onopen = () => {
    connected.value = true
  }
  socket.onmessage = (event) => {
    try {
      const log: JobLog = JSON.parse(event.data)
      logs.value = [log, ...logs.value].slice(0, MAX_ITEMS)
    } catch {
      // ignore malformed frames
    }
  }
  // ponytail: fixed-delay reconnect, no exponential backoff/jitter — same
  // simplification as LiveActivityFeed, upgrade both together if needed.
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

const statusVariant = (status: JobLog['status']) => (status === 'success' ? 'success' : 'danger')
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
      <p class="text-sm font-semibold">{{ $t('jobLogs.liveFeed.title') }}</p>
      <p class="text-xs text-foreground/40">{{ connected ? $t('jobLogs.liveFeed.connected') : $t('jobLogs.liveFeed.reconnecting') }}</p>
    </div>

    <div v-if="loading && logs.length === 0" class="py-10 text-center text-sm text-foreground/40">
      {{ $t('common.loading') }}
    </div>
    <div v-else-if="logs.length === 0" class="py-10 text-center text-sm text-foreground/40">
      {{ $t('jobLogs.liveFeed.noRunsYet') }}
    </div>

    <TransitionGroup
      v-else
      tag="div"
      name="live-row"
      class="flex flex-col gap-2 max-h-[70vh] overflow-y-auto"
    >
      <div
        v-for="log in logs"
        :key="log.id"
        class="w-full rounded-lg border border-foreground/10 p-2.5 text-left text-xs"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-1.5 min-w-0">
            <Badge variant="secondary" look="outline" class="text-[10px] shrink-0">{{ log.name }}</Badge>
          </div>
          <Badge :variant="statusVariant(log.status)" look="outline" class="text-[10px] shrink-0">
            {{ log.status }}
          </Badge>
        </div>
        <div v-if="log.message" class="mt-1.5 break-words text-foreground/70">{{ log.message }}</div>
        <div class="mt-1.5 flex items-center justify-between text-foreground/40">
          <span>{{ log.type }}</span>
          <span>{{ timeAgo(log.started_at) }}</span>
        </div>
      </div>
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
