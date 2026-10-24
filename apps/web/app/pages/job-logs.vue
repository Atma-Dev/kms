<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Button } from '~/base/ui/button'
import { Table, TableBody, TableHeader, TableRow } from '~/base/ui/table'
import { Lucide } from '~/base/ui/lucide'
import { Badge } from '~/base/ui/badge'
import { Chart, getColor } from '~/base/ui/chart'
import type { ChartConfiguration } from 'chart.js'
import { useJobLogService } from '~/services/JobLogService'
import LiveJobLogFeed from '~/components/features/job-logs/LiveJobLogFeed.vue'
import type JobLog from '~/types/entities/job_log'

definePageMeta({
  title: 'jobLogs.title',
  pageSubTitle: 'jobLogs.subtitle',
  breadcrumb: ['Job Logs'],
})

const { t } = useI18n()
const { logs, meta, loading, definitions, stats, getAll, getDefinitions, getStats } = useJobLogService()
onMounted(getDefinitions)

const statsRange = ref<'24h' | '7d' | '30d'>('24h')
watch(statsRange, () => getStats(statsRange.value), { immediate: true })
const rangeOptions: { id: '24h' | '7d' | '30d'; name: string }[] = [
  { id: '24h', name: t('jobLogs.range24h') },
  { id: '7d', name: t('jobLogs.range7d') },
  { id: '30d', name: t('jobLogs.range30d') },
]

// Chart.vue builds its ChartJs instance once on mount and never reacts to
// config changes — remount it on every stats refresh via :key.
const chartKey = computed(() => `${statsRange.value}:${stats.value?.trend.length ?? 0}:${stats.value?.summary.total ?? 0}`)
const trendChartConfig = computed<ChartConfiguration<'line'>>(() => ({
  type: 'line',
  data: {
    labels: (stats.value?.trend ?? []).map((p) => p.bucket),
    datasets: [
      { label: t('jobLogs.statSuccess'), data: (stats.value?.trend ?? []).map((p) => p.success), borderColor: () => getColor('--color-success', 1), backgroundColor: () => getColor('--color-success', 0.15), tension: 0.3, fill: true },
      { label: t('jobLogs.statFailed'), data: (stats.value?.trend ?? []).map((p) => p.failed), borderColor: () => getColor('--color-danger', 1), backgroundColor: () => getColor('--color-danger', 0.15), tension: 0.3, fill: true },
      { label: t('jobLogs.statSkipped'), data: (stats.value?.trend ?? []).map((p) => p.skipped), borderColor: () => getColor('--color-foreground', 0.5), backgroundColor: () => getColor('--color-foreground', 0.08), tension: 0.3, fill: true },
    ],
  },
  options: {
    maintainAspectRatio: false,
    plugins: { legend: { position: 'bottom', labels: { boxWidth: 10, font: { size: 11 } } } },
    scales: {
      x: { ticks: { font: { size: 10 }, color: () => getColor('--color-foreground', 0.6), maxRotation: 0, autoSkip: true }, grid: { display: false } },
      y: { beginAtZero: true, ticks: { font: { size: 11 }, color: () => getColor('--color-foreground', 0.5), precision: 0 }, grid: { color: () => getColor('--color-foreground', 0.1) } },
    },
  },
}))

const typeOptions = [
  { id: 'schedule', name: 'Schedule' },
  { id: 'queue', name: 'Queue' },
]
const statusOptions = [
  { id: 'success', name: 'Success' },
  { id: 'failed', name: 'Failed' },
]
// Registered job names, so the filter only ever offers a name that exists —
// rather than a free-text field nothing validates against.
const nameOptions = computed(() => definitions.value.map((d) => ({ id: d.name, name: d.name })))

const { filters, hasActiveFilters, reset } = useServerList({
  filters: { type: '', name: '', status: '' },
  fetch: (page, perPage, applied) => getAll(page, perPage, applied),
})

const statusVariant = (status: JobLog['status']) => (status === 'success' ? 'success' : 'danger')
</script>

