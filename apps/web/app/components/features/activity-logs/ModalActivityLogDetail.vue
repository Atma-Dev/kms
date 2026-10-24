<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import type ActivityLog from '~/types/entities/activity_log'

const isShowModal = ref(false)
const log = ref<ActivityLog | null>(null)

const handleModal = (isShow: boolean, payload?: ActivityLog) => {
  isShowModal.value = isShow
  if (payload) log.value = payload
}

// Bodies are stored pre-redacted and possibly truncated (see SanitizeBody in
// api-kms), so a truncated one may not parse — fall back to the raw
// string rather than showing nothing.
const pretty = (raw: string | null) => {
  if (!raw) return null
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

defineExpose({ handleModal })
</script>

<template>
  <DialogRoot
    :open="isShowModal"
    :onOpenChange="(d: { open: boolean }) => { if (!d.open) handleModal(false) }"
  >
    <DialogContent class="sm:max-w-2xl">
      <DialogTitle>
        {{ log?.method }} {{ log?.path }}
      </DialogTitle>

      <div v-if="log" class="flex flex-col gap-4 mt-4 max-h-[70vh] overflow-auto">
        <div>
          <p class="text-xs font-semibold uppercase tracking-wider text-foreground/50 mb-1">{{ $t('activityLogs.detailModal.requestLabel') }}</p>
          <pre
            v-if="pretty(log.request_body)"
            class="rounded-lg border border-foreground/10 bg-foreground/5 p-3 text-xs whitespace-pre-wrap break-all"
          >{{ pretty(log.request_body) }}</pre>
          <p v-else class="text-sm text-foreground/40">{{ $t('activityLogs.detailModal.empty') }}</p>
        </div>

        <div>
          <p class="text-xs font-semibold uppercase tracking-wider text-foreground/50 mb-1">{{ $t('activityLogs.detailModal.responseLabel') }}</p>
          <pre
            v-if="pretty(log.response_body)"
            class="rounded-lg border border-foreground/10 bg-foreground/5 p-3 text-xs whitespace-pre-wrap break-all"
          >{{ pretty(log.response_body) }}</pre>
          <p v-else class="text-sm text-foreground/40">{{ $t('activityLogs.detailModal.empty') }}</p>
        </div>
      </div>

      <div class="flex justify-end mt-7">
        <Button variant="primary" @click="handleModal(false)">{{ $t('common.close') }}</Button>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
