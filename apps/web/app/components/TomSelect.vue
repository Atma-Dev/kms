<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useAttrs } from 'vue'
import { input } from '~/base/ui/styles/input.styles'

type OptionItem = { id: number | string; name: string }

const props = withDefaults(
    defineProps<{
        modelValue?: string | number
        options?: OptionItem[]
        placeholder?: string
        id?: string
        name?: string
        // Passed straight to tom-select's `render` config, so a caller can
        // draw richer options (an icon preview) without another wrapper.
        render?: Record<string, unknown>
    }>(),
    {
        modelValue: undefined,
        options: () => [],
        placeholder: 'Select...',
        id: undefined,
        name: undefined,
        render: undefined,
    },
)

const emit = defineEmits(['update:modelValue'])

const selectEl = ref<HTMLSelectElement | null>(null)
type TomSelectInstance = {
    setValue: (v: string) => void
    destroy: () => void
    clearOptions: () => void
    addOption: (opts: unknown) => void
    refreshOptions: (flag?: boolean) => void
}

let tsInstance: TomSelectInstance | null = null
let tsDropdownEl: HTMLElement | null = null
let resizeHandler: (() => void) | null = null
let scrollHandler: (() => void) | null = null

async function initTomSelect() {
    const module = await import('tom-select')
    try {
        await import('tom-select/dist/css/tom-select.css')
    } catch (err) {
        console.warn('tom-select CSS could not be loaded automatically:', err)
    }
    const TomSelectClass = module.default || module

    const optionItems = (props.options || []).map((o) => ({
        value: String((o as OptionItem).id),
        text: (o as OptionItem).name,
    }))

    const TomSelectCtor = TomSelectClass as unknown as {
        new (el: HTMLSelectElement, opts: unknown): TomSelectInstance
    }
    tsInstance = new TomSelectCtor(selectEl.value as HTMLSelectElement, {
        options: optionItems,
        valueField: 'value',
        labelField: 'text',
        searchField: ['text'],
        placeholder: props.placeholder,
        allowEmptyOption: true,
        ...(props.render ? { render: props.render } : {}),
        onChange: (value: string) => {
            emit('update:modelValue', value)
        },
    })

    if (props.modelValue && tsInstance) {
        tsInstance.setValue(String(props.modelValue))
    }

    try {
        const instAny = tsInstance as unknown as {
            dropdown: HTMLElement | null
            control: HTMLElement | null
            wrapper: HTMLElement | null
            on?: (ev: string, cb: () => void) => void
        }

        tsDropdownEl = instAny.dropdown
        const wrapperEl = instAny.wrapper
        const controlEl = instAny.control

        if (wrapperEl && input) {
            const cls = String(input).split(/\s+/).filter(Boolean)
            wrapperEl.classList.add(...cls, 'ts-custom-input')
        }

        const reposition = () => {
            if (!tsDropdownEl || !controlEl) return
            const rect = controlEl.getBoundingClientRect()
            tsDropdownEl.style.position = 'absolute'
            tsDropdownEl.style.left = rect.left + 'px'
            tsDropdownEl.style.top = rect.bottom + window.scrollY + 'px'
            tsDropdownEl.style.width = rect.width + 'px'
            tsDropdownEl.style.zIndex = '99999'
        }

        const attachDropdown = () => {
            if (!tsDropdownEl) return
            try {
                if (tsDropdownEl.parentElement !== document.body)
                    document.body.appendChild(tsDropdownEl)
            } catch {}
            try {
                tsDropdownEl.style.position = 'absolute'
                tsDropdownEl.style.transform = 'none'
                tsDropdownEl.style.removeProperty('transform')
                tsDropdownEl.style.boxSizing = 'border-box'
                tsDropdownEl.style.right = 'auto'
            } catch {}

            reposition()
            requestAnimationFrame(() => {
                reposition()
                setTimeout(reposition, 50)
            })

            if (!resizeHandler) {
                resizeHandler = () => reposition()
                scrollHandler = () => reposition()
                window.addEventListener('resize', resizeHandler)
                window.addEventListener('scroll', scrollHandler, true)
            }
        }

        if (tsDropdownEl) attachDropdown()

        if (instAny.on) {
            instAny.on('dropdown_open', () => {
                attachDropdown()
                setTimeout(reposition, 0)
                setTimeout(reposition, 100)
                setTimeout(reposition, 300)
            })
        }
    } catch {
        // ignore
    }
}

onMounted(() => {
    initTomSelect().catch((err) => {
        console.error('Failed to load tom-select:', err)
    })
})

onBeforeUnmount(() => {
    if (tsInstance) {
        try { tsInstance.destroy() } catch {}
        tsInstance = null
    }
    if (resizeHandler) {
        window.removeEventListener('resize', resizeHandler)
        resizeHandler = null
    }
    if (scrollHandler) {
        window.removeEventListener('scroll', scrollHandler, true)
        scrollHandler = null
    }
    if (tsDropdownEl && tsDropdownEl.parentElement === document.body) {
        try { document.body.removeChild(tsDropdownEl) } catch {}
        tsDropdownEl = null
    }
})

watch(
    () => props.options,
    (newOpts) => {
        if (!tsInstance) return
        const optionItems = newOpts.map((o) => ({ value: String(o.id), text: o.name }))
        tsInstance.clearOptions()
        tsInstance.addOption(optionItems)
        tsInstance.refreshOptions(false)
    },
    { deep: true },
)

watch(
    () => props.modelValue,
    (val) => {
        if (!tsInstance) return
        tsInstance.setValue(val ? String(val) : '')
    },
)

const otherAttrs = useAttrs()
</script>

<template>
    <select ref="selectEl" :id="props.id" :name="props.name" v-bind="otherAttrs">
        <option value="">{{ props.placeholder }}</option>
        <option v-for="opt in props.options" :key="opt.id" :value="String(opt.id)">
            {{ opt.name }}
        </option>
    </select>
</template>

<style scoped></style>

<style>
.ts-custom-input {
    cursor: pointer;
    display: block !important;
    padding: 0 !important;
}

.ts-custom-input .ts-control {
    display: flex !important;
    align-items: center !important;
    gap: 0.5rem !important;
    flex-wrap: nowrap !important;
    border: none !important;
    padding: 0.5rem 0.75rem !important;
    border-radius: 0 !important;
    background: transparent !important;
    box-shadow: none !important;
    min-height: 100% !important;
    cursor: pointer !important;
}

.ts-custom-input .ts-control input {
    border: none !important;
    outline: none !important;
    background: transparent !important;
    padding: 0 !important;
    margin: 0 !important;
    min-width: 2ch !important;
    line-height: 1 !important;
    align-self: center !important;
}

.ts-custom-input .ts-control input::placeholder {
    color: color-mix(in srgb, var(--color-foreground), transparent 20%);
}

.ts-dropdown {
    z-index: 99999 !important;
}

.ts-dropdown.single {
    border-radius: 12px;
    overflow: hidden;
}

.ts-dropdown .option,
.ts-dropdown .optgroup-header,
.ts-dropdown .no-results,
.ts-dropdown .create {
    padding: 8px 12px !important;
}
</style>
