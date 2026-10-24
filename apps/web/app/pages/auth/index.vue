<script lang="ts" setup>
import { Lucide } from '~/base/ui/lucide'
import { Input } from '~/base/ui/input'
import logoUrl from '~/assets/images/logo.svg'
import { openOAuthPopup } from '~/utils/oauthPopup'
import { authErrorMessageKey } from '~/utils/auth'
import { useAuthService } from '~/services/AuthService'
import { isFetchResponseError } from '~/utils/helper'
import type { ResponseError } from '~/types/response'

definePageMeta({
  layout: false,
  public: true,
})

const { t } = useI18n()

const features = computed(() => [
  {
    icon: 'FolderKanban' as const,
    title: t('auth.featureAccountsTitle'),
    description: t('auth.featureAccountsDescription'),
  },
  {
    icon: 'Workflow' as const,
    title: t('auth.featureOrdersTitle'),
    description: t('auth.featureOrdersDescription'),
  },
  {
    icon: 'ShieldCheck' as const,
    title: t('auth.featureAccessTitle'),
    description: t('auth.featureAccessDescription'),
  },
])

const config = useRuntimeConfig()
const { login, finishLogin } = useAuthService()

// One error line for both paths: the password form and the Google popup can
// only fail one at a time, and neither has more than one thing to say.
const loginError = ref('')

const credentials = reactive({ email: '', password: '' })
const showPassword = ref(false)
const signingIn = ref(false)

// The API answers a wrong email and a wrong password identically (it will not
// tell a caller which addresses exist), so there is one message to show and
// no field to attach it to.
const loginWithPassword = async () => {
  if (signingIn.value) return
  signingIn.value = true
  loginError.value = ''

  try {
    await login({ ...credentials })
    await navigateTo('/home')
  } catch (e) {
    loginError.value = isFetchResponseError(e) ? (e._data as ResponseError).message : t('auth.errorUnexpected')
  } finally {
    signingIn.value = false
  }
}

// Opens Google's redirect in a popup so the tab never leaves the app —
// /auth/callback posts the result back once it lands inside the popup (see
// openOAuthPopup / callback.vue). Falls back to the old full-page redirect
// only when a popup could never be opened (blocked), not when the user
// closes it deliberately.
const signingInWithGoogle = ref(false)

const loginWithGoogle = async () => {
  loginError.value = ''
  const redirectUrl = `${config.public.apiBase}/api/auth/google/redirect`

  signingInWithGoogle.value = true
  const result = await openOAuthPopup(redirectUrl)

  if (result.status === 'blocked') {
    signingInWithGoogle.value = false
    navigateTo(redirectUrl, { external: true })
    return
  }
  if (result.status === 'cancelled') {
    signingInWithGoogle.value = false
    return
  }

  if (result.error) {
    signingInWithGoogle.value = false
    if (result.error === 'membership_inactive') {
      navigateTo('/auth/membership-inactive')
      return
    }
    if (result.error === 'no_account') {
      navigateTo('/auth/no-account')
      return
    }
    loginError.value = t(authErrorMessageKey(result.error))
    return
  }
  if (!result.access_token) {
    signingInWithGoogle.value = false
    return
  }

  try {
    await finishLogin(result.access_token)
    navigateTo('/home')
  } catch {
    loginError.value = t('auth.errorVerifyFailed')
  } finally {
    signingInWithGoogle.value = false
  }
}

const busy = computed(() => signingIn.value || signingInWithGoogle.value)
</script>

