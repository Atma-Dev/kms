import type { BadgeVariants } from '~/base/ui/styles/badge.styles'
import type { QueueJobStatus } from '~/types/entities/queue_job'

type BadgeVariant = NonNullable<BadgeVariants['variant']>

const STATUS_VARIANT: Record<QueueJobStatus, BadgeVariant> = {
  pending: 'pending',
  running: 'primary',
  success: 'success',
  failed: 'danger',
  canceled: 'secondary',
}

export const queueJobStatusVariant = (status: QueueJobStatus): BadgeVariant => STATUS_VARIANT[status] ?? 'secondary'

export const formatDurationMs = (ms: number | null): string => {
  if (ms === null) return '—'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

export const formatAgeSeconds = (seconds: number | null): string => {
  if (seconds === null) return '—'
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  return `${Math.floor(seconds / 3600)}h`
}
