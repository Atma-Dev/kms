export interface AuditTrail {
  id: string
  object_table: string
  object_id: string
  before: unknown
  after: unknown
  tag: unknown
  additional: unknown
}

// Not a BaseEntity: an audit row is append-only and never updated, so it
// has no meaningful updated_at.
export default interface Audit {
  id: string
  action: string
  message: string
  app_name: string
  actor_table: string
  actor_id: string
  actor_name: string
  actor_email: string
  actor_phone: string
  ip_address: string
  method: string
  endpoint: string
  user_agent: string
  entry_object_table: string
  entry_object_id: string
  tag: unknown
  additional: unknown
  trails: AuditTrail[]
  created_at: string
}