<template>
  <div class="flex min-h-screen">
    <div class="fixed right-4 top-4 z-20 lg:right-6 lg:top-6">
      <LanguageSwitcher />
    </div>

    <!-- BEGIN: Left brand panel -->
    <div
      class="relative hidden w-1/2 flex-col justify-between overflow-hidden bg-[#0a0a14] p-12 text-white lg:flex"
      style="background-image: radial-gradient(circle, rgb(255 255 255 / 0.06) 1px, transparent 1px); background-size: 22px 22px;"
    >
      <div class="pointer-events-none absolute -right-32 top-0 h-[32rem] w-[32rem] rounded-full bg-primary/50 blur-[120px]"></div>
      <div class="pointer-events-none absolute -left-32 bottom-0 h-[28rem] w-[28rem] rounded-full bg-accent/30 blur-[120px]"></div>

      <a href="" class="relative flex items-center gap-3">
        <div class="flex h-9 w-9 items-center justify-center rounded-lg bg-white/10">
          <img class="h-5 w-5" :src="logoUrl" alt="Atma KMS" />
        </div>
        <div class="leading-tight">
          <div class="text-sm font-semibold">Atma KMS</div>
          <div class="text-[10px] uppercase tracking-widest text-white/40">{{ $t('auth.platform') }}</div>
        </div>
      </a>

      <div class="relative">
        <h2 class="text-4xl font-semibold leading-tight">
          {{ $t('auth.heroTitle') }}
        </h2>
        <p class="mt-3 max-w-sm text-white/50">
          {{ $t('auth.heroSubtitle') }}
        </p>

        <div class="mt-10 flex flex-col gap-4">
          <div v-for="feature in features" :key="feature.title" class="flex items-start gap-3">
            <div class="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-white/10">
              <Lucide :icon="feature.icon" class="size-4" />
            </div>
            <div>
              <div class="text-sm font-medium">{{ feature.title }}</div>
              <div class="text-sm text-white/40">{{ feature.description }}</div>
            </div>
          </div>
        </div>
      </div>

      <p class="relative text-xs text-white/30">© {{ new Date().getFullYear() }} Atma</p>
    </div>
    <!-- END: Left brand panel -->

    <!-- BEGIN: Right form panel -->
    <div class="relative flex w-full flex-col items-center justify-center overflow-hidden bg-background px-6 py-16 lg:w-1/2">
      <!-- Softens the hard edge against the dark brand panel: starts at that
           panel's own near-black tone (bridges the black→white jump), fades
           through the same primary glow as the two circles over there. -->
      <div class="pointer-events-none absolute inset-y-0 left-0 hidden w-72 bg-gradient-to-r from-[#0a0a14]/[.12] via-primary/[.08] to-transparent lg:block"></div>

      <div class="w-full max-w-sm">
        <a href="" class="mb-10 flex items-center gap-2 lg:hidden">
          <img class="h-6 w-6" :src="logoUrl" alt="Atma KMS" />
          <span class="text-sm font-medium">
            Atma <span class="font-light opacity-60">KMS</span>
          </span>
        </a>

        <h1 class="text-2xl font-semibold">{{ $t('auth.signIn') }}</h1>
        <p class="mt-1 text-sm opacity-50">{{ $t('auth.signInSubtitle') }}</p>

        <form class="mt-8 flex flex-col gap-4" @submit.prevent="loginWithPassword">
          <div class="grid gap-1.5">
            <label class="text-sm opacity-60" for="email">{{ $t('auth.email') }} <RequiredMark /></label>
            <Input id="email" v-model="credentials.email" type="email" autocomplete="email" required :placeholder="$t('auth.emailPlaceholder')" />
          </div>

          <div class="grid gap-1.5">
            <label class="text-sm opacity-60" for="password">{{ $t('auth.password') }} <RequiredMark /></label>
            <div class="relative">
              <Input
                id="password"
                v-model="credentials.password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                required
                class="pr-10"
                :placeholder="$t('auth.passwordPlaceholder')"
              />
              <button
                type="button"
                class="absolute inset-y-0 right-0 flex w-10 items-center justify-center opacity-50 hover:opacity-80"
                :aria-label="$t(showPassword ? 'auth.hidePassword' : 'auth.showPassword')"
                @click="showPassword = !showPassword"
              >
                <Lucide :icon="showPassword ? 'EyeOff' : 'Eye'" class="size-4" />
              </button>
            </div>
          </div>

          <button
            type="submit"
            class="flex w-full items-center justify-center rounded-lg bg-primary px-4 py-3 text-sm font-medium text-white transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-60"
            :disabled="busy"
          >
            <Lucide v-if="signingIn" icon="LoaderCircle" class="mr-2 size-4 animate-spin" />
            {{ $t('auth.signIn') }}
          </button>

          <div class="relative flex items-center justify-center py-1">
            <div class="absolute inset-0 flex items-center">
              <div class="w-full border-t border-foreground/10"></div>
            </div>
            <span class="relative bg-background px-3 text-xs uppercase tracking-wide opacity-40">{{ $t('auth.or') }}</span>
          </div>

          <button
            type="button"
            class="flex w-full items-center justify-center gap-2 rounded-lg border border-foreground/15 bg-background px-4 py-3 text-sm font-medium text-foreground transition-colors hover:bg-foreground/[.03] disabled:cursor-not-allowed disabled:opacity-60"
            :disabled="busy"
            @click="loginWithGoogle"
          >
            <Lucide v-if="signingInWithGoogle" icon="LoaderCircle" class="size-4 animate-spin" />
            <svg v-else class="h-4 w-4 shrink-0" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
              <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4" />
              <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853" />
              <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05" />
              <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335" />
            </svg>
            {{ $t('auth.signInWithGoogle') }}
          </button>

          <p v-if="loginError" class="text-sm text-danger">{{ loginError }}</p>
        </form>

        <p class="mt-10 text-xs opacity-40 lg:hidden">{{ $t('auth.copyright', { year: new Date().getFullYear() }) }}</p>
      </div>
    </div>
    <!-- END: Right form panel -->

  </div>
</template>
