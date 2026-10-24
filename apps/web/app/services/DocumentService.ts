import { useFlashStore } from '~/stores/flash'
import type Document from '~/types/entities/document'
import type Tag from '~/types/entities/tag'
import type { AccessLevel, AccessSubjectType, DocumentAccess } from '~/types/entities/attachment'
import type {
  DiffLine,
  DocumentAction,
  DocumentContent,
  DocumentDetail,
  DocumentVersion,
} from '~/types/entities/document'
import type {
  ResponseData,
  ResponseDataCollectionWithoutPagination,
  ResponseSingleData,
  ServerPaginationMeta,
} from '~/types/response'

interface DocumentListResponse extends ResponseData {
  payload: {
    data: Document[]
    meta: ServerPaginationMeta
  }
}

export interface DocumentRequest {
  title: string
  category_id: string | null
  content: DocumentContent
  change_note?: string | null
  is_pinned?: boolean
}

export interface DocumentFilters {
  status?: string
  category_id?: string
  search?: string
  sort?: string
  direction?: string
}

const buildQuery = (page: number, perPage: number, filters: DocumentFilters) => ({
  page,
  per_page: perPage,
  ...(filters.status ? { status: filters.status } : {}),
  ...(filters.category_id ? { category_id: filters.category_id } : {}),
  ...(filters.search ? { search: filters.search } : {}),
  ...(filters.sort ? { sort: filters.sort } : {}),
  ...(filters.direction ? { direction: filters.direction } : {}),
})

const defaultMeta: ServerPaginationMeta = {
  current_page: 1,
  per_page: 10,
  total: 0,
  last_page: 1,
  from: 0,
  to: 0,
}

const failure = (message: string) => ({ code: 'ERROR', message, status_code: 500, timestamp: '' })

