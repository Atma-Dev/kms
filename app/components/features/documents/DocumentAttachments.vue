<script setup lang="ts">
import { Button } from '~/base/ui/button'
import { Lucide } from '~/base/ui/lucide'
import { useAttachmentService } from '~/services/AttachmentService'
import { useAttachmentUrl } from '~/composables/useAttachmentUrl'

const props = defineProps<{ documentId: string; canEdit: boolean }>()

const { attachments, loading, uploading, getByDocument, upload, deleteById } = useAttachmentService()
const fileInput = ref<HTMLInputElement | null>(null)
const { token, resolve } = useAttachmentUrl()

const load = () => getByDocument(props.documentId)
onMounted(load)
watch(() => props.documentId, load)

const onPick = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  // Reset first: picking the same file twice in a row fires no change event
  // otherwise, and the second upload would silently do nothing.
  input.value = ''
  if (!file) return
  if (await upload(props.documentId, file)) await load()
}

const onDelete = async (id: string) => {
  if (await deleteById(id)) await load()
}

/** Bytes to something a person reads. */
const humanSize = (bytes: number): string => {
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit++
  }
  return `${value.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`
}

const iconFor = (type: string) => {
  if (type.startsWith('image/')) return 'Image'
  if (type === 'application/pdf') return 'FileText'
  if (type.includes('sheet') || type.includes('excel') || type === 'text/csv') return 'Sheet'
  if (type.includes('zip')) return 'FileArchive'
  return 'File'
}
</script>

<template>
  <div>
    <div class="mb-3 flex items-center justify-between">
      <p class="text-sm font-semibold">{{ $t('documents.attachmentsTitle') }}</p>
      <template v-if="canEdit">
        <input ref="fileInput" type="file" class="hidden" @change="onPick" />
        <Button size="sm" look="outline" variant="ghost" :disabled="uploading" @click="fileInput?.click()">
          <Lucide icon="Paperclip" class="mr-1.5 size-4" />
          {{ uploading ? $t('documents.uploading') : $t('documents.attachFile') }}
        </Button>
      </template>
    </div>

    <p v-if="loading" class="text-sm text-foreground/40">{{ $t('common.loading') }}</p>
    <p v-else-if="!attachments.length" class="text-sm text-foreground/40">
      {{ $t('documents.noAttachments') }}
    </p>

    <ul v-else class="divide-y divide-foreground/5">
      <li v-for="file in attachments" :key="file.id" class="flex items-center gap-2 py-2">
        <Lucide :icon="iconFor(file.file_type)" class="size-4 shrink-0 text-foreground/40" />
        <!-- The store is not a static folder: this URL is an endpoint that
             checks the document first, so it needs the app's own auth. -->
        <a
          :href="resolve(file.url)"
          class="min-w-0 flex-1 truncate text-sm hover:text-primary hover:underline"
          :title="file.file_name"
          :class="token ? '' : 'pointer-events-none opacity-60'"
          download
        >
          {{ file.file_name }}
        </a>
        <span class="shrink-0 text-xs text-foreground/35">{{ humanSize(file.file_size) }}</span>
        <button
          v-if="canEdit"
          type="button"
          class="shrink-0 rounded p-1 text-foreground/40 hover:bg-danger/10 hover:text-danger"
          :aria-label="$t('common.delete')"
          :title="$t('common.delete')"
          @click="onDelete(file.id)"
        >
          <Lucide icon="Trash2" class="size-3.5" />
        </button>
      </li>
    </ul>
  </div>
</template>
