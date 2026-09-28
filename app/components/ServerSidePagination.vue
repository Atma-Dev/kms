<script setup lang="ts">
import { Box } from '~/base/ui/box'
import {
  PaginationContext,
  PaginationEllipsis,
  PaginationItem,
  PaginationNextTrigger,
  PaginationPrevTrigger,
  PaginationRoot,
} from '~/base/ui/pagination'
import { NativeSelect, NativeSelectOption } from '~/base/ui/native-select'

import type { ServerPaginationMeta } from '~/types/response'

const route = useRoute()

withDefaults(
  defineProps<{
    meta: ServerPaginationMeta
    loading?: boolean
    isWithBox?: boolean
    perPageOptions?: number[]
  }>(),
  {
    loading: false,
    isWithBox: false,
    perPageOptions: () => [10, 20, 50, 100],
  },
)

const currentPerPage = computed(() => Number(route.query.per_page) || 10)

const handlePageChange = (details: { page: number }) => {
  navigateTo({ query: { ...route.query, page: details.page } })
}

const handlePerPageChange = (event: Event) => {
  const value = (event.target as HTMLSelectElement).value
  navigateTo({ query: { ...route.query, page: 1, per_page: value } })
}
</script>

<template>
  <component :is="isWithBox ? Box : 'div'">
    <!-- Initial loading (no data yet) -->
    <div
      v-if="loading && meta.total === 0"
      class="flex items-center justify-center gap-3 py-16 text-foreground/50"
    >
      <svg class="h-5 w-5 animate-spin" viewBox="0 0 24 24" fill="none">
        <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
        <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
      </svg>
      <span class="text-sm">{{ $t('common.loading') }}</span>
    </div>

    <!-- Empty -->
    <EmptyData v-else-if="!loading && meta.total === 0" />

    <!-- Content + Pagination -->
    <template v-else>
      <div class="relative overflow-x-auto">
        <!-- Page-change loading overlay -->
        <div
          v-if="loading"
          class="absolute inset-0 z-10 flex items-center justify-center bg-background/60 backdrop-blur-[1px]"
        >
          <svg class="h-5 w-5 animate-spin text-foreground/50" viewBox="0 0 24 24" fill="none">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
          </svg>
        </div>
        <slot />
      </div>

      <div class="mt-4 flex flex-wrap items-center justify-between gap-3 px-2">
        <PaginationRoot
          :key="`page-${Number(route.query.page) || 1}-size-${meta.per_page}-total-${meta.total}`"
          :count="meta.total"
          :pageSize="meta.per_page"
          :page="Number(route.query.page) || 1"
          :siblingCount="2"
          :onPageChange="handlePageChange"
        >
          <PaginationPrevTrigger>{{ $t('common.previous') }}</PaginationPrevTrigger>
          <PaginationContext v-slot="{ pagination }">
            <template v-for="(page, i) in pagination?.pages" :key="i">
              <PaginationItem v-if="page.type === 'page'" v-bind="{ ...page }">
                {{ page.value }}
              </PaginationItem>
              <PaginationEllipsis v-else :index="i" />
            </template>
          </PaginationContext>
          <PaginationNextTrigger>{{ $t('common.next') }}</PaginationNextTrigger>
        </PaginationRoot>

        <div class="flex items-center gap-3">
          <span class="text-sm text-foreground/50">
            {{ $t('common.showingEntries', { from: meta.from, to: meta.to, total: meta.total }) }}
          </span>
          <NativeSelect
            :value="currentPerPage"
            class="w-20 text-sm py-1"
            @change="handlePerPageChange"
          >
            <NativeSelectOption
              v-for="opt in perPageOptions"
              :key="opt"
              :value="opt"
            >
              {{ opt }}
            </NativeSelectOption>
          </NativeSelect>
        </div>
      </div>
    </template>
  </component>
</template>