import type { DocumentAction, DocumentStatus } from '~/types/entities/document'

/**
 * Status → badge colour. Kept here rather than in each page so the same status
 * never reads as two different things on two screens.
 */
export const statusVariant = (status: DocumentStatus): string =>
  ({
    draft: 'secondary',
    in_review: 'warning',
    published: 'success',
    archived: 'danger',
  })[status] ?? 'secondary'

/**
 * How each lifecycle button is drawn. The list of *which* buttons to show
 * comes from the API (`available_actions`) — this only says what they look
 * like, so the rules stay in one place.
 */
export const actionStyle: Record<DocumentAction, { icon: string; variant: string }> = {
  'submit-review': { icon: 'Send', variant: 'primary' },
  approve: { icon: 'Check', variant: 'success' },
  reject: { icon: 'X', variant: 'danger' },
  publish: { icon: 'Upload', variant: 'success' },
  unpublish: { icon: 'Download', variant: 'warning' },
  archive: { icon: 'Archive', variant: 'danger' },
  restore: { icon: 'RotateCcw', variant: 'primary' },
}

/** Only reject requires a reason — see FR-4 and document.Service.Transition. */
export const actionNeedsReason = (action: DocumentAction): boolean => action === 'reject'
