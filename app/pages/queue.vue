<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Button } from '~/base/ui/button'
import { Table, TableBody, TableHeader, TableRow } from '~/base/ui/table'
import { Lucide } from '~/base/ui/lucide'
import { Badge } from '~/base/ui/badge'
import { useQueueService } from '~/services/QueueService'
import { queueJobStatusVariant, formatDurationMs, formatAgeSeconds } from '~/utils/queue'
import LiveQueueFeed from '~/components/features/queue/LiveQueueFeed.vue'
import type QueueJob from '~/types/entities/queue_job'
import type { ModalDelete } from '#components'
import { Permission } from '~/enums/Permission'

definePageMeta({
  title: 'queue.title',
  pageSubTitle: 'queue.subtitle',
  breadcrumb: ['Queue'],
})

const authStore = useAuthStore()
const canManage = computed(() => authStore.hasPermission(Permission.MANAGEMENT_QUEUE_MANAGE))

const { jobs, meta, stats, loading, getAll, getStats, retry, cancel, retryFailed, purge } = useQueueService()
onMounted(getStats)

const statusOptions = [
  { id: 'pending', name: 'Pending' },
  { id: 'running', name: 'Running' },
  { id: 'success', name: 'Success' },
  { id: 'failed', name: 'Failed' },
  { id: 'canceled', name: 'Canceled' },
]
// Registered handler names, so the filter only offers a name that can
// actually appear — same reasoning as job-logs' name filter.
const nameOptions = computed(() => (stats.value?.registered_jobs ?? []).map((name) => ({ id: name, name })))
const queueOptions = computed(() => (stats.value?.queues ?? []).map((q) => ({ id: q, name: q })))

const { filters, hasActiveFilters, reset, refresh } = useServerList({
  filters: { queue: '', name: '', status: '' },
  fetch: (page, perPage, applied) => getAll(page, perPage, applied),
})

const refreshAll = () => {
  refresh()
  getStats()
}

const onRetry = async (id: string) => {
  if (await retry(id)) refreshAll()
}
const onCancel = async (id: string) => {
  if (await cancel(id)) refreshAll()
}

// retryFailed just re-queues rows — not destructive, fires directly. purge
// permanently deletes rows, so it goes through the shared delete-confirm
// modal like every other destructive action in this app.
const modalPurgeRef = useTemplateRef<InstanceType<typeof ModalDelete> | null>('modalPurgeRef')

const onRetryFailed = async () => {
  if (await retryFailed()) refreshAll()
}
const onPurge = async () => {
  if (await purge(['success', 'failed', 'canceled'])) refreshAll()
}

const canRetry = (job: QueueJob) => job.status === 'success' || job.status === 'failed' || job.status === 'canceled'
const canCancel = (job: QueueJob) => job.status === 'pending'
</script>

