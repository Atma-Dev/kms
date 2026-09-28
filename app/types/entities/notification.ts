import type BaseEntity from '~/types/entities/base_entity'

export default interface AppNotification extends BaseEntity {
  type: string
  title: string
  body: string
  data: Record<string, unknown> | null
  is_read: boolean
  f_created_at: string
}
