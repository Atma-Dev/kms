export default interface Attachment {
  id: string
  file_name: string
  file_type: string
  file_size: number
  /**
   * The only way to reach the bytes. The store is not a static folder — every
   * read goes through an endpoint that checks the document first — so there is
   * no path to build a URL from, and none is sent.
   */
  url: string
  uploaded_by: string
  created_at: string
}

export type AccessSubjectType = 'user' | 'role'
export type AccessLevel = 'view' | 'edit' | 'manage'

export interface DocumentAccess {
  id: string
  subject_type: AccessSubjectType
  subject_id: string
  permission: AccessLevel
  created_at: string
}
