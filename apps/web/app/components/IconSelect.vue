<script setup lang="ts">
import { h, render as renderVnode } from 'vue'
import { icons } from 'lucide-vue-next'

const model = defineModel<string>({ default: '' })

defineProps<{ placeholder?: string; id?: string }>()

// The canonical PascalCase names — the same keys `Lucide` resolves against,
// so whatever is picked here renders elsewhere without translation.
const options = Object.keys(icons).map((name) => ({ id: name, name }))

// tom-select wants markup, lucide ships Vue components: render one into a
// detached node and hand tom-select the element it produced.
const renderOption = (data: { value: string; text: string }) => {
  const el = document.createElement('div')
  // Inline styles, not utility classes: tom-select owns this node's CSS and
  // the control is only one line tall, so the label has to stay on that line.
  Object.assign(el.style, {
    display: 'flex',
    alignItems: 'center',
    gap: '0.5rem',
    minWidth: '0',
    overflow: 'hidden',
  })

  const icon = icons[data.value as keyof typeof icons]
  if (icon) {
    const host = document.createElement('span')
    Object.assign(host.style, { display: 'flex', flex: '0 0 auto' })
    // `size` is lucide's own prop — width/height get overwritten by it.
    renderVnode(h(icon, { size: 16 }), host)
    el.append(host)
  }

  const label = document.createElement('span')
  Object.assign(label.style, {
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    whiteSpace: 'nowrap',
  })
  label.textContent = data.text
  el.append(label)
  return el
}
</script>

<template>
  <TomSelect
    v-model="model"
    :id="id"
    :options="options"
    :placeholder="placeholder"
    :render="{ option: renderOption, item: renderOption }"
  />
</template>
