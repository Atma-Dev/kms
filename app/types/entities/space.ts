import type BaseEntity from './base_entity'
import type { ProtectionLevel } from './protection'

export default interface Space extends BaseEntity {
  name: string
  slug: string
  description: string | null
  icon: string | null
  is_archived: boolean
  /**
   * The default every document in this space is served at unless it carries an
   * override of its own (Modul 4 FR-6).
   */
  protection_level: ProtectionLevel
  created_by: string
  created_at: string
  updated_at: string
}
