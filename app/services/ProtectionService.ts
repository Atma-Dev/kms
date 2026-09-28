import { useFlashStore } from '~/stores/flash'
import type {
  AccessLogEntry,
  AccessLogFilters,
  CreatedViewSession,
  DocumentPages,
  ProtectionLevel,
  SharedDocument,
  ViewSession,
} from '~/types/entities/protection'
import type { DocumentDetail } from '~/types/entities/document'
import type {
  ResponseData,
  ResponseDataCollectionWithoutPagination,
  ResponseSingleData,
  ServerPaginationMeta,
} from '~/types/response'

interface AccessLogResponse extends ResponseData {
  payload: {
    data: AccessLogEntry[]
    meta: ServerPaginationMeta
  }
}

const defaultMeta: ServerPaginationMeta = {
  current_page: 1,
  per_page: 10,
  total: 0,
  last_page: 1,
  from: 0,
  to: 0,
}

const failure = (message: string) => ({ code: 'ERROR', message, status_code: 500, timestamp: '' })

/**
 * Modul 4: the watermarked renders, the access trail, the protection level and
 * the share links.
 *
 * A service of its own rather than more of DocumentService, for the same
 * reason the API put them in app/protection: these are about what happens to a
 * document once someone can read it, and the screens that use them (an audit
 * table, a share dialog) are not the screens that edit documents.
 */
