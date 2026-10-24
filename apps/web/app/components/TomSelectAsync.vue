<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { input } from '~/base/ui/styles/input.styles'

export interface AsyncOption {
  id: string
  code: string
  name: string
}

const props = defineProps<{
  modelValue?: string
  placeholder?: string
  loadFn: (query: string) => Promise<AsyncOption[]>
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'select', option: AsyncOption | null): void
}>()

const selectEl = ref<HTMLSelectElement | null>(null)

type TomSelectInstance = {
  setValue: (v: string, silent?: boolean) => void
  clear: (silent?: boolean) => void
  destroy: () => void
  options: Record<string, unknown>
  on?: (ev: string, cb: (...args: unknown[]) => void) => void
  dropdown: HTMLElement | null
  control: HTMLElement | null
  wrapper: HTMLElement | null
}

let tsInstance: TomSelectInstance | null = null
let tsDropdownEl: HTMLElement | null = null
let resizeHandler: (() => void) | null = null
let scrollHandler: (() => void) | null = null

async function initTomSelect() {
  const module = await import('tom-select')
  try { await import('tom-select/dist/css/tom-select.css') } catch {}
  const TomSelectCtor = (module.default || module) as unknown as {
    new (el: HTMLSelectElement, opts: unknown): TomSelectInstance
  }

  tsInstance = new TomSelectCtor(selectEl.value as HTMLSelectElement, {
    valueField: 'id',
    labelField: 'display',
    searchField: ['display'],
    placeholder: props.placeholder ?? 'Cari reseller...',
    preload: false,
    shouldLoad: (q: string) => q.length >= 2,
    load: (query: string, callback: (data: unknown[]) => void) => {
      props.loadFn(query)
        .then((items) =>
          callback(
            items.map((o) => ({ id: o.id, display: `${o.code} – ${o.name}`, code: o.code, name: o.name })),
          ),
        )
        .catch(() => callback([]))
    },
    onChange: (value: string) => {
      emit('update:modelValue', value)
      if (!value) {
        emit('select', null)
        return
      }
      const opt = tsInstance?.options[value] as (AsyncOption & { display?: string }) | undefined
      if (opt) emit('select', { id: opt.id, code: opt.code, name: opt.name })
    },
  })

  // Apply project input styles and portal the dropdown
  try {
    const instAny = tsInstance as unknown as { dropdown: HTMLElement | null; control: HTMLElement | null; wrapper: HTMLElement | null }
    tsDropdownEl = instAny.dropdown
    const wrapperEl = instAny.wrapper
    const controlEl = instAny.control

    if (wrapperEl && input) {
      wrapperEl.classList.add(...String(input).split(/\s+/).filter(Boolean), 'ts-custom-input')
    }

    const reposition = () => {
      if (!tsDropdownEl || !controlEl) return
      const rect = controlEl.getBoundingClientRect()
      tsDropdownEl.style.position = 'absolute'
      tsDropdownEl.style.left = `${rect.left}px`
      tsDropdownEl.style.top = `${rect.bottom + window.scrollY}px`
      tsDropdownEl.style.width = `${rect.width}px`
      tsDropdownEl.style.zIndex = '99999'
    }

    const attachDropdown = () => {
      if (!tsDropdownEl) return
      try { if (tsDropdownEl.parentElement !== document.body) document.body.appendChild(tsDropdownEl) } catch {}
      Object.assign(tsDropdownEl.style, { position: 'absolute', transform: 'none', boxSizing: 'border-box', right: 'auto' })
      reposition()
      requestAnimationFrame(() => { reposition(); setTimeout(reposition, 50) })
      if (!resizeHandler) {
        resizeHandler = () => reposition()
        scrollHandler = () => reposition()
        window.addEventListener('resize', resizeHandler)
        window.addEventListener('scroll', scrollHandler, true)
      }
    }

    if (tsDropdownEl) attachDropdown()
    if (tsInstance?.on) {
      tsInstance.on('dropdown_open', () => {
        attachDropdown()
        setTimeout(reposition, 0)
        setTimeout(reposition, 100)
      })
    }
  } catch {}
}

onMounted(() => { initTomSelect().catch((e) => console.error('TomSelectAsync init failed:', e)) })

onBeforeUnmount(() => {
  tsInstance?.destroy()
  tsInstance = null
  if (resizeHandler) { window.removeEventListener('resize', resizeHandler); resizeHandler = null }
  if (scrollHandler) { window.removeEventListener('scroll', scrollHandler, true); scrollHandler = null }
  if (tsDropdownEl?.parentElement === document.body) { try { document.body.removeChild(tsDropdownEl) } catch {} }
  tsDropdownEl = null
})

watch(
  () => props.modelValue,
  (val) => {
    if (!tsInstance) return
    if (!val) tsInstance.clear(true)
    else tsInstance.setValue(val, true)
  },
)
</script>

<template>
  <select ref="selectEl" />
</template>

<style>
.ts-custom-input { cursor: pointer; display: block !important; padding: 0 !important; }
.ts-custom-input .ts-control { display: flex !important; align-items: center !important; gap: 0.5rem !important; flex-wrap: nowrap !important; border: none !important; padding: 0.5rem 0.75rem !important; border-radius: 0 !important; background: transparent !important; box-shadow: none !important; min-height: 100% !important; cursor: pointer !important; }
.ts-custom-input .ts-control input { border: none !important; outline: none !important; background: transparent !important; padding: 0 !important; margin: 0 !important; min-width: 2ch !important; line-height: 1 !important; align-self: center !important; }
.ts-custom-input .ts-control input::placeholder { color: color-mix(in srgb, var(--color-foreground), transparent 20%); }
.ts-dropdown { z-index: 99999 !important; }
.ts-dropdown.single { border-radius: 12px; overflow: hidden; }
.ts-dropdown .option, .ts-dropdown .optgroup-header, .ts-dropdown .no-results, .ts-dropdown .create { padding: 8px 12px !important; }
</style>
