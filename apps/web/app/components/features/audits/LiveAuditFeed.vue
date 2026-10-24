<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { useAuditService } from '~/services/AuditService'
import { timeAgo } from '~/utils/helper'
import type Audit from '~/types/entities/audit'

const emit = defineEmits<{ view: [audit: Audit] }>()

const MAX_ITEMS = 15
const RECONNECT_DELAY_MS = 3000

// Separate service instance from the main table below — this always shows
// the newest audits regardless of whatever the table's filters currently
// narrow it to, and never drives the table itself: the table stays exactly
// as the last manual page/filter left it, this panel is just "something new
// happened, go take a look" if you want it.
const { audits, loading, getAll } = useAuditService()

const connected = ref(false)
let socket: WebSocket | null = null
let closedByUs = false

// Mirrors activity-logs' LiveActivityFeed — page-scoped, so inlined rather
// than pulled into its own composable.
const connect = async () => {
  if (!localStorage.getItem('access_token')) return

  const token = await fetchWsToken()
  if (!token) return

  const config = useRuntimeConfig()
  const wsBase = config.public.apiBase.replace(/^http/, 'ws')
  socket = new WebSocket(`${wsBase}/api/management/audits/ws?token=${encodeURIComponent(token)}`)

  socket.onopen = () => {
    connected.value = true
  }
  socket.onmessage = (event) => {
    try {
      const audit: Audit = JSON.parse(event.data)
      audits.value = [audit, ...audits.value].slice(0, MAX_ITEMS)
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
      <p class="text-sm font-semibold">{{ $t('audits.liveFeed.title') }}</p>
      <p class="text-xs text-foreground/40">{{ connected ? $t('audits.liveFeed.connected') : $t('audits.liveFeed.reconnecting') }}</p>
    </div>

    <div v-if="loading && audits.length === 0" class="py-10 text-center text-sm text-foreground/40">
      {{ $t('common.loading') }}
    </div>
    <div v-else-if="audits.length === 0" class="py-10 text-center text-sm text-foreground/40">
      {{ $t('audits.liveFeed.noActivityYet') }}
    </div>

    <TransitionGroup
      v-else
      tag="div"
      name="live-row"
      class="flex flex-col gap-2 max-h-[70vh] overflow-y-auto"
    >
      <button
        v-for="audit in audits"
        :key="audit.id"
        type="button"
        class="w-full rounded-lg border border-foreground/10 p-2.5 text-left text-xs hover:bg-foreground/5 transition-colors"
        @click="emit('view', audit)"
      >
        <div class="flex items-center justify-between gap-2">
          <span class="truncate font-medium">{{ audit.action }}</span>
        </div>
        <div class="mt-1.5 flex items-center justify-between text-foreground/40">
          <span class="truncate">
            {{ audit.actor_name || $t('audits.liveFeed.anonymous') }}
            <template v-if="audit.entry_object_table"> · {{ audit.entry_object_table }} #{{ audit.entry_object_id }}</template>
          </span>
          <span class="shrink-0 ml-2">{{ timeAgo(audit.created_at) }}</span>
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
