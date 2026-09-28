import { useFlashStore } from '~/stores/flash'
import type { SearchHistoryEntry, SearchResult, SearchSuggestion } from '~/types/entities/search'
import type {
  ResponseData,
  ResponseDataCollectionWithoutPagination,
  ServerPaginationMeta,
} from '~/types/response'

interface SearchListResponse extends ResponseData {
  payload: {
    data: SearchResult[]
    meta: ServerPaginationMeta
  }
}

export interface SearchFilters {
  q?: string
  space_id?: string
  category_id?: string
  tag?: string
  author_id?: string
  from?: string
  to?: string
  sort?: string
}

// Empty filters are dropped rather than sent blank: the API reads a present
// parameter as a filter, and `?author_id=` would be a filter on nobody.
const buildQuery = (page: number, perPage: number, filters: SearchFilters) => {
  const query: Record<string, string | number> = { page, per_page: perPage }
  for (const [key, value] of Object.entries(filters)) {
    if (value) query[key] = value
  }
  return query
}

const defaultMeta: ServerPaginationMeta = {
  current_page: 1,
  per_page: 10,
  total: 0,
  last_page: 1,
  from: 0,
  to: 0,
}

const failure = (message: string) => ({ code: 'ERROR', message, status_code: 500, timestamp: '' })

export const useSearchService = () => {
  const { $api } = useNuxtApp()
  const flashStore = useFlashStore()

  const results = ref<SearchResult[]>([])
  const meta = ref<ServerPaginationMeta>({ ...defaultMeta })
  const loading = ref(false)

  const suggestions = ref<SearchSuggestion[]>([])
  const history = ref<SearchHistoryEntry[]>([])

  const getAll = async (page = 1, perPage = 10, filters: SearchFilters = {}) => {
    loading.value = true
    try {
      const response = await $api<SearchListResponse>('api/search', {
        method: 'GET',
        query: buildQuery(page, perPage, filters),
      })
      results.value = response.payload.data
      meta.value = response.payload.meta
    } catch (e: any) {
      flashStore.setFailed(e?.data ?? failure('Failed to search'))
      results.value = []
      meta.value = { ...defaultMeta }
    } finally {
      loading.value = false
    }
  }

  /**
   * Autocomplete. Failures are swallowed: this fires on a keystroke, and a
   * flash message per character typed is worse than a dropdown that stays
   * empty for a moment.
   */
  const suggest = async (q: string) => {
    if (!q.trim()) {
      suggestions.value = []
      return
    }
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<SearchSuggestion>>(
        'api/search/suggest',
        { method: 'GET', query: { q } },
      )
      suggestions.value = response.payload.data
    } catch {
      suggestions.value = []
    }
  }

  /** Recent searches — same reasoning as `suggest` for the silent failure. */
  const getHistory = async () => {
    try {
      const response = await $api<ResponseDataCollectionWithoutPagination<SearchHistoryEntry>>(
        'api/search/history',
        { method: 'GET' },
      )
      history.value = response.payload.data
    } catch {
      history.value = []
    }
  }

  const deleteHistory = async (id: string) => {
    // Removed here as well as on the server: the list is already on screen and
    // refetching it to drop one row is a round trip the user watches happen.
    history.value = history.value.filter((entry) => entry.id !== id)
    try {
      await $api<ResponseData>(`api/search/history/${id}`, { method: 'DELETE' })
    } catch {
      await getHistory()
    }
  }

  return { results, meta, loading, suggestions, history, getAll, suggest, getHistory, deleteHistory }
}
