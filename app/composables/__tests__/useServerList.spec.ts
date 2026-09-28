import { describe, expect, it, vi } from 'vitest'
import * as vue from 'vue'
import { defineComponent, h, ref } from 'vue'
import { createMemoryHistory, createRouter, useRoute, useRouter } from 'vue-router'
import { mount } from '@vue/test-utils'
import { useServerList } from '../useServerList'

// The composable relies on Nuxt auto-imports; expose them as globals here.
Object.assign(globalThis, vue, { useRoute, useRouter })

const settle = async () => {
  for (let i = 0; i < 5; i++) {
    await vue.nextTick()
    await new Promise((resolve) => setTimeout(resolve, 5))
  }
}

const setup = async () => {
  const fetch = vi.fn()
  const activeTab = ref('all')
  let api: ReturnType<typeof useServerList<{ search: string; status: string }>>

  const Page = defineComponent({
    setup() {
      api = useServerList({
        filters: { search: '', status: '' },
        debounce: ['search'],
        deps: [activeTab],
        fetch: (page, perPage, applied) => fetch(page, perPage, { ...applied }, activeTab.value),
      })
      return () => h('div')
    },
  })

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:pathMatch(.*)*', component: Page }],
  })
  router.push('/deposits')
  await router.isReady()

  mount(Page, { global: { plugins: [router] } })
  await settle()

  return { fetch, activeTab, router, api: api! }
}

describe('useServerList', () => {
  it('fetches once on mount', async () => {
    const { fetch } = await setup()
    expect(fetch).toHaveBeenCalledTimes(1)
    expect(fetch).toHaveBeenLastCalledWith(1, 10, { search: '', status: '' }, 'all')
  })

  it('fetches once per filter change and writes it to the URL', async () => {
    const { fetch, api, router } = await setup()
    api.filters.status = 'OPEN'
    await settle()

    expect(fetch).toHaveBeenCalledTimes(2)
    expect(fetch).toHaveBeenLastCalledWith(1, 10, { search: '', status: 'OPEN' }, 'all')
    expect(router.currentRoute.value.query.status).toBe('OPEN')
  })

  it('coalesces a dep change + reset into a single request', async () => {
    const { fetch, activeTab, api, router } = await setup()
    api.filters.status = 'OPEN'
    await settle()
    expect(fetch).toHaveBeenCalledTimes(2)

    // What the page does on tab change.
    activeTab.value = 'assigned-to-me'
    api.reset()
    await settle()

    expect(fetch).toHaveBeenCalledTimes(3)
    expect(fetch).toHaveBeenLastCalledWith(1, 10, { search: '', status: '' }, 'assigned-to-me')
    expect(router.currentRoute.value.query.status).toBeUndefined()
    expect(api.filters.status).toBe('')
  })

  it('debounces free-text filters into one request', async () => {
    const { fetch, api } = await setup()
    api.filters.search = 'a'
    await vue.nextTick()
    api.filters.search = 'ab'
    await vue.nextTick()
    api.filters.search = 'abc'
    await new Promise((resolve) => setTimeout(resolve, 700))
    await settle()

    expect(fetch).toHaveBeenCalledTimes(2)
    expect(fetch).toHaveBeenLastCalledWith(1, 10, { search: 'abc', status: '' }, 'all')
  })

  it('refetches and syncs the inputs when the URL changes underneath it', async () => {
    const { fetch, api, router } = await setup()

    // A navigation the page did not initiate — a shared link, or back/forward.
    router.push('/deposits?page=1&status=OPEN')
    await settle()

    expect(fetch).toHaveBeenCalledTimes(2)
    expect(fetch).toHaveBeenLastCalledWith(1, 10, { search: '', status: 'OPEN' }, 'all')
    expect(api.filters.status).toBe('OPEN')

    router.back()
    await settle()

    expect(fetch).toHaveBeenCalledTimes(3)
    expect(fetch).toHaveBeenLastCalledWith(1, 10, { search: '', status: '' }, 'all')
    expect(api.filters.status).toBe('')
  })
})
