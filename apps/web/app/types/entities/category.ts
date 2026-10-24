import type BaseEntity from './base_entity'

/**
 * One node of a space's category tree. `children` is always an array — the API
 * serialises a leaf as `[]`, never null, so a recursive component can walk it
 * without a type check at every level.
 */
export default interface Category extends BaseEntity {
  space_id: string
  parent_id: string | null
  name: string
  slug: string
  order: number
  children: Category[]
  /**
   * Documents in this category *and everything beneath it* — which is what
   * clicking it lists. A count of only direct children would read 0 beside a
   * category holding a whole branch.
   */
  document_count: number
  created_at: string
  updated_at: string
}

/** The tree endpoint's envelope: the roots plus the two numbers no node owns. */
export interface CategoryTree {
  categories: Category[]
  total_document_count: number
  uncategorised_count: number
}
