<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { TableBody, TableHeader, TableRow } from '~/base/ui/table'
import { Lucide } from '~/base/ui/lucide'
import type { ClientSidePaginationProps } from '~/types/response'
import ClientSideFilter from '~/components/ClientSideFilter.vue'
import { usePermissionService } from '~/services/PermissionService'
import type Permission from '~/types/entities/permission'

definePageMeta({
  title: 'account.permissions.title',
  pageSubTitle: 'account.permissions.subtitle',
  breadcrumb: ['Account', 'Permissions'],
})

const { getAll, permissionsCollection, loading } = usePermissionService()

onMounted(() => getAll())

const searchKey = ref('')
const filteredData = computed((): Permission[] => {
  const key = searchKey.value.toLowerCase()
  return key === ''
    ? permissionsCollection.value
    : permissionsCollection.value.filter(p => p.name?.toLowerCase().includes(key))
})
</script>

<template>
  <div class="grid grid-cols-12">
    <Box class="col-span-12 overflow-auto lg:overflow-visible">
      <div class="mb-4 flex flex-wrap justify-end">
        <ClientSideFilter v-model="searchKey" />
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center gap-3 py-16 text-foreground/50">
        <svg class="h-5 w-5 animate-spin" viewBox="0 0 24 24" fill="none">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
        </svg>
        <span class="text-sm">{{ $t('common.loading') }}</span>
      </div>

      <ClientSidePagination
        v-else
        :data="filteredData"
        v-slot="slotProps: ClientSidePaginationProps<Permission>"
      >
        <TableHeader>
          <TableRow>
            <TH icon="Hash">{{ $t('common.no') }}</TH>
            <TH icon="FileText">{{ $t('common.name') }}</TH>
            <TH icon="AlignLeft">{{ $t('account.permissions.columnDescription') }}</TH>
            <TH icon="AlignLeft">{{ $t('common.createdAt') }}</TH>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="(permission, key) in slotProps.data" :key="key">
            <TD>{{ slotProps.start + key }}</TD>
            <TD>
              <p class="font-medium">{{ permission.name }}</p>
              <SmallInfo>{{ permission.id }}</SmallInfo>
            </TD>
            <TD>{{ permission.description ?? '-' }}</TD>
            <TD>{{ permission.created_at }}</TD>
          </TableRow>
        </TableBody>
      </ClientSidePagination>
    </Box>
  </div>
</template>
