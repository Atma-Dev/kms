import type BaseEntity from './base_entity'
import type Tag from './tag'
import type { ProtectionLevel } from './protection'

export type DocumentStatus = 'draft' | 'in_review' | 'published' | 'archived'

/** The lifecycle moves, matching document.Action on the API. */
export type DocumentAction =
  | 'submit-review'
  | 'approve'
  | 'reject'
  | 'publish'
  | 'unpublish'
  | 'archive'
  | 'restore'

/**
 * The editor's node tree (TipTap/ProseMirror), stored and returned as-is. It
 * is deliberately untyped past "a JSON object": modelling node types here
 * would mean editing this file every time the editor gains a mark.
 */
export type DocumentContent = Record<string, unknown>

/** The list row — no content, so a page of twenty stays small. */
export default interface Document extends BaseEntity {
  space_id: string
  category_id: string | null
  title: string
  slug: string
  excerpt: string | null
  status: DocumentStatus
  author_id: string
  view_count: number
  is_pinned: boolean
  published_at: string | null
  created_at: string
  updated_at: string
}

export interface DocumentDetail extends Document {
  content: DocumentContent
  last_editor_id: string | null
  current_version_id: string | null
  /**
   * The lifecycle moves this caller may make right now, already filtered by
   * their permissions server-side. Render exactly these buttons — re-deriving
   * the rules here is how the two drift apart.
   */
  available_actions: DocumentAction[]
  can_edit: boolean
  tags: Tag[]
  /**
   * The document carries access grants, so this reader is seeing it because
   * they are named on it. Who else is on the list is gated separately — the
   * client only ever needs to know that a lock is on.
   */
  is_restricted: boolean
  /**
   * The level actually in force (Modul 4 FR-6), with the space default already
   * applied — render from this, never from the override below.
   */
  protection_level: ProtectionLevel
  /** The document's own setting; null means it inherits its space. */
  protection_level_override: ProtectionLevel | null
  /**
   * The text was withheld because the document is strict and this caller may
   * only read it — load the page images instead. Without the flag an empty
   * content tree would be indistinguishable from an empty document.
   */
  content_hidden: boolean
}

export interface DocumentVersion {
  id: string
  version_number: number
  editor_id: string | null
  change_note: string | null
  created_at: string
}

export type DiffOp = 'equal' | 'add' | 'remove'

export interface DiffLine {
  op: DiffOp
  text: string
  from_line: number | null
  to_line: number | null
}