<template>
  <div class="grid grid-cols-12 gap-6">
    <div v-if="stats" class="col-span-12 grid grid-cols-2 gap-4 lg:grid-cols-4">
      <Box raised="single">
        <div class="text-2xl font-medium leading-8">{{ stats.summary.total }}</div>
        <div class="mt-1.5 text-xs uppercase opacity-70">{{ $t('jobLogs.statTotal') }}</div>
      </Box>
      <Box raised="single">
        <div class="text-2xl font-medium leading-8 text-success">{{ stats.summary.success_rate.toFixed(0) }}%</div>
        <div class="mt-1.5 text-xs uppercase opacity-70">{{ $t('jobLogs.statSuccessRate') }}</div>
      </Box>
      <Box raised="single">
        <div class="text-2xl font-medium leading-8 text-danger">{{ stats.summary.failed }}</div>
        <div class="mt-1.5 text-xs uppercase opacity-70">{{ $t('jobLogs.statFailed') }}</div>
      </Box>
      <Box raised="single">
        <div class="text-2xl font-medium leading-8">{{ stats.summary.avg_duration_ms }}ms</div>
        <div class="mt-1.5 text-xs uppercase opacity-70">{{ $t('jobLogs.statAvgDuration') }}</div>
      </Box>
    </div>

    <Box class="col-span-12">
      <div class="mb-3 flex items-center justify-between">
        <p class="text-sm font-semibold">{{ $t('jobLogs.trendTitle') }}</p>
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
      </div>
      <Chart :key="chartKey" class="h-[260px]" :config="trendChartConfig" />
    </Box>

    <Box v-if="definitions.length" class="col-span-12">
      <p class="mb-2 text-sm font-semibold">{{ $t('jobLogs.registeredTitle') }}</p>
      <div class="flex flex-wrap gap-2">
        <div
          v-for="def in definitions"
          :key="def.name"
          class="flex items-center gap-2 rounded-lg border border-foreground/10 px-3 py-1.5 text-xs"
        >
          <Badge variant="secondary" look="outline" class="text-[10px]">{{ def.type }}</Badge>
          <span class="font-medium">{{ def.name }}</span>
          <span class="text-foreground/40">· {{ $t('jobLogs.everyInterval', { interval: def.interval }) }}</span>
        </div>
      </div>
    </Box>

    <Box class="col-span-12 lg:col-span-9 overflow-auto lg:overflow-visible">
      <div class="mb-4 flex justify-end">
        <FilterPopover :active="hasActiveFilters">
          <TomSelect v-model="filters.type" :options="typeOptions" :placeholder="$t('jobLogs.allTypesPlaceholder')" />
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
              <TH icon="Tag">{{ $t('jobLogs.columnType') }}</TH>
              <TH icon="Braces">{{ $t('jobLogs.columnName') }}</TH>
              <TH icon="CircleCheck">{{ $t('common.status') }}</TH>
              <TH icon="MessageSquare">{{ $t('jobLogs.columnMessage') }}</TH>
              <TH icon="Clock">{{ $t('jobLogs.columnStartedAt') }}</TH>
              <TH icon="Clock">{{ $t('jobLogs.columnFinishedAt') }}</TH>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="(log, index) in logs" :key="log.id">
              <TD>{{ meta.from + index }}</TD>
              <TD>
                <Badge variant="secondary" look="outline" class="text-xs">{{ log.type }}</Badge>
              </TD>
              <TD>{{ log.name }}</TD>
              <TD>
                <Badge :variant="statusVariant(log.status)" look="outline" class="text-xs">
                  {{ log.status }}
                </Badge>
              </TD>
              <TD class="max-w-xs whitespace-normal break-words">{{ log.message ?? '—' }}</TD>
              <TD>{{ log.started_at }}</TD>
              <TD>{{ log.finished_at ?? '—' }}</TD>
            </TableRow>
          </TableBody>
        </Table>
      </ServerSidePagination>
    </Box>

    <div class="col-span-12 lg:col-span-3">
      <LiveJobLogFeed />
    </div>
  </div>
</template>

<style scoped></style>
