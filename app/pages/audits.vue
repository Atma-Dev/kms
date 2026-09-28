<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Input } from '~/base/ui/input'
import { Button } from '~/base/ui/button'
import { Table, TableBody, TableHeader, TableRow } from '~/base/ui/table'
import { Lucide } from '~/base/ui/lucide'
import { useAuditService } from '~/services/AuditService'
import { useLookupService } from '~/services/LookupService'
import ModalAuditDetail from '~/components/features/audits/ModalAuditDetail.vue'
import LiveAuditFeed from '~/components/features/audits/LiveAuditFeed.vue'
import type Audit from '~/types/entities/audit'

definePageMeta({
  title: 'audits.title',
  pageSubTitle: 'audits.subtitle',
  breadcrumb: ['Audits'],
})

const { audits, meta, loading, getAll } = useAuditService()

// Same dropdown as activity-logs' user filter — lookups.users.show is a
// smaller capability than management.audits.show, so it's hidden rather
// than requested if the caller doesn't hold it.
const { options: userOptions, can: canFilterByUser, load: loadUsers } = useLookupService('users')
onMounted(loadUsers)

const { filters, hasActiveFilters, reset } = useServerList({
  filters: { entry_object_table: '', entry_object_id: '', actor_id: '', action: '', date_from: '', date_to: '' },
  debounce: ['entry_object_table', 'entry_object_id', 'action'],
  fetch: (page, perPage, applied) => getAll(page, perPage, applied),
})

const modalDetailRef = useTemplateRef<InstanceType<typeof ModalAuditDetail> | null>('modalDetailRef')
const viewDetail = (audit: Audit) => modalDetailRef.value?.handleModal(true, audit)
</script>

<template>
  <div class="grid grid-cols-12 gap-6">
    <Box class="col-span-12 lg:col-span-9 overflow-auto lg:overflow-visible">
      <div class="mb-4 flex justify-end">
        <FilterPopover :active="hasActiveFilters">
          <Input v-model="filters.entry_object_table" type="text" :placeholder="$t('audits.entryObjectTablePlaceholder')" class="w-full" />
          <Input v-model="filters.entry_object_id" type="text" :placeholder="$t('audits.entryObjectIdPlaceholder')" class="w-full" />
          <TomSelect v-if="canFilterByUser" v-model="filters.actor_id" :options="userOptions" :placeholder="$t('audits.allActorsPlaceholder')" />
          <Input v-model="filters.action" type="text" :placeholder="$t('audits.actionPlaceholder')" class="w-full" />
          <Input v-model="filters.date_from" type="date" class="w-full" :aria-label="$t('audits.dateFromLabel')" />
          <Input v-model="filters.date_to" type="date" class="w-full" :aria-label="$t('audits.dateToLabel')" />

          <template #footer>
            <Button look="outline" variant="ghost" size="sm" :disabled="!hasActiveFilters" @click="reset">
              <Lucide class="mr-1.5 size-4" icon="X" />
              {{ $t('common.reset') }}
            </Button>
          </template>
        </FilterPopover>
      </div>

      <ServerSidePagination :meta="meta" :loading="loading">
        <Table class="-mt-2" variant="boxed">
          <TableHeader>
            <TableRow>
              <TH icon="Hash">{{ $t('common.no') }}</TH>
              <TH icon="User">{{ $t('audits.columnActor') }}</TH>
              <TH icon="Zap">{{ $t('audits.columnAction') }}</TH>
              <TH icon="Database">{{ $t('audits.columnEntryObject') }}</TH>
              <TH icon="Clock">{{ $t('common.createdAt') }}</TH>
              <TH icon="Settings">{{ $t('common.actions') }}</TH>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="(audit, index) in audits" :key="audit.id">
              <TD>{{ meta.from + index }}</TD>
              <TD>
                <span v-if="audit.actor_name">{{ audit.actor_name }}</span>
                <span v-else class="text-sm text-foreground/40">—</span>
              </TD>
              <TD>{{ audit.action }}</TD>
              <TD>
                <span v-if="audit.entry_object_table">{{ audit.entry_object_table }} #{{ audit.entry_object_id }}</span>
                <span v-else class="text-sm text-foreground/40">—</span>
              </TD>
              <TD>{{ audit.created_at }}</TD>
              <TD>
                <Button look="outline" variant="ghost" @click="viewDetail(audit)">
                  <Lucide icon="Eye" />
                </Button>
              </TD>
            </TableRow>
          </TableBody>
        </Table>
      </ServerSidePagination>
    </Box>

    <div class="col-span-12 lg:col-span-3">
      <!-- Live feed only — never touches the table above. New audits show up
           here as they happen; the table stays exactly as the last manual
           page/filter left it until you refresh or re-filter it yourself. -->
      <LiveAuditFeed @view="viewDetail" />
    </div>
  </div>

  <ModalAuditDetail ref="modalDetailRef" />
</template>

<style scoped></style>
