<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Input } from '~/base/ui/input'
import { Button } from '~/base/ui/button'
import TomSelect from '~/components/TomSelect.vue'
import { TableBody, TableHeader, TableRow } from '~/base/ui/table'
import { Lucide } from '~/base/ui/lucide'
import { useUserService } from '~/services/UserService'
import { useRoleService } from '~/services/RoleService'
import { Badge } from '~/base/ui/badge'
import type User from '~/types/entities/user'
import type { ClientSidePaginationProps } from '~/types/response'
import type { UserRole } from '~/types/entities/user'
import ModalFormUser from '~/components/features/ModalFormUser.vue'
import { isSuperAdminRole } from '~/utils/role'
import { Permission } from '~/enums/Permission'

definePageMeta({
  title: 'account.users.title',
  pageSubTitle: 'account.users.subtitle',
  breadcrumb: ['Account', 'Users'],
})

const authStore = useAuthStore()
// The user form assigns roles, and shows each role's permission count — the
// full role list, not a lookup, and gated by management.roles.show rather than
// the management.users.show this page runs on. Skipped without it (a 403 here
// would take the whole page down) and the edit button goes with it.
const canReadRoles = computed(() => authStore.hasPermission(Permission.MANAGEMENT_ROLES_SHOW))
const canUpdate = computed(
  () => authStore.hasPermission(Permission.MANAGEMENT_USERS_UPDATE) && canReadRoles.value,
)

const { users, loading, getAll } = useUserService()
const { rolesCollection, getAll: getRoles } = useRoleService()

onMounted(() => {
  getAll()
  if (canReadRoles.value) getRoles()
})

// Client-side now: the endpoint returns every user in one call, so filtering
// here costs nothing and the list responds without a round trip.
const filters = reactive({ search: '', membership_status: '' })
const hasActiveFilters = computed(
  () => filters.search !== '' || filters.membership_status !== '',
)
const reset = () => {
  filters.search = ''
  filters.membership_status = ''
}

const filteredData = computed((): User[] => {
  const search = filters.search.toLowerCase()
  return users.value.filter((user) => {
    if (filters.membership_status !== '' && user.membership_status !== filters.membership_status)
      return false
    if (search === '') return true
    return (
      user.full_name?.toLowerCase().includes(search) || user.email?.toLowerCase().includes(search)
    )
  })
})

const { t } = useI18n()
const membershipStatusOptions = computed(() => [
  { id: 'ACTIVE', name: t('account.users.membershipActive') },
  { id: 'INACTIVE', name: t('account.users.membershipInactive') },
])

const modalFormRef = useTemplateRef<InstanceType<typeof ModalFormUser> | null>('modalFormRef')

const ROLE_PREVIEW_LIMIT = 3

const isSuperAdmin = (role: UserRole) => isSuperAdminRole(role.name)

// Super-Admin leads the preview so it never collapses into "+N more".
const sortedRoles = (user: User) =>
  [...user.roles].sort((a, b) => Number(isSuperAdmin(b)) - Number(isSuperAdmin(a)))

const visibleRoles = (user: User) => sortedRoles(user).slice(0, ROLE_PREVIEW_LIMIT)
const hiddenRoleCount = (user: User) => Math.max(0, user.roles.length - ROLE_PREVIEW_LIMIT)
</script>

<template>
  <div class="grid grid-cols-12 gap-6">
    <Box class="col-span-12 overflow-auto lg:overflow-visible">
      <div class="mb-4 flex flex-wrap items-end justify-end gap-3">
        <div class="relative">
          <Input
            v-model="filters.search"
            type="text"
            :placeholder="$t('account.users.searchPlaceholder')"
            class="w-52 pr-8"
          />
          <Lucide
            class="absolute inset-y-0 right-0 my-auto mr-2.5 size-4 text-foreground/40"
            icon="Search"
          />
        </div>

        <div class="w-44">
          <TomSelect
            v-model="filters.membership_status"
            :options="membershipStatusOptions"
            :placeholder="$t('account.users.allMembershipStatus')"
          />
        </div>

        <!-- Always rendered — disabling rather than hiding keeps the inputs
             above from shifting when filters come and go. -->
        <Button look="outline" variant="ghost" :disabled="!hasActiveFilters" @click="reset">
          <Lucide class="mr-1.5 size-4" icon="X" />
          {{ $t('common.reset') }}
        </Button>
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
        v-slot="slotProps: ClientSidePaginationProps<User>"
      >
        <TableHeader>
          <TableRow>
            <TH icon="Hash">{{ $t('common.no') }}</TH>
            <TH icon="User">{{ $t('common.name') }}</TH>
            <TH icon="Mail">{{ $t('account.users.columnEmail') }}</TH>
            <TH icon="UserCog">{{ $t('account.users.columnRoles') }}</TH>
            <TH icon="IdCard">{{ $t('account.users.columnMarketingId') }}</TH>
            <TH icon="ShieldCheck">{{ $t('account.users.columnMembership') }}</TH>
            <TH icon="Settings">{{ $t('common.actions') }}</TH>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="(user, index) in slotProps.data" :key="user.id">
            <TD>{{ slotProps.start + index }}</TD>
            <TD>
              <p class="font-medium">{{ user.full_name }}</p>
              <SmallInfo>{{ user.id }}</SmallInfo>
            </TD>
            <TD>{{ user.email }}</TD>
            <TD>
              <div class="flex flex-wrap gap-1">
                <Badge
                  v-for="role in visibleRoles(user)"
                  :key="role.id"
                  :variant="isSuperAdmin(role) ? 'warning' : 'secondary'"
                  look="outline"
                  class="text-xs"
                >
                  <Lucide v-if="isSuperAdmin(role)" icon="Crown" />
                  {{ role.name }}
                </Badge>
                <Badge
                  v-if="hiddenRoleCount(user) > 0"
                  variant="ghost"
                  look="outline"
                  class="text-xs"
                >
                  {{ $t('account.users.moreRoles', { count: hiddenRoleCount(user) }) }}
                </Badge>
                <span v-if="user.roles.length === 0" class="text-sm text-foreground/40">—</span>
              </div>
            </TD>
            <TD>
              <Badge v-if="user.marketing_id" variant="secondary" look="outline" class="text-xs">
                {{ user.marketing_id }}
              </Badge>
              <span v-else class="text-sm text-foreground/40">—</span>
            </TD>
            <TD>
              <span
                v-if="user.membership_status === 'ACTIVE'"
                class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-success/10 text-success text-xs font-medium"
              >
                <Lucide icon="ShieldCheck" class="w-3 h-3" />
                {{ $t('account.users.membershipActive') }}
              </span>
              <span
                v-else
                class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-foreground/10 text-foreground/50 text-xs font-medium"
              >
                <Lucide icon="ShieldOff" class="w-3 h-3" />
                {{ $t('account.users.membershipInactive') }}
              </span>
            </TD>
            <TD>
              <div class="flex items-center gap-1">
                <ButtonEdit v-if="canUpdate" @click-edit="modalFormRef?.handleModal(true, user)" />
              </div>
            </TD>
          </TableRow>
        </TableBody>
      </ClientSidePagination>
    </Box>
  </div>

  <ModalFormUser ref="modalFormRef" :roles="rolesCollection" @submit="getAll()" />
</template>

<style scoped></style>
