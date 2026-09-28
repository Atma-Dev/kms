<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Badge } from '~/base/ui/badge'
import { useQueueService } from '~/services/QueueService'
import { queueJobStatusVariant } from '~/utils/queue'
import { timeAgo } from '~/utils/helper'
import type QueueJob from '~/types/entities/queue_job'

const MAX_ITEMS = 15
const RECONNECT_DELAY_MS = 3000

// Separate service instance from the main table, same reason as
// LiveJobLogFeed: this always shows the newest activity regardless of the
// table's active filters.
const { jobs, loading, getAll } = useQueueService()

const connected = ref(false)
let socket: WebSocket | null = null
let closedByUs = false

// A queue frame is a state transition of an existing row (enqueue -> claim
// -> finish), not always a new row — dedupe by id before prepending, unlike
// the append-only job-log/audit feeds.
const upsert = (job: QueueJob) => {
  jobs.value = [job, ...jobs.value.filter((j) => j.id !== job.id)].slice(0, MAX_ITEMS)
}

const connect = async () => {
  if (!localStorage.getItem('access_token')) return

  const token = await fetchWsToken()
  if (!token) return

  const config = useRuntimeConfig()
  const wsBase = config.public.apiBase.replace(/^http/, 'ws')
  socket = new WebSocket(`${wsBase}/api/management/queue/ws?token=${encodeURIComponent(token)}`)

  socket.onopen = () => {
    connected.value = true
  }
  socket.onmessage = (event) => {
    try {
      upsert(JSON.parse(event.data) as QueueJob)
    } catch {
      // ignore malformed frames
    }
  }
  // ponytail: fixed-delay reconnect, no exponential backoff/jitter — same
  // simplification as LiveJobLogFeed/LiveActivityFeed.
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
      <p class="text-sm font-semibold">{{ $t('queue.liveFeed.title') }}</p>
      <p class="text-xs text-foreground/40">{{ connected ? $t('queue.liveFeed.connected') : $t('queue.liveFeed.reconnecting') }}</p>
    </div>

    <div v-if="loading && jobs.length === 0" class="py-10 text-center text-sm text-foreground/40">
      {{ $t('common.loading') }}
    </div>
    <div v-else-if="jobs.length === 0" class="py-10 text-center text-sm text-foreground/40">
      {{ $t('queue.liveFeed.noJobsYet') }}
    </div>

    <TransitionGroup
      v-else
      tag="div"
      name="live-row"
      class="flex flex-col gap-2 max-h-[70vh] overflow-y-auto"
    >
      <div
        v-for="job in jobs"
        :key="job.id"
        class="w-full rounded-lg border border-foreground/10 p-2.5 text-left text-xs"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-1.5 min-w-0">
            <Badge variant="secondary" look="outline" class="text-[10px] shrink-0">{{ job.name }}</Badge>
          </div>
          <Badge :variant="queueJobStatusVariant(job.status)" look="outline" class="text-[10px] shrink-0">
            {{ job.status }}
          </Badge>
        </div>
        <div v-if="job.last_error" class="mt-1.5 break-words text-foreground/70">{{ job.last_error }}</div>
        <div class="mt-1.5 flex items-center justify-between text-foreground/40">
          <span>{{ job.queue }} · {{ $t('queue.attemptShort', { n: job.attempts, max: job.max_attempts }) }}</span>
          <span>{{ timeAgo(job.available_at) }}</span>
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
