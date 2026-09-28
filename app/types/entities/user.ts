import type BaseEntity from '~/types/entities/base_entity'

/**
 * Slim role shape embedded in the user payload — the backend only sends
 * `id` and `name` here, not the full Role entity.
 */
export interface UserRole {
  id: string
  name: string
}

export default interface User extends BaseEntity {
  first_name: string
  last_name: string
  full_name?: string
  email: string
  marketing_id: string | null
  /** Whether the account is allowed to use the app at all — login checks it. */
  membership_status: 'ACTIVE' | 'INACTIVE'
  deactivated_at: string | null
  roles: UserRole[]
  /** Flattened across every role, deduped — see PermissionSeeder in ../api-kms. */
  permissions: string[]
  access_token?: string
}
