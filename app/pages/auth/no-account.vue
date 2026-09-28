<script lang="ts" setup>
import { Lucide } from '~/base/ui/lucide'

definePageMeta({
  layout: false,
  public: true,
})
</script>

<template>
  <!-- The provider (Google) verified who this is, but no local
       user row matches that email at all — sso never pushed a membership for
       it, not "sso revoked it" (see membership-inactive.vue for that case).
       Reached from both the popup login flow (pages/auth/index.vue) and the
       full-page redirect fallback (pages/auth/callback.vue). -->
  <div class="flex h-screen items-center justify-center">
    <div class="flex flex-col items-center gap-4 text-center px-6 max-w-sm">
      <div class="flex h-14 w-14 items-center justify-center rounded-full bg-warning/10">
        <Lucide icon="UserX" class="h-7 w-7 text-warning" />
      </div>
      <h1 class="text-base font-semibold">{{ $t('auth.noAccountTitle') }}</h1>
      <p class="text-sm text-foreground/70">{{ $t('auth.errorNoAccount') }}</p>
      <a
        href="/auth"
        class="mt-2 rounded-md border border-foreground/15 px-5 py-2.5 text-sm font-medium text-foreground/80 transition-colors hover:bg-foreground/5"
      >
        {{ $t('auth.backToSignIn') }}
      </a>
    </div>
  </div>
</template>
