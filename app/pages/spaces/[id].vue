<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Button } from '~/base/ui/button'
import { Input } from '~/base/ui/input'
import { Lucide } from '~/base/ui/lucide'
import { Badge } from '~/base/ui/badge'
import { useSpaceService } from '~/services/SpaceService'
import { useCategoryService, type CategoryRequest } from '~/services/CategoryService'
import { useDocumentService } from '~/services/DocumentService'
import type Category from '~/types/entities/category'
import type { DocumentStatus } from '~/types/entities/document'
import CategoryTree from '~/components/features/categories/CategoryTree.vue'
import ModalFormCategory from '~/components/features/categories/ModalFormCategory.vue'
import ModalFormDocument from '~/components/features/documents/ModalFormDocument.vue'
import { Permission } from '~/enums/Permission'
import { statusVariant } from '~/utils/document'

definePageMeta({
  title: 'spaces.detailTitle',
  breadcrumb: ['Spaces', 'Detail'],
})

const route = useRoute()
const spaceId = computed(() => route.params.id as string)

const authStore = useAuthStore()
const canAddCategory = computed(() => authStore.hasPermission(Permission.MASTERS_CATEGORIES_STORE))
const canCreateDocument = computed(() => authStore.hasPermission(Permission.DOCUMENTS_STORE))
// The bin answers to the permission that filled it, not to reading.
const canSeeBin = computed(() => authStore.hasPermission(Permission.DOCUMENTS_DESTROY))

const { space, getById } = useSpaceService()
onMounted(() => getById(spaceId.value))

const {
  tree,
  totalDocumentCount,
  uncategorisedCount,
  loading: categoriesLoading,
  submitting: categorySubmitting,
  getTree,
  addNew: addCategory,
  updateById: updateCategory,
  deleteById: deleteCategory,
  reorder,
} = useCategoryService(spaceId)
onMounted(getTree)

const { documents, meta, loading, submitting, getAll, getDeleted, undelete } = useDocumentService()

// The selected category is a filter on the document list, so it belongs in the
// URL with the other filters — a shared link then opens on the same branch.
const selectedCategoryId = computed({
  get: () => filters.category_id || null,
  set: (value: string | null) => {
    filters.category_id = value ?? ''
  },
})

// 'deleted' is not a status — it is a different list entirely, on its own
// endpoint. It sits with the status pills because that is where someone looks
// for "where did my document go", not because it belongs to the same field.
const DELETED = 'deleted'

const statusOptions: { id: DocumentStatus | '' | typeof DELETED; labelKey: string }[] = [
  { id: '', labelKey: 'documents.status.all' },
  { id: 'draft', labelKey: 'documents.status.draft' },
  { id: 'in_review', labelKey: 'documents.status.in_review' },
  { id: 'published', labelKey: 'documents.status.published' },
  { id: 'archived', labelKey: 'documents.status.archived' },
  { id: DELETED, labelKey: 'documents.status.deleted' },
]

const showingBin = computed(() => filters.status === DELETED)

const { filters, hasActiveFilters, reset, refresh } = useServerList({
  filters: { search: '', status: '', category_id: '', sort: 'updated_at', direction: 'desc' },
  debounce: ['search'],
  fetch: (page, perPage, applied) =>
    applied.status === DELETED
      ? getDeleted(spaceId.value, page, perPage)
      : getAll(spaceId.value, page, perPage, applied),
})

const toggleCategory = (id: string | null) => {
  // Clicking the selected category clears the filter, so the tree doubles as
  // its own reset without a second control.
  selectedCategoryId.value = selectedCategoryId.value === id ? null : id
}

const categoryModal = ref<InstanceType<typeof ModalFormCategory> | null>(null)
const documentModal = ref<InstanceType<typeof ModalFormDocument> | null>(null)

const onCreateCategory = async (request: CategoryRequest) => {
  if (await addCategory(request)) {
    categoryModal.value?.close()
    await getTree()
  }
}

