<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Input } from '~/base/ui/input'
import { Button } from '~/base/ui/button'
import { Table, TableBody, TableHeader, TableRow } from '~/base/ui/table'
import { Lucide } from '~/base/ui/lucide'
import { Badge } from '~/base/ui/badge'
import { useActivityLogService } from '~/services/ActivityLogService'
import { useLookupService } from '~/services/LookupService'
import ModalActivityLogDetail from '~/components/features/activity-logs/ModalActivityLogDetail.vue'
import LiveActivityFeed from '~/components/features/activity-logs/LiveActivityFeed.vue'
import type ActivityLog from '~/types/entities/activity_log'

definePageMeta({
  title: 'activityLogs.title',
  pageSubTitle: 'activityLogs.subtitle',
  breadcrumb: ['Activity Logs'],
})

const { logs, meta, loading, getAll } = useActivityLogService()

// Feeds the user filter's dropdown. Gated by lookups.users.show, which is
// not the management.activity-logs.show this page runs on — so the dropdown is
// hidden (and its request skipped) rather than 403-ing the whole page, which
// is what happened while this read the full user-management list instead.
//
// user_id must not be sent without the permission either: the API refuses
// that parameter rather than ignoring it. Hiding the control is what keeps it
// out of `filters`.
const { options: userOptions, can: canFilterByUser, load: loadUsers } = useLookupService('users')
onMounted(loadUsers)

const methodOptions = [
  { id: 'POST', name: 'POST' },
  { id: 'PATCH', name: 'PATCH' },
  { id: 'PUT', name: 'PUT' },
  { id: 'DELETE', name: 'DELETE' },
]

const { filters, hasActiveFilters, reset } = useServerList({
  filters: { search: '', method: '', status_code: '', user_id: '', date_from: '', date_to: '' },
  debounce: ['search', 'status_code'],
  fetch: (page, perPage, applied) => getAll(page, perPage, applied),
})

const statusVariant = (status: number) => {
  if (status >= 500) return 'danger'
  if (status >= 400) return 'warning'
  return 'success'
}

const modalDetailRef = useTemplateRef<InstanceType<typeof ModalActivityLogDetail> | null>('modalDetailRef')
const viewDetail = (log: ActivityLog) => modalDetailRef.value?.handleModal(true, log)
</script>

<template>
  <div class="grid grid-cols-12 gap-6">
    <Box class="col-span-12 lg:col-span-9 overflow-auto lg:overflow-visible">
      <div class="mb-4 flex justify-end">
        <FilterPopover :active="hasActiveFilters">
          <div class="relative">
            <Input
              v-model="filters.search"
              type="text"
              :placeholder="$t('activityLogs.pathPlaceholder')"
              class="w-full pr-8"
            />
            <Lucide
              class="absolute inset-y-0 right-0 my-auto mr-2.5 size-4 text-foreground/40"
              icon="Search"
            />
          </div>

          <TomSelect v-model="filters.method" :options="methodOptions" :placeholder="$t('activityLogs.allMethodsPlaceholder')" />
          <Input v-model="filters.status_code" type="number" :placeholder="$t('activityLogs.statusCodePlaceholder')" class="w-full" />
          <TomSelect v-if="canFilterByUser" v-model="filters.user_id" :options="userOptions" :placeholder="$t('activityLogs.allUsersPlaceholder')" />
          <Input v-model="filters.date_from" type="date" class="w-full" :aria-label="$t('activityLogs.dateFromLabel')" />
          <Input v-model="filters.date_to" type="date" class="w-full" :aria-label="$t('activityLogs.dateToLabel')" />

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
              <TH icon="User">{{ $t('activityLogs.columnUser') }}</TH>
              <TH icon="ArrowRightLeft">{{ $t('activityLogs.columnMethod') }}</TH>
              <TH icon="Link">{{ $t('activityLogs.columnPath') }}</TH>
              <TH icon="CircleCheck">{{ $t('common.status') }}</TH>
              <TH icon="Globe">{{ $t('activityLogs.columnIp') }}</TH>
              <TH icon="Clock">{{ $t('common.createdAt') }}</TH>
              <TH icon="Settings">{{ $t('common.actions') }}</TH>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="(log, index) in logs" :key="log.id">
              <TD>{{ meta.from + index }}</TD>
              <TD>
                <span v-if="log.user">{{ log.user.full_name }}</span>
                <span v-else class="text-sm text-foreground/40">—</span>
              </TD>
              <TD>
                <Badge variant="secondary" look="outline" class="text-xs">{{ log.method }}</Badge>
              </TD>
              <TD>{{ log.path }}</TD>
              <TD>
                <Badge :variant="statusVariant(log.status_code)" look="outline" class="text-xs">
                  {{ log.status_code }}
                </Badge>
              </TD>
              <TD>{{ log.ip }}</TD>
              <TD>{{ log.created_at }}</TD>
              <TD>
                <Button look="outline" variant="ghost" @click="viewDetail(log)">
                  <Lucide icon="Eye" />
                </Button>
              </TD>
            </TableRow>
          </TableBody>
        </Table>
      </ServerSidePagination>
    </Box>

    <div class="col-span-12 lg:col-span-3">
      <LiveActivityFeed @view="viewDetail" />
    </div>
  </div>

  <ModalActivityLogDetail ref="modalDetailRef" />
</template>

<style scoped></style>