export const useProtectionService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const logs = ref<AccessLogEntry[]>([])
  const meta = ref<ServerPaginationMeta>({ ...defaultMeta })
  const sessions = ref<ViewSession[]>([])
  const loading = ref(false)
  const submitting = ref(false)

  /**
   * Downloads the PDF (FR-2).
   *
   * Fetched as a blob through `$api` rather than pointed at with an `<a href>`:
   * the export needs the Authorization header, which a browser-issued
   * navigation does not send — the same problem app/attachment solved with a
   * `?token=` escape hatch. A one-shot object URL avoids adding a second.
   */
  const exportPdf = async (id: string, filename: string): Promise<boolean> => {
    submitting.value = true
    try {
      const blob = await $api<Blob>(`api/documents/${id}/export-pdf`, {
        method: 'GET',
        responseType: 'blob',
      })

      const url = URL.createObjectURL(blob)
      const link = window.document.createElement('a')
      link.href = url
      link.download = `${filename}.pdf`
      link.click()
      // Revoked immediately: the blob is the reader's own watermarked copy,
      // and leaving the URL alive leaves it readable to anything that can run
      // script on this page for as long as the tab is open.
      URL.revokeObjectURL(url)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to export the document'))
      return false
    } finally {
      submitting.value = false
    }
  }

  /** The strict viewer's page images (FR-3), watermarked server-side. */
  const getPages = async (id: string): Promise<string[]> => {
    loading.value = true
    try {
      const response = await $api<ResponseSingleData<DocumentPages>>(`api/documents/${id}/pages`, {
        method: 'GET',
      })
      return response.payload.data.pages
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to render the document'))
      return []
    } finally {
      loading.value = false
    }
  }

  /**
   * Sets a document's level (FR-6). `null` clears the override and puts the
   * document back on its space's default — inheriting is the absence of a
   * setting, which is why this takes a nullable rather than a third value.
   */
  const setDocumentLevel = async (
    id: string,
    level: ProtectionLevel | null,
  ): Promise<DocumentDetail | null> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<DocumentDetail>>(
        `api/documents/${id}/protection-setting`,
        { method: 'PUT', body: { protection_level: level } },
      )
      flashStore.setSuccess(response)
      return response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to set the protection level'))
      return null
    } finally {
      submitting.value = false
    }
  }

  /**
   * Sets a space's default (FR-6).
   *
   * Not nullable, unlike the per-document call: a space has nothing to inherit
   * from, so it always holds a value. Its own endpoint rather than a field on
   * the space form, because PUT /spaces/{id} is a full replace and sending the
   * whole space to change one enum would carry this client's copy of the name
   * and description with it.
   */
  const setSpaceLevel = async (spaceId: string, level: ProtectionLevel): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseData>(`api/spaces/${spaceId}/protection-setting`, {
        method: 'PUT',
        body: { protection_level: level },
      })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to set the protection level'))
      return false
    } finally {
      submitting.value = false
    }
  }

  const getAccessLog = async (
    id: string,
    page: number,
    perPage: number,
    filters: AccessLogFilters = {},
  ) => {
    loading.value = true
    try {
      const response = await $api<AccessLogResponse>(`api/documents/${id}/access-log`, {
        method: 'GET',
        query: {
          page,
          per_page: perPage,
          ...(filters.action ? { action: filters.action } : {}),
          ...(filters.from ? { from: filters.from } : {}),
          ...(filters.to ? { to: filters.to } : {}),
        },
      })
      logs.value = response.payload.data
      meta.value = response.payload.meta
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load the access log'))
    } finally {
      loading.value = false
    }
  }

  /** The same trail read by person instead of by document. */
  const getUserAccessLog = async (
    userId: string,
    page: number,
    perPage: number,
    filters: AccessLogFilters = {},
  ) => {
    loading.value = true
    try {
      const response = await $api<AccessLogResponse>(`api/management/users/${userId}/access-log`, {
        method: 'GET',
        query: {
          page,
          per_page: perPage,
          ...(filters.action ? { action: filters.action } : {}),
          ...(filters.from ? { from: filters.from } : {}),
          ...(filters.to ? { to: filters.to } : {}),
        },
      })
      logs.value = response.payload.data
      meta.value = response.payload.meta
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load the access log'))
    } finally {
      loading.value = false
    }
  }

  /**
   * Reports a print (FR-4).
   *
   * The only access the client tells the server about, because a browser print
   * never reaches the API at all. Failures are swallowed: a reader who is
   * printing should not get an error banner because a log row did not save,
   * and the row is a hint rather than evidence either way.
   */
  const reportPrint = async (id: string): Promise<void> => {
    try {
      await $api<ResponseData>(`api/documents/${id}/access-log`, {
        method: 'POST',
        body: { action: 'print' },
      })
    } catch {
      // Deliberately silent — see above.
    }
  }

  const getViewSessions = async (id: string) => {
    loading.value = true
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<ViewSession>>(
        `api/documents/${id}/view-sessions`,
        { method: 'GET' },
      )
      sessions.value = response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load the share links'))
    } finally {
      loading.value = false
    }
  }

  /**
   * Creates a share link (FR-5). The token in the response is the only copy
   * there will ever be — the API stores its hash — so whatever calls this has
   * to show it before it is thrown away.
   */
  const createViewSession = async (
    id: string,
    expiresInHours: number,
    label: string | null,
  ): Promise<CreatedViewSession | null> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<CreatedViewSession>>(
        `api/documents/${id}/view-sessions`,
        { method: 'POST', body: { expires_in_hours: expiresInHours, label } },
      )
      flashStore.setSuccess(response)
      return response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to create the share link'))
      return null
    } finally {
      submitting.value = false
    }
  }

  const revokeViewSession = async (id: string, sessionId: string): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseData>(`api/documents/${id}/view-sessions/${sessionId}`, {
        method: 'DELETE',
      })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to revoke the share link'))
      return false
    } finally {
      submitting.value = false
    }
  }

  /**
   * Opens a shared document by token. The one call in the app that needs no
   * account — and the page using it is the one route marked `public`.
   */
  const openShared = async (token: string): Promise<SharedDocument | null> => {
    loading.value = true
    try {
      const response = await $api<ResponseSingleData<SharedDocument>>(`api/view-session/${token}`, {
        method: 'GET',
      })
      return response.payload.data
    } catch (e: any) {
      // No flash: the recipient has no app around them to show it in. The page
      // renders the failure itself.
      return null
    } finally {
      loading.value = false
    }
  }

  return {
    logs, meta, sessions, loading, submitting,
    exportPdf, getPages, setDocumentLevel, setSpaceLevel,
    getAccessLog, getUserAccessLog, reportPrint,
    getViewSessions, createViewSession, revokeViewSession, openShared,
  }
}