const onUpdateCategory = async (id: string, request: CategoryRequest) => {
  if (await updateCategory(id, request)) {
    categoryModal.value?.close()
    await getTree()
  }
}

const onRemoveCategory = async (category: Category) => {
  if (await deleteCategory(category.id)) {
    if (selectedCategoryId.value === category.id) selectedCategoryId.value = null
    await getTree()
    refresh()
  }
}

/** Finds the sibling list a category belongs to, so a move reorders the right level. */
const siblingsOf = (parentId: string | null): Category[] => {
  if (parentId === null) return tree.value
  const walk = (nodes: Category[]): Category[] | null => {
    for (const node of nodes) {
      if (node.id === parentId) return node.children
      const found = walk(node.children)
      if (found) return found
    }
    return null
  }
  return walk(tree.value) ?? []
}

const onMoveCategory = async (category: Category, direction: -1 | 1) => {
  const siblings = siblingsOf(category.parent_id)
  const index = siblings.findIndex((c) => c.id === category.id)
  const target = index + direction
  if (index < 0 || target < 0 || target >= siblings.length) return

  // Sends the whole ordered list, which is what the API takes — it renumbers
  // the level in one transaction rather than trusting two positions to agree.
  const ids = siblings.map((c) => c.id)
  ;[ids[index], ids[target]] = [ids[target]!, ids[index]!]
  await reorder(category.parent_id, ids)
}

const onDocumentCreated = (id: string) => navigateTo(`/documents/${id}`)

const onUndelete = async (id: string) => {
  if (await undelete(id)) {
    await Promise.all([getTree(), refresh()])
  }
}
</script>

