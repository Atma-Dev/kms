<script setup lang="ts">
import { DialogCloseTrigger, DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Lucide } from '~/base/ui/lucide'
import { ref } from 'vue'

const emit = defineEmits<{
  (e: 'submit'): void
}>()
const isShowModal = ref(false)
const handleModal = (isShow: boolean) => {
  isShowModal.value = isShow
}
const onConfirm = () => {
  emit('submit')
  handleModal(false)
}
defineExpose({ handleModal })
</script>

<template>
  <DialogRoot
    :open="isShowModal"
    :onOpenChange="
      (details: { open: boolean }) => {
        isShowModal = details.open
      }
    "
  >
    <DialogContent>
      <DialogTitle>{{ $t('common.deleteTitle') }}</DialogTitle>
      <div class="grid gap-4 mt-4">{{ $t('common.deleteConfirmMessage') }}</div>
      <div class="flex gap-2 justify-end mt-7">
        <DialogCloseTrigger @click="handleModal(false)">
          <Lucide icon="SquareX" />
          {{ $t('common.close') }}
        </DialogCloseTrigger>
        <Button variant="primary" @click="onConfirm()">
          <Lucide icon="Save" />
          {{ $t('common.confirm') }}
        </Button>
      </div>
      <DialogCloseTrigger />
    </DialogContent>
  </DialogRoot>
</template>

<style scoped></style>
