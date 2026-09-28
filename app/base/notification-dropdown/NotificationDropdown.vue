<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { useNotificationStore } from '~/stores/notification'
import _ from 'lodash'

interface Props {
  class?: string
  boxClass?: string
}

const props = defineProps<Props>()

const notificationStore = useNotificationStore()
const { items } = storeToRefs(notificationStore)
const recent = computed(() => _.take(items.value, 5))
</script>

<template>
  <div
    v-bind="$attrs"
    :class="[
      'invisible opacity-0 scale-95 transition-all duration-200 delay-0 group-hover/notifications:visible group-hover/notifications:opacity-100 group-hover/notifications:scale-100 group-hover/notifications:delay-200',
      props.class,
    ]"
  >
    <Box
      :class="`before:shadow-foreground/5 z-50 flex w-96 flex-col gap-2.5 px-6 py-5 before:rounded-2xl before:shadow-xl before:backdrop-blur after:rounded-2xl ${props.boxClass ?? ''}`"
    >
      <div class="flex place-content-between items-center">
        <div class="font-medium">Notifications</div>
        <NuxtLink class="text-primary" to="/notifications">View All</NuxtLink>
      </div>
      <div v-if="!recent.length" class="mt-1 text-sm opacity-70">No notifications yet.</div>
      <div v-else class="mt-1 flex flex-col gap-2.5">
        <a
          v-for="notification in recent"
          :key="notification.id"
          class="hover:border-foreground/10 hover:bg-foreground/5 -mx-2 flex items-center gap-3.5 rounded-2xl border border-transparent p-2 cursor-pointer"
          :class="{ 'bg-foreground/5': !notification.is_read }"
          @click="notificationStore.markRead(notification.id)"
        >
          <div class="flex flex-col gap-1 w-full">
            <div class="flex place-content-between items-center">
              <div class="font-medium">{{ notification.title }}</div>
              <div class="text-xs opacity-70">{{ notification.f_created_at }}</div>
            </div>
            <div class="line-clamp-2 opacity-70">
              {{ notification.body }}
            </div>
          </div>
        </a>
      </div>
    </Box>
  </div>
</template>
