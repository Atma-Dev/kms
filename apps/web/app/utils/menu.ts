import type { Permission } from '~/enums/Permission'

interface MenuLike {
  permission?: Permission
  sub_menu?: MenuLike[]
}

/**
 * Drops menu entries the user's permission doesn't cover, and any group
 * label ("nav.groups.x" — a bare string entry) left with nothing under it.
 */
export const filterMenuByPermission = <T extends MenuLike>(
  menu: (string | T)[],
  can: (permission: Permission) => boolean,
): (string | T)[] => {
  const filterItem = (item: T): T | null => {
    if (item.permission && !can(item.permission)) return null
    if (item.sub_menu) {
      const subMenu = (item.sub_menu as T[]).map(filterItem).filter((m): m is T => m !== null)
      if (subMenu.length === 0) return null
      return { ...item, sub_menu: subMenu }
    }
    return item
  }

  const result: (string | T)[] = []
  let pendingGroup: string | null = null

  for (const entry of menu) {
    if (typeof entry === 'string') {
      pendingGroup = entry
      continue
    }
    const filtered = filterItem(entry)
    if (!filtered) continue
    if (pendingGroup !== null) {
      result.push(pendingGroup)
      pendingGroup = null
    }
    result.push(filtered)
  }
  return result
}