export const useDocumentService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const documents = ref<Document[]>([])
  const document = ref<DocumentDetail | null>(null)
  const versions = ref<DocumentVersion[]>([])
  const meta = ref<ServerPaginationMeta>({ ...defaultMeta })
  const loading = ref(false)
  const submitting = ref(false)

  const getAll = async (spaceId: string, page: number, perPage: number, filters: DocumentFilters = {}) => {
    loading.value = true
    try {
      const response = await $api<DocumentListResponse>(`api/spaces/${spaceId}/documents`, {
        method: 'GET',
        query: buildQuery(page, perPage, filters),
      })
      documents.value = response.payload.data
      meta.value = response.payload.meta
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load documents'))
    } finally {
      loading.value = false
    }
  }

  const getById = async (id: string) => {
    loading.value = true
    try {
      const response = await $api<ResponseSingleData<DocumentDetail>>(`api/documents/${id}`, { method: 'GET' })
      document.value = response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load document'))
    } finally {
      loading.value = false
    }
  }

  /**
   * The documents a reader of this one is most likely to want next (FR-4).
   *
   * Returned rather than stored on the service: the panel that shows them is
   * the only thing that wants them, and putting them in shared state would
   * outlive the document they belong to.
   *
   * A failure is swallowed — a sidebar of suggestions is not worth an error
   * banner over the document the reader actually came for.
   */
  const getRelated = async (id: string): Promise<Document[]> => {
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<Document>>(
        `api/documents/${id}/related`,
        { method: 'GET' },
      )
      return response.payload.data
    } catch {
      return []
    }
  }

  const addNew = async (spaceId: string, request: DocumentRequest): Promise<DocumentDetail | null> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<DocumentDetail>>(`api/spaces/${spaceId}/documents`, {
        method: 'POST',
        body: request,
      })
      flashStore.setSuccess(response)
      return response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to create document'))
      return null
    } finally {
      submitting.value = false
    }
  }

  // Every write that returns a document replaces `document` from the response
  // rather than patching the local copy: status, available_actions and the
  // version pointer all move together, and rebuilding them here would be a
  // second implementation of the lifecycle rules.
  const updateById = async (id: string, request: DocumentRequest): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<DocumentDetail>>(`api/documents/${id}`, {
        method: 'PUT',
        body: request,
      })
      flashStore.setSuccess(response)
      document.value = response.payload.data
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to update document'))
      return false
    } finally {
      submitting.value = false
    }
  }

  /**
   * Pins or unpins. A dedicated call rather than a PUT: sending the whole
   * document to flip one boolean would carry this client's copy of the content
   * with it and overwrite anything edited in between.
   */
  const setPinned = async (id: string, pinned: boolean): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<DocumentDetail>>(`api/documents/${id}/pin`, {
        method: 'PATCH',
        body: { is_pinned: pinned },
      })
      flashStore.setSuccess(response)
      document.value = response.payload.data
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to pin document'))
      return false
    } finally {
      submitting.value = false
    }
  }

  /** The bin for one space. Gated on documents.destroy, not on reading. */
  const getDeleted = async (spaceId: string, page: number, perPage: number) => {
    loading.value = true
    try {
      const response = await $api<DocumentListResponse>(`api/spaces/${spaceId}/documents/deleted`, {
        method: 'GET',
        query: { page, per_page: perPage },
      })
      documents.value = response.payload.data
      meta.value = response.payload.meta
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load deleted documents'))
    } finally {
      loading.value = false
    }
  }

  /** Comes back as a draft, whatever it was when deleted. */
  const undelete = async (id: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<DocumentDetail>>(`api/documents/${id}/undelete`, {
        method: 'POST',
      })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to restore document'))
      return false
    } finally {
      submitting.value = false
    }
  }

  /** One call for every lifecycle move — the API has a route per action. */
  const transition = async (id: string, action: DocumentAction, reason?: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<DocumentDetail>>(`api/documents/${id}/${action}`, {
        method: 'POST',
        body: { reason: reason ?? null },
      })
      flashStore.setSuccess(response)
      document.value = response.payload.data
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure(`Failed to ${action} document`))
      return false
    } finally {
      submitting.value = false
    }
  }

  const deleteById = async (id: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseData>(`api/documents/${id}`, { method: 'DELETE' })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to delete document'))
      return false
    } finally {
      submitting.value = false
    }
  }

  const getVersions = async (id: string) => {
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<DocumentVersion>>(
        `api/documents/${id}/versions`,
        { method: 'GET' },
      )
      versions.value = response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load versions'))
    }
  }

  /**
   * Either side may be the literal `current`: snapshots are taken *before* an
   * overwrite, so the live content is not itself a version row — "what changed
   * since v3" is otherwise unanswerable.
   */
  const compare = async (id: string, from: string, to: string): Promise<DiffLine[]> => {
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<DiffLine>>(
        `api/documents/${id}/versions/compare`,
        { method: 'GET', query: { from, to } },
      )
      return response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to compare versions'))
      return []
    }
  }

  const rollback = async (id: string, versionNumber: number, changeNote?: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<DocumentDetail>>(
        `api/documents/${id}/versions/${versionNumber}/rollback`,
        { method: 'POST', body: { change_note: changeNote ?? null } },
      )
      flashStore.setSuccess(response)
      document.value = response.payload.data
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to roll back document'))
      return false
    } finally {
      submitting.value = false
    }
  }

  /**
   * Replaces the whole tag set by name. A name nobody has used becomes a tag —
   * the vocabulary is open (FR-6) — so the answer can differ from what was
   * sent whenever a name resolved to an existing tag with different casing.
   * The document is patched from it rather than from the request.
   */
  const syncTags = async (id: string, names: string[]): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<Tag>>(`api/documents/${id}/tags`, {
        method: 'POST',
        body: { names },
      })
      flashStore.setSuccess(response)
      if (document.value) document.value.tags = response.payload.data
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to update tags'))
      return false
    } finally {
      submitting.value = false
    }
  }

  const getAccess = async (id: string): Promise<DocumentAccess[]> => {
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<DocumentAccess>>(
        `api/documents/${id}/access`,
        { method: 'GET' },
      )
      return response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load access'))
      return []
    }
  }

  /** The first grant restricts the document; it never widens one (FR-9). */
  const grantAccess = async (
    id: string,
    subjectType: AccessSubjectType,
    subjectId: string,
    permission: AccessLevel,
  ): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<DocumentAccess>>(`api/documents/${id}/access`, {
        method: 'POST',
        body: { subject_type: subjectType, subject_id: subjectId, permission },
      })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to grant access'))
      return false
    } finally {
      submitting.value = false
    }
  }

  const revokeAccess = async (id: string, accessId: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseData>(`api/documents/${id}/access/${accessId}`, { method: 'DELETE' })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to revoke access'))
      return false
    } finally {
      submitting.value = false
    }
  }

  return {
    documents, document, versions, meta, loading, submitting,
    syncTags, getAccess, grantAccess, revokeAccess,
    setPinned, getDeleted, undelete,
    getAll, getById, getRelated, addNew, updateById, transition, deleteById,
    getVersions, compare, rollback,
  }
}
