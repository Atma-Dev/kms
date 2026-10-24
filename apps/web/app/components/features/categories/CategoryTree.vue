<script setup lang="ts">
import { Lucide } from '~/base/ui/lucide'
import { Button } from '~/base/ui/button'
import type Category from '~/types/entities/category'
import { Permission } from '~/enums/Permission'

const props = defineProps<{
  nodes: Category[]
  /** Depth of this level, 1-based. The API caps nesting at 3. */
  depth?: number
  selectedId?: string | null
}>()

const emit = defineEmits<{
  (e: 'select', id: string | null): void
  (e: 'add-child', parent: Category): void
  (e: 'edit', category: Category): void
  (e: 'remove', category: Category): void
  (e: 'move', category: Category, direction: -1 | 1): void
}>()

const MAX_DEPTH = 3
const depth = computed(() => props.depth ?? 1)
const canNest = computed(() => depth.value < MAX_DEPTH)

const authStore = useAuthStore()
const canUpdate = computed(() => authStore.hasPermission(Permission.MASTERS_CATEGORIES_UPDATE))
const canStore = computed(() => authStore.hasPermission(Permission.MASTERS_CATEGORIES_STORE))
const canDestroy = computed(() => authStore.hasPermission(Permission.MASTERS_CATEGORIES_DESTROY))
</script>

<template>
  <ul :class="depth > 1 ? 'ml-4 border-l border-foreground/10 pl-3' : ''">
    <li v-for="(node, index) in nodes" :key="node.id" class="py-0.5">
      <div
        class="group flex items-center gap-1 rounded-lg px-2 py-1.5"
        :class="selectedId === node.id ? 'bg-primary/10 text-primary' : 'hover:bg-foreground/5'"
      >
        <button type="button" class="flex min-w-0 flex-1 items-center gap-2 text-left" @click="emit('select', node.id)">
          <Lucide :icon="node.children.length ? 'FolderOpen' : 'Folder'" class="size-4 shrink-0 opacity-60" />
          <span class="truncate text-sm">{{ node.name }}</span>
          <!-- The whole subtree, matching what clicking this lists. Hidden at
               zero: an empty category is obvious from the absence, and a row
               of "0"s makes the ones that matter harder to find. The count
               gives way to the action buttons on hover. -->
          <span
            v-if="node.document_count > 0"
            class="ml-auto shrink-0 rounded-full bg-foreground/5 px-1.5 py-0.5 text-[10px] tabular-nums text-foreground/45 group-hover:hidden"
          >
            {{ node.document_count }}
          </span>
        </button>

        <!-- Reordering is up/down buttons, not drag: no drag library is
             installed, and buttons work on touch and with a keyboard for free.
             The API takes the whole ordered list either way, so swapping in
             drag later changes nothing server-side. -->
        <div class="flex shrink-0 gap-0.5 opacity-0 transition group-hover:opacity-100 focus-within:opacity-100">
          <button
            v-if="canUpdate"
            type="button"
            class="rounded p-1 hover:bg-foreground/10 disabled:opacity-20"
            :disabled="index === 0"
            :aria-label="$t('categories.moveUp')"
            :title="$t('categories.moveUp')"
            @click="emit('move', node, -1)"
          >
            <Lucide icon="ChevronUp" class="size-3.5" />
          </button>
          <button
            v-if="canUpdate"
            type="button"
            class="rounded p-1 hover:bg-foreground/10 disabled:opacity-20"
            :disabled="index === nodes.length - 1"
            :aria-label="$t('categories.moveDown')"
            :title="$t('categories.moveDown')"
            @click="emit('move', node, 1)"
          >
            <Lucide icon="ChevronDown" class="size-3.5" />
          </button>
          <button
            v-if="canStore && canNest"
            type="button"
            class="rounded p-1 hover:bg-foreground/10"
            :aria-label="$t('categories.addChild')"
            :title="$t('categories.addChild')"
            @click="emit('add-child', node)"
          >
            <Lucide icon="Plus" class="size-3.5" />
          </button>
          <button
            v-if="canUpdate"
            type="button"
            class="rounded p-1 hover:bg-foreground/10"
            :aria-label="$t('common.edit')"
            :title="$t('common.edit')"
            @click="emit('edit', node)"
          >
            <Lucide icon="Pencil" class="size-3.5" />
          </button>
          <button
            v-if="canDestroy"
            type="button"
            class="rounded p-1 text-danger hover:bg-danger/10"
            :aria-label="$t('common.delete')"
            :title="$t('common.delete')"
            @click="emit('remove', node)"
          >
            <Lucide icon="Trash2" class="size-3.5" />
          </button>
        </div>
      </div>

      <CategoryTree
        v-if="node.children.length"
        :nodes="node.children"
        :depth="depth + 1"
        :selected-id="selectedId"
        @select="emit('select', $event)"
        @add-child="emit('add-child', $event)"
        @edit="emit('edit', $event)"
        @remove="emit('remove', $event)"
        @move="(c, d) => emit('move', c, d)"
      />
    </li>

    <li v-if="!nodes.length && depth === 1" class="px-2 py-3 text-sm text-foreground/40">
      {{ $t('categories.empty') }}
    </li>
  </ul>
</template>
