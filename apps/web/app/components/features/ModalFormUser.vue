<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Input } from '~/base/ui/input'
import { Lucide } from '~/base/ui/lucide'
import { Badge } from '~/base/ui/badge'
import type { PropType } from 'vue'
import type Role from '~/types/entities/role'
import type User from '~/types/entities/user'
import { useUserService } from '~/services/UserService'
import { isSuperAdminRole } from '~/utils/role'

const props = defineProps({
  roles: {
    required: true,
    type: Object as PropType<Role[]>,
  },
})

const emit = defineEmits<{ (e: 'submit'): void }>()

const { updateById, submitting } = useUserService()

const isShowModal = ref(false)
const editingUser = ref<User | null>(null)
const selectedRoleIds = ref<string[]>([])
const marketingId = ref('')
const roleSearchKey = ref('')

const filteredRoles = computed(() => {
  const query = roleSearchKey.value.trim().toLowerCase()
  if (!query) return props.roles
  return props.roles.filter((role) => role.name?.toLowerCase().includes(query))
})

const toggleRole = (id: string, checked: boolean) => {
  selectedRoleIds.value = checked
    ? Array.from(new Set([...selectedRoleIds.value, id]))
    : selectedRoleIds.value.filter((roleId) => roleId !== id)
}

const handleModal = (isShow: boolean, user?: User) => {
  isShowModal.value = isShow
  roleSearchKey.value = ''
  editingUser.value = user ?? null
  // The list payload only carries {id, name} per role, which is all we need.
  selectedRoleIds.value = user ? user.roles.map((role) => role.id) : []
  marketingId.value = user?.marketing_id ?? ''
}

const onSubmit = async () => {
  if (!editingUser.value) return

  const ok = await updateById(editingUser.value.id, {
    role_ids: selectedRoleIds.value,
    // Trimmed so a whitespace-only value clears the field instead of storing blanks.
    marketing_id: marketingId.value.trim(),
  })
  if (ok) {
    handleModal(false)
    emit('submit')
  }
}

defineExpose({ handleModal })
</script>

<template>
  <DialogRoot
    :open="isShowModal"
    :onOpenChange="(details: { open: boolean }) => { isShowModal = details.open }"
  >
    <DialogContent class="sm:max-w-2xl">
      <DialogTitle>
        <div class="flex items-center justify-between w-full">
          <h2 class="text-base font-medium">{{ $t('account.users.updateUser') }}</h2>
          <button
            type="button"
            class="text-foreground/40 hover:text-foreground/70"
            @click="handleModal(false)"
          >
            <Lucide icon="X" class="w-4 h-4" />
          </button>
        </div>
      </DialogTitle>

      <div v-if="editingUser" class="mt-6">
        <div class="rounded-xl border border-foreground/10 p-4">
          <p class="font-medium">{{ editingUser.full_name }}</p>
          <SmallInfo>{{ editingUser.email }}</SmallInfo>
        </div>

        <div class="mt-5">
          <label class="text-sm font-bold" for="marketing_id">{{ $t('account.users.columnMarketingId') }}</label>
          <div class="text-xs text-foreground/40 mt-0.5 mb-2">
            {{ $t('account.users.marketingIdHint') }}
          </div>
          <Input
            id="marketing_id"
            v-model="marketingId"
            type="text"
            :placeholder="$t('account.users.marketingIdPlaceholder')"
          />
          <FormFeedback feedbackKey="marketing_id" />
        </div>

        <div
          class="flex flex-wrap items-center justify-between gap-3 pb-1 border-b border-foreground/10 mt-5"
        >
          <div>
            <h3 class="text-sm font-bold">{{ $t('account.users.columnRoles') }}</h3>
            <div class="text-xs text-foreground/40 mt-0.5">
              {{ $t('account.users.selectRolesHint') }}
            </div>
          </div>
          <div class="flex items-center gap-3">
            <span class="text-xs text-foreground/40 font-medium whitespace-nowrap">
              {{ $t('common.selectedCount', { count: selectedRoleIds.length }) }}
            </span>
            <ClientSideFilter v-model="roleSearchKey" />
          </div>
        </div>

        <div v-if="filteredRoles.length === 0" class="text-xs text-foreground/40 italic text-center py-6">
          {{ $t('account.users.noRolesMatch') }}
        </div>

        <div v-else class="grid grid-cols-1 sm:grid-cols-2 gap-2 mt-3 overflow-y-auto max-h-[50vh]">
          <label
            v-for="role in filteredRoles"
            :key="role.id"
            class="flex items-start gap-2 rounded-xl border border-foreground/10 p-3 cursor-pointer hover:border-primary/40"
          >
            <input
              type="checkbox"
              class="form-checkbox mt-0.5 shrink-0 h-4 w-4 rounded border-slate-300 text-primary"
              :checked="selectedRoleIds.includes(role.id)"
              @change="toggleRole(role.id, ($event.target as HTMLInputElement).checked)"
            />
            <span class="flex-1 min-w-0 select-none">
              <Badge
                :variant="isSuperAdminRole(role.name) ? 'warning' : 'secondary'"
                look="outline"
                class="text-xs"
              >
                <Lucide v-if="isSuperAdminRole(role.name)" icon="Crown" />
                {{ role.name }}
              </Badge>
              <span class="block text-[10px] text-foreground/40 font-medium mt-1">
                {{ $t('account.users.permissionsCount', { count: role.permissions.length }) }}
              </span>
            </span>
          </label>
        </div>

        <FormFeedback feedbackKey="role_ids" />
      </div>

      <div class="flex gap-2 justify-end mt-7">
        <Button look="outline" variant="ghost" @click="handleModal(false)">
          <Lucide icon="X" />
          {{ $t('common.cancel') }}
        </Button>
        <Button variant="primary" :disabled="submitting" @click="onSubmit">
          <svg v-if="submitting" class="mr-1.5 h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
          </svg>
          <Lucide v-else icon="Save" />
          {{ submitting ? $t('common.saving') : $t('common.confirm') }}
        </Button>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