<template>
  <div class="grid grid-cols-12 gap-6">
    <div v-if="stats" class="col-span-12 grid grid-cols-2 gap-4 lg:grid-cols-5">
      <Box raised="single">
        <div class="text-2xl font-medium leading-8">{{ stats.by_status.pending ?? 0 }}</div>
        <div class="mt-1.5 text-xs uppercase opacity-70">{{ $t('queue.statPending') }}</div>
      </Box>
      <Box raised="single">
        <div class="text-2xl font-medium leading-8">{{ stats.by_status.running ?? 0 }}</div>
        <div class="mt-1.5 text-xs uppercase opacity-70">{{ $t('queue.statRunning') }}</div>
      </Box>
      <Box raised="single">
        <div class="text-2xl font-medium leading-8 text-danger">{{ stats.by_status.failed ?? 0 }}</div>
        <div class="mt-1.5 text-xs uppercase opacity-70">{{ $t('queue.statFailed') }}</div>
      </Box>
      <Box raised="single">
        <div class="text-2xl font-medium leading-8">{{ stats.delayed_count }}</div>
        <div class="mt-1.5 text-xs uppercase opacity-70">{{ $t('queue.statDelayed') }}</div>
      </Box>
      <Box raised="single">
        <div class="text-2xl font-medium leading-8">{{ formatAgeSeconds(stats.oldest_pending_age_seconds) }}</div>
        <div class="mt-1.5 text-xs uppercase opacity-70">{{ $t('queue.statOldestPending') }}</div>
      </Box>
    </div>

    <Box v-if="stats?.registered_jobs.length" class="col-span-12">
      <p class="mb-2 text-sm font-semibold">{{ $t('queue.registeredTitle') }}</p>
      <div class="flex flex-wrap gap-2">
        <Badge v-for="name in stats.registered_jobs" :key="name" variant="secondary" look="outline" class="text-xs">
          {{ name }}
        </Badge>
      </div>
    </Box>
    <Box v-else class="col-span-12">
      <p class="text-sm text-foreground/50">{{ $t('queue.noHandlersRegistered') }}</p>
    </Box>

    <Box class="col-span-12 lg:col-span-9 overflow-auto lg:overflow-visible">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div v-if="canManage" class="flex gap-2">
          <Button look="outline" variant="ghost" size="sm" @click="onRetryFailed">
            <Lucide icon="RotateCcw" class="mr-1.5 size-4" />
            {{ $t('queue.retryAllFailed') }}
          </Button>
          <Button look="outline" variant="ghost" size="sm" @click="modalPurgeRef?.handleModal(true)">
            <Lucide icon="Trash2" class="mr-1.5 size-4" />
            {{ $t('queue.purgeFinished') }}
          </Button>
        </div>
        <div v-else />

        <FilterPopover :active="hasActiveFilters">
          <TomSelect v-model="filters.queue" :options="queueOptions" :placeholder="$t('queue.allQueuesPlaceholder')" />
          <TomSelect v-model="filters.name" :options="nameOptions" :placeholder="$t('jobLogs.allJobsPlaceholder')" />
          <TomSelect v-model="filters.status" :options="statusOptions" :placeholder="$t('common.status')" />

          <template #footer>
            <Button look="outline" variant="ghost" size="sm" :disabled="!hasActiveFilters" @click="reset">
              <Lucide class="mr-1.5 size-4" icon="X" />
              {{ $t('common.reset') }}
            </Button>
          </template>
        </FilterPopover>
      </div>

      <ServerSidePagination :meta="meta" :loading="loading">
        <Table class="-mt-2" variant="boxed">
          <TableHeader>
            <TableRow>
              <TH icon="Hash">{{ $t('common.no') }}</TH>
              <TH icon="Layers">{{ $t('queue.columnQueue') }}</TH>
              <TH icon="Braces">{{ $t('jobLogs.columnName') }}</TH>
              <TH icon="CircleCheck">{{ $t('common.status') }}</TH>
              <TH icon="Repeat">{{ $t('queue.columnAttempts') }}</TH>
              <TH icon="MessageSquare">{{ $t('queue.columnLastError') }}</TH>
              <TH icon="Timer">{{ $t('queue.columnDuration') }}</TH>
              <TH v-if="canManage" icon="Settings">{{ $t('common.actions') }}</TH>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="(job, index) in jobs" :key="job.id">
              <TD>{{ meta.from + index }}</TD>
              <TD><Badge variant="secondary" look="outline" class="text-xs">{{ job.queue }}</Badge></TD>
              <TD>{{ job.name }}</TD>
              <TD>
                <Badge :variant="queueJobStatusVariant(job.status)" look="outline" class="text-xs">
                  {{ job.status }}
                </Badge>
              </TD>
              <TD>{{ job.attempts }}/{{ job.max_attempts }}</TD>
              <TD class="max-w-xs whitespace-normal break-words">{{ job.last_error ?? '—' }}</TD>
              <TD>{{ formatDurationMs(job.duration_ms) }}</TD>
              <TD v-if="canManage">
                <div class="flex items-center gap-1">
                  <a
                    v-if="canRetry(job)"
                    href="#"
                    class="mr-3 flex items-center"
                    @click.prevent="onRetry(job.id)"
                  >
                    <Lucide class="mr-1 h-4 w-4" icon="RotateCcw" />
                    {{ $t('queue.actionRetry') }}
                  </a>
                  <a
                    v-if="canCancel(job)"
                    href="#"
                    class="mr-3 flex items-center text-danger"
                    @click.prevent="onCancel(job.id)"
                  >
                    <Lucide class="mr-1 h-4 w-4" icon="Ban" />
                    {{ $t('queue.actionCancel') }}
                  </a>
                  <span v-if="!canRetry(job) && !canCancel(job)" class="text-foreground/30">—</span>
                </div>
              </TD>
            </TableRow>
          </TableBody>
        </Table>
      </ServerSidePagination>
    </Box>

    <div class="col-span-12 lg:col-span-3">
      <LiveQueueFeed />
    </div>
  </div>

  <ModalDelete ref="modalPurgeRef" @submit="onPurge" />
</template>

<style scoped></style>