<template>
  <div class="grid grid-cols-12 gap-6">
    <div class="col-span-12">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-3">
          <NuxtLink to="/spaces" class="rounded-lg p-2 text-foreground/50 hover:bg-foreground/5">
            <Lucide icon="ArrowLeft" class="size-4" />
          </NuxtLink>
          <div>
            <h2 class="text-lg font-medium">{{ space?.name ?? '—' }}</h2>
            <p class="text-xs text-foreground/40">{{ space?.description || $t('spaces.noDescription') }}</p>
          </div>
          <Badge v-if="space?.is_archived" variant="secondary" look="outline" class="text-[10px]">
            {{ $t('spaces.archived') }}
          </Badge>
        </div>

        <Button v-if="canCreateDocument" variant="primary" @click="documentModal?.handleModal(true)">
          <Lucide icon="Plus" />
          {{ $t('documents.createDocument') }}
        </Button>
      </div>
    </div>

    <Box class="col-span-12 lg:col-span-3">
      <div class="mb-2 flex items-center justify-between">
        <p class="text-sm font-semibold">{{ $t('categories.title') }}</p>
        <button
          v-if="canAddCategory"
          type="button"
          class="rounded p-1.5 text-foreground/50 hover:bg-foreground/5"
          :aria-label="$t('categories.createCategory')"
          :title="$t('categories.createCategory')"
          @click="categoryModal?.handleModal(true)"
        >
          <Lucide icon="Plus" class="size-4" />
        </button>
      </div>

      <button
        type="button"
        class="mb-1 flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm"
        :class="!selectedCategoryId ? 'bg-primary/10 text-primary' : 'hover:bg-foreground/5'"
        @click="selectedCategoryId = null"
      >
        <Lucide icon="Layers" class="size-4 opacity-60" />
        {{ $t('categories.allDocuments') }}
        <span
          v-if="totalDocumentCount > 0"
          class="ml-auto rounded-full bg-foreground/5 px-1.5 py-0.5 text-[10px] tabular-nums text-foreground/45"
        >
          {{ totalDocumentCount }}
        </span>
      </button>

      <!-- Documents filed under no category are reachable from "all documents"
           and nowhere else, so the number is worth stating rather than leaving
           it to be found by subtraction. -->
      <p v-if="uncategorisedCount > 0" class="mb-1 px-2 text-[11px] text-foreground/35">
        {{ $t('categories.uncategorised', { count: uncategorisedCount }) }}
      </p>

      <div v-if="categoriesLoading" class="px-2 py-3 text-sm text-foreground/40">{{ $t('common.loading') }}</div>
      <CategoryTree
        v-else
        :nodes="tree"
        :selected-id="selectedCategoryId"
        @select="toggleCategory"
        @add-child="(parent) => categoryModal?.handleModal(true, { parent })"
        @edit="(category) => categoryModal?.handleModal(true, { category })"
        @remove="onRemoveCategory"
        @move="onMoveCategory"
      />
    </Box>

    <Box class="col-span-12 lg:col-span-9">
      <div class="mb-4 flex flex-wrap items-center gap-2">
        <div class="relative">
          <Input
            v-model="filters.search"
            type="text"
            :placeholder="$t('documents.searchPlaceholder')"
            class="w-56 pr-8"
          />
          <Lucide class="absolute inset-y-0 right-0 my-auto mr-2.5 size-4 text-foreground/40" icon="Search" />
        </div>

        <Button
          v-for="option in statusOptions"
          v-show="option.id !== DELETED || canSeeBin"
          :key="option.id"
          size="sm"
          :look="filters.status === option.id ? 'filled' : 'outline'"
          :variant="filters.status === option.id ? 'primary' : 'ghost'"
          @click="filters.status = option.id"
        >
          {{ $t(option.labelKey) }}
        </Button>

        <Button v-if="hasActiveFilters" look="outline" variant="ghost" size="sm" class="ml-auto" @click="reset">
          <Lucide class="mr-1.5 size-4" icon="X" />
          {{ $t('common.reset') }}
        </Button>
      </div>

      <ServerSidePagination :meta="meta" :loading="loading">
        <div class="divide-y divide-foreground/5">
          <component
            :is="showingBin ? 'div' : NuxtLink"
            v-for="doc in documents"
            :key="doc.id"
            :to="showingBin ? undefined : `/documents/${doc.id}`"
            class="flex items-start gap-3 py-3 transition"
            :class="showingBin ? '' : 'hover:bg-foreground/[0.02]'"
          >
            <Lucide
              :icon="doc.is_pinned ? 'Pin' : 'FileText'"
              class="mt-0.5 size-4 shrink-0"
              :class="doc.is_pinned ? 'text-primary' : 'text-foreground/30'"
            />
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <span class="truncate font-medium">{{ doc.title }}</span>
                <Badge :variant="statusVariant(doc.status)" look="outline" class="text-[10px]">
                  {{ $t(`documents.status.${doc.status}`) }}
                </Badge>
              </div>
              <p class="mt-0.5 line-clamp-1 text-sm text-foreground/50">{{ doc.excerpt || '—' }}</p>
            </div>
            <div class="shrink-0 text-right text-xs text-foreground/40">
              <div>{{ doc.updated_at }}</div>
              <div class="mt-0.5 flex items-center justify-end gap-1">
                <Lucide icon="Eye" class="size-3" />
                {{ doc.view_count }}
              </div>
            </div>

            <Button
              v-if="showingBin"
              size="sm"
              look="outline"
              variant="ghost"
              class="shrink-0"
              :disabled="submitting"
              @click="onUndelete(doc.id)"
            >
              <Lucide icon="ArchiveRestore" class="mr-1.5 size-4" />
              {{ $t('documents.restoreDeleted') }}
            </Button>
          </component>
        </div>

        <div v-if="!loading && !documents.length" class="py-12 text-center text-sm text-foreground/40">
          {{ showingBin ? $t('documents.binEmpty') : $t('documents.empty') }}
        </div>
      </ServerSidePagination>
    </Box>

    <ModalFormCategory
      ref="categoryModal"
      :submitting="categorySubmitting"
      @create="onCreateCategory"
      @update="onUpdateCategory"
    />
    <ModalFormDocument ref="documentModal" :space-id="spaceId" :tree="tree" @created="onDocumentCreated" />
  </div>
</template>
