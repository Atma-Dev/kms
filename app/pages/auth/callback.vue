<script lang="ts" setup>
import { authErrorMessageKey } from '~/utils/auth'
import { OAUTH_CALLBACK_MESSAGE_TYPE } from '~/utils/oauthPopup'
import { useAuthService } from '~/services/AuthService'

definePageMeta({
  layout: false,
  public: true,
})

const { t } = useI18n()

const route = useRoute()
const { finishLogin } = useAuthService()

const errorMessage = ref<string | null>(null)

onMounted(async () => {
  const accessToken = route.query.access_token as string | undefined
  const error = route.query.error as string | undefined

  // Opened as a popup by /auth (see openOAuthPopup) — hand the raw result
  // back to the window that opened us instead of finishing the login here,
  // since the store/composable state that matters lives there.
  if (window.opener && window.opener !== window) {
    window.opener.postMessage(
      { type: OAUTH_CALLBACK_MESSAGE_TYPE, access_token: accessToken, error },
      window.location.origin,
    )
    window.close()
    return
  }

  if (error === 'membership_inactive') {
    navigateTo('/auth/membership-inactive')
    return
  }

  if (error === 'no_account') {
    navigateTo('/auth/no-account')
    return
  }

  if (error) {
    errorMessage.value = t(authErrorMessageKey(error))
    return
  }

  if (!accessToken) {
    navigateTo('/auth')
    return
  }

  try {
    await finishLogin(accessToken)
    navigateTo('/home')
  } catch (e) {
    errorMessage.value = t('auth.errorVerifyFailed')
  }
})
</script>

<template>
  <div class="flex h-screen items-center justify-center">
    <div v-if="errorMessage" class="flex flex-col items-center gap-4 text-center px-6 max-w-sm">
      <div class="flex h-14 w-14 items-center justify-center rounded-full bg-danger/10">
        <svg class="h-7 w-7 text-danger" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
      </div>
      <p class="text-sm text-foreground/70">{{ errorMessage }}</p>
      <a
        href="/auth"
        class="mt-2 rounded-md bg-primary px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-primary/90"
      >
        {{ $t('auth.callbackRetry') }}
      </a>
    </div>

    <div v-else class="flex flex-col items-center gap-3 text-center opacity-70">
      <svg class="h-6 w-6 animate-spin text-primary" viewBox="0 0 24 24" fill="none">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
      </svg>
      <span class="text-sm">{{ $t('auth.signingIn') }}</span>
    </div>
  </div>
</template>
