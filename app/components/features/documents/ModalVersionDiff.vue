<script setup lang="ts">
import { DialogContent, DialogRoot, DialogTitle } from '~/base/ui/dialog'
import { Button } from '~/base/ui/button'
import { Lucide } from '~/base/ui/lucide'
import type { DiffLine, DocumentVersion } from '~/types/entities/document'

const props = defineProps<{ versions: DocumentVersion[] }>()
const emit = defineEmits<{ (e: 'compare', from: string, to: string): void }>()

const isShowModal = ref(false)
const from = ref('')
const to = ref('')
const lines = ref<DiffLine[]>([])
const loading = ref(false)

/**
 * `current` is offered alongside the numbered versions: snapshots are taken
 * *before* an overwrite, so the live content is not itself a version row and
 * "what changed since v3" would otherwise be unanswerable.
 */
const CURRENT = 'current'

const options = computed(() => [
  { id: CURRENT, name: 'Current' },
  ...props.versions.map((v) => ({ id: String(v.version_number), name: `v${v.version_number}` })),
])

const handleModal = (isShow: boolean) => {
  isShowModal.value = isShow
  lines.value = []
  if (!isShow) return
  // Defaults to the most useful comparison: the newest snapshot against what
  // is live now.
  from.value = props.versions.length ? String(props.versions[0]!.version_number) : CURRENT
  to.value = CURRENT
  run()
}

const run = () => {
  loading.value = true
  emit('compare', from.value, to.value)
}

const setResult = (result: DiffLine[]) => {
  lines.value = result
  loading.value = false
}

const opStyle = (op: DiffLine['op']) =>
  ({
    add: 'bg-success/10 text-success',
    remove: 'bg-danger/10 text-danger line-through decoration-danger/40',
    equal: 'text-foreground/60',
  })[op]

const opSign = (op: DiffLine['op']) => ({ add: '+', remove: '−', equal: ' ' })[op]

const changeCount = computed(() => lines.value.filter((l) => l.op !== 'equal').length)

defineExpose({ handleModal, setResult })
</script>

<template>
  <DialogRoot :open="isShowModal" :onOpenChange="(d: { open: boolean }) => { isShowModal = d.open }">
    <DialogContent class="sm:max-w-3xl">
      <DialogTitle>
        <div class="flex w-full items-center justify-between">
          <h2 class="text-base font-medium">{{ $t('documents.compareTitle') }}</h2>
          <button type="button" class="text-foreground/40 hover:text-foreground/70" @click="handleModal(false)">
            <Lucide icon="X" class="h-4 w-4" />
          </button>
        </div>
      </DialogTitle>

      <div class="mt-4 flex flex-wrap items-end gap-3">
        <div class="w-32">
          <label class="text-xs text-foreground/50">{{ $t('documents.compareFrom') }}</label>
          <TomSelect v-model="from" class="mt-1" :options="options" />
        </div>
        <Lucide icon="ArrowRight" class="mb-2.5 size-4 text-foreground/30" />
        <div class="w-32">
          <label class="text-xs text-foreground/50">{{ $t('documents.compareTo') }}</label>
          <TomSelect v-model="to" class="mt-1" :options="options" />
        </div>
        <Button size="sm" variant="primary" :disabled="loading" @click="run">
          <Lucide icon="GitCompare" class="mr-1.5 size-4" />
          {{ $t('documents.compare') }}
        </Button>
        <span class="mb-2 ml-auto text-xs text-foreground/40">
          {{ $t('documents.changeCount', { count: changeCount }) }}
        </span>
      </div>

      <div class="mt-4 max-h-[55vh] overflow-auto rounded-xl border border-foreground/10">
        <div v-if="loading" class="p-6 text-center text-sm text-foreground/40">{{ $t('common.loading') }}</div>
        <div v-else-if="!lines.length" class="p-6 text-center text-sm text-foreground/40">
          {{ $t('documents.noChanges') }}
        </div>
        <div v-else class="divide-y divide-foreground/5 font-mono text-xs">
          <div v-for="(line, index) in lines" :key="index" class="flex gap-3 px-3 py-1.5" :class="opStyle(line.op)">
            <span class="w-8 shrink-0 select-none text-right opacity-40">{{ line.from_line ?? '' }}</span>
            <span class="w-8 shrink-0 select-none text-right opacity-40">{{ line.to_line ?? '' }}</span>
            <span class="w-3 shrink-0 select-none">{{ opSign(line.op) }}</span>
            <span class="whitespace-pre-wrap break-words">{{ line.text }}</span>
          </div>
        </div>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
