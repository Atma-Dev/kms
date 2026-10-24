<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Lucide } from '~/base/ui/lucide'
import { useAuthService } from '~/services/AuthService'
import ModalChangePassword from '~/components/features/account/ModalChangePassword.vue'

interface Props {
  class?: string
  boxClass?: string
}

const props = defineProps<Props>()
const { logout } = useAuthService()
const { user, fullName } = storeToRefs(useAuthStore())

// The modal teleports to <body>, so it is unaffected by this dropdown being
// hidden again (CSS opacity/visibility, not v-if) the moment the pointer leaves.
const changePasswordModal = useTemplateRef('changePasswordModal')
</script>

<template>
  <div
    v-bind="$attrs"
    :class="[
      'invisible opacity-0 scale-95 transition-all duration-200 delay-0 group-hover/profile:visible group-hover/profile:opacity-100 group-hover/profile:scale-100 group-hover/profile:delay-200',
      props.class,
    ]"
  >
    <Box
      :class="`text-foreground before:shadow-foreground/5 flex w-64 flex-col gap-2.5 px-6 py-5 before:rounded-2xl before:shadow-xl before:backdrop-blur after:rounded-2xl ${props.boxClass ?? ''}`"
    >
      <div class="flex flex-col gap-0.5">
        <div class="font-medium">{{ fullName }}</div>
        <div class="mt-0.5 text-xs opacity-70">{{ user?.email }}</div>
      </div>
      <div class="bg-foreground/5 h-px"></div>
      <div class="flex flex-col gap-0.5">
        <a
          class="hover:bg-foreground/5 -mx-3 flex gap-2.5 rounded-lg px-4 py-1.5"
          href="#"
          @click.prevent="changePasswordModal?.handleModal(true)"
        >
          <Lucide icon="KeyRound" /> {{ $t('auth.changePassword') }}
        </a>
        <a class="hover:bg-foreground/5 -mx-3 flex gap-2.5 rounded-lg px-4 py-1.5" href="#" @click.prevent="logout()">
          <Lucide icon="Power" /> {{ $t('common.logout') }}
        </a>
      </div>
    </Box>

    <ModalChangePassword ref="changePasswordModal" />
  </div>
</template>
