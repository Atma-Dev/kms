import { alertRootVariants } from '~/base/ui/styles/alert.styles'
import type { VariantProps } from 'class-variance-authority'
import type { Icon } from '~/base/ui/lucide'
import type { ResponseData, ResponseError } from '~/types/response'

export const useFlashStore = defineStore('flashStore', () => {
  const title = ref<string | null>(null)
  const failedMessage = ref<string | null>(null)
  const successMessage = ref<string | null>(null)
  const variant = ref<VariantProps<typeof alertRootVariants>['variant'] | null>(null)
  const icon = ref<Icon | null>(null)

  const errors = ref<Record<string, string[]> | null>(null)
  const setSuccess = (responseData: ResponseData) => {
    clearMessages()
    title.value = responseData.code
    successMessage.value = responseData.message
    variant.value = 'success'
    icon.value = 'CheckCircle'
  }

  const setFailed = (responseError: ResponseError) => {
    clearMessages()
    title.value = responseError.code
    failedMessage.value = responseError.message
    if (responseError.errors) {
      errors.value = responseError.errors
    }
    variant.value = 'danger'
    icon.value = 'OctagonAlertIcon'
  }

  const clearMessages = () => {
    failedMessage.value = null
    successMessage.value = null
    variant.value = null
  }

  const isKeyErrors = (key: string):boolean=>{
    return errors.value ? key in errors.value : false
  }

  const getErrorsByKey = (key: string):string[]=>{
    return errors.value?.[key] ?? []
  }
  return {
    icon,
    variant,
    failedMessage,
    successMessage,
    title,
    errors,
    setSuccess,
    setFailed,
    clearMessages,
    isKeyErrors,
    getErrorsByKey,
  }
})
