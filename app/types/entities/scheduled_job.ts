export default interface ScheduledJob {
  id: string
  name: string
  cron_expression: string
  enabled: boolean
  next_run_at: string | null
  last_run_at: string | null
  last_status: string | null
  last_message: string | null
  last_duration_ms: number | null
}
