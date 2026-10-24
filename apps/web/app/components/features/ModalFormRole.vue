<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Label } from '~/base/ui/label'
import { Input } from '~/base/ui/input'
import { Lucide } from '~/base/ui/lucide'
import type { PropType } from 'vue'
import type Permission from '~/types/entities/permission'
import type Role from '~/types/entities/role'
import { type RoleRequest, useRoleService } from '~/services/RoleService'

const props = defineProps({
  permissions: {
    required: true,
    type: Object as PropType<Permission[]>,
  },
})

const emit = defineEmits<{ (e: 'submit'): void }>()

const { addNew, updateById, submitting } = useRoleService()

const isShowModal = ref(false)
const modalType = ref<'create' | 'edit'>('create')
const editingId = ref<string | null>(null)
const isMutable = ref(true)
const permissionSearchKey = ref('')

const getInitialForm = (): RoleRequest => ({ name: '', permission_ids: [] })
const form = ref<RoleRequest>(getInitialForm())

/**
 * Feature group of a permission, e.g. `management.users.show` -> `management - users`.
 * Derived from the name because the API does not send `feature_group` yet.
 */
const featureGroupOf = (permission: Permission): string => {
  if (permission.feature_group) return permission.feature_group

  const segments = permission.name.split('.')
  segments.pop()
  return segments.join(' - ').replace(/_/g, ' ')
}

const groupedPermissions = computed(() => {
  const groups: Record<string, Permission[]> = {}

  for (const permission of props.permissions) {
    const group = featureGroupOf(permission)
    ;(groups[group] ??= []).push(permission)
  }

  return groups
})

const filteredGroupedPermissions = computed<Record<string, Permission[]>>(() => {
  const query = permissionSearchKey.value.trim().toLowerCase()
  if (!query) return groupedPermissions.value

  const result: Record<string, Permission[]> = {}

  for (const [groupName, permissions] of Object.entries(groupedPermissions.value)) {
    // a group name match keeps the whole group, otherwise match per permission
    const matched = groupName.toLowerCase().includes(query)
      ? permissions
      : permissions.filter(
          (permission) =>
            permission.name?.toLowerCase().includes(query) ||
            permission.description?.toLowerCase().includes(query),
        )

    if (matched.length > 0) result[groupName] = matched
  }

  return result
})

const selectedCountIn = (permissions: Permission[]) =>
  permissions.filter((permission) => form.value.permission_ids.includes(permission.id)).length

const isGroupFullySelected = (permissions: Permission[]) =>
  permissions.length > 0 &&
  permissions.every((permission) => form.value.permission_ids.includes(permission.id))

const toggleGroup = (permissions: Permission[], checked: boolean) => {
  const groupIds = permissions.map((permission) => permission.id)
  form.value.permission_ids = checked
    ? Array.from(new Set([...form.value.permission_ids, ...groupIds]))
    : form.value.permission_ids.filter((id) => !groupIds.includes(id))
}

const togglePermission = (id: string, checked: boolean) => {
  form.value.permission_ids = checked
    ? Array.from(new Set([...form.value.permission_ids, id]))
    : form.value.permission_ids.filter((permissionId) => permissionId !== id)
}

const handleModal = (isShow: boolean, role?: Role) => {
  isShowModal.value = isShow
  form.value = getInitialForm()
  editingId.value = null
  isMutable.value = true
  permissionSearchKey.value = ''

  if (role) {
    modalType.value = 'edit'
    editingId.value = role.id
    isMutable.value = role.is_mutable
    form.value.name = role.name
    form.value.permission_ids = role.permissions.map((p) => p.id)
  } else {
    modalType.value = 'create'
  }
}

