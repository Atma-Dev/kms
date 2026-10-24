<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Label } from '~/base/ui/label'
import { Textarea } from '~/base/ui/textarea'
import { Lucide } from '~/base/ui/lucide'
import type { DocumentAction } from '~/types/entities/document'

const props = defineProps<{ submitting: boolean }>()
const emit = defineEmits<{ (e: 'confirm', action: DocumentAction, reason: string): void }>()

const isShowModal = ref(false)
const action = ref<DocumentAction>('reject')
const reason = ref('')

const handleModal = (isShow: boolean, nextAction?: DocumentAction) => {
  isShowModal.value = isShow
  reason.value = ''
  if (nextAction) action.value = nextAction
}

const onConfirm = () => emit('confirm', action.value, reason.value)

defineExpose({ handleModal })
</script>

<template>
  <DialogRoot :open="isShowModal" :onOpenChange="(d: { open: boolean }) => { isShowModal = d.open }">
    <DialogContent class="sm:max-w-md">
      <DialogTitle>{{ $t('documents.rejectTitle') }}</DialogTitle>

      <div class="mt-4">
        <Label for="reject-reason">{{ $t('documents.reasonLabel') }} <RequiredMark /></Label>
        <Textarea
          id="reject-reason"
          v-model="reason"
          class="mt-2"
          rows="4"
          :placeholder="$t('documents.reasonPlaceholder')"
        />
        <!-- The API refuses a rejection with no reason: sending one back with
             nothing to act on defeats the point of a review. -->
        <p class="mt-1 text-xs text-foreground/40">{{ $t('documents.reasonHint') }}</p>
        <FormFeedback feedbackKey="reason" />
      </div>

      <div class="mt-6 flex justify-end gap-2">
        <Button look="outline" variant="ghost" @click="handleModal(false)">{{ $t('common.cancel') }}</Button>
        <Button variant="danger" :disabled="props.submitting || !reason.trim()" @click="onConfirm">
          <Lucide icon="X" class="mr-1.5 h-4 w-4" />
          {{ $t('documents.action.reject') }}
        </Button>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
