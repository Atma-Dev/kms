<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Button } from '~/base/ui/button'
import { Badge } from '~/base/ui/badge'
import { Lucide } from '~/base/ui/lucide'
import { Table, TableBody, TableHeader, TableRow } from '~/base/ui/table'
import { useJobLogService } from '~/services/JobLogService'
import { useQueueService } from '~/services/QueueService'
import { formatDurationMs, formatAgeSeconds } from '~/utils/queue'
import StatTile from '~/components/features/jobs/StatTile.vue'
import JobTrendChart from '~/components/features/jobs/JobTrendChart.vue'
import type { JobLogStatus } from '~/types/entities/job_log'
import type { BadgeVariants } from '~/base/ui/styles/badge.styles'

definePageMeta({
  title: 'jobsDashboard.title',
  pageSubTitle: 'jobsDashboard.subtitle',
  breadcrumb: ['Jobs Dashboard'],
})

const { t } = useI18n()

const { stats: runStats, getStats: fetchRunStats } = useJobLogService()
const { stats: queueStats, getStats: fetchQueueStats } = useQueueService()

const statsRange = ref<'24h' | '7d' | '30d'>('24h')
const rangeOptions: { id: '24h' | '7d' | '30d'; name: string }[] = [
  { id: '24h', name: t('jobLogs.range24h') },
  { id: '7d', name: t('jobLogs.range7d') },
  { id: '30d', name: t('jobLogs.range30d') },
]

const loading = ref(false)
const load = async () => {
  loading.value = true
  try {
    await Promise.all([fetchRunStats(statsRange.value), fetchQueueStats()])
  } finally {
    loading.value = false
  }
}
watch(statsRange, load)
onMounted(load)

const summary = computed(() => runStats.value?.summary)
const successRateVariant = computed(() => {
  if (!summary.value || summary.value.total === 0) return 'secondary'
  if (summary.value.success_rate >= 99) return 'success'
  if (summary.value.success_rate >= 90) return 'warning'
  return 'danger'
})

const backlogVariant = computed(() => {
  const age = queueStats.value?.oldest_pending_age_seconds
  if (age === null || age === undefined) return 'secondary'
  if (age > 300) return 'danger'
  if (age > 60) return 'warning'
  return 'success'
})

type BadgeVariant = NonNullable<BadgeVariants['variant']>
const STATUS_VARIANT: Record<JobLogStatus, BadgeVariant> = {
  success: 'success',
  failed: 'danger',
  skipped: 'secondary',
}
const jobStatusVariant = (status: JobLogStatus): BadgeVariant =>
  STATUS_VARIANT[status] ?? 'secondary'
</script>

