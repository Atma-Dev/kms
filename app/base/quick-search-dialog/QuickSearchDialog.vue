<script setup lang="ts">
import { Lucide, type Icon } from '~/base/ui/lucide'
import { Input } from '~/base/ui/input'
import { DialogRoot, DialogContent } from '~/base/ui/dialog'
import mainMenu, { type Menu } from '~/main/side-menu'
import { filterMenuByPermission } from '~/utils/menu'
import { useSearchService } from '~/services/SearchService'
import { Permission } from '~/enums/Permission'
import { debounce } from 'lodash'

const { open } = defineProps<{
  open?: boolean
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
}>()

const { t } = useI18n()
const query = ref('')
const { hasPermission } = useAuthStore()

// The thing the user actually picked — not the text they typed to find it.
interface RecentItem {
  path: string
  label: string
  subtitle?: string
  icon?: Icon
}

const RECENT_ITEMS_KEY = 'quick_search_recent'
const MAX_RECENT_ITEMS = 5
const recentItems = ref<RecentItem[]>(JSON.parse(localStorage.getItem(RECENT_ITEMS_KEY) ?? '[]'))

const addRecentItem = (item: RecentItem) => {
  const next = [item, ...recentItems.value.filter((i) => i.path !== item.path)].slice(0, MAX_RECENT_ITEMS)
  recentItems.value = next
  localStorage.setItem(RECENT_ITEMS_KEY, JSON.stringify(next))
}

const clearRecentItems = () => {
  recentItems.value = []
  localStorage.removeItem(RECENT_ITEMS_KEY)
}

// Documents, not just pages (Modul 3 FR-5 and FR-6). The menu below is a
// static list this file can filter itself; documents are not, so they come
// from the API — suggestions while typing, and this user's last searches when
// the box is empty.
const canSearchDocuments = hasPermission(Permission.DOCUMENTS_SHOW)
const { suggestions, history, suggest, getHistory, deleteHistory } = useSearchService()

// Debounced, because this fires per keystroke. 250ms is short enough that the
// list feels attached to the typing and long enough that a typed word is one
// request rather than six.
const suggestDebounced = debounce((q: string) => suggest(q), 250)

watch(query, (q) => {
  if (!canSearchDocuments) return
  if (!q.trim()) {
    suggestDebounced.cancel()
    suggestions.value = []
    return
  }
  suggestDebounced(q)
})

// Dialog is reused across opens (v-model:open toggles the same instance), so
// state has to be cleared here rather than relying on remount.
watch(
  () => open,
  (isOpen) => {
    if (!isOpen) {
      suggestDebounced.cancel()
      query.value = ''
      return
    }
    // Loaded per open rather than once: another tab, or this one an hour ago,
    // has been adding to the same list.
    if (canSearchDocuments) getHistory()
  },
)

onUnmounted(() => suggestDebounced.cancel())

const runSearch = async (q: string) => {
  emit('update:open', false)
  await navigateTo({ path: '/search', query: { q } })
}

const openDocument = async (id: string) => {
  emit('update:open', false)
  await navigateTo(`/documents/${id}`)
}

// Menu is a flat-ish static list (route_name/title only, no search data of its
// own), so filtering it client side needs no request. Same permission filter
// SideMenu/TopMenu use — otherwise a page the user can't open still shows up
// here and clicking it 403s.
const flatMenu = computed(() => {
  const items: Menu[] = []
  const walk = (list: (string | Menu)[]) =>
    list.forEach((m) => {
      if (typeof m === 'string') return
      if (m.route_name && m.title) items.push(m)
      if (m.sub_menu) walk(m.sub_menu)
    })
  walk(filterMenuByPermission(mainMenu, hasPermission))
  return items
})

const matchedMenu = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return []
  // Match against the translated label, not the raw i18n key — otherwise
  // searching in Indonesian (or for any word not in the key itself) never
  // finds a match.
  return flatMenu.value.filter((m) => t(m.title!).toLowerCase().includes(q)).slice(0, 5)
})

const menuPath = (m: Menu) => `/${m.route_name}`

const goTo = async (item: RecentItem) => {
  addRecentItem(item)
  emit('update:open', false)
  await navigateTo(item.path)
}
</script>

