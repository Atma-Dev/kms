<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Button } from '~/base/ui/button'
import { Lucide } from '~/base/ui/lucide'
import { Table, TableBody, TableHeader, TableRow } from '~/base/ui/table'
import { Permission } from '~/enums/Permission'
import { useCacheService } from '~/services/CacheService'

definePageMeta({
  title: 'account.cache.title',
  pageSubTitle: 'account.cache.subtitle',
  breadcrumb: ['Account', 'Cache'],
})

const authStore = useAuthStore()
const canClear = computed(() => authStore.hasPermission(Permission.MANAGEMENT_CACHE_DESTROY))

// The result of a clear is a flash message, rendered by SimpleAlert in the
// theme layout.
const { entries, loading, clearingKey, getAll, clear } = useCacheService()

onMounted(() => getAll())

// Keys are "lookup:areas" — the prefix is the same for every row, so only the
// resource half is worth a column.
const resourceOf = (key: string) => key.split(':').slice(1).join(':') || key

// A countdown reads better than a timestamp here: the question is always "how
// stale can this still be", never "when exactly was it stored".
const formatCountdown = (seconds: number) => {
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  return `${Math.floor(minutes / 60)}h ${minutes % 60}m`
}

const handleClear = async (key = '') => {
  if (await clear(key)) await getAll()
}
</script>

<template>
  <div class="grid grid-cols-12 gap-6">
    <Box class="col-span-12 overflow-auto lg:overflow-visible">
      <div class="mb-4 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <SmallInfo class="max-w-2xl">{{ $t('account.cache.lookupDescription') }}</SmallInfo>

        <div class="flex shrink-0 gap-2">
          <Button variant="secondary" :disabled="loading" @click="getAll">
            <Lucide icon="RotateCw" :class="loading ? 'animate-spin' : ''" />
            {{ $t('common.refresh') }}
          </Button>
          <Button
            v-if="canClear"
            variant="danger"
            :disabled="clearingKey !== null || entries.length === 0"
            @click="handleClear()"
          >
            <Lucide icon="Trash" />
            {{ $t('account.cache.clearAllButton') }}
          </Button>
        </div>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center gap-3 py-16 text-foreground/50">
        <svg class="h-5 w-5 animate-spin" viewBox="0 0 24 24" fill="none">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
        </svg>
        <span class="text-sm">{{ $t('common.loading') }}</span>
      </div>

      <!-- Nothing cached is the normal state right after a clear, not an error -->
      <div v-else-if="entries.length === 0" class="py-16 text-center text-sm text-foreground/50">
        {{ $t('account.cache.empty') }}
      </div>

      <Table v-else class="-mt-2" variant="boxed">
        <TableHeader>
          <TableRow>
            <TH icon="Hash">{{ $t('common.no') }}</TH>
            <TH icon="Database">{{ $t('account.cache.columnResource') }}</TH>
            <TH icon="List">{{ $t('account.cache.columnItems') }}</TH>
            <TH icon="Clock">{{ $t('account.cache.columnExpires') }}</TH>
            <TH v-if="canClear" icon="Settings">{{ $t('common.actions') }}</TH>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow v-for="(entry, index) in entries" :key="entry.key">
            <TD>{{ index + 1 }}</TD>
            <TD>
              <p class="font-medium">{{ resourceOf(entry.key) }}</p>
              <SmallInfo>{{ entry.key }}</SmallInfo>
            </TD>
            <TD>{{ $t('account.cache.itemCount', { count: entry.count }) }}</TD>
            <TD>{{ formatCountdown(entry.expires_in_seconds) }}</TD>
            <TD v-if="canClear">
              <Button
                variant="secondary"
                size="sm"
                :disabled="clearingKey !== null"
                @click="handleClear(entry.key)"
              >
                <Lucide icon="Eraser" :class="clearingKey === entry.key ? 'animate-spin' : ''" />
                {{ $t('account.cache.clearButton') }}
              </Button>
            </TD>
          </TableRow>
        </TableBody>
      </Table>
    </Box>
  </div>
</template>