<template>
  <div class="grid grid-cols-12 gap-6">
    <div class="col-span-12 flex flex-wrap items-center justify-between gap-3">
      <p class="text-sm font-medium opacity-70">{{ $t('jobsDashboard.runsHeading') }}</p>
      <div class="flex items-center gap-2">
        <div class="flex gap-1">
          <Button
            v-for="opt in rangeOptions"
            :key="opt.id"
            size="sm"
            :look="statsRange === opt.id ? 'filled' : 'outline'"
            :variant="statsRange === opt.id ? 'primary' : 'ghost'"
            @click="statsRange = opt.id"
          >
            {{ opt.name }}
          </Button>
        </div>
        <Button look="outline" size="sm" :disabled="loading" @click="load">
          <Lucide icon="RefreshCw" class="mr-1.5 size-4" />
          {{ $t('common.refresh') }}
        </Button>
      </div>
    </div>

    <StatTile
      class="col-span-6 xl:col-span-3"
      icon="Activity"
      :label="$t('jobsDashboard.totalRuns')"
      :value="summary ? String(summary.total) : '—'"
    />
    <StatTile
      class="col-span-6 xl:col-span-3"
      icon="CircleCheck"
      :label="$t('jobsDashboard.successRate')"
      :value="summary ? `${summary.success_rate.toFixed(1)}%` : '—'"
      :variant="successRateVariant"
    />
    <StatTile
      class="col-span-6 xl:col-span-3"
      icon="CircleX"
      :label="$t('jobsDashboard.failedRuns')"
      :value="summary ? String(summary.failed) : '—'"
      :variant="summary && summary.failed > 0 ? 'danger' : 'secondary'"
    />
    <StatTile
      class="col-span-6 xl:col-span-3"
      icon="Timer"
      :label="$t('jobsDashboard.avgDuration')"
      :value="formatDurationMs(summary?.avg_duration_ms ?? null)"
    />

    <div class="col-span-12 -mb-2">
      <p class="text-sm font-medium opacity-70">{{ $t('jobsDashboard.queueHeading') }}</p>
    </div>

    <StatTile
      class="col-span-6 xl:col-span-3"
      icon="Hourglass"
      :label="$t('queue.statPending')"
      :value="queueStats ? String(queueStats.by_status.pending ?? 0) : '—'"
      :hint="
        queueStats && queueStats.delayed_count > 0
          ? $t('jobsDashboard.delayedHint', { count: queueStats.delayed_count })
          : ''
      "
    />
    <StatTile
      class="col-span-6 xl:col-span-3"
      icon="Play"
      :label="$t('queue.statRunning')"
      :value="queueStats ? String(queueStats.by_status.running ?? 0) : '—'"
      :variant="queueStats && (queueStats.by_status.running ?? 0) > 0 ? 'primary' : 'secondary'"
    />
    <StatTile
      class="col-span-6 xl:col-span-3"
      icon="TriangleAlert"
      :label="$t('queue.statFailed')"
      :value="queueStats ? String(queueStats.by_status.failed ?? 0) : '—'"
      :variant="queueStats && (queueStats.by_status.failed ?? 0) > 0 ? 'danger' : 'secondary'"
    />
    <StatTile
      class="col-span-6 xl:col-span-3"
      icon="Clock"
      :label="$t('jobsDashboard.backlogAge')"
      :value="formatAgeSeconds(queueStats?.oldest_pending_age_seconds ?? null)"
      :variant="backlogVariant"
    />

    <Box class="col-span-12 p-5 xl:col-span-8">
      <p class="mb-4 font-medium">{{ $t('jobLogs.trendTitle') }}</p>
      <JobTrendChart :points="runStats?.trend ?? []" />
    </Box>

    <Box class="col-span-12 self-start p-5 xl:col-span-4">
      <p class="mb-4 font-medium">{{ $t('jobsDashboard.queueByJob') }}</p>
      <div
        v-if="!queueStats || queueStats.by_name.length === 0"
        class="py-8 text-center text-sm opacity-40"
      >
        {{ $t('jobsDashboard.queueEmpty') }}
      </div>
      <div v-else class="flex flex-col gap-2">
        <div
          v-for="row in queueStats.by_name"
          :key="row.name"
          class="rounded-lg border border-foreground/10 p-3"
        >
          <div class="flex items-center justify-between gap-2">
            <span class="truncate text-sm font-medium">{{ row.name }}</span>
            <Badge v-if="row.failed > 0" variant="danger" look="outline" class="shrink-0">
              {{ $t('jobsDashboard.failedCount', { count: row.failed }) }}
            </Badge>
          </div>
          <div class="mt-1.5 flex gap-3 text-xs opacity-60">
            <span>{{ $t('queue.statPending') }}: {{ row.pending }}</span>
            <span>{{ $t('queue.statRunning') }}: {{ row.running }}</span>
            <span>{{ $t('jobLogs.statSuccess') }}: {{ row.success }}</span>
          </div>
        </div>
      </div>
    </Box>

    <Box class="col-span-12 overflow-auto p-5 lg:overflow-visible">
      <p class="mb-4 font-medium">{{ $t('jobsDashboard.perJobTitle') }}</p>
      <Table class="-mt-2" variant="boxed">
        <TableHeader>
          <TableRow>
            <TH icon="Timer">{{ $t('jobLogs.columnName') }}</TH>
            <TH icon="Hash">{{ $t('jobsDashboard.columnRuns') }}</TH>
            <TH icon="CircleX">{{ $t('jobsDashboard.columnFailed') }}</TH>
            <TH icon="Gauge">{{ $t('jobsDashboard.columnAvg') }}</TH>
            <TH icon="ChevronsUp">{{ $t('jobsDashboard.columnMax') }}</TH>
            <TH icon="Clock">{{ $t('jobsDashboard.columnLastRun') }}</TH>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="row in runStats?.by_job ?? []" :key="`${row.type}:${row.name}`">
            <TD>
              <p class="font-medium">{{ row.name }}</p>
              <SmallInfo>{{ row.type }}</SmallInfo>
            </TD>
            <TD>
              {{ row.total }}
              <SmallInfo v-if="row.skipped > 0">{{
                $t('jobsDashboard.skippedCount', { count: row.skipped })
              }}</SmallInfo>
            </TD>
            <TD>
              <Badge :variant="row.failed > 0 ? 'danger' : 'secondary'" look="outline">{{
                row.failed
              }}</Badge>
            </TD>
            <TD>{{ formatDurationMs(row.avg_duration_ms) }}</TD>
            <TD>{{ formatDurationMs(row.max_duration_ms) }}</TD>
            <TD>
              <div class="flex items-center gap-2">
                <Badge
                  v-if="row.last_status"
                  :variant="jobStatusVariant(row.last_status)"
                  look="outline"
                >
                  {{ row.last_status }}
                </Badge>
                <span class="text-sm opacity-70">{{ row.f_last_run_at ?? '—' }}</span>
              </div>
            </TD>
          </TableRow>
        </TableBody>
      </Table>

      <EmptyData v-if="!loading && (runStats?.by_job.length ?? 0) === 0" />
    </Box>
  </div>
</template>

<style scoped></style>
