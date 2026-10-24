export type JobLogType = 'schedule' | 'queue'
export type JobLogStatus = 'success' | 'failed' | 'skipped'

export default interface JobLog {
  id: string
  type: JobLogType
  name: string
  status: JobLogStatus
  message: string | null
  started_at: string
  finished_at: string | null
}

export interface JobDefinition {
  type: JobLogType
  name: string
  interval: string
}

export interface JobLogTrendPoint {
  bucket: string
  success: number
  failed: number
  skipped: number
}

export interface JobLogRollup {
  type: JobLogType
  name: string
  total: number
  success: number
  failed: number
  skipped: number
  avg_duration_ms: number
  max_duration_ms: number
  last_status: JobLogStatus | null
  last_run_at: string | null
  f_last_run_at: string | null
}

export interface JobLogStats {
  summary: {
    total: number
    success: number
    failed: number
    skipped: number
    success_rate: number
    avg_duration_ms: number
    max_duration_ms: number
  }
  trend: JobLogTrendPoint[]
  by_job: JobLogRollup[]
}
