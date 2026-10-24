<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Button } from '~/base/ui/button'
import { Input } from '~/base/ui/input'
import { Lucide } from '~/base/ui/lucide'
import { Badge } from '~/base/ui/badge'
import { useSearchService } from '~/services/SearchService'
import { useLookupService } from '~/services/LookupService'
import { statusVariant } from '~/utils/document'

definePageMeta({
  title: 'search.title',
  pageSubTitle: 'search.subtitle',
  breadcrumb: ['Search'],
})

const { results, meta, loading, getAll } = useSearchService()

const { filters, hasActiveFilters, reset } = useServerList({
  filters: { q: '', space_id: '', category_id: '', tag: '', author_id: '', from: '', to: '', sort: '' },
  debounce: ['q'],
  fetch: (page, perPage, applied) => getAll(page, perPage, applied),
})

// Each dropdown answers to its own lookups.* permission, which is smaller
// than the right to manage the resource behind it — so a filter the caller
// may not fill is hidden rather than requested and 403'd.
const { options: spaceOptions, can: canFilterBySpace, load: loadSpaces } = useLookupService('spaces')
const { options: tagOptions, can: canFilterByTag, load: loadTags } = useLookupService('tags')
const { options: authorOptions, can: canFilterByAuthor, load: loadAuthors } = useLookupService('users')

// Categories belong to one space, so the lookup is handed the filter's own
// space id by reference and reloaded when it changes — offering another
// space's categories would be a filter that can never match.
const categoryQuery = reactive({ space_id: '' })
const {
  options: categoryOptions,
  can: canFilterByCategory,
  load: loadCategories,
} = useLookupService('categories', categoryQuery)

watch(
  () => filters.space_id,
  (spaceId) => {
    categoryQuery.space_id = spaceId
    filters.category_id = ''
    if (spaceId) loadCategories()
  },
)

onMounted(() => {
  loadSpaces()
  loadTags()
  loadAuthors()
  // A shared link can arrive with a space already filtered, and its categories
  // have to be loaded for the second dropdown to show what it is set to.
  if (filters.space_id) {
    categoryQuery.space_id = filters.space_id
    loadCategories()
  }
})

// Relevance only means something when something was searched for. With an
// empty box the API sorts by newest whatever this says, so the option is
// hidden rather than offered and quietly ignored.
const sortOptions = computed(() => [
  ...(filters.q ? [{ value: '', label: 'search.sortRelevance' }] : []),
  { value: 'newest', label: 'search.sortNewest' },
  { value: 'popular', label: 'search.sortPopular' },
])
</script>

<template>
  <div class="grid grid-cols-12 gap-6">
    <Box class="col-span-12">
      <div class="mb-5 flex flex-wrap items-center gap-3">
        <div class="relative min-w-0 flex-1">
          <Lucide class="absolute inset-y-0 left-0 my-auto ml-3 size-4 text-foreground/40" icon="Search" />
          <Input
            v-model="filters.q"
            type="search"
            :placeholder="$t('search.placeholder')"
            class="w-full pl-9"
            autofocus
          />
        </div>

        <div class="flex items-center gap-1 rounded-lg border border-foreground/10 p-0.5">
          <button
            v-for="option in sortOptions"
            :key="option.value"
            type="button"
            class="rounded-md px-2.5 py-1.5 text-xs font-medium transition"
            :class="
              filters.sort === option.value
                ? 'bg-primary/10 text-primary'
                : 'text-foreground/50 hover:text-foreground'
            "
            @click="filters.sort = option.value"
          >
            {{ $t(option.label) }}
          </button>
        </div>

        <FilterPopover :active="hasActiveFilters">
          <TomSelect
            v-if="canFilterBySpace"
            v-model="filters.space_id"
            :options="spaceOptions"
            :placeholder="$t('search.allSpaces')"
          />
          <TomSelect
            v-if="canFilterByCategory && filters.space_id"
            v-model="filters.category_id"
            :options="categoryOptions"
            :placeholder="$t('search.allCategories')"
          />
          <TomSelect
            v-if="canFilterByTag"
            v-model="filters.tag"
            :options="tagOptions"
            :placeholder="$t('search.allTags')"
          />
          <TomSelect
            v-if="canFilterByAuthor"
            v-model="filters.author_id"
            :options="authorOptions"
            :placeholder="$t('search.allAuthors')"
          />
          <Input v-model="filters.from" type="date" class="w-full" :aria-label="$t('search.updatedFrom')" />
          <Input v-model="filters.to" type="date" class="w-full" :aria-label="$t('search.updatedTo')" />

          <template #footer>
            <Button look="outline" variant="ghost" size="sm" :disabled="!hasActiveFilters" @click="reset">
              <Lucide class="mr-1.5 size-4" icon="X" />
              {{ $t('common.reset') }}
            </Button>
          </template>
        </FilterPopover>
      </div>

      <ServerSidePagination :meta="meta" :loading="loading">
        <div class="flex flex-col divide-y divide-foreground/5">
          <NuxtLink
            v-for="result in results"
            :key="result.id"
            :to="`/documents/${result.id}`"
            class="group -mx-2 rounded-lg px-2 py-3.5 transition hover:bg-foreground/[0.03]"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="truncate font-medium group-hover:text-primary">{{ result.title }}</div>
                <!-- The API escapes the document's text and adds the <mark>
                     itself; re-highlighting here would need the client to know
                     how the query was tokenised, which only the index does. -->
                <p class="mt-1 line-clamp-2 text-sm text-foreground/60 [&_mark]:bg-primary/20 [&_mark]:text-inherit" v-html="result.snippet" />
              </div>
              <Badge :variant="statusVariant(result.status)" look="outline" class="shrink-0 text-[10px]">
                {{ $t(`documents.status.${result.status}`) }}
              </Badge>
            </div>

            <div class="mt-2 flex items-center gap-3 text-xs text-foreground/40">
              <span class="flex items-center gap-1">
                <Lucide icon="Eye" class="size-3.5" />
                {{ result.view_count }}
              </span>
              <span class="flex items-center gap-1">
                <Lucide icon="Clock" class="size-3.5" />
                {{ result.updated_at }}
              </span>
            </div>
          </NuxtLink>
        </div>

        <div v-if="!loading && !results.length" class="py-14 text-center">
          <Lucide icon="SearchX" class="mx-auto size-8 text-foreground/20" />
          <p class="mt-3 text-sm text-foreground/40">
            {{ filters.q ? $t('search.noResults', { query: filters.q }) : $t('search.startTyping') }}
          </p>
        </div>
      </ServerSidePagination>
    </Box>
  </div>
</template>
