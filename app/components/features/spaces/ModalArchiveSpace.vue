<script setup lang="ts">
import { DialogCloseTrigger, DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Lucide } from '~/base/ui/lucide'
import type Space from '~/types/entities/space'
import { useSpaceService } from '~/services/SpaceService'

const emit = defineEmits<{ (e: 'archived'): void }>()

const { archiveById, submitting } = useSpaceService()

const isShowModal = ref(false)
const selected = ref<Space | null>(null)

const handleModal = (isShow: boolean, space?: Space) => {
  isShowModal.value = isShow
  selected.value = space ?? null
}

const onConfirm = async () => {
  if (!selected.value) return
  if (await archiveById(selected.value.id)) {
    handleModal(false)
    emit('archived')
  }
}

defineExpose({ handleModal })
</script>

<template>
  <DialogRoot :open="isShowModal" :onOpenChange="(d: { open: boolean }) => { isShowModal = d.open }">
    <DialogContent class="sm:max-w-md">
      <DialogTitle>{{ $t('spaces.archiveTitle') }}</DialogTitle>

      <div class="flex flex-col items-center gap-4 py-4 text-center">
        <div class="flex h-14 w-14 items-center justify-center rounded-full bg-danger/10">
          <Lucide icon="Archive" class="h-7 w-7 text-danger" />
        </div>
        <p class="text-sm text-foreground/70">
          {{ $t('spaces.archiveConfirm', { space: selected?.name }) }}
        </p>
        <!-- Says what archiving does, because "delete" is what the button next
             to it usually means: the documents stay readable. -->
        <p class="text-xs text-foreground/40">{{ $t('spaces.archiveHint') }}</p>
      </div>

      <div class="mt-2 flex justify-end gap-2">
        <DialogCloseTrigger as-child @click="handleModal(false)">
          <Button look="outline" variant="ghost">{{ $t('common.cancel') }}</Button>
        </DialogCloseTrigger>
        <Button variant="danger" :disabled="submitting" @click="onConfirm">
          <Lucide icon="Archive" class="mr-1.5 h-4 w-4" />
          {{ $t('spaces.archive') }}
        </Button>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
