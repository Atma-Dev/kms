<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Button } from '~/base/ui/button'
import { Lucide, type Icon } from '~/base/ui/lucide'
import { useNotificationService } from '~/services/NotificationService'
import { useNotificationStore } from '~/stores/notification'
import type AppNotification from '~/types/entities/notification'

definePageMeta({
  title: 'notifications.title',
  pageSubTitle: 'notifications.subtitle',
  breadcrumb: ['Notifications'],
})

const { t } = useI18n()
const { notifications, meta, loading, getAll, markRead, markAllRead } = useNotificationService()
const notificationStore = useNotificationStore()

useServerList({
  fetch: (page, perPage) => getAll(page, perPage),
})

const hasUnread = computed(() => notifications.value.some((n) => !n.is_read))

// Only one type exists today (invoice.status_changed) — a prefix match keeps
// this from needing an update every time a new invoice.* notification is added.
const iconForType = (type: string): Icon => (type.startsWith('invoice.') ? 'Wallet' : 'Bell')

const timeOf = (isoDate: string) =>
  new Date(isoDate).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })

const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()

// Notifications arrive newest-first from the API — grouping by insertion
// order (Map) keeps that ordering without a separate sort.
const groupLabel = (isoDate: string) => {
  const date = new Date(isoDate)
  const diffDays = Math.round((startOfDay(new Date()) - startOfDay(date)) / 86400000)
  if (diffDays === 0) return t('notifications.today')
  if (diffDays === 1) return t('notifications.yesterday')
  return date.toLocaleDateString(undefined, { day: 'numeric', month: 'long', year: 'numeric' })
}

const groups = computed(() => {
  const byLabel = new Map<string, AppNotification[]>()
  for (const notification of notifications.value) {
    const label = groupLabel(notification.created_at)
    if (!byLabel.has(label)) byLabel.set(label, [])
    byLabel.get(label)!.push(notification)
  }
  return Array.from(byLabel, ([label, items]) => ({ label, items }))
})

// Optimistic locally, then the API call — this page keeps its own list
// separate from the dropdown's (different page sizes), so the store is only
// re-synced afterward to keep the bell badge and dropdown feed accurate.
const handleMarkRead = async (id: string) => {
  const notification = notifications.value.find((n) => n.id === id)
  if (!notification || notification.is_read) return
  notification.is_read = true
  await markRead(id)
  notificationStore.fetch()
}

const handleMarkAllRead = async () => {
  notifications.value.forEach((n) => (n.is_read = true))
  await markAllRead()
  notificationStore.fetch()
}
</script>

<template>
  <Box class="overflow-auto lg:overflow-visible">
    <div class="mb-4 flex items-center justify-end">
      <Button v-if="hasUnread" look="outline" size="sm" @click="handleMarkAllRead">
        {{ $t('notifications.markAllRead') }}
      </Button>
    </div>

    <ServerSidePagination :meta="meta" :loading="loading">
      <div v-if="!notifications.length && !loading" class="py-10 text-center text-sm opacity-70">
        {{ $t('notifications.empty') }}
      </div>
      <div v-else class="flex flex-col gap-5">
        <div v-for="group in groups" :key="group.label">
          <div class="mb-2 flex items-center gap-3">
            <span class="text-xs font-medium uppercase tracking-wide opacity-50">{{ group.label }}</span>
            <div class="h-px flex-1 bg-foreground/10" />
          </div>
          <div class="flex flex-col divide-y divide-foreground/10">
            <a
              v-for="notification in group.items"
              :key="notification.id"
              class="flex cursor-pointer items-start gap-3 py-3"
              @click="handleMarkRead(notification.id)"
            >
              <span
                class="mt-3.5 size-1.5 shrink-0 rounded-full"
                :class="notification.is_read ? 'bg-transparent' : 'bg-primary'"
              />
              <div
                class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg"
                :class="notification.is_read ? 'bg-foreground/10 text-foreground/50' : 'bg-primary/10 text-primary'"
              >
                <Lucide :icon="iconForType(notification.type)" class="h-4.5 w-4.5" />
              </div>
              <div class="flex w-full min-w-0 flex-col gap-1">
                <div class="flex place-content-between items-center gap-3">
                  <div :class="notification.is_read ? 'font-normal' : 'font-medium'">
                    {{ notification.title }}
                  </div>
                  <div class="shrink-0 text-xs opacity-70">{{ timeOf(notification.created_at) }}</div>
                </div>
                <div class="opacity-70">{{ notification.body }}</div>
              </div>
            </a>
          </div>
        </div>
      </div>
    </ServerSidePagination>
  </Box>
</template>
