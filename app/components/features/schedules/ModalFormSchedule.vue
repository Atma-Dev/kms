<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Label } from '~/base/ui/label'
import { Input } from '~/base/ui/input'
import { Lucide } from '~/base/ui/lucide'
import type ScheduledJob from '~/types/entities/scheduled_job'
import { type ScheduleUpdateRequest, useScheduleService } from '~/services/ScheduleService'

const emit = defineEmits<{ (e: 'submit'): void }>()

const { updateById, submitting } = useScheduleService()
const flashStore = useFlashStore()

const isShowModal = ref(false)
const editingId = ref<string | null>(null)
const editingName = ref('')

const getInitialForm = (): ScheduleUpdateRequest => ({ cron_expression: '', enabled: true })
const form = ref<ScheduleUpdateRequest>(getInitialForm())

const handleModal = (isShow: boolean, schedule?: ScheduledJob) => {
  isShowModal.value = isShow
  flashStore.errors = null

  if (schedule) {
    editingId.value = schedule.id
    editingName.value = schedule.name
    form.value = { cron_expression: schedule.cron_expression, enabled: schedule.enabled }
  } else {
    editingId.value = null
    editingName.value = ''
    form.value = getInitialForm()
  }
}

const onSubmit = async () => {
  if (!editingId.value) return
  if (await updateById(editingId.value, form.value)) {
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
    <DialogContent>
      <DialogTitle>
        <div class="flex items-center justify-between w-full">
          <h2 class="text-base font-medium">{{ $t('schedules.editSchedule', { name: editingName }) }}</h2>
          <button
            type="button"
            class="text-foreground/40 hover:text-foreground/70"
            @click="handleModal(false)"
          >
            <Lucide icon="X" class="w-4 h-4" />
          </button>
        </div>
      </DialogTitle>

      <div class="grid gap-4 mt-6">
        <div>
          <Label for="schedule-cron">{{ $t('schedules.fieldCronExpression') }} <RequiredMark /></Label>
          <Input id="schedule-cron" v-model="form.cron_expression" class="mt-2 font-mono" placeholder="*/5 * * * *" />
          <p class="mt-1 text-xs text-foreground/40">{{ $t('schedules.cronHint') }}</p>
          <FormFeedback feedbackKey="cron_expression" />
        </div>
        <label class="flex items-center gap-2 text-sm">
          <input type="checkbox" v-model="form.enabled" class="size-4 rounded border-foreground/20" />
          {{ $t('schedules.fieldEnabled') }}
        </label>
      </div>

      <div class="flex gap-2 justify-end mt-7">
        <Button look="outline" variant="ghost" @click="handleModal(false)">
          <Lucide icon="X" />
          {{ $t('common.cancel') }}
        </Button>
        <Button variant="primary" :disabled="submitting" @click="onSubmit">
          <Lucide icon="Save" />
          {{ submitting ? $t('common.saving') : $t('common.confirm') }}
        </Button>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
