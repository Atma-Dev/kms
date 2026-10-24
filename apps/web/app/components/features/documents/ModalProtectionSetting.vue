<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Label } from '~/base/ui/label'
import { Lucide } from '~/base/ui/lucide'
import { useProtectionService } from '~/services/ProtectionService'
import type { ProtectionLevel } from '~/types/entities/protection'
import type { DocumentDetail } from '~/types/entities/document'

/**
 * The per-document protection level (FR-6).
 *
 * Three choices, not two: "inherit" is the absence of an override, which is
 * what puts the document back on its space's default. A form that only offered
 * normal and strict would have no way to undo a setting.
 */
const props = defineProps<{
  documentId: string
  /** The document's own override, null when it inherits. */
  current: ProtectionLevel | null
  /** What it resolves to today — shown so "inherit" is not a mystery. */
  effective: ProtectionLevel
}>()
const emit = defineEmits<{ (e: 'saved', document: DocumentDetail): void }>()

const { setDocumentLevel, submitting } = useProtectionService()

const isShowModal = ref(false)
// TomSelect emits a string, so the override is held as one and cast on the way
// out: '' is the wire form of "inherit".
const level = ref('')

const { t } = useI18n()
const levelOptions = computed(() => [
  { id: '', name: t('documents.protection.inherit', { level: t(`documents.protection.${props.effective}`) }) },
  { id: 'normal', name: t('documents.protection.normal') },
  { id: 'strict', name: t('documents.protection.strict') },
])

const handleModal = (isShow: boolean) => {
  isShowModal.value = isShow
  if (isShow) level.value = props.current ?? ''
}

const save = async () => {
  const saved = await setDocumentLevel(props.documentId, (level.value || null) as ProtectionLevel | null)
  if (!saved) return
  emit('saved', saved)
  handleModal(false)
}

defineExpose({ handleModal })
</script>

<template>
  <DialogRoot :open="isShowModal" :onOpenChange="(d: { open: boolean }) => handleModal(d.open)">
    <DialogContent class="sm:max-w-md">
      <DialogTitle>
        <div class="flex w-full items-center justify-between">
          <h2 class="text-base font-medium">{{ $t('documents.protection.title') }}</h2>
          <button type="button" class="text-foreground/40 hover:text-foreground/70" @click="handleModal(false)">
            <Lucide icon="X" class="h-4 w-4" />
          </button>
        </div>
      </DialogTitle>

      <div class="mt-4">
        <Label>{{ $t('documents.protection.levelLabel') }}</Label>
        <div class="mt-2">
          <TomSelect v-model="level" :options="levelOptions" :placeholder="$t('documents.protection.levelPlaceholder')" />
        </div>
        <FormFeedback feedbackKey="protection_level" />
      </div>

      <!-- Says what each level really buys, because the difference is exactly
           where a protection feature gets oversold. -->
      <ul class="mt-4 grid gap-2 text-xs text-foreground/60">
        <li class="rounded-xl bg-foreground/5 px-3 py-2">
          <span class="font-medium text-foreground/80">{{ $t('documents.protection.normal') }}</span> —
          {{ $t('documents.protection.normalExplain') }}
        </li>
        <li class="rounded-xl bg-foreground/5 px-3 py-2">
          <span class="font-medium text-foreground/80">{{ $t('documents.protection.strict') }}</span> —
          {{ $t('documents.protection.strictExplain') }}
        </li>
      </ul>

      <div class="mt-5 flex justify-end gap-2">
        <Button look="outline" variant="ghost" :disabled="submitting" @click="handleModal(false)">
          {{ $t('common.cancel') }}
        </Button>
        <Button variant="primary" :disabled="submitting" @click="save">
          {{ submitting ? $t('common.saving') : $t('common.save') }}
        </Button>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
