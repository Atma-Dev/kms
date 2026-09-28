import { useFlashStore } from '~/stores/flash'
import type { ResponseSingleData } from '~/types/response'

export const useProfileService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const submitting = ref(false)

  /**
   * Changes the caller's own password. The current one goes with it: the API
   * re-checks it, so a session left open on someone else's screen cannot be
   * used to lock its owner out.
   *
   * Returns false on failure — the api plugin already flashed the reason
   * (a wrong current password comes back as a 401 with its own message).
   */
  const updatePassword = async (request: {
    current_password: string
    password: string
    password_confirmation: string
  }): Promise<boolean> => {
    submitting.value = true
    try {
      const response = await $api<ResponseSingleData<null>>('api/profiles/password', {
        method: 'PATCH',
        body: request,
      })
      flashStore.setSuccess(response)
      return true
    } catch {
      return false
    } finally {
      submitting.value = false
    }
  }

  return { submitting, updatePassword }
}