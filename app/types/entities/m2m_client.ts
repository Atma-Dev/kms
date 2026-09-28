import type BaseEntity from '~/types/entities/base_entity'

export default interface M2MClient extends BaseEntity {
  client: string
  /** Route scopes this client may call — "*" grants every scope. */
  scope: string[]
  revoked_at: string | null
}
