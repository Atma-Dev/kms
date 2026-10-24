<script setup lang="ts">
import { Chart, getColor } from '~/base/ui/chart'
import type { ChartConfiguration } from 'chart.js'
import type { JobLogTrendPoint } from '~/types/entities/job_log'

const props = defineProps<{ points: JobLogTrendPoint[] }>()
const { t } = useI18n()

// Chart.vue builds its ChartJs instance once on mount and never reacts to
// config changes — remount it on every data refresh via :key.
const chartKey = computed(() => `${props.points.length}:${props.points.at(-1)?.bucket ?? ''}`)

const labels = computed(() => props.points.map((point) => point.bucket))

const config = computed<ChartConfiguration<'bar'>>(() => ({
  type: 'bar',
  data: {
    labels: labels.value,
    datasets: [
      {
        label: t('jobLogs.statSuccess'),
        data: props.points.map((p) => p.success),
        backgroundColor: getColor('--color-success', 0.75),
        borderRadius: 2,
      },
      {
        label: t('jobLogs.statFailed'),
        data: props.points.map((p) => p.failed),
        backgroundColor: getColor('--color-danger', 0.85),
        borderRadius: 2,
      },
      {
        label: t('jobLogs.statSkipped'),
        data: props.points.map((p) => p.skipped),
        backgroundColor: getColor('--color-warning', 0.8),
        borderRadius: 2,
      },
    ],
  },
  options: {
    maintainAspectRatio: false,
    plugins: { legend: { position: 'bottom', labels: { boxWidth: 10, font: { size: 11 } } } },
    scales: {
      x: {
        stacked: true,
        ticks: {
          font: { size: 10 },
          color: () => getColor('--color-foreground', 0.6),
          maxRotation: 0,
          autoSkip: true,
        },
        grid: { display: false },
      },
      y: {
        stacked: true,
        beginAtZero: true,
        ticks: {
          font: { size: 11 },
          color: () => getColor('--color-foreground', 0.5),
          precision: 0,
        },
        grid: { color: () => getColor('--color-foreground', 0.1) },
      },
    },
  },
}))
</script>

<template>
  <Chart :key="chartKey" class="h-[260px]" :config="config" />
</template>
