import type BaseEntity from '~/types/entities/base_entity'
import type User from '~/types/entities/user'

export default interface ActivityLog extends BaseEntity {
  user: User | null
  method: string
  path: string
  status_code: number
  ip: string
  request_body: string | null
  response_body: string | null
}
