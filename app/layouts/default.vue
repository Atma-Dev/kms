<template>
  <ThemesLayout>
    <slot />
  </ThemesLayout>

  <!-- Mounted once, here, because this layout is the only one that opens the
       notification socket. Teleports to body, so it sits bottom-right over
       whatever page is showing (placement is set in base/ui/toast/toaster). -->
  <ToasterContainer :toaster="toaster" v-slot="{ toast }">
    <ToastRoot :key="toast.id">
      <div class="cursor-pointer" @click="open(toast.meta?.href)">
        <ToastTitle>{{ toast.title }}</ToastTitle>
        <ToastDescription>{{ toast.description }}</ToastDescription>
      </div>
      <ToastCloseTrigger />
    </ToastRoot>
  </ToasterContainer>
</template>

<script setup lang="ts">
import ThemesLayout from '~/themes'
import {
  toaster,
  ToasterContainer,
  ToastRoot,
  ToastTitle,
  ToastDescription,
  ToastCloseTrigger,
} from '~/base/ui/toast'

// The user cached in localStorage survives reloads, so it can go stale when an
// admin changes a role or deactivates the account. Refresh it once per app load
// — this layout only mounts for authenticated pages (`/auth` sets layout: false).
const { fetchMe } = useAuthStore()
const { fetch: fetchNotifications } = useNotificationStore()

const open = (href?: string) => {
  if (href) navigateTo(href)
}

onMounted(() => {
  fetchMe()
  fetchNotifications()
  onUnmounted(useNotificationSocket())
})
</script>
