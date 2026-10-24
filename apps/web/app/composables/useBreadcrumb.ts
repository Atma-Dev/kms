import { computed } from 'vue'
import { useRoute } from 'vue-router'

/** Breadcrumb trail from route meta: `breadcrumb: ['Master', 'Regions']`, falling back to `title`. */
export const useBreadcrumb = () => {
  const route = useRoute()

  return computed<string[]>(() => {
    if (Array.isArray(route.meta.breadcrumb)) return route.meta.breadcrumb as string[]
    return route.meta.title ? [route.meta.title as string] : []
  })
}