<template>
  <DialogRoot :open="open" @openChange="(details) => emit('update:open', details.open)">
    <DialogContent class="sm:max-w-2xl p-0">
      <div class="relative border-b border-dashed border-foreground/15">
        <Lucide class="absolute inset-y-0 my-auto ml-4 opacity-70 z-50" icon="Search" />
        <Input
          v-model="query"
          class="w-full pl-12 bg-transparent border-none pr-19 h-14 focus:ring-offset-transparent focus-visible:ring-transparent shadow-none"
          type="text"
          placeholder="Search pages..."
          autofocus
        />
        <div
          class="absolute inset-y-0 right-0 flex items-center h-6 px-2 my-auto mr-4 border rounded-lg border-foreground/30 text-xs opacity-70"
        >
          esc
        </div>
      </div>

      <div v-if="!query.trim()">
        <div v-if="recentItems.length" class="px-5 py-4">
          <div class="flex items-center">
            <div class="text-xs uppercase opacity-70">Recently Visited</div>
            <a class="ml-auto text-xs opacity-70 cursor-pointer" href="" @click.prevent="clearRecentItems">
              Clear
            </a>
          </div>
          <div class="mt-3.5 flex flex-col gap-1">
            <a
              v-for="item in recentItems"
              :key="item.path"
              class="hover:border-foreground/10 hover:bg-foreground/5 -mx-1 flex items-center gap-2.5 rounded-xl border border-transparent p-1 cursor-pointer"
              href=""
              @click.prevent="goTo(item)"
            >
              <div
                class="flex items-center justify-center overflow-hidden border rounded-lg border-primary/10 bg-primary/10 text-primary size-8"
              >
                <Lucide class="stroke-1" :icon="item.icon ?? 'History'" />
              </div>
              <div class="font-medium truncate text-sm">
                {{ item.label }}
              </div>
            </a>
          </div>
        </div>
        <div v-if="history.length" class="border-t border-dashed border-foreground/10 px-5 py-4">
          <div class="text-xs uppercase opacity-70">{{ $t('search.recentSearches') }}</div>
          <div class="mt-3.5 flex flex-col gap-1">
            <div
              v-for="entry in history"
              :key="entry.id"
              class="hover:border-foreground/10 hover:bg-foreground/5 -mx-1 flex items-center gap-2.5 rounded-xl border border-transparent p-1"
            >
              <a class="flex min-w-0 flex-1 items-center gap-2.5 cursor-pointer" href="" @click.prevent="runSearch(entry.query)">
                <div
                  class="flex items-center justify-center overflow-hidden border rounded-lg border-primary/10 bg-primary/10 text-primary size-8"
                >
                  <Lucide class="stroke-1" icon="Search" />
                </div>
                <div class="font-medium truncate text-sm">{{ entry.query }}</div>
              </a>
              <button
                type="button"
                class="shrink-0 rounded p-1.5 opacity-50 hover:bg-foreground/5 hover:opacity-100"
                :aria-label="$t('common.delete')"
                @click.stop="deleteHistory(entry.id)"
              >
                <Lucide class="size-4" icon="X" />
              </button>
            </div>
          </div>
        </div>

        <div v-if="!recentItems.length && !history.length" class="px-5 py-8 text-center text-sm opacity-60">
          Start typing to search pages and documents.
        </div>
      </div>
      <!-- One branch, not three: "nothing matched" still has to offer the full
           search, because this dropdown only completes titles while /search
           reads the content as well — which is exactly where the thing being
           looked for turns out to be. -->
      <div v-else class="px-5 py-4 pb-6">
        <template v-if="matchedMenu.length">
        <div class="text-xs uppercase opacity-70">Pages</div>
        <div class="mt-3.5 flex flex-col gap-1">
          <a
            v-for="menu in matchedMenu"
            :key="menu.route_name"
            class="hover:border-foreground/10 hover:bg-foreground/5 -mx-1 flex items-center gap-2.5 rounded-xl border border-transparent p-1 cursor-pointer"
            href=""
            @click.prevent="goTo({ path: menuPath(menu), label: t(menu.title!), icon: menu.icon })"
          >
            <div
              class="flex items-center justify-center overflow-hidden border rounded-lg border-primary/10 bg-primary/10 text-primary size-8"
            >
              <Lucide v-if="menu.icon" class="stroke-1" :icon="menu.icon" />
            </div>
            <div class="font-medium truncate text-sm">
              {{ $t(menu.title!) }}
            </div>
          </a>
        </div>
        </template>

        <template v-if="suggestions.length">
          <div class="mt-5 text-xs uppercase opacity-70">{{ $t('search.documents') }}</div>
          <div class="mt-3.5 flex flex-col gap-1">
            <a
              v-for="doc in suggestions"
              :key="doc.id"
              class="hover:border-foreground/10 hover:bg-foreground/5 -mx-1 flex items-center gap-2.5 rounded-xl border border-transparent p-1 cursor-pointer"
              href=""
              @click.prevent="openDocument(doc.id)"
            >
              <div
                class="flex items-center justify-center overflow-hidden border rounded-lg border-primary/10 bg-primary/10 text-primary size-8"
              >
                <Lucide class="stroke-1" icon="FileText" />
              </div>
              <div class="font-medium truncate text-sm">{{ doc.title }}</div>
            </a>
          </div>
        </template>

        <p v-if="!matchedMenu.length && !suggestions.length" class="py-4 text-center text-sm opacity-60">
          No results for "{{ query }}".
        </p>

        <a
          v-if="canSearchDocuments"
          class="mt-4 -mx-1 flex items-center gap-2 rounded-xl p-1 text-sm text-primary cursor-pointer hover:underline"
          href=""
          @click.prevent="runSearch(query)"
        >
          <Lucide class="size-4" icon="Search" />
          {{ $t('search.seeAllResults', { query }) }}
        </a>
      </div>
    </DialogContent>
  </DialogRoot>
</template>