const onSubmit = async () => {
  const ok =
    modalType.value === 'create'
      ? await addNew(form.value)
      : await updateById(editingId.value!, form.value)

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
    <DialogContent class="sm:max-w-320">
      <DialogTitle>
        <div class="flex items-center justify-between w-full">
          <h2 class="text-base font-medium">
            {{
              modalType === 'create'
                ? $t('account.roles.createRole')
                : isMutable
                  ? $t('account.roles.updateRole')
                  : $t('account.roles.updateRolePermissions')
            }}
          </h2>
          <button
            type="button"
            class="text-foreground/40 hover:text-foreground/70"
            @click="handleModal(false)"
          >
            <Lucide icon="X" class="w-4 h-4" />
          </button>
        </div>
      </DialogTitle>

      <div class="grid grid-cols-12 gap-4 gap-y-3 mt-6 overflow-y-auto max-h-[70vh]">
        <div class="col-span-12 sm:col-span-6">
          <Label for="role-name">{{ $t('common.name') }} <RequiredMark /></Label>
          <Input
            id="role-name"
            v-model="form.name"
            class="mt-2"
            :placeholder="$t('account.roles.namePlaceholder')"
            :disabled="!isMutable"
          />
          <div v-if="!isMutable" class="text-xs text-foreground/40 mt-1">
            {{ $t('account.roles.nameProtected') }}
          </div>
          <FormFeedback feedbackKey="name" />
        </div>

        <div
          class="col-span-12 flex flex-wrap items-center justify-between gap-3 pb-1 border-b border-foreground/10 mt-2"
        >
          <div>
            <h3 class="text-sm font-bold">{{ $t('account.roles.columnPermissions') }}</h3>
            <div class="text-xs text-foreground/40 mt-0.5">
              {{ $t('account.roles.selectPermissionsHint') }}
            </div>
          </div>
          <div class="flex items-center gap-3">
            <span class="text-xs text-foreground/40 font-medium whitespace-nowrap">
              {{ $t('common.selectedCount', { count: form.permission_ids.length }) }}
            </span>
            <ClientSideFilter v-model="permissionSearchKey" />
          </div>
        </div>

        <div v-if="Object.keys(filteredGroupedPermissions).length === 0" class="col-span-12">
          <div class="text-xs text-foreground/40 italic text-center py-6">
            {{ $t('account.roles.noPermissionsMatch') }}
          </div>
        </div>

        <div
          v-for="(groupPermissions, groupName) in filteredGroupedPermissions"
          :key="groupName"
          class="col-span-12 sm:col-span-6"
        >
          <div class="rounded-xl border border-foreground/10 p-4 h-full">
            <div class="flex items-center gap-3 pb-3 mb-3 border-b border-foreground/10">
              <div
                class="w-7 h-7 rounded-lg bg-primary/10 flex items-center justify-center text-primary shrink-0"
              >
                <Lucide icon="KeyRound" class="w-3.5 h-3.5" />
              </div>
              <label class="font-bold text-sm flex-1 min-w-0 break-words capitalize">
                {{ groupName }}
              </label>
              <span class="text-[10px] text-foreground/40 font-semibold shrink-0">
                {{ selectedCountIn(groupPermissions) }}/{{ groupPermissions.length }}
              </span>
              <label
                class="flex items-center gap-1.5 text-[10px] font-semibold text-foreground/40 hover:text-primary cursor-pointer shrink-0 select-none"
              >
                <input
                  type="checkbox"
                  class="form-checkbox w-3.5 h-3.5 rounded border-slate-300 text-primary cursor-pointer"
                  :checked="isGroupFullySelected(groupPermissions)"
                  @change="
                    toggleGroup(groupPermissions, ($event.target as HTMLInputElement).checked)
                  "
                />
                {{ $t('common.all') }}
              </label>
            </div>

            <div class="grid grid-cols-1 gap-2">
              <label
                v-for="permission in groupPermissions"
                :key="permission.id"
                class="flex items-start gap-2 cursor-pointer"
              >
                <input
                  type="checkbox"
                  class="form-checkbox mt-0.5 shrink-0 h-4 w-4 rounded border-slate-300 text-primary"
                  :checked="form.permission_ids.includes(permission.id)"
                  @change="
                    togglePermission(permission.id, ($event.target as HTMLInputElement).checked)
                  "
                />
                <span class="flex-1 min-w-0 select-none">
                  <span class="block text-xs leading-relaxed break-words">
                    {{ permission.description || permission.name }}
                  </span>
                  <span class="block text-[10px] text-foreground/40 font-medium mt-0.5 break-words">
                    {{ permission.name }}
                  </span>
                </span>
              </label>
            </div>
          </div>
        </div>

        <div class="col-span-12">
          <FormFeedback feedbackKey="permission_ids" />
        </div>
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