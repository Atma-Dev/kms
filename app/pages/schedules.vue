<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Table, TableBody, TableHeader, TableRow } from '~/base/ui/table'
import { Lucide } from '~/base/ui/lucide'
import { Badge } from '~/base/ui/badge'
import { useScheduleService } from '~/services/ScheduleService'
import ModalFormSchedule from '~/components/features/schedules/ModalFormSchedule.vue'
import type ScheduledJob from '~/types/entities/scheduled_job'
import { Permission } from '~/enums/Permission'

definePageMeta({
  title: 'schedules.title',
  pageSubTitle: 'schedules.subtitle',
  breadcrumb: ['Schedules'],
})

const authStore = useAuthStore()
const canManage = computed(() => authStore.hasPermission(Permission.MANAGEMENT_SCHEDULES_MANAGE))

const { schedules, loading, getAll, runNow } = useScheduleService()
const refresh = () => getAll()
onMounted(refresh)

const modalFormRef = useTemplateRef<InstanceType<typeof ModalFormSchedule> | null>('modalFormRef')

const statusVariant = (status: string | null) => (status === 'success' ? 'success' : status === 'skipped' ? 'secondary' : 'danger')

const runningId = ref<string | null>(null)
const onRunNow = async (schedule: ScheduledJob) => {
  runningId.value = schedule.id
  try {
    if (await runNow(schedule.id)) refresh()
  } finally {
    runningId.value = null
  }
}
</script>

<template>
  <Box class="overflow-auto lg:overflow-visible">
    <div v-if="loading" class="flex items-center justify-center gap-3 py-16 text-foreground/50">
      <svg class="h-5 w-5 animate-spin" viewBox="0 0 24 24" fill="none">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
      </svg>
      <span class="text-sm">{{ $t('common.loading') }}</span>
    </div>

    <Table v-else variant="boxed">
      <TableHeader>
        <TableRow>
          <TH icon="Hash">{{ $t('common.no') }}</TH>
          <TH icon="Braces">{{ $t('jobLogs.columnName') }}</TH>
          <TH icon="Clock4">{{ $t('schedules.columnCron') }}</TH>
          <TH icon="Power">{{ $t('schedules.columnEnabled') }}</TH>
          <TH icon="Clock">{{ $t('schedules.columnNextRun') }}</TH>
          <TH icon="History">{{ $t('schedules.columnLastRun') }}</TH>
          <TH icon="MessageSquare">{{ $t('jobLogs.columnMessage') }}</TH>
          <TH v-if="canManage" icon="Settings">{{ $t('common.actions') }}</TH>
        </TableRow>
      </TableHeader>
      <TableBody>
        <TableRow v-for="(schedule, index) in schedules" :key="schedule.id">
          <TD>{{ index + 1 }}</TD>
          <TD class="font-medium">{{ schedule.name }}</TD>
          <TD class="font-mono text-xs">{{ schedule.cron_expression }}</TD>
          <TD>
            <Badge :variant="schedule.enabled ? 'success' : 'secondary'" look="outline" class="text-xs">
              {{ schedule.enabled ? $t('common.enabled') : $t('common.disabled') }}
            </Badge>
          </TD>
          <TD>{{ schedule.next_run_at ?? '—' }}</TD>
          <TD>
            <span>{{ schedule.last_run_at ?? '—' }}</span>
            <Badge
              v-if="schedule.last_status"
              :variant="statusVariant(schedule.last_status)"
              look="outline"
              class="ms-1.5 text-xs"
            >
              {{ schedule.last_status }}
            </Badge>
          </TD>
          <TD class="max-w-xs whitespace-normal break-words">{{ schedule.last_message ?? '—' }}</TD>
          <TD v-if="canManage">
            <div class="flex items-center gap-1">
              <ButtonEdit @click-edit="modalFormRef?.handleModal(true, schedule)" />
              <a
                href="#"
                class="flex items-center"
                :class="{ 'pointer-events-none opacity-40': runningId === schedule.id }"
                @click.prevent="onRunNow(schedule)"
              >
                <Lucide class="mr-1 h-4 w-4" icon="Play" />
                {{ $t('schedules.actionRunNow') }}
              </a>
            </div>
          </TD>
        </TableRow>
      </TableBody>
    </Table>
  </Box>

  <ModalFormSchedule ref="modalFormRef" @submit="refresh" />
</template>
