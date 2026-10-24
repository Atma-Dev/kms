export type QueueJobStatus = 'pending' | 'running' | 'success' | 'failed' | 'canceled'

export default interface QueueJob {
  id: string
  queue: string
  name: string
  status: QueueJobStatus
  attempts: number
  max_attempts: number
  payload: unknown
  payload_redacted: boolean
  available_at: string
  started_at: string | null
  finished_at: string | null
  last_error: string | null
  duration_ms: number | null
}

export interface QueueStats {
  by_status: Partial<Record<QueueJobStatus, number>>
  delayed_count: number
  oldest_pending_age_seconds: number | null
  by_queue: { queue: string; count: number }[]
  by_name: {
    name: string
    count: number
    pending: number
    running: number
    success: number
    failed: number
  }[]
  registered_jobs: string[]
  queues: string[]
}
