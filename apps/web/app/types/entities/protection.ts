/**
 * Modul 4: how a document defends itself once it is readable.
 *
 * What each level actually promises is worth being clear about, because the
 * UI is where the difference is easiest to overstate:
 *
 * - `normal`: the document renders as itself, with a watermark overlaid in
 *   this browser. Anyone who opens dev tools can delete that overlay. It makes
 *   a careless copy carry a name; it is not proof of anything.
 * - `strict`: the server sends page images with the watermark already in the
 *   pixels and withholds the text entirely. There is nothing to select, and
 *   nothing the client could strip.
 *
 * Neither stops a photograph of the screen, and the UI should never suggest
 * otherwise.
 */
export type ProtectionLevel = 'normal' | 'strict'

/** What a reader did with a document (FR-4). */
export type AccessAction = 'view' | 'download' | 'print' | 'export_pdf'

export interface AccessLogEntry {
  id: string
  document_id: string
  document: string
  user_id: string | null
  /** Empty for a reader who came through a share link, or a deleted account. */
  user: string
  user_email: string
  action: AccessAction
  ip_address: string | null
  user_agent: string | null
  created_at: string
}

export interface AccessLogFilters {
  action?: string
  from?: string
  to?: string
}

/** A live share link (FR-5). Never carries the token — see below. */
export interface ViewSession {
  id: string
  document_id: string
  label: string | null
  created_by: string
  creator: string
  expires_at: string
  created_at: string
}

/**
 * The create response, and the only place the token ever appears: the API
 * stores just its hash, so a link that is not copied now cannot be recovered.
 */
export interface CreatedViewSession extends ViewSession {
  token: string
}

/** The strict viewer's payload: one data URL per page, in reading order. */
export interface DocumentPages {
  count: number
  pages: string[]
}

/** What a share-link recipient gets — one document, and nothing around it. */
export interface SharedDocument {
  title: string
  content: Record<string, unknown>
  watermark: string
  protection_level: ProtectionLevel
  expires_at: string
  pages?: string[]
}
