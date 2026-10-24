<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Button } from '~/base/ui/button'
import { Input } from '~/base/ui/input'
import { Lucide } from '~/base/ui/lucide'
import { Badge } from '~/base/ui/badge'
import { useSpaceService } from '~/services/SpaceService'
import type Space from '~/types/entities/space'
import ModalFormSpace from '~/components/features/spaces/ModalFormSpace.vue'
import ModalArchiveSpace from '~/components/features/spaces/ModalArchiveSpace.vue'
import { Permission } from '~/enums/Permission'

definePageMeta({
  title: 'spaces.title',
  pageSubTitle: 'spaces.subtitle',
  breadcrumb: ['Spaces'],
})

const authStore = useAuthStore()
const canStore = computed(() => authStore.hasPermission(Permission.MASTERS_SPACES_STORE))
const canUpdate = computed(() => authStore.hasPermission(Permission.MASTERS_SPACES_UPDATE))
const canArchive = computed(() => authStore.hasPermission(Permission.MASTERS_SPACES_DESTROY))

const { spaces, meta, loading, submitting, getAll, unarchiveById } = useSpaceService()

const { filters, hasActiveFilters, reset, refresh } = useServerList({
  filters: { search: '', include_archived: '' },
  debounce: ['search'],
  fetch: (page, perPage, applied) => getAll(page, perPage, applied),
})

// The card is clickable, but the links inside it still own cmd/ctrl-click —
// otherwise opening a space in a new tab also navigates the current one.
const openSpace = (space: Space, event: MouseEvent) => {
  if (event.metaKey || event.ctrlKey || event.shiftKey) return
  navigateTo(`/spaces/${space.id}`)
}

const formModal = ref<InstanceType<typeof ModalFormSpace> | null>(null)
const archiveModal = ref<InstanceType<typeof ModalArchiveSpace> | null>(null)
</script>

<template>
  <div class="grid grid-cols-12 gap-6">
    <Box class="col-span-12">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-2">
          <div class="relative">
            <Input
              v-model="filters.search"
              type="text"
              :placeholder="$t('spaces.searchPlaceholder')"
              class="w-56 pr-8"
            />
            <Lucide class="absolute inset-y-0 right-0 my-auto mr-2.5 size-4 text-foreground/40" icon="Search" />
          </div>
          <Button
            size="sm"
            :look="filters.include_archived ? 'filled' : 'outline'"
            :variant="filters.include_archived ? 'primary' : 'ghost'"
            @click="filters.include_archived = filters.include_archived ? '' : '1'"
          >
            <Lucide class="mr-1.5 size-4" icon="Archive" />
            {{ $t('spaces.showArchived') }}
          </Button>
          <Button v-if="hasActiveFilters" look="outline" variant="ghost" size="sm" @click="reset">
            <Lucide class="mr-1.5 size-4" icon="X" />
            {{ $t('common.reset') }}
          </Button>
        </div>

        <Button v-if="canStore" variant="primary" @click="formModal?.handleModal(true)">
          <Lucide icon="Plus" />
          {{ $t('spaces.createSpace') }}
        </Button>
      </div>

      <ServerSidePagination :meta="meta" :loading="loading">
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
          <div
            v-for="space in spaces"
            :key="space.id"
            class="flex cursor-pointer flex-col rounded-xl border border-foreground/10 p-4 transition hover:border-primary/40"
            @click="openSpace(space, $event)"
          >
            <div class="flex items-start justify-between gap-2">
              <NuxtLink :to="`/spaces/${space.id}`" class="flex min-w-0 items-center gap-2.5">
                <div class="grid size-9 shrink-0 place-items-center rounded-lg bg-primary/10 text-primary">
                  <Lucide icon="BookOpen" class="size-4" />
                </div>
                <div class="min-w-0">
                  <div class="truncate font-medium">{{ space.name }}</div>
                  <div class="truncate text-xs text-foreground/40">/{{ space.slug }}</div>
                </div>
              </NuxtLink>

              <Badge v-if="space.is_archived" variant="secondary" look="outline" class="shrink-0 text-[10px]">
                {{ $t('spaces.archived') }}
              </Badge>
            </div>

            <p class="mt-3 line-clamp-2 min-h-[2.5rem] text-sm text-foreground/60">
              {{ space.description || $t('spaces.noDescription') }}
            </p>

            <div class="mt-3 flex items-center justify-between border-t border-foreground/5 pt-3">
              <NuxtLink :to="`/spaces/${space.id}`" class="text-xs font-medium text-primary hover:underline">
                {{ $t('spaces.openSpace') }}
              </NuxtLink>
              <div class="flex gap-1">
                <button
                  v-if="canUpdate"
                  type="button"
                  class="rounded p-1.5 text-foreground/50 hover:bg-foreground/5"
                  :aria-label="$t('common.edit')"
                  :title="$t('common.edit')"
                  @click.stop="formModal?.handleModal(true, space)"
                >
                  <Lucide icon="Pencil" class="size-4" />
                </button>
                <button
                  v-if="canUpdate && space.is_archived"
                  type="button"
                  class="rounded p-1.5 text-foreground/50 hover:bg-foreground/5 hover:text-primary"
                  :disabled="submitting"
                  :aria-label="$t('spaces.unarchive')"
                  :title="$t('spaces.unarchive')"
                  @click="unarchiveById(space).then((ok) => ok && refresh())"
                >
                  <Lucide icon="ArchiveRestore" class="size-4" />
                </button>
                <button
                  v-if="canArchive && !space.is_archived"
                  type="button"
                  class="rounded p-1.5 text-foreground/50 hover:bg-danger/10 hover:text-danger"
                  :aria-label="$t('spaces.archive')"
                  :title="$t('spaces.archive')"
                  @click.stop="archiveModal?.handleModal(true, space)"
                >
                  <Lucide icon="Archive" class="size-4" />
                </button>
              </div>
            </div>
          </div>
        </div>

        <div v-if="!loading && !spaces.length" class="py-12 text-center text-sm text-foreground/40">
          {{ $t('spaces.empty') }}
        </div>
      </ServerSidePagination>
    </Box>

    <ModalFormSpace ref="formModal" @saved="refresh" />
    <ModalArchiveSpace ref="archiveModal" @archived="refresh" />
  </div>
</template>
