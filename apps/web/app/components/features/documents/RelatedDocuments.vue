<script setup lang="ts">
import { Lucide } from '~/base/ui/lucide'
import { useDocumentService } from '~/services/DocumentService'
import type Document from '~/types/entities/document'

const { documentId } = defineProps<{ documentId: string }>()

const { getRelated } = useDocumentService()
const related = ref<Document[]>([])
const loading = ref(true)

// Refetched when the route moves between documents: clicking a suggestion
// keeps this component mounted, and without the watch the panel would keep
// recommending the document that was left behind.
watch(
  () => documentId,
  async (id) => {
    loading.value = true
    related.value = await getRelated(id)
    loading.value = false
  },
  { immediate: true },
)
</script>

<template>
  <div>
    <p class="mb-3 text-sm font-semibold">{{ $t('documents.relatedTitle') }}</p>

    <p v-if="loading" class="text-sm text-foreground/40">{{ $t('common.loading') }}</p>

    <p v-else-if="!related.length" class="text-sm text-foreground/40">
      {{ $t('documents.noRelated') }}
    </p>

    <ul v-else class="divide-y divide-foreground/5">
      <li v-for="doc in related" :key="doc.id">
        <NuxtLink
          :to="`/documents/${doc.id}`"
          class="group -mx-1 flex items-start gap-2 rounded-lg px-1 py-2.5 transition hover:bg-foreground/[0.03]"
        >
          <Lucide icon="FileText" class="mt-0.5 size-4 shrink-0 text-foreground/30" />
          <div class="min-w-0">
            <div class="truncate text-sm font-medium group-hover:text-primary">{{ doc.title }}</div>
            <p v-if="doc.excerpt" class="mt-0.5 line-clamp-2 text-xs text-foreground/50">{{ doc.excerpt }}</p>
          </div>
        </NuxtLink>
      </li>
    </ul>
  </div>
</template>
