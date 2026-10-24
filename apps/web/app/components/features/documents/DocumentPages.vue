<script setup lang="ts">
import { Lucide } from '~/base/ui/lucide'
import { useProtectionService } from '~/services/ProtectionService'

/**
 * The strict viewer (FR-3): a document as page images rendered by the server,
 * with the watermark already in the pixels.
 *
 * There is no text here to select, and the watermark cannot be stripped
 * without stripping the words with it — unlike the overlay in
 * DocumentProtection, which is a deterrent in the reader's own browser.
 *
 * The cost is real and worth naming: no text search inside the page, no screen
 * reader, no reflow on a narrow screen. That is why strict is a per-document
 * decision rather than the default.
 */
const props = defineProps<{
  documentId: string
  /** Pages already in hand — the shared-link page has them in its response. */
  pages?: string[]
}>()

const { getPages, loading } = useProtectionService()
const rendered = ref<string[]>(props.pages ?? [])

onMounted(async () => {
  if (props.pages?.length) return
  rendered.value = await getPages(props.documentId)
})
</script>

<template>
  <div>
    <p v-if="loading && !rendered.length" class="py-16 text-center text-sm text-foreground/40">
      {{ $t('documents.protection.rendering') }}
    </p>

    <p v-else-if="!rendered.length" class="py-16 text-center text-sm text-foreground/40">
      {{ $t('documents.protection.renderFailed') }}
    </p>

    <div v-else class="grid gap-4">
      <!-- draggable=false and the blocked context menu are the same basic
           deterrent as the text viewer's: they stop a drag-to-desktop and a
           right-click save, and nothing more. -->
      <img
        v-for="(page, index) in rendered"
        :key="index"
        :src="page"
        :alt="$t('documents.protection.pageAlt', { page: index + 1 })"
        class="w-full rounded-xl border border-foreground/10 shadow-sm"
        draggable="false"
        @contextmenu.prevent
        @dragstart.prevent
      />

      <p class="flex items-center justify-center gap-1.5 text-xs text-foreground/40">
        <Lucide icon="ShieldCheck" class="size-3.5" />
        {{ $t('documents.protection.pagesFooter', { count: rendered.length }) }}
      </p>
    </div>
  </div>
</template>
