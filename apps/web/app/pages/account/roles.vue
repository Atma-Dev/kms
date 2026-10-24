<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Badge } from '~/base/ui/badge'
import { TableBody, TableHeader, TableRow } from '~/base/ui/table'
import { Button } from '~/base/ui/button'
import { Lucide } from '~/base/ui/lucide'
import { useRoleService } from '~/services/RoleService'
import type Role from '~/types/entities/role'
import type { ClientSidePaginationProps } from '~/types/response'
import { usePermissionService } from '~/services/PermissionService'
import type { ModalDelete } from '#components'
import ModalFormRole from '~/components/features/ModalFormRole.vue'
import { Permission } from '~/enums/Permission'

definePageMeta({
  title: 'account.roles.title',
  pageSubTitle: 'account.roles.subtitle',
  breadcrumb: ['Account', 'Roles'],
})

const authStore = useAuthStore()
// The role form's permission picker needs feature_group and description, so
// it reads the full permission list rather than a lookup — and that answers to
// management.permissions.show, not the management.roles.show this page runs
// on. Without it the fetch is skipped (it would otherwise 403 the whole page)
// and the buttons that open the form are hidden, since the form would have
// nothing to pick from.
const canReadPermissions = computed(() => authStore.hasPermission(Permission.MANAGEMENT_PERMISSIONS_SHOW))
const canStore = computed(
  () => authStore.hasPermission(Permission.MANAGEMENT_ROLES_STORE) && canReadPermissions.value,
)
const canUpdate = computed(
  () => authStore.hasPermission(Permission.MANAGEMENT_ROLES_UPDATE) && canReadPermissions.value,
)
const canDestroy = computed(() => authStore.hasPermission(Permission.MANAGEMENT_ROLES_DESTROY))

const { getAll, deleteById, rolesCollection, loading, submitting } = useRoleService()
const { permissionsCollection, getAll: getPermissions } = usePermissionService()

onMounted(async () => {
  await Promise.all([getAll(), canReadPermissions.value ? getPermissions() : Promise.resolve()])
})

const searchKey = ref('')
const filteredData = computed((): Role[] => {
  const key = searchKey.value.toLowerCase()
  return key === ''
    ? rolesCollection.value
    : rolesCollection.value.filter((r) => r.name?.toLowerCase().includes(key))
})

const modalFormRef = useTemplateRef<InstanceType<typeof ModalFormRole> | null>('modalFormRef')
const modalDeleteRef = useTemplateRef<InstanceType<typeof ModalDelete> | null>('modalDeleteRef')
const selectedRoleId = ref('')

const PERMISSION_PREVIEW_LIMIT = 3

const visiblePermissions = (role: Role) => role.permissions.slice(0, PERMISSION_PREVIEW_LIMIT)
const hiddenPermissionCount = (role: Role) =>
  Math.max(0, role.permissions.length - PERMISSION_PREVIEW_LIMIT)
</script>

<template>
  <div class="grid grid-cols-12 gap-6">
    <Box class="col-span-12 overflow-auto lg:overflow-visible">
      <div class="flex flex-wrap items-center justify-between gap-3 mb-4">
        <Button v-if="canStore" @click="modalFormRef?.handleModal(true)">
          <Lucide icon="Plus" />
          {{ $t('account.roles.addRole') }}
        </Button>
        <ClientSideFilter v-model="searchKey" />
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center gap-3 py-16 text-foreground/50">
        <svg class="h-5 w-5 animate-spin" viewBox="0 0 24 24" fill="none">
          <circle
            class="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            stroke-width="4"
          />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
        </svg>
        <span class="text-sm">{{ $t('common.loading') }}</span>
      </div>

      <ClientSidePagination
        v-else
        :data="filteredData"
        v-slot="slotProps: ClientSidePaginationProps<Role>"
      >
        <TableHeader>
          <TableRow>
            <TH icon="Hash">{{ $t('common.no') }}</TH>
            <TH icon="UserCog">{{ $t('account.roles.columnName') }}</TH>
            <TH icon="ShieldCheck">{{ $t('account.roles.columnPermissions') }}</TH>
            <TH icon="Lock">{{ $t('account.roles.columnMutable') }}</TH>
            <TH icon="Lock">{{ $t('common.createdAt') }}</TH>
            <TH icon="Settings">{{ $t('common.actions') }}</TH>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="(role, index) in slotProps.data" :key="role.id">
            <TD>{{ slotProps.start + index }}</TD>
            <TD>
              <p class="font-medium">{{ role.name }}</p>
              <SmallInfo>{{ role.id }}</SmallInfo>
            </TD>
            <TD>
              <div class="flex flex-wrap gap-1">
                <Badge
                  v-for="perm in visiblePermissions(role)"
                  :key="perm.id"
                  variant="secondary"
                  look="outline"
                  class="text-xs"
                >
                  {{ perm.name }}
                </Badge>
                <Badge
                  v-if="hiddenPermissionCount(role) > 0"
                  variant="ghost"
                  look="outline"
                  class="text-xs"
                >
                  {{ $t('account.roles.morePermissions', { count: hiddenPermissionCount(role) }) }}
                </Badge>
                <span v-if="role.permissions.length === 0" class="text-sm text-foreground/40"
                  >—</span
                >
              </div>
            </TD>
            <TD>
              <span
                v-if="role.is_mutable"
                class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-success/10 text-success text-xs font-medium"
              >
                <Lucide icon="LockOpen" class="w-3 h-3" />
                {{ $t('account.roles.mutable') }}
              </span>
              <span
                v-else
                class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-foreground/10 text-foreground/50 text-xs font-medium"
              >
                <Lucide icon="Lock" class="w-3 h-3" />
                {{ $t('account.roles.immutable') }}
              </span>
            </TD>
            <TD>
              {{ role.created_at }}
            </TD>
            <TD>
              <div class="flex items-center gap-1">
                <ButtonEdit v-if="canUpdate" @click-edit="modalFormRef?.handleModal(true, role)" />
                <ButtonDelete
                  v-if="role.is_mutable && canDestroy"
                  @click-delete="
                    () => {
                      modalDeleteRef?.handleModal(true)
                      selectedRoleId = role.id
                    }
                  "
                />
              </div>
            </TD>
          </TableRow>
        </TableBody>
      </ClientSidePagination>
    </Box>
  </div>

  <ModalDelete
    ref="modalDeleteRef"
    @submit="
      async () => {
        const ok = await deleteById(selectedRoleId)
        if (ok) await getAll()
      }
    "
  />

  <ModalFormRole ref="modalFormRef" :permissions="permissionsCollection" @submit="getAll()" />
</template>
