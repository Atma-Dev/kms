import type BaseEntity from './base_entity'

export default interface Tag extends BaseEntity {
  name: string
  slug: string
  /** Only present on the tag master — the chips on a document have no use for it. */
  document_count?: number
  created_at: string
  updated_at: string
}
