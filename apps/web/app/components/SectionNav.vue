<script setup lang="ts">
import { Box } from '~/base/ui/box'
import { Lucide, type Icon } from '~/base/ui/lucide'

// Left-hand nav for shell pages (/master, /account). Menus are real routes,
// not local tabs: each list owns its own `?page=`/`?search=` via `useServerList`,
// and switching menu must not carry the other one's query along.
defineProps<{
  menus: { title: string; description: string; to: string; icon: Icon }[]
}>()

const route = useRoute()
</script>

<template>
  <div class="grid grid-cols-12 gap-6">
    <Box class="col-span-12 h-fit p-2 lg:col-span-3 xl:col-span-2">
      <nav class="flex flex-col gap-1">
        <NuxtLink
          v-for="menu in menus"
          :key="menu.to"
          :to="menu.to"
          class="flex items-center gap-3 rounded-lg px-3 py-2.5 transition-colors"
          :class="
            route.path === menu.to
              ? 'bg-primary/10 text-primary font-medium'
              : 'text-foreground/70 hover:bg-foreground/5'
          "
        >
          <Lucide :icon="menu.icon" class="size-4 shrink-0" />
          <span class="min-w-0">
            <span class="block text-sm">{{ $t(menu.title) }}</span>
            <span class="block text-xs text-foreground/40">{{ $t(menu.description) }}</span>
          </span>
        </NuxtLink>
      </nav>
    </Box>

    <div class="col-span-12 lg:col-span-9 xl:col-span-10">
      <NuxtPage />
    </div>
  </div>
</template>
