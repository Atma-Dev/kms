import { useFlashStore } from '~/stores/flash'
import type Attachment from '~/types/entities/attachment'
import type {
  ResponseData,
  ResponseDataCollectionWithoutPagination,
  ResponseSingleData,
} from '~/types/response'

const failure = (message: string) => ({ code: 'ERROR', message, status_code: 500, timestamp: '' })

export const useAttachmentService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const attachments = ref<Attachment[]>([])
  const loading = ref(false)
  const uploading = ref(false)

  const getByDocument = async (documentId: string) => {
    loading.value = true
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<Attachment>>(
        `api/documents/${documentId}/attachments`,
        { method: 'GET' },
      )
      attachments.value = response.payload.data
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to load attachments'))
    } finally {
      loading.value = false
    }
  }

  // FormData, never a JSON body: the file is bytes, and Content-Type is set by
  // the browser so the multipart boundary matches what it actually wrote.
  const upload = async (documentId: string, file: File): Promise<boolean> => {
    uploading.value = true
    try {
      const body = new FormData()
      body.append('file', file)
      const response = await $api<ResponseSingleData<Attachment>>(
        `api/documents/${documentId}/attachments`,
        { method: 'POST', body },
      )
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to upload file'))
      return false
    } finally {
      uploading.value = false
    }
  }

  /**
   * Uploads an image for the editor to embed (FR-8) and returns its URL.
   *
   * Filed against the space rather than the document: the editor needs the URL
   * while the document is still being written.
   */
  const uploadInlineImage = async (spaceId: string, file: File): Promise<string | null> => {
    try {
      const body = new FormData()
      body.append('file', file)
      body.append('space_id', spaceId)
      const response = await $api<ResponseSingleData<{ url: string }>>('api/uploads/inline-image', {
        method: 'POST',
        body,
      })
      return response.payload.data.url
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to upload image'))
      return null
    }
  }

  const deleteById = async (id: string): Promise<boolean> => {
    try {
      const response = await $api<ResponseData>(`api/attachments/${id}`, { method: 'DELETE' })
      flashStore.setSuccess(response)
      return true
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to delete file'))
      return false
    }
  }

  return { attachments, loading, uploading, getByDocument, upload, uploadInlineImage, deleteById }
}
